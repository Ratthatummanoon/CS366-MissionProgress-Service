package models

// OutboxEntry represents a pending event in the outbox table.
type OutboxEntry struct {
	OutboxID     string `json:"outbox_id" dynamodbav:"outbox_id"`
	CreatedAt    string `json:"created_at" dynamodbav:"created_at"`
	EventType    string `json:"event_type" dynamodbav:"event_type"`
	EventPayload string `json:"event_payload" dynamodbav:"event_payload"`
	Status       string `json:"status" dynamodbav:"status"`
	RetryCount   int    `json:"retry_count" dynamodbav:"retry_count"`
	LastError    string `json:"last_error,omitempty" dynamodbav:"last_error,omitempty"`
	TTL          int64  `json:"ttl,omitempty" dynamodbav:"ttl,omitempty"`
}
