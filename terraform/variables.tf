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
  default     = "mission-progress-api-key-2024"
}

variable "incident_service_url" {
  description = "URL of the IncidentTracking Service"
  type        = string
  default     = "http://localhost:9999"
}
