package worker

import (
	"fmt"
	"log"
	"os"
	"time"

	"handling_large_files_go/internal/db"
	"handling_large_files_go/internal/parse"
	"handling_large_files_go/internal/queue"
)

// StartWorkers starts N consumer goroutines that pull jobs from RabbitMQ
func StartWorkers() {
	workers := 3
	for i := 0; i < workers; i++ {
		id := i
		go func() {
			consumer := fmt.Sprintf("worker-%d", id)
			log.Printf("starting rabbit consumer %s", consumer)
			err := queue.ConsumeJobs(consumer, func(jobID, filePath string) error {
				log.Printf("consumer %s got job %s -> %s", consumer, jobID, filePath)
				// ensure DB connected
				if db.DB == nil {
					if err := db.Connect(); err != nil {
						log.Println("DB connect error:", err)
						return err
					}
				}
				// attempt to mark processing only if pending
				if err := markProcessing(jobID); err != nil {
					log.Printf("mark processing skipped/err for job %s: %v", jobID, err)
					// ack message by returning nil so it is removed from queue
					return nil
				}

				// timer to check processing time
				start := time.Now()

				// process file
				invalidPath, err := processFile(filePath)
				if err != nil {
					log.Println("processing error:", err)
					if err := incrementRetry(jobID); err != nil {
						log.Println("increment retry err:", err)
					}
					return err
				}

				elapsed := time.Since(start)
				log.Printf("consumer %s processed job %s in %s", consumer, jobID, elapsed)

				// mark success and set invalid file path if any
				if err := markSuccess(jobID, invalidPath); err != nil {
					log.Println("mark success err:", err)
				}
				// cleanup temp file
				_ = os.Remove(filePath)
				return nil
			})
			if err != nil {
				log.Printf("consumer %s exited: %v", consumer, err)
				time.Sleep(5 * time.Second)
				// retry connecting
				go func() { StartWorkers() }()
				return
			}
		}()
	}
	// keep process alive
	select {}
}

func markProcessing(jobID string) error {
	res, err := db.DB.Exec("UPDATE upload_jobs SET status='processing', updated_at=now() WHERE job_id=$1 AND status='pending'", jobID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("job not pending")
	}
	return nil
}

func incrementRetry(jobID string) error {
	res, err := db.DB.Exec("UPDATE upload_jobs SET retries = retries + 1, updated_at = now() WHERE job_id=$1 RETURNING retries", jobID)
	if err != nil {
		return err
	}
	_ = res
	// check retries and move to failed if >3
	var retries int
	row := db.DB.QueryRow("SELECT retries FROM upload_jobs WHERE job_id=$1", jobID)
	if err := row.Scan(&retries); err == nil {
		if retries > 3 {
			_, _ = db.DB.Exec("UPDATE upload_jobs SET status='failed', updated_at=now() WHERE job_id=$1", jobID)
		}
	}
	return nil
}

func markSuccess(jobID, invalidPath string) error {
	_, err := db.DB.Exec("UPDATE upload_jobs SET status='success', invalid_file_path=$2, updated_at=now() WHERE job_id=$1", jobID, invalidPath)
	return err
}

func processFile(path string) (invalidPath string, err error) {
	// stream parse and batch upsert
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// parse returns invalidRows file path if any
	invalid, err := parse.ProcessXLSX(path)
	if err != nil {
		return invalid, err
	}
	return invalid, nil
}
