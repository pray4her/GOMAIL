package queue

import "context"

const (
	// TaskCreatedQueue is the queue for newly created batch tasks that need dispatching.
	TaskCreatedQueue = "tasks:created"
	// EmailSendingQueue is the queue for individual emails ready to be sent.
	EmailSendingQueue = "email:sending"
	// TaskScheduledQueue is the sorted set for tasks scheduled for future delivery.
	TaskScheduledQueue = "tasks:scheduled"
	// RecipientSyncQueue is the queue for recipient create/update events to be synced to Elasticsearch.
	RecipientSyncQueue = "recipients:sync"
	// RecipientImportQueue is the queue for recipient import tasks to be processed.
	RecipientImportQueue = "recipients:import"
)

// QueueService defines the interface for a message queue.
// This allows for different implementations (e.g., Redis, RabbitMQ, Kafka).
type QueueService interface {
	// Enqueue adds a message to the specified queue.
	Enqueue(ctx context.Context, queueName string, message string) error

	// Dequeue removes and returns a message from the specified queue.
	// It should block until a message is available.
	Dequeue(ctx context.Context, queueName string) (string, error)

	// EnqueueScheduled adds a message to a sorted set, scored by its execution time.
	EnqueueScheduled(ctx context.Context, queueName string, message string, score float64) error

	// DequeueDue retrieves all messages from a sorted set whose score is up to the given max score.
	DequeueDue(ctx context.Context, queueName string, maxScore float64) ([]string, error)
}
