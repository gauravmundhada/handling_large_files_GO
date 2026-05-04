● handling_large_files_go

  Lightweight Go service to accept large Excel uploads, enqueue background processing jobs via RabbitMQ and process them with worker consumers that update a Postgres DB.

  Features

   - HTTP multipart upload endpoint (/upload)
   - Enqueues jobs to RabbitMQ (queue: upload_jobs)
   - Worker consumers process files, update job status, and write invalid-row spreadsheets
   - Uses local temp storage for uploads

  Requirements

   - Go
    1.20+
   - PostgreSQL
   - RabbitMQ (AMQP on port 5672; management UI on 15672)

  Environment

  Set these before running (replace placeholders and URL-encode special chars):

   - DATABASE_URL
   export DATABASE_URL="postgres://DB_USER:DB_PASS@DB_HOST:5432/DB_NAME"
   - RABBITMQ_URL
   export RABBITMQ_URL="amqp://RMQ_USER:RMQ_PASS@RMQ_HOST:5672/VHOST"
    - Root vhost: use %2f for / (example: /%2f)
    - Use amqps:// if TLS is required

  Quickstart (local)

   1. Ensure Postgres and RabbitMQ are running and reachable.
   2. From project root:
   export DATABASE_URL="postgres://user:pass@localhost:5432/mydb"
   export RABBITMQ_URL="amqp://user:pass@localhost:5672/%2f"
   go run main.go
   3. Upload a sample file:
   curl -F "file=@test_employees.xlsx" http://localhost:8080/upload (http://localhost:8080/upload)

  Logs will show enqueue/publish attempts. The API responds immediately with a job_id.

  Running workers

  Workers consume from the upload_jobs queue. Start the worker binary/process (see cmd/main.go or worker runner) so consumers run in the same process or separately depending on
  your deployment.

  Building

  go build ./cmd/server

  Troubleshooting

  Common publish/connect issues:

   - EOF / Exception (501) typically means the app connected to the management HTTP port (15672) instead of AMQP (5672). Verify:
    - nc -vz rabbit.host 5672
    - curl -I http://rabbit.host:15672/ (http://rabbit.host:15672/)  (management UI)
    - Ensure RABBITMQ_URL uses port 5672 and correct scheme (amqp:// or amqps://)
   - Use management API to confirm vhosts and permissions:
   curl -u user:pass http://localhost:15672/api/aliveness-test/%2f (http://localhost:15672/api/aliveness-test/%2f)
   curl -u user:pass http://localhost:15672/api/permissions/%2f/ (http://localhost:15672/api/permissions/%2f/)<user>



  Architecture

  See architecture.md for component diagrams, data flow, and TODOs.

