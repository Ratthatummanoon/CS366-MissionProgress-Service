package response

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"

	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/models"
)

var corsHeaders = map[string]string{
	"Content-Type":                 "application/json",
	"Access-Control-Allow-Origin":  "*",
	"Access-Control-Allow-Methods": "GET,POST,OPTIONS",
	"Access-Control-Allow-Headers": "x-api-key,X-Rescue-Team-ID,Content-Type",
}

// JSON returns a successful API Gateway proxy response.
func JSON(statusCode int, body interface{}) events.APIGatewayProxyResponse {
	data, _ := json.Marshal(body)
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    corsHeaders,
		Body:       string(data),
	}
}

// Error returns an error API Gateway proxy response.
func Error(statusCode int, code, message string) events.APIGatewayProxyResponse {
	return JSON(statusCode, models.ErrorResponse{
		Error:   code,
		Code:    code,
		Message: message,
	})
}
