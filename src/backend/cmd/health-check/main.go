package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type ServiceStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "available" | "unavailable" | "not_configured"
}

type ServiceURLs struct {
	RescueTeam     string `json:"rescueTeamUrl,omitempty"`
	ManageDispatch string `json:"manageDispatchUrl,omitempty"`
	RescueRequest  string `json:"rescueRequestUrl,omitempty"`
}

type HealthResponse struct {
	Services    []ServiceStatus `json:"services"`
	ServiceURLs ServiceURLs     `json:"service_urls"`
	CheckedAt   string          `json:"checked_at"`
}

func checkService(name, baseURL string) ServiceStatus {
	if baseURL == "" {
		return ServiceStatus{Name: name, Status: "not_configured"}
	}

	c := &http.Client{Timeout: 2 * time.Second}
	resp, err := c.Get(baseURL)
	if err != nil {
		return ServiceStatus{Name: name, Status: "unavailable"}
	}
	resp.Body.Close()
	return ServiceStatus{Name: name, Status: "available"}
}

func handler(_ context.Context, _ events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	rescueRequestURL  := os.Getenv("RESCUE_REQUEST_SERVICE_URL")
	manageDispatchURL := os.Getenv("MANAGE_DISPATCH_SERVICE_URL")
	rescueTeamURL     := os.Getenv("RESCUE_TEAM_SERVICE_URL")

	services := []ServiceStatus{
		checkService("RescueRequest", rescueRequestURL),
		checkService("ManageDispatch", manageDispatchURL),
		checkService("RescueTeam", rescueTeamURL),
	}

	body, _ := json.Marshal(HealthResponse{
		Services: services,
		ServiceURLs: ServiceURLs{
			RescueTeam:     rescueTeamURL,
			ManageDispatch: manageDispatchURL,
			RescueRequest:  rescueRequestURL,
		},
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
	})

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(body),
	}, nil
}

func main() {
	lambda.Start(handler)
}
