package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var DB *sql.DB

func Connect() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL not set")
	}
	var err error
	DB, err = sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	DB.SetMaxOpenConns(20)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(30 * time.Minute)

	if err := DB.Ping(); err != nil {
		return err
	}
	log.Println("connected to database")
	return nil
}

// EnsureMigration is a lightweight helper to create tables if not exists
func EnsureMigration() error {
	q := `
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS employees (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    email text NOT NULL,
    department text,
    grade text,
    date_of_joining date,
    date_of_leaving date,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    UNIQUE(name,email)
);

CREATE TABLE IF NOT EXISTS upload_jobs (
    job_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    file_path text NOT NULL,
    status text NOT NULL,
    retries int DEFAULT 0,
    invalid_file_path text,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now()
);
`
	_, err := DB.Exec(q)
	return err
}
