output "api_gateway_invoke_url" {
  description = "API Gateway invoke URL"
  value       = aws_api_gateway_stage.v1.invoke_url
}

output "api_key_value" {
  description = "API Key value for authentication"
  value       = local.api_key_value
  sensitive   = true
}

output "frontend_url" {
  description = "Frontend website URL"
  value       = aws_s3_bucket_website_configuration.frontend.website_endpoint
}

output "frontend_bucket" {
  description = "Frontend S3 bucket name"
  value       = aws_s3_bucket.frontend.id
}

output "evidence_bucket" {
  description = "Evidence S3 bucket name"
  value       = aws_s3_bucket.evidence.id
}
