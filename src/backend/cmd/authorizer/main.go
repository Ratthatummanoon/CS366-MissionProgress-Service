package main

import (
	"context"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var validAPIKey string

func init() {
	validAPIKey = os.Getenv("VALID_API_KEY")
}

func handler(ctx context.Context, request events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	apiKey := request.Headers["x-api-key"]
	if apiKey == "" {
		apiKey = request.Headers["X-Api-Key"]
	}
	rescueTeamID := request.Headers["X-Rescue-Team-ID"]
	if rescueTeamID == "" {
		rescueTeamID = request.Headers["x-rescue-team-id"]
	}

	// Validate
	if apiKey == "" || rescueTeamID == "" || apiKey != validAPIKey {
		return generatePolicy("user", "Deny", request.MethodArn), nil
	}

	// Allow - principalId = rescue_team_id
	resp := generatePolicy(rescueTeamID, "Allow", request.MethodArn)
	resp.Context = map[string]interface{}{
		"rescueTeamId": rescueTeamID,
	}
	return resp, nil
}

func generatePolicy(principalID, effect, resource string) events.APIGatewayCustomAuthorizerResponse {
	// Use wildcard for the resource to cover all methods/paths under this API
	arnParts := strings.Split(resource, "/")
	if len(arnParts) > 1 {
		resource = arnParts[0] + "/*"
	}

	return events.APIGatewayCustomAuthorizerResponse{
		PrincipalID: principalID,
		PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
			Version: "2012-10-17",
			Statement: []events.IAMPolicyStatement{
				{
					Action:   []string{"execute-api:Invoke"},
					Effect:   effect,
					Resource: []string{resource},
				},
			},
		},
	}
}

func main() {
	lambda.Start(handler)
}
