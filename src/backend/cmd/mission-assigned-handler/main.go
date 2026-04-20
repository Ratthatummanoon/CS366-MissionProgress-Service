package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"

	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/models"
	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/repository"
)

// EventBridgeEvent represents an EventBridge event envelope.
type EventBridgeEvent struct {
	Source     string          `json:"source"`
	DetailType string         `json:"detail-type"`
	Detail     json.RawMessage `json:"detail"`
}

// MissionAssignedPayload is the payload from Dispatch service.
type MissionAssignedPayload struct {
	MissionID    string `json:"mission_id"`
	RescueUnitID string `json:"rescue_unit_id"`
	IncidentID   string `json:"incident_id"`
	AssignedAt   string `json:"assigned_at"`
}

var (
	missionRepo  *repository.MissionRepo
	timelineRepo *repository.TimelineRepo
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
}

func handler(ctx context.Context, event EventBridgeEvent) error {
	log.Printf("INFO: Received event: source=%s, detail-type=%s", event.Source, event.DetailType)

	// 1. Parse payload
	var payload MissionAssignedPayload
	if err := json.Unmarshal(event.Detail, &payload); err != nil {
		return fmt.Errorf("unmarshal MissionAssignedEvent detail: %w", err)
	}

	// 2. Validate required fields
	if payload.MissionID == "" || payload.IncidentID == "" || payload.RescueUnitID == "" {
		return fmt.Errorf("missing required fields: mission_id=%s, incident_id=%s, rescue_unit_id=%s",
			payload.MissionID, payload.IncidentID, payload.RescueUnitID)
	}

	assignedAt := payload.AssignedAt
	if assignedAt == "" {
		assignedAt = "unknown"
	}

	// 3. Create mission record (idempotent — skip if already exists)
	mission := &models.MissionAssignment{
		MissionID:         payload.MissionID,
		IncidentID:        payload.IncidentID,
		RescueTeamID:      payload.RescueUnitID,
		CurrentStatus:     "DISPATCHED",
		LatestImpactLevel: 0,
		StartedAt:         assignedAt,
		LastUpdatedAt:     assignedAt,
	}

	err := missionRepo.CreateMissionIdempotent(ctx, mission)
	if err != nil {
		// Check if it's a conditional check failure (already exists) — that's OK
		if strings.Contains(err.Error(), "ConditionalCheckFailedException") {
			log.Printf("INFO: Mission %s already exists — skipping (idempotent)", payload.MissionID)
			return nil
		}
		return fmt.Errorf("create mission: %w", err)
	}

	log.Printf("INFO: Created mission %s for incident %s, team %s",
		payload.MissionID, payload.IncidentID, payload.RescueUnitID)

	// 4. Create initial timeline entry
	entry := &models.TimelineEntry{
		MissionID:   payload.MissionID,
		Timestamp:   assignedAt,
		LogID:       uuid.New().String(),
		ActionType:  "MISSION_ASSIGNED",
		Description: fmt.Sprintf("Mission assigned to %s", payload.RescueUnitID),
		PerformedBy: "SYSTEM",
	}

	if err := timelineRepo.AddTimelineEntry(ctx, entry); err != nil {
		log.Printf("ERROR: add timeline entry: %v", err)
		// Don't fail the whole handler — mission was created successfully
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
