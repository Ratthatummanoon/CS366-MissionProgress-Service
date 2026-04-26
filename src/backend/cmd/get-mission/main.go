package main

import (
	"context"
	"log"
	"os"
	"sync"

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
	missionRepo          *repository.MissionRepo
	timelineRepo         *repository.TimelineRepo
	rescueRequestClient  *client.RescueRequestClient
	manageDispatchClient *client.ManageDispatchClient
	rescueTeamClient     *client.RescueTeamClient
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
	manageDispatchClient = client.NewManageDispatchClient()
	rescueTeamClient = client.NewRescueTeamClient()
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// 1. Parse request_id from path
	requestID := request.PathParameters["request_id"]
	if requestID == "" {
		return response.Error(400, "MISSING_PARAMETER", "request_id is required"), nil
	}

	// 2. Query mission by request_id
	mission, err := missionRepo.GetMissionByRequestID(ctx, requestID)
	if err != nil {
		log.Printf("ERROR: query mission: %v", err)
		return response.Error(500, "INTERNAL_ERROR", "Failed to query mission"), nil
	}
	if mission == nil {
		return response.Error(404, "REQUEST_NOT_FOUND", "No mission found for request: "+requestID), nil
	}

	// 3. Call all external services in parallel (degraded mode on failure)
	// Sequential calls would take up to ~8s worst-case; parallel reduces it to ~2.7s.
	var (
		dataSource   = "full"
		description  string
		location     string
		incidentType string

		dispatchStatus string
		priorityLevel  int

		teamName     string
		teamType     string
		capabilities []string
		equipment    []string
		teamLocation *models.TeamLocationSnap
	)

	var wg sync.WaitGroup
	wg.Add(3)

	// 3a. RescueRequest Service — fetch request description/location/type
	go func() {
		defer wg.Done()
		requestDetail := rescueRequestClient.GetRequestDetail(ctx, requestID)
		if requestDetail != nil {
			description = requestDetail.Master.Description
			location = client.FormatLocation(requestDetail.Master)
			incidentType = requestDetail.Master.RequestType
		} else {
			dataSource = "partial"
			log.Printf("INFO: RescueRequestService unavailable - returning partial data for requestID=%s", requestID)
		}
	}()

	// 3b. ManageDispatch Service — fetch dispatch status/priority
	go func() {
		defer wg.Done()
		if mission.DispatchID == "" {
			return
		}
		dispatchList := manageDispatchClient.GetDispatchByTeamAndRequest(ctx, mission.RescueTeamID)
		if dispatchList != nil {
			for _, item := range dispatchList.Items {
				if item.DispatchID == mission.DispatchID {
					dispatchStatus = item.Status
					priorityLevel = item.PriorityLevel
					break
				}
			}
		} else {
			log.Printf("INFO: ManageDispatchService unavailable - skipping dispatch enrichment for missionID=%s", mission.MissionID)
		}
	}()

	// 3c. RescueTeam Service — fetch team info
	go func() {
		defer wg.Done()
		teamDetail := rescueTeamClient.GetTeamDetail(ctx, mission.RescueTeamID)
		if teamDetail != nil {
			teamName = teamDetail.TeamName
			teamType = teamDetail.TeamType
			capabilities = teamDetail.Capabilities
			equipment = teamDetail.Equipment
			teamLocation = &models.TeamLocationSnap{
				Lat: teamDetail.Location.Lat,
				Lng: teamDetail.Location.Lng,
			}
		} else {
			log.Printf("INFO: RescueTeamService unavailable - returning partial team data for teamID=%s", mission.RescueTeamID)
			dataSource = "partial"
		}
	}()

	wg.Wait()

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
		RequestID:         mission.RequestID,
		IncidentID:        mission.IncidentID,
		MissionID:         mission.MissionID,
		DispatchID:        mission.DispatchID,
		RescueTeamID:      mission.RescueTeamID,
		TeamName:          teamName,
		TeamType:          teamType,
		Capabilities:      capabilities,
		Equipment:         equipment,
		TeamLocation:      teamLocation,
		PriorityLevel:     priorityLevel,
		DispatchStatus:    dispatchStatus,
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
