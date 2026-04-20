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

variable "prioritization_sqs_arn" {
  description = "SQS ARN for Prioritization Service consumer"
  type        = string
  default     = ""
}
