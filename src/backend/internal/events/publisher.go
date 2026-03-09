package events

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	ebtypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/google/uuid"

	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/models"
	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/repository"
)

const source = "MissionProgressService"

// Publisher publishes events to EventBridge with Outbox fallback.
type Publisher struct {
	ebClient   *eventbridge.Client
	outboxRepo *repository.OutboxRepo
	busName    string
}

// NewPublisher creates a new event publisher.
func NewPublisher(ebClient *eventbridge.Client, outboxRepo *repository.OutboxRepo) *Publisher {
	busName := os.Getenv("EVENT_BUS_NAME")
	if busName == "" {
		busName = "mission-progress-events"
	}
	return &Publisher{
		ebClient:   ebClient,
		outboxRepo: outboxRepo,
		busName:    busName,
	}
}

// PublishMissionStatusChanged publishes the MissionStatusChanged event.
func (p *Publisher) PublishMissionStatusChanged(ctx context.Context, event models.MissionStatusChangedEvent) {
	p.publish(ctx, "MissionStatusChanged", event)
}

// PublishBackupRequested publishes the MissionBackupRequested event.
func (p *Publisher) PublishBackupRequested(ctx context.Context, event models.MissionBackupRequestedEvent) {
	p.publish(ctx, "MissionBackupRequested", event)
}

// PublishImpactLevelUpdated publishes the ImpactLevelUpdated event.
func (p *Publisher) PublishImpactLevelUpdated(ctx context.Context, event models.ImpactLevelUpdatedEvent) {
	p.publish(ctx, "ImpactLevelUpdated", event)
}

func (p *Publisher) publish(ctx context.Context, detailType string, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ERROR: marshal event payload: %v", err)
		return
	}

	_, err = p.ebClient.PutEvents(ctx, &eventbridge.PutEventsInput{
		Entries: []ebtypes.PutEventsRequestEntry{
			{
				Source:       aws.String(source),
				DetailType:  aws.String(detailType),
				Detail:      aws.String(string(data)),
				EventBusName: aws.String(p.busName),
			},
		},
	})
	if err != nil {
		log.Printf("ERROR: EventBridge publish failed for %s: %v - saving to outbox", detailType, err)
		p.saveToOutbox(ctx, detailType, string(data))
		return
	}
	log.Printf("INFO: Published %s event to EventBridge", detailType)
}

func (p *Publisher) saveToOutbox(ctx context.Context, eventType string, payload string) {
	now := time.Now().UTC().Format(time.RFC3339)
	entry := &models.OutboxEntry{
		OutboxID:     uuid.New().String(),
		CreatedAt:    now,
		EventType:    eventType,
		EventPayload: payload,
		Status:       "PENDING",
		RetryCount:   0,
	}
	if err := p.outboxRepo.SaveOutboxEntry(ctx, entry); err != nil {
		log.Printf("ERROR: Failed to save outbox entry: %v", err)
	} else {
		log.Printf("INFO: Saved %s event to outbox for retry", eventType)
	}
}
