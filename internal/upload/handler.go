package upload

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"handling_large_files_go/internal/db"
	"handling_large_files_go/internal/queue"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// UploadHandler accepts multipart file and enqueues a job
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30) // 1GB limit
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".xlsx" && ext != ".xls" {
		http.Error(w, "only Excel files (.xlsx, .xls) are allowed", http.StatusBadRequest)
		return
	}

	jobID := uuid.New().String()
	tmpDir := os.TempDir()
	filePath := filepath.Join(tmpDir, fmt.Sprintf("upload-%s-%s", jobID, filepath.Base(header.Filename)))
	out, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "cannot create temp file", http.StatusInternalServerError)
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}

	// insert job record
	if db.DB == nil {
		if err := db.Connect(); err != nil {
			log.Println("DB connect error:", err)
		}
	}
	q := `INSERT INTO upload_jobs (job_id, file_path, status, created_at, updated_at) VALUES ($1,$2,'pending',now(),now())`
	if _, err := db.DB.Exec(q, jobID, filePath); err != nil {
		log.Println("failed to insert job:", err)
		http.Error(w, "failed to create job", http.StatusInternalServerError)
		return
	}

	// publish to RabbitMQ (non-blocking)
	go func(j string, p string) {
		if err := queue.PublishJob(j, p); err != nil {
			log.Println("publish job err:", err)
		} else {
			log.Printf("enqueued job %s -> %s\n", j, p)
		}
	}(jobID, filePath)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"job_id": jobID})
}

// StatusHandler returns status and if invalid file exists, streams it as attachment
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["job_id"]
	if jobID == "" {
		http.Error(w, "job_id required", http.StatusBadRequest)
		return
	}
	if db.DB == nil {
		if err := db.Connect(); err != nil {
			log.Println("DB connect error:", err)
		}
	}
	var status string
	var invalidPath sql.NullString
	row := db.DB.QueryRow("SELECT status, invalid_file_path FROM upload_jobs WHERE job_id = $1", jobID)
	if err := row.Scan(&status, &invalidPath); err != nil {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	if status == "processing" || status == "pending" {
		h := map[string]string{"job_id": jobID, "status": status}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(h)
		return
	}
	// if completed and invalid file exists, stream as attachment
	if invalidPath.Valid && invalidPath.String != "" && status == "success" {
		f, err := os.Open(invalidPath.String)
		if err != nil {
			http.Error(w, "failed to open invalid file", http.StatusInternalServerError)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", "attachment; filename=invalid_rows.xlsx")
		io.Copy(w, f)
		return
	}
	// otherwise return status
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"job_id": jobID, "status": status})
}
