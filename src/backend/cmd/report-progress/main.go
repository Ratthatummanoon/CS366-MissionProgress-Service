package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/google/uuid"

	evtpub "github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/events"
	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/models"
	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/repository"
	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/response"
	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/statemachine"
)

var (
	missionRepo  *repository.MissionRepo
	timelineRepo *repository.TimelineRepo
	publisher    *evtpub.Publisher
)

func init() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("unable to load AWS config: %v", err)
	}

	ddbClient := dynamodb.NewFromConfig(cfg)
	ebClient := eventbridge.NewFromConfig(cfg)

	tableMission := os.Getenv("TABLE_MISSION")
	tableTimeline := os.Getenv("TABLE_TIMELINE")
	tableOutbox := os.Getenv("TABLE_OUTBOX")

	missionRepo = repository.NewMissionRepo(ddbClient, tableMission)
	timelineRepo = repository.NewTimelineRepo(ddbClient, tableTimeline)
	outboxRepo := repository.NewOutboxRepo(ddbClient, tableOutbox)
	publisher = evtpub.NewPublisher(ebClient, outboxRepo)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// 1. Parse incident_id from path
	incidentID := request.PathParameters["incident_id"]
	if incidentID == "" {
		return response.Error(400, "MISSING_PARAMETER", "incident_id is required"), nil
	}

	// 2. Parse X-Rescue-Team-ID from authorizer context or headers
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

	// 3. Parse & validate request body
	var req models.ReportProgressRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return response.Error(400, "INVALID_BODY", "Invalid request body"), nil
	}
	if req.Status == "" {
		return response.Error(400, "MISSING_PARAMETER", "new_status is required"), nil
	}
	if !statemachine.ValidateStatus(req.Status) {
		return response.Error(400, "INVALID_STATUS", "Invalid status value: "+req.Status), nil
	}

	// 4. Query mission by incident_id
	mission, err := missionRepo.GetMissionByIncidentID(ctx, incidentID)
	if err != nil {
		log.Printf("ERROR: query mission: %v", err)
		return response.Error(500, "INTERNAL_ERROR", "Failed to query mission"), nil
	}
	if mission == nil {
		return response.Error(404, "INCIDENT_NOT_FOUND", "No mission found for incident: "+incidentID), nil
	}

	// 5. Validate state transition
	oldStatus := mission.CurrentStatus
	if !statemachine.IsValidTransition(oldStatus, req.Status) {
		return response.Error(400, "INVALID_STATE_TRANSITION",
			"Cannot transition from "+oldStatus+" to "+req.Status), nil
	}

	// 6. Update mission status
	now := time.Now().UTC().Format(time.RFC3339)
	mission.CurrentStatus = req.Status
	mission.LastUpdatedAt = now
	if req.NewImpactLevel != nil {
		mission.LatestImpactLevel = *req.NewImpactLevel
	}
	if err := missionRepo.UpdateMissionStatus(ctx, mission); err != nil {
		log.Printf("ERROR: update mission: %v", err)
		return response.Error(500, "INTERNAL_ERROR", "Failed to update mission status"), nil
	}

	// 7. Add timeline entry
	entry := &models.TimelineEntry{
		MissionID:   mission.MissionID,
		Timestamp:   now,
		LogID:       uuid.New().String(),
		ActionType:  "STATUS_CHANGE",
		Description: req.Note,
		PerformedBy: rescueTeamID,
		GPSLocation: req.CurrentLocation,
	}
	if err := timelineRepo.AddTimelineEntry(ctx, entry); err != nil {
		log.Printf("ERROR: add timeline entry: %v", err)
	}

	// 8. Publish events (non-blocking with outbox fallback)
	publisher.PublishMissionStatusChanged(ctx, models.MissionStatusChangedEvent{
		MissionID:    mission.MissionID,
		IncidentID:   mission.IncidentID,
		RescueTeamID: mission.RescueTeamID,
		OldStatus:    oldStatus,
		NewStatus:    req.Status,
		ChangedAt:    now,
		ChangedBy:    rescueTeamID,
	})

	if req.Status == "NEED_BACKUP" {
		publisher.PublishBackupRequested(ctx, models.MissionBackupRequestedEvent{
			MissionID:    mission.MissionID,
			IncidentID:   mission.IncidentID,
			RescueTeamID: mission.RescueTeamID,
			RequestedAt:  now,
			RequestedBy:  rescueTeamID,
			Location:     req.CurrentLocation,
		})
	}

	if req.NewImpactLevel != nil {
		publisher.PublishImpactLevelUpdated(ctx, models.ImpactLevelUpdatedEvent{
			MissionID:    mission.MissionID,
			IncidentID:   mission.IncidentID,
			RescueTeamID: mission.RescueTeamID,
			OldLevel:     0,
			NewLevel:     *req.NewImpactLevel,
			UpdatedAt:    now,
			UpdatedBy:    rescueTeamID,
		})
	}

	// 9. Return success response
	return response.JSON(200, models.ReportProgressResponse{
		Message:    "Progress reported successfully",
		MissionID:  mission.MissionID,
		IncidentID: mission.IncidentID,
		OldStatus:  oldStatus,
		NewStatus:  req.Status,
		UpdatedAt:  now,
	}), nil
}

func main() {
	lambda.Start(handler)
}
