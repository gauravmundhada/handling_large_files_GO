-- migrations/001_init.sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE employees (
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

CREATE TABLE upload_jobs (
    job_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    file_path text NOT NULL,
    status text NOT NULL,
    retries int DEFAULT 0,
    invalid_file_path text,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now()
);
