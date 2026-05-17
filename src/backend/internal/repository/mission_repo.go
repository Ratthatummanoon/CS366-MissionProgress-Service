package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/models"
)

// ErrConditionalCheckFailed is returned when an optimistic-lock condition fails,
// meaning another request already changed the mission status concurrently.
var ErrConditionalCheckFailed = errors.New("conditional check failed: mission status changed by concurrent request")

// MissionRepo handles DynamoDB operations for MissionAssignment.
type MissionRepo struct {
	client    *dynamodb.Client
	tableName string
}

// NewMissionRepo creates a new MissionRepo.
func NewMissionRepo(client *dynamodb.Client, tableName string) *MissionRepo {
	return &MissionRepo{client: client, tableName: tableName}
}

// UpdateMissionStatus updates the current_status, latest_impact_level, and last_updated_at.
// expectedStatus is used as an optimistic-lock condition: the update only succeeds if
// current_status in DynamoDB still equals expectedStatus at write time.
// Returns ErrConditionalCheckFailed if a concurrent request already changed the status.
func (r *MissionRepo) UpdateMissionStatus(ctx context.Context, mission *models.MissionAssignment, expectedStatus string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"mission_id": &types.AttributeValueMemberS{Value: mission.MissionID},
		},
		UpdateExpression:    aws.String("SET current_status = :s, latest_impact_level = :il, last_updated_at = :u"),
		ConditionExpression: aws.String("current_status = :expected"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":s":        &types.AttributeValueMemberS{Value: mission.CurrentStatus},
			":il":       &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", mission.LatestImpactLevel)},
			":u":        &types.AttributeValueMemberS{Value: mission.LastUpdatedAt},
			":expected": &types.AttributeValueMemberS{Value: expectedStatus},
		},
	})
	if err != nil {
		var condErr *types.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			return ErrConditionalCheckFailed
		}
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
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put mission: %w", err)
	}
	return nil
}

// GetMissionByRequestID queries the request-index GSI for a mission.
func (r *MissionRepo) GetMissionByRequestID(ctx context.Context, requestID string) (*models.MissionAssignment, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	output, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("request-index"),
		KeyConditionExpression: aws.String("request_id = :rid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":rid": &types.AttributeValueMemberS{Value: requestID},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("query mission by request_id: %w", err)
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

// GetMissionByRequestIDAndTeamID queries the request-index GSI and filters by rescue_team_id.
// Used by the report-progress handler to enforce team ownership: a team can only update
// its own mission. Supports multiple missions per request_id (e.g. backup team scenarios)
// by scanning all matching request_id rows and filtering for the calling team.
func (r *MissionRepo) GetMissionByRequestIDAndTeamID(ctx context.Context, requestID, teamID string) (*models.MissionAssignment, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	output, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("request-index"),
		KeyConditionExpression: aws.String("request_id = :rid"),
		FilterExpression:       aws.String("rescue_team_id = :tid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":rid": &types.AttributeValueMemberS{Value: requestID},
			":tid": &types.AttributeValueMemberS{Value: teamID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("query mission by request_id and team_id: %w", err)
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

// GetMissionByDispatchID queries the dispatch-index GSI for a mission by dispatch_id.
// Used for idempotency check in mission-assigned-handler.
func (r *MissionRepo) GetMissionByDispatchID(ctx context.Context, dispatchID string) (*models.MissionAssignment, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	output, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("dispatch-index"),
		KeyConditionExpression: aws.String("dispatch_id = :did"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":did": &types.AttributeValueMemberS{Value: dispatchID},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("query mission by dispatch_id: %w", err)
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

// GetMissionsByTeamID queries the team-index GSI for all missions of a team.
func (r *MissionRepo) GetMissionsByTeamID(ctx context.Context, teamID string, statusFilter string) ([]models.MissionAssignment, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("team-index"),
		KeyConditionExpression: aws.String("rescue_team_id = :tid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":tid": &types.AttributeValueMemberS{Value: teamID},
		},
	}

	if statusFilter != "" {
		input.FilterExpression = aws.String("current_status = :st")
		input.ExpressionAttributeValues[":st"] = &types.AttributeValueMemberS{Value: statusFilter}
	}

	output, err := r.client.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("query missions by team_id: %w", err)
	}

	var missions []models.MissionAssignment
	if err := attributevalue.UnmarshalListOfMaps(output.Items, &missions); err != nil {
		return nil, fmt.Errorf("unmarshal missions: %w", err)
	}
	return missions, nil
}
