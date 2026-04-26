package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"

	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/client"
	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/models"
	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/repository"
)

// EventBridgeEvent represents an EventBridge event envelope.
type EventBridgeEvent struct {
	Source     string          `json:"source"`
	DetailType string         `json:"detail-type"`
	Detail     json.RawMessage `json:"detail"`
}

// MissionAssignedPayload คือ payload จาก DispatchOrderCreated event ของ Manage Dispatch Service
// field names ใช้ camelCase ตาม Manage Dispatch contract
type MissionAssignedPayload struct {
	DispatchID    string `json:"dispatchId"`
	RequestID     string `json:"requestId"`
	TeamID        string `json:"teamId"`
	PriorityLevel int    `json:"priorityLevel"`
	Status        string `json:"status"`
	DispatchedAt  string `json:"dispatchedAt"`
}

var (
	missionRepo         *repository.MissionRepo
	timelineRepo        *repository.TimelineRepo
	rescueRequestClient *client.RescueRequestClient
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
	rescueRequestClient = client.NewRescueRequestClient()
}

func handler(ctx context.Context, event EventBridgeEvent) error {
	log.Printf("INFO: Received event: source=%s, detail-type=%s", event.Source, event.DetailType)

	// 1. Parse payload
	var payload MissionAssignedPayload
	if err := json.Unmarshal(event.Detail, &payload); err != nil {
		return fmt.Errorf("unmarshal MissionAssignedEvent detail: %w", err)
	}

	// 2. Validate required fields
	if payload.DispatchID == "" || payload.RequestID == "" || payload.TeamID == "" {
		return fmt.Errorf("missing required fields: dispatchId=%s, requestId=%s, teamId=%s",
			payload.DispatchID, payload.RequestID, payload.TeamID)
	}

	assignedAt := payload.DispatchedAt
	if assignedAt == "" {
		assignedAt = "unknown"
	}

	// 3. Idempotency check — query by dispatch_id (not by generated mission_id)
	// EventBridge delivers at-least-once; the same DispatchOrderCreated event may arrive multiple times.
	existing, err := missionRepo.GetMissionByDispatchID(ctx, payload.DispatchID)
	if err != nil {
		log.Printf("ERROR: idempotency check failed for dispatchId=%s: %v", payload.DispatchID, err)
		return fmt.Errorf("idempotency check: %w", err)
	}
	if existing != nil {
		log.Printf("INFO: Mission for dispatchId=%s already exists (missionId=%s) — skipping (idempotent)",
			payload.DispatchID, existing.MissionID)
		return nil
	}

	// 4. Fetch incidentId from RescueRequest Service (degraded: empty string on failure)
	incidentID := ""
	if requestDetail := rescueRequestClient.GetRequestDetail(ctx, payload.RequestID); requestDetail != nil {
		incidentID = requestDetail.Master.IncidentID
		log.Printf("INFO: Fetched incidentId=%s for requestId=%s", incidentID, payload.RequestID)
	} else {
		log.Printf("WARN: RescueRequestService unavailable for requestId=%s — incidentId will be empty", payload.RequestID)
	}

	// 5. Generate mission_id ใหม่ (Manage Dispatch ไม่ส่ง mission_id มาให้)
	generatedMissionID := "MISS-" + uuid.New().String()[:8]

	// 6. Create mission record
	mission := &models.MissionAssignment{
		MissionID:         generatedMissionID,
		DispatchID:        payload.DispatchID,
		RequestID:         payload.RequestID,
		IncidentID:        incidentID,
		RescueTeamID:      payload.TeamID,
		PriorityLevel:     payload.PriorityLevel,
		CurrentStatus:     "DISPATCHED",
		LatestImpactLevel: 0,
		StartedAt:         assignedAt,
		LastUpdatedAt:     assignedAt,
	}

	if err := missionRepo.CreateMission(ctx, mission); err != nil {
		return fmt.Errorf("create mission: %w", err)
	}

	log.Printf("INFO: Created mission %s for dispatch %s, team %s, incident %s",
		generatedMissionID, payload.DispatchID, payload.TeamID, incidentID)

	// 7. Create initial timeline entry
	entry := &models.TimelineEntry{
		MissionID:   generatedMissionID,
		Timestamp:   assignedAt,
		LogID:       uuid.New().String(),
		ActionType:  "MISSION_ASSIGNED",
		Description: fmt.Sprintf("Dispatch %s assigned to team %s", payload.DispatchID, payload.TeamID),
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
