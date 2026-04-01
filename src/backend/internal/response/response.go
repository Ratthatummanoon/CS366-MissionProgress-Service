package response

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"

	"github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/models"
)

var corsHeaders = map[string]string{
	"Content-Type":                 "application/json",
	"Access-Control-Allow-Origin":  "*",
	"Access-Control-Allow-Methods": "GET,POST,OPTIONS",
	"Access-Control-Allow-Headers": "x-api-key,X-Rescue-Team-ID,Content-Type",
}

// newTraceID generates a unique trace ID for each response.
func newTraceID() string {
	return uuid.New().String()
}

// buildHeaders copies corsHeaders and adds X-Trace-Id.
func buildHeaders(traceID string) map[string]string {
	headers := make(map[string]string, len(corsHeaders)+1)
	for k, v := range corsHeaders {
		headers[k] = v
	}
	headers["X-Trace-Id"] = traceID
	return headers
}

// JSON returns a successful API Gateway proxy response.
func JSON(statusCode int, body interface{}) events.APIGatewayProxyResponse {
	traceID := newTraceID()
	data, _ := json.Marshal(body)
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    buildHeaders(traceID),
		Body:       string(data),
	}
}

// Error returns an error API Gateway proxy response.
func Error(statusCode int, code, message string) events.APIGatewayProxyResponse {
	traceID := newTraceID()
	data, _ := json.Marshal(models.ErrorResponse{
		Error:   code,
		Code:    code,
		Message: message,
		TraceID: traceID,
	})
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    buildHeaders(traceID),
		Body:       string(data),
	}
}