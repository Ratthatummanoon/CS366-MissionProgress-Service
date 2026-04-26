variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "Project name prefix for resources"
  type        = string
  default     = "mission-progress"
}

variable "lab_role_arn" {
  description = "ARN of the LabRole IAM role"
  type        = string
  default     = ""
}

variable "api_key_value" {
  description = "API Key value for authentication"
  type        = string
  sensitive   = true
  default     = ""
}

variable "rescue_request_service_url" {
  description = "URL of the RescueRequest Service"
  type        = string
  default     = "http://localhost:9998"
}

variable "rescue_request_service_token" {
  description = "Bearer token for authenticating with RescueRequest Service (staff access)"
  type        = string
  sensitive   = true
  default     = ""
}

variable "incident_tracking_sqs_arn" {
  description = "SQS ARN for IncidentTracking Service consumer"
  type        = string
  default     = ""
}

variable "dispatch_sqs_arn" {
  description = "SQS ARN for Dispatch Management Service consumer"
  type        = string
  default     = ""
}

variable "manage_dispatch_service_url" {
  description = "URL of the Manage Dispatch Service"
  type        = string
  default     = "http://localhost:9997"
}

variable "manage_dispatch_service_token" {
  description = "Bearer token for authenticating with Manage Dispatch Service"
  type        = string
  sensitive   = true
  default     = ""
}

variable "rescue_team_service_url" {
  description = "URL of the RescueTeam Service"
  type        = string
  default     = "http://localhost:9996"
}

variable "rescue_team_service_token" {
  description = "Bearer token for authenticating with RescueTeam Service"
  type        = string
  sensitive   = true
  default     = ""
}

variable "prioritization_sqs_arn" {
  description = "SQS ARN for Prioritization Service consumer"
  type        = string
  default     = ""
}
