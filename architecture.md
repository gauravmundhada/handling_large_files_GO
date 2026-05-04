Project: handling_large_files_go

Overview

This service accepts large Excel uploads via an HTTP API, enqueues background processing jobs to RabbitMQ, and processes files in worker processes which save results to Postgres. The design separates quick HTTP response (enqueue + persist job) from CPU-/IO-heavy work (workers). Temporary files are stored on local disk (system temp dir) during processing.

Components

- HTTP API (cmd/main.go, internal/upload): handles multipart uploads, validates file types, writes temp file, creates DB job record, and publishes a job message to RabbitMQ.
- Queue (internal/queue): RabbitMQ client encapsulation. Declares queue "upload_jobs", publishes job messages, and contains consumer helper.
- Workers (internal/worker): consumer goroutines that call ConsumeJobs and invoke handler logic (processing/parsing Excel, DB updates).
- Database (internal/db): Postgres used for job metadata, job status, retry counts, and storing paths to invalid-files.
- Storage: local filesystem for temporary files and invalid-row spreadsheets.

Data flow (simplified)

1. Client -> POST /upload (multipart/form-data)
2. Server: save temp file, INSERT upload_jobs (status=pending)
3. Server: PublishJob(jobID|filePath) -> RabbitMQ queue "upload_jobs"
4. Worker consumer: receive message -> process file -> update DB (processing->success|failed) -> write invalid file if needed

Environment variables

- DATABASE_URL: postgres://USER:PASS@HOST:5432/DB_NAME
- RABBITMQ_URL: amqp://USER:PASS@HOST:5672/VHOST (use amqps:// for TLS; URL-encode "/" as %2f)
- Other: standard process envs for port, log level (check main.go)

Operational notes

- Ensure RABBITMQ_URL points to AMQP port (5672); management UI runs on 15672.
- Worker scale: spawn N consumers (configurable). Each consumer uses prefetch=1 for fair dispatch.
- Durable queue and persistent messages: queue is declared durable; messages are published non-persistent in code (check amqp.Publishing). If durability required, set DeliveryMode.
- Restart behavior: InitRabbitMQ includes retries; ensure service restarts pick up updated env variables.

Failure modes & troubleshooting

- EOF / 501: often caused by connecting to HTTP management port (15672) instead of AMQP (5672) or wrong scheme (amqp vs amqps). Verify using nc and management API /api/aliveness-test.
- Permission issues: check RabbitMQ vhosts/users/permissions via management API.
- Disk space: monitor /tmp for large uploads.

Observability

- Application logs include connection attempts and publish errors. Add metrics (Prometheus) around queue length, job processing time, and failure rates.

Security

- Never log full RABBITMQ_URL or DATABASE_URL. Log only host and scheme.
- Use TLS (amqps) and secure DB connections in production.

TODO

- Make message publishing persistent (DeliveryMode=2) if required.
- Add graceful shutdown to close RabbitMQ connections and drain consumers.
- Add configuration via config file or env parsing library and improve retry/backoff behavior.
