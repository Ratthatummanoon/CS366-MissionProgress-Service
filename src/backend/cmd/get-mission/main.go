package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/client"
	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/models"
	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/repository"
	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/response"
)

var (
	missionRepo    *repository.MissionRepo
	timelineRepo   *repository.TimelineRepo
	incidentClient *client.IncidentClient
)

func init() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("unable to load AWS config: %v", err)
	}

	ddbClient := dynamodb.NewFromConfig(cfg)
	tableMission := os.Getenv("TABLE_MISSION")
	tableTimeline := os.Getenv("TABLE_TIMELINE")

	missionRepo = repository.NewMissionRepo(ddbClient, tableMission)
	timelineRepo = repository.NewTimelineRepo(ddbClient, tableTimeline)
	incidentClient = client.NewIncidentClient()
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// 1. Parse incident_id from path
	incidentID := request.PathParameters["incident_id"]
	if incidentID == "" {
		return response.Error(400, "MISSING_PARAMETER", "incident_id is required"), nil
	}

	// 2. Query mission by incident_id
	mission, err := missionRepo.GetMissionByIncidentID(ctx, incidentID)
	if err != nil {
		log.Printf("ERROR: query mission: %v", err)
		return response.Error(500, "INTERNAL_ERROR", "Failed to query mission"), nil
	}
	if mission == nil {
		return response.Error(404, "INCIDENT_NOT_FOUND", "No mission found for incident: "+incidentID), nil
	}

	// 3. Call IncidentTracking Service (degraded mode on failure)
	dataSource := "full"
	var description, location, incidentType string

	incidentDetail := incidentClient.GetIncidentDetail(incidentID)
	if incidentDetail != nil {
		description = incidentDetail.Description
		location = incidentDetail.Location
		incidentType = incidentDetail.IncidentType
	} else {
		dataSource = "partial"
		log.Printf("INFO: IncidentTracking unavailable - returning partial data for %s", incidentID)
	}

	// 4. Query timeline entries sorted by timestamp
	timeline, err := timelineRepo.GetTimelineByMissionID(ctx, mission.MissionID)
	if err != nil {
		log.Printf("ERROR: query timeline: %v", err)
		timeline = []models.TimelineEntry{}
	}
	if timeline == nil {
		timeline = []models.TimelineEntry{}
	}

	// 5. Return combined response
	return response.JSON(200, models.GetMissionResponse{
		IncidentID:        mission.IncidentID,
		MissionID:         mission.MissionID,
		RescueTeamID:      mission.RescueTeamID,
		CurrentStatus:     mission.CurrentStatus,
		LatestImpactLevel: mission.LatestImpactLevel,
		StartedAt:         mission.StartedAt,
		LastUpdatedAt:     mission.LastUpdatedAt,
		Description:       description,
		Location:          location,
		IncidentType:      incidentType,
		Timeline:          timeline,
		DataSource:        dataSource,
	}), nil
}

func main() {
	lambda.Start(handler)
}
