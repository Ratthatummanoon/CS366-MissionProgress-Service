package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/models"
	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/repository"
	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/response"
	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/statemachine"
)

var (
	missionRepo *repository.MissionRepo
)

func init() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("unable to load AWS config: %v", err)
	}

	ddbClient := dynamodb.NewFromConfig(cfg)
	tableMission := os.Getenv("TABLE_MISSION")

	missionRepo = repository.NewMissionRepo(ddbClient, tableMission)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// 1. Parse X-Rescue-Team-ID from authorizer context
	rescueTeamID := ""
	if authCtx, ok := request.RequestContext.Authorizer["principalId"]; ok {
		if v, ok := authCtx.(string); ok {
			rescueTeamID = v
		}
	}
	if rescueTeamID == "" {
		rescueTeamID = request.Headers["X-Rescue-Team-ID"]
	}
	if rescueTeamID == "" {
		rescueTeamID = request.Headers["x-rescue-team-id"]
	}
	if rescueTeamID == "" {
		return response.Error(400, "MISSING_PARAMETER", "X-Rescue-Team-ID is required"), nil
	}

	// 2. Parse optional status filter from query params
	statusFilter := request.QueryStringParameters["status"]
	if statusFilter != "" && !statemachine.ValidateStatus(statusFilter) {
		return response.Error(400, "INVALID_STATUS", "Invalid status filter: "+statusFilter), nil
	}

	// 3. Query missions by team ID
	missions, err := missionRepo.GetMissionsByTeamID(ctx, rescueTeamID, statusFilter)
	if err != nil {
		log.Printf("ERROR: query missions: %v", err)
		return response.Error(500, "INTERNAL_ERROR", "Failed to query missions"), nil
	}
	if missions == nil {
		missions = []models.MissionAssignment{}
	}

	// 4. Return response (always 200, empty array if no missions)
	return response.JSON(200, models.ListMissionsResponse{
		TeamID:        rescueTeamID,
		TotalMissions: len(missions),
		Missions:      missions,
	}), nil
}

func main() {
	lambda.Start(handler)
}
