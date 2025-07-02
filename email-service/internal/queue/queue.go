package queue

import "context"

const (
	// TaskCreatedQueue is the queue for newly created batch tasks that need dispatching.
	TaskCreatedQueue = "tasks:created"
	// EmailSendingQueue is the queue for individual emails ready to be sent.
	EmailSendingQueue = "email:sending"
)

// QueueService defines the interface for a message queue.
// This allows for different implementations (e.g., Redis, RabbitMQ, Kafka).
type QueueService interface {
	// Enqueue adds a message to the specified queue.
	Enqueue(ctx context.Context, queueName string, message string) error

	// Dequeue removes and returns a message from the specified queue.
	// It should block until a message is available.
	Dequeue(ctx context.Context, queueName string) (string, error)
}
