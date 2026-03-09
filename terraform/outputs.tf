output "api_gateway_invoke_url" {
  description = "API Gateway invoke URL"
  value       = aws_api_gateway_stage.v1.invoke_url
}

output "api_key_value" {
  description = "API Key value for authentication"
  value       = var.api_key_value
  sensitive   = true
}
