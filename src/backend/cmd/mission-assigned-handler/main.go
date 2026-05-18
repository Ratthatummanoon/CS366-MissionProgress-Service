package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"

	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/client"
	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/models"
	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/repository"
)

// SNSDispatchMessage คือ envelope ของ SNS message จาก Manage Dispatch Service.
type SNSDispatchMessage struct {
	Header struct {
		MessageType string `json:"messageType"`
		TraceID     string `json:"traceId"`
	} `json:"header"`
	Body DispatchBody `json:"body"`
}

// snsNotification คือ JSON wrapper ที่ SNS ใส่ใน SQS message body
// เมื่อใช้ SNS → SQS subscription
type snsNotification struct {
	Type     string `json:"Type"`
	Message  string `json:"Message"`
	TopicArn string `json:"TopicArn"`
}

// DispatchBody คือ payload ของ DispatchOrderCreated event ตาม Manage Dispatch contract.
type DispatchBody struct {
	DispatchID    string `json:"dispatchId"`
	Status        string `json:"status"`
	RequestID     string `json:"requestId"`
	TeamID        string `json:"teamId"`
	PriorityLevel string `json:"priorityLevel"` // "HIGH", "NORMAL", "LOW", "CRITICAL"
	DispatchedAt  string `json:"dispatchedAt"`
	Timestamp     string `json:"timestamp"`
}

// priorityToInt maps Manage Dispatch priority strings → int (CRITICAL=4, HIGH=3, NORMAL=2, LOW=1).
func priorityToInt(p string) int {
	switch p {
	case "CRITICAL":
		return 4
	case "HIGH":
		return 3
	case "NORMAL", "MEDIUM":
		return 2
	case "LOW":
		return 1
	default:
		return 0
	}
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

func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	for _, record := range sqsEvent.Records {
		// SQS message body คือ JSON ของ SNS notification (SNS → SQS subscription)
		// ต้อง unwrap: SQS body → SNS notification → SNS Message → actual payload
		var notification snsNotification
		if err := json.Unmarshal([]byte(record.Body), &notification); err != nil {
			log.Printf("ERROR: unmarshal SQS body as SNS notification: %v", err)
			return fmt.Errorf("unmarshal SQS body: %w", err)
		}
		log.Printf("INFO: Received SQS record from topic=%s", notification.TopicArn)
		if err := processRecord(ctx, notification.Message); err != nil {
			log.Printf("ERROR: processing record: %v", err)
			return err
		}
	}
	return nil
}

func processRecord(ctx context.Context, rawMessage string) error {
	// 1. Parse SNS envelope
	var msg SNSDispatchMessage
	if err := json.Unmarshal([]byte(rawMessage), &msg); err != nil {
		return fmt.Errorf("unmarshal SNS message: %w", err)
	}

	if msg.Header.MessageType != "DispatchOrderCreated" {
		log.Printf("INFO: Skipping unknown messageType=%s", msg.Header.MessageType)
		return nil
	}

	payload := msg.Body
	log.Printf("INFO: Received DispatchOrderCreated: dispatchId=%s, requestId=%s, teamId=%s",
		payload.DispatchID, payload.RequestID, payload.TeamID)

	// 2. Validate required fields
	if payload.DispatchID == "" || payload.RequestID == "" || payload.TeamID == "" {
		return fmt.Errorf("missing required fields: dispatchId=%s, requestId=%s, teamId=%s",
			payload.DispatchID, payload.RequestID, payload.TeamID)
	}

	assignedAt := payload.DispatchedAt
	if assignedAt == "" {
		assignedAt = payload.Timestamp
	}
	if assignedAt == "" {
		assignedAt = "unknown"
	}

	// 3. Idempotency check — query by dispatch_id
	// SNS delivers at-least-once; the same DispatchOrderCreated event may arrive multiple times.
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

	// 5. Generate mission_id
	generatedMissionID := "MISS-" + uuid.New().String()[:8]

	// 6. Create mission record
	mission := &models.MissionAssignment{
		MissionID:         generatedMissionID,
		DispatchID:        payload.DispatchID,
		RequestID:         payload.RequestID,
		IncidentID:        incidentID,
		RescueTeamID:      payload.TeamID,
		PriorityLevel:     priorityToInt(payload.PriorityLevel),
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
