package models

import "time"

// Employee represents a row in employees table
type Employee struct {
	ID            string     `db:"id" json:"id"`
	Name          string     `db:"name" json:"name"`
	Email         string     `db:"email" json:"email"`
	Department    string     `db:"department" json:"department"`
	Grade         string     `db:"grade" json:"grade"`
	DateOfJoining *time.Time `db:"date_of_joining" json:"date_of_joining"`
	DateOfLeaving *time.Time `db:"date_of_leaving" json:"date_of_leaving"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at" json:"updated_at"`
}

// UploadJob tracks a file processing job
type UploadJob struct {
	JobID       string    `db:"job_id" json:"job_id"`
	FilePath    string    `db:"file_path" json:"file_path"`
	Status      string    `db:"status" json:"status"`
	Retries     int       `db:"retries" json:"retries"`
	InvalidPath string    `db:"invalid_file_path" json:"invalid_file_path"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}
