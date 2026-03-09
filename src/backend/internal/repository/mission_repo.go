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

// MissionRepo handles DynamoDB operations for MissionAssignment.
type MissionRepo struct {
	client    *dynamodb.Client
	tableName string
}

// NewMissionRepo creates a new MissionRepo.
func NewMissionRepo(client *dynamodb.Client, tableName string) *MissionRepo {
	return &MissionRepo{client: client, tableName: tableName}
}

// GetMissionByIncidentID queries the incident-index GSI for a mission.
func (r *MissionRepo) GetMissionByIncidentID(ctx context.Context, incidentID string) (*models.MissionAssignment, error) {
	output, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("incident-index"),
		KeyConditionExpression: aws.String("incident_id = :iid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":iid": &types.AttributeValueMemberS{Value: incidentID},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("query mission by incident_id: %w", err)
	}
	if len(output.Items) == 0 {
		return nil, nil
	}

	var mission models.MissionAssignment
	if err := attributevalue.UnmarshalMap(output.Items[0], &mission); err != nil {
		return nil, fmt.Errorf("unmarshal mission: %w", err)
	}
	return &mission, nil
}

// UpdateMissionStatus updates the current_status, latest_impact_level, and last_updated_at.
func (r *MissionRepo) UpdateMissionStatus(ctx context.Context, mission *models.MissionAssignment) error {
	_, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"mission_id": &types.AttributeValueMemberS{Value: mission.MissionID},
		},
		UpdateExpression: aws.String("SET current_status = :s, latest_impact_level = :il, last_updated_at = :u"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":s":  &types.AttributeValueMemberS{Value: mission.CurrentStatus},
			":il": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", mission.LatestImpactLevel)},
			":u":  &types.AttributeValueMemberS{Value: mission.LastUpdatedAt},
		},
	})
	if err != nil {
		return fmt.Errorf("update mission status: %w", err)
	}
	return nil
}

// CreateMission inserts a new mission assignment.
func (r *MissionRepo) CreateMission(ctx context.Context, mission *models.MissionAssignment) error {
	item, err := attributevalue.MarshalMap(mission)
	if err != nil {
		return fmt.Errorf("marshal mission: %w", err)
	}
	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put mission: %w", err)
	}
	return nil
}
