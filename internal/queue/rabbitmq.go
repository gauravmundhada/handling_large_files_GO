package queue

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var conn *amqp.Connection
var ch *amqp.Channel

// InitRabbitMQ creates a connection and channel to RabbitMQ with basic retries and clearer errors.
func InitRabbitMQ() error {
	urlStr := os.Getenv("RABBITMQ_URL")
	if urlStr == "" {
		return fmt.Errorf("RABBITMQ_URL not set")
	}

	u, err := url.Parse(urlStr)
	if err == nil {
		// Log scheme and host only to avoid leaking credentials
		log.Printf("connecting to RabbitMQ %s://%s", u.Scheme, u.Host)
	}

	// retry dial a few times with exponential backoff
	var dialErr error
	for i := 0; i < 3; i++ {
		conn, dialErr = amqp.Dial(urlStr)
		if dialErr == nil {
			break
		}
		log.Printf("rabbitmq dial attempt %d failed: %v", i+1, dialErr)
		time.Sleep(time.Duration(1<<i) * time.Second)
	}
	if dialErr != nil {
		return fmt.Errorf("failed to connect to rabbitmq: %w", dialErr)
	}

	ch, err = conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	_, err = ch.QueueDeclare("upload_jobs", true, false, false, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("queue declare failed: %w", err)
	}
	return nil
}

func PublishJob(jobID, filePath string) error {
	body := jobID + "|" + filePath

	if ch == nil {
		if err := InitRabbitMQ(); err != nil {
			return err
		}
	}

	if ch == nil {
		return fmt.Errorf("rabbitmq channel not initialized")
	}

	if err := ch.Publish("", "upload_jobs", false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(body),
	}); err != nil {
		return fmt.Errorf("publish failed: %w", err)
	}
	return nil
}

// ConsumeJobs connects to RabbitMQ and delivers jobs to handler. This function blocks until the connection is closed or an error occurs.
// handler should return nil on success; on error the caller (worker) is expected to record retries and decide further action.
func ConsumeJobs(consumer string, handler func(jobID, filePath string) error) error {
	urlStr := os.Getenv("RABBITMQ_URL")
	if urlStr == "" {
		log.Println("RABBITMQ_URL not set, skipping consume")
		return nil
	}
	conn, err := amqp.Dial(urlStr)
	if err != nil {
		return fmt.Errorf("failed to dial rabbitmq: %w", err)
	}
	// keep connection open until error
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}
	// QoS: prefetch 1 per consumer for fair dispatch
	if err := ch.Qos(1, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to set qos: %w", err)
	}
	q, err := ch.QueueDeclare("upload_jobs", true, false, false, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("queue declare failed: %w", err)
	}
	msgs, err := ch.Consume(q.Name, consumer, false, false, false, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("consume failed: %w", err)
	}
	for d := range msgs {
		body := string(d.Body)
		parts := strings.SplitN(body, "|", 2)
		if len(parts) != 2 {
			log.Println("invalid message body, acking")
			d.Ack(false)
			continue
		}
		jobID := parts[0]
		filePath := parts[1]
		if err := handler(jobID, filePath); err != nil {
			// handler records retries/failures in DB; ack to remove from queue
			d.Ack(false)
			continue
		}
		// success
		d.Ack(false)
	}
	// cleanup
	ch.Close()
	conn.Close()
	return nil
}
