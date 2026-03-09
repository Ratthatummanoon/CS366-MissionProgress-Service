package repository

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

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
	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put outbox entry: %w", err)
	}
	return nil
}
