package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	ebtypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"

	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/repository"
)

const (
	source     = "MissionProgressService"
	maxRetries = 5
)

var (
	outboxRepo *repository.OutboxRepo
	ebClient   *eventbridge.Client
	busName    string
)

func init() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("unable to load AWS config: %v", err)
	}

	ddbClient := dynamodb.NewFromConfig(cfg)
	ebClient = eventbridge.NewFromConfig(cfg)

	tableOutbox := os.Getenv("TABLE_OUTBOX")
	outboxRepo = repository.NewOutboxRepo(ddbClient, tableOutbox)

	busName = os.Getenv("EVENT_BUS_NAME")
	if busName == "" {
		busName = "mission-progress-events"
	}
}

func handler(ctx context.Context) error {
	entries, err := outboxRepo.GetPendingOutboxEntries(ctx)
	if err != nil {
		log.Printf("ERROR: fetch pending outbox entries: %v", err)
		return err
	}

	if len(entries) == 0 {
		log.Printf("INFO: No pending outbox entries")
		return nil
	}

	log.Printf("INFO: Processing %d pending outbox entries", len(entries))

	for _, entry := range entries {
		// Validate that event_payload is valid JSON
		var payload json.RawMessage
		if err := json.Unmarshal([]byte(entry.EventPayload), &payload); err != nil {
			log.Printf("ERROR: invalid event payload for %s: %v", entry.OutboxID, err)
			updateErr := outboxRepo.UpdateOutboxEntryStatus(ctx, entry.OutboxID, "FAILED", entry.RetryCount+1, "invalid payload: "+err.Error())
			if updateErr != nil {
				log.Printf("ERROR: update outbox entry %s: %v", entry.OutboxID, updateErr)
			}
			continue
		}

		// Publish to EventBridge
		ebCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, pubErr := ebClient.PutEvents(ebCtx, &eventbridge.PutEventsInput{
			Entries: []ebtypes.PutEventsRequestEntry{
				{
					Source:       aws.String(source),
					DetailType:  aws.String(entry.EventType),
					Detail:      aws.String(entry.EventPayload),
					EventBusName: aws.String(busName),
				},
			},
		})
		cancel()

		if pubErr != nil {
			newRetryCount := entry.RetryCount + 1
			if newRetryCount >= maxRetries {
				log.Printf("ERROR: Max retries reached for outbox entry %s, marking as FAILED", entry.OutboxID)
				updateErr := outboxRepo.UpdateOutboxEntryStatus(ctx, entry.OutboxID, "FAILED", newRetryCount, pubErr.Error())
				if updateErr != nil {
					log.Printf("ERROR: update outbox entry %s: %v", entry.OutboxID, updateErr)
				}
			} else {
				log.Printf("WARNING: Retry %d/%d for outbox entry %s: %v", newRetryCount, maxRetries, entry.OutboxID, pubErr)
				updateErr := outboxRepo.UpdateOutboxEntryStatus(ctx, entry.OutboxID, "PENDING", newRetryCount, pubErr.Error())
				if updateErr != nil {
					log.Printf("ERROR: update outbox entry %s: %v", entry.OutboxID, updateErr)
				}
			}
			continue
		}

		// Success — mark as SENT
		log.Printf("INFO: Successfully published outbox entry %s (%s)", entry.OutboxID, entry.EventType)
		updateErr := outboxRepo.UpdateOutboxEntryStatus(ctx, entry.OutboxID, "SENT", entry.RetryCount, "")
		if updateErr != nil {
			log.Printf("ERROR: update outbox entry %s after success: %v", entry.OutboxID, updateErr)
		}
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
