package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/models"
	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/repository"
	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/response"
)

var (
	missionRepo    *repository.MissionRepo
	s3PresignClient *s3.PresignClient
	evidenceBucket string
)

var allowedContentTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

func init() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("unable to load AWS config: %v", err)
	}

	ddbClient := dynamodb.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg)

	tableMission := os.Getenv("TABLE_MISSION")
	evidenceBucket = os.Getenv("EVIDENCE_BUCKET")

	missionRepo = repository.NewMissionRepo(ddbClient, tableMission)
	s3PresignClient = s3.NewPresignClient(s3Client)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// 1. Parse request_id from path
	requestID := request.PathParameters["request_id"]
	if requestID == "" {
		return response.Error(400, "MISSING_PARAMETER", "request_id is required"), nil
	}

	// 2. Parse X-Rescue-Team-ID from authorizer context
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

	// 3. Parse request body
	var req models.PresignedURLRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return response.Error(400, "INVALID_BODY", "Invalid request body"), nil
	}
	if req.FileName == "" {
		return response.Error(400, "MISSING_PARAMETER", "file_name is required"), nil
	}
	if req.ContentType == "" {
		return response.Error(400, "MISSING_PARAMETER", "content_type is required"), nil
	}

	// 4. Validate content type
	if !allowedContentTypes[req.ContentType] {
		return response.Error(400, "INVALID_CONTENT_TYPE",
			"content_type must be one of: image/jpeg, image/png, image/webp"), nil
	}

	// 5. Check mission exists
	mission, err := missionRepo.GetMissionByRequestID(ctx, requestID)
	if err != nil {
		log.Printf("ERROR: query mission: %v", err)
		return response.Error(500, "INTERNAL_ERROR", "Failed to query mission"), nil
	}
	if mission == nil {
		return response.Error(404, "REQUEST_NOT_FOUND", "No mission found for request: "+requestID), nil
	}

	// 6. Generate S3 key
	timestamp := time.Now().Unix()
	imageKey := fmt.Sprintf("evidence/%s/%s/%d-%s", mission.MissionID, rescueTeamID, timestamp, req.FileName)

	// 7. Generate presigned PUT URL
	presignResult, err := s3PresignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(evidenceBucket),
		Key:         aws.String(imageKey),
		ContentType: aws.String(req.ContentType),
	}, s3.WithPresignExpires(300*time.Second))
	if err != nil {
		log.Printf("ERROR: generate presigned URL: %v", err)
		return response.Error(500, "PRESIGN_FAILED", "Failed to generate upload URL. Mission can still operate in text-only mode."), nil
	}

	// 8. Return response
	return response.JSON(200, models.PresignedURLResponse{
		UploadURL: presignResult.URL,
		ImageKey:  imageKey,
		ExpiresIn: 300,
		Message:   "Upload URL generated successfully. Use PUT method to upload.",
	}), nil
}

func main() {
	lambda.Start(handler)
}
