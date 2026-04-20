package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/models"
)

// OutboxRepo handles DynamoDB operations for EventOutbox.
type OutboxRepo struct {
	client    *dynamodb.Client
	tableName string
}

// NewOutboxRepo creates a new OutboxRepo.
func NewOutboxRepo(client *dynamodb.Client, tableName string) *OutboxRepo {
	return &OutboxRepo{client: client, tableName: tableName}
}

// SaveOutboxEntry saves a pending event to the outbox table.
func (r *OutboxRepo) SaveOutboxEntry(ctx context.Context, entry *models.OutboxEntry) error {
	item, err := attributevalue.MarshalMap(entry)
	if err != nil {
		return fmt.Errorf("marshal outbox entry: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put outbox entry: %w", err)
	}
	return nil
}

// GetPendingOutboxEntries queries the status-index GSI for PENDING entries.
func (r *OutboxRepo) GetPendingOutboxEntries(ctx context.Context) ([]models.OutboxEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	output, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("status-index"),
		KeyConditionExpression: aws.String("#s = :status"),
		ExpressionAttributeNames: map[string]string{
			"#s": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: "PENDING"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("query pending outbox entries: %w", err)
	}

	var entries []models.OutboxEntry
	if err := attributevalue.UnmarshalListOfMaps(output.Items, &entries); err != nil {
		return nil, fmt.Errorf("unmarshal outbox entries: %w", err)
	}
	return entries, nil
}

// UpdateOutboxEntryStatus updates the status and retry_count of an outbox entry.
func (r *OutboxRepo) UpdateOutboxEntryStatus(ctx context.Context, outboxID string, status string, retryCount int, lastError string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"outbox_id": &types.AttributeValueMemberS{Value: outboxID},
		},
		UpdateExpression: aws.String("SET #s = :status, retry_count = :rc, last_error = :le"),
		ExpressionAttributeNames: map[string]string{
			"#s": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: status},
			":rc":     &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", retryCount)},
			":le":     &types.AttributeValueMemberS{Value: lastError},
		},
	})
	if err != nil {
		return fmt.Errorf("update outbox entry status: %w", err)
	}
	return nil
}
