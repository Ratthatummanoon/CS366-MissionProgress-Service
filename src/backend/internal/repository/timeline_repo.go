package repository

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/models"
)

// TimelineRepo handles DynamoDB operations for MissionTimeline.
type TimelineRepo struct {
	client    *dynamodb.Client
	tableName string
}

// NewTimelineRepo creates a new TimelineRepo.
func NewTimelineRepo(client *dynamodb.Client, tableName string) *TimelineRepo {
	return &TimelineRepo{client: client, tableName: tableName}
}

// AddTimelineEntry inserts a new timeline entry.
func (r *TimelineRepo) AddTimelineEntry(ctx context.Context, entry *models.TimelineEntry) error {
	item, err := attributevalue.MarshalMap(entry)
	if err != nil {
		return fmt.Errorf("marshal timeline entry: %w", err)
	}
	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put timeline entry: %w", err)
	}
	return nil
}

// GetTimelineByMissionID queries timeline entries sorted by timestamp.
func (r *TimelineRepo) GetTimelineByMissionID(ctx context.Context, missionID string) ([]models.TimelineEntry, error) {
	output, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("mission_id = :mid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":mid": &types.AttributeValueMemberS{Value: missionID},
		},
		ScanIndexForward: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("query timeline: %w", err)
	}

	var entries []models.TimelineEntry
	if err := attributevalue.UnmarshalListOfMaps(output.Items, &entries); err != nil {
		return nil, fmt.Errorf("unmarshal timeline entries: %w", err)
	}
	return entries, nil
}
