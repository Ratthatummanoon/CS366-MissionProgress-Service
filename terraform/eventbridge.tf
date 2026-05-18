# ---------------------------------------------------
# Custom Event Bus
# ---------------------------------------------------
resource "aws_cloudwatch_event_bus" "mission_events" {
  name = "mission-progress-events"

  tags = {
    Project = var.project_name
  }
}

# ---------------------------------------------------
# CloudWatch Log Groups for event targets
# ---------------------------------------------------
resource "aws_cloudwatch_log_group" "mission_status_changed" {
  name              = "/aws/events/${var.project_name}/mission-status-changed"
  retention_in_days = 7

  tags = {
    Project = var.project_name
  }
}

resource "aws_cloudwatch_log_group" "backup_requested" {
  name              = "/aws/events/${var.project_name}/backup-requested"
  retention_in_days = 7

  tags = {
    Project = var.project_name
  }
}

resource "aws_cloudwatch_log_group" "impact_level_updated" {
  name              = "/aws/events/${var.project_name}/impact-level-updated"
  retention_in_days = 7

  tags = {
    Project = var.project_name
  }
}

# ---------------------------------------------------
# Rule: MissionStatusChanged → CloudWatch Log
# ---------------------------------------------------
resource "aws_cloudwatch_event_rule" "mission_status_changed" {
  name           = "mission-status-changed-rule"
  event_bus_name = aws_cloudwatch_event_bus.mission_events.name
  description    = "Capture MissionStatusChanged events"

  event_pattern = jsonencode({
    source      = ["MissionProgressService"]
    detail-type = ["MissionStatusChanged"]
  })

  tags = {
    Project = var.project_name
  }
}

resource "aws_cloudwatch_event_target" "mission_status_changed_log" {
  rule           = aws_cloudwatch_event_rule.mission_status_changed.name
  event_bus_name = aws_cloudwatch_event_bus.mission_events.name
  target_id      = "mission-status-changed-log"
  arn            = aws_cloudwatch_log_group.mission_status_changed.arn
}

# ---------------------------------------------------
# API Destination: IncidentTracking → mission-status-receiver
# ---------------------------------------------------
resource "aws_cloudwatch_event_connection" "incident_tracking" {
  name               = "incident-tracking-connection"
  authorization_type = "API_KEY"

  auth_parameters {
    api_key {
      key   = "x-api-key"
      value = var.incident_tracking_api_key
    }
  }
}

resource "aws_cloudwatch_event_api_destination" "incident_tracking" {
  name                             = "incident-tracking-mission-status-receiver"
  connection_arn                   = aws_cloudwatch_event_connection.incident_tracking.arn
  invocation_endpoint              = var.incident_tracking_receiver_url
  http_method                      = "POST"
  invocation_rate_limit_per_second = 20
}

resource "aws_cloudwatch_event_target" "mission_status_changed_incident_tracking" {
  rule           = aws_cloudwatch_event_rule.mission_status_changed.name
  event_bus_name = aws_cloudwatch_event_bus.mission_events.name
  target_id      = "mission-status-changed-incident-tracking"
  arn            = aws_cloudwatch_event_api_destination.incident_tracking.arn
  role_arn       = local.lab_role_arn
}

# ---------------------------------------------------
# Rule: MissionBackupRequested → CloudWatch Log
# ---------------------------------------------------
resource "aws_cloudwatch_event_rule" "backup_requested" {
  name           = "backup-requested-rule"
  event_bus_name = aws_cloudwatch_event_bus.mission_events.name
  description    = "Capture MissionBackupRequested events"

  event_pattern = jsonencode({
    source      = ["MissionProgressService"]
    detail-type = ["MissionBackupRequested"]
  })

  tags = {
    Project = var.project_name
  }
}

resource "aws_cloudwatch_event_target" "backup_requested_log" {
  rule           = aws_cloudwatch_event_rule.backup_requested.name
  event_bus_name = aws_cloudwatch_event_bus.mission_events.name
  target_id      = "backup-requested-log"
  arn            = aws_cloudwatch_log_group.backup_requested.arn
}

# ---------------------------------------------------
# Rule: ImpactLevelUpdated → CloudWatch Log
# ---------------------------------------------------
resource "aws_cloudwatch_event_rule" "impact_level_updated" {
  name           = "impact-level-updated-rule"
  event_bus_name = aws_cloudwatch_event_bus.mission_events.name
  description    = "Capture ImpactLevelUpdated events"

  event_pattern = jsonencode({
    source      = ["MissionProgressService"]
    detail-type = ["ImpactLevelUpdated"]
  })

  tags = {
    Project = var.project_name
  }
}

resource "aws_cloudwatch_event_target" "impact_level_updated_log" {
  rule           = aws_cloudwatch_event_rule.impact_level_updated.name
  event_bus_name = aws_cloudwatch_event_bus.mission_events.name
  target_id      = "impact-level-updated-log"
  arn            = aws_cloudwatch_log_group.impact_level_updated.arn
}

# ---------------------------------------------------
# Resource Policy: Allow EventBridge to write to CloudWatch Logs
# ---------------------------------------------------
data "aws_iam_policy_document" "eventbridge_logs" {
  statement {
    effect = "Allow"
    principals {
      type        = "Service"
      identifiers = ["events.amazonaws.com"]
    }
    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]
    resources = [
      "${aws_cloudwatch_log_group.mission_status_changed.arn}:*",
      "${aws_cloudwatch_log_group.backup_requested.arn}:*",
      "${aws_cloudwatch_log_group.impact_level_updated.arn}:*",
    ]
  }
}

resource "aws_cloudwatch_log_resource_policy" "eventbridge_logs" {
  policy_name     = "${var.project_name}-eventbridge-logs"
  policy_document = data.aws_iam_policy_document.eventbridge_logs.json
}

# ---------------------------------------------------
# Scheduled Rule: Outbox Processor (every 1 minute)
# ---------------------------------------------------
resource "aws_cloudwatch_event_rule" "outbox_processor_schedule" {
  name                = "${var.project_name}-outbox-processor-schedule"
  description         = "Trigger outbox-processor Lambda every 1 minute"
  schedule_expression = "rate(1 minute)"

  tags = {
    Project = var.project_name
  }
}

resource "aws_cloudwatch_event_target" "outbox_processor_target" {
  rule      = aws_cloudwatch_event_rule.outbox_processor_schedule.name
  target_id = "outbox-processor-lambda"
  arn       = aws_lambda_function.outbox_processor.arn
}

# ---------------------------------------------------
# SNS Subscription: ManageDispatch → mission-assigned-handler Lambda
# Topic: request-dispatch-v1 (ARN: var.dispatch_sns_topic_arn)
# ---------------------------------------------------
resource "aws_sns_topic_subscription" "dispatch_order_created" {
  count     = var.dispatch_sns_topic_arn != "" ? 1 : 0
  topic_arn = var.dispatch_sns_topic_arn
  protocol  = "lambda"
  endpoint  = aws_lambda_function.mission_assigned_handler.arn

  filter_policy = jsonencode({
    messageType = ["DispatchOrderCreated"]
  })
}

# ---------------------------------------------------
# Lambda Target: MissionStatusChanged → IncidentTracking (direct)
# ---------------------------------------------------
resource "aws_cloudwatch_event_target" "mission_status_changed_incident_lambda" {
  count          = var.incident_tracking_lambda_arn != "" ? 1 : 0
  rule           = aws_cloudwatch_event_rule.mission_status_changed.name
  event_bus_name = aws_cloudwatch_event_bus.mission_events.name
  target_id      = "mission-status-changed-incident-lambda"
  arn            = var.incident_tracking_lambda_arn
}

resource "aws_lambda_permission" "allow_eventbridge_incident_status_changed" {
  count         = var.incident_tracking_lambda_arn != "" ? 1 : 0
  statement_id  = "AllowEventBridgeIncidentStatusChanged"
  action        = "lambda:InvokeFunction"
  function_name = var.incident_tracking_lambda_arn
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.mission_status_changed.arn
}

# ---------------------------------------------------
# NOTE: Dispatch RESOLVED path — now uses sync PATCH (no SQS/EventBridge routing needed)
# ---------------------------------------------------

# ---------------------------------------------------
# SQS Target: MissionStatusChanged → RescueRequest
# ---------------------------------------------------
# SQS target (deprecated — RescueRequest เปลี่ยนเป็น EventBridge cross-account bus)
# resource "aws_cloudwatch_event_target" "mission_status_changed_rescue_request_sqs" { ... }

resource "aws_cloudwatch_event_target" "mission_status_changed_rescue_request_bus" {
  count          = var.rescue_request_event_bus_arn != "" ? 1 : 0
  rule           = aws_cloudwatch_event_rule.mission_status_changed.name
  event_bus_name = aws_cloudwatch_event_bus.mission_events.name
  target_id      = "mission-status-changed-rescue-request-bus"
  arn            = var.rescue_request_event_bus_arn
  role_arn       = local.lab_role_arn
}

# ---------------------------------------------------
# SQS Targets: MissionBackupRequested → Prioritization
# ---------------------------------------------------
resource "aws_cloudwatch_event_target" "backup_requested_prioritization_sqs" {
  count          = var.prioritization_sqs_arn != "" ? 1 : 0
  rule           = aws_cloudwatch_event_rule.backup_requested.name
  event_bus_name = aws_cloudwatch_event_bus.mission_events.name
  target_id      = "backup-requested-prioritization-sqs"
  arn            = var.prioritization_sqs_arn
}

# ---------------------------------------------------
# Lambda Targets: ImpactLevelUpdated → IncidentTracking (direct) + Prioritization (SQS)
# ---------------------------------------------------
resource "aws_cloudwatch_event_target" "impact_level_updated_incident_lambda" {
  count          = var.incident_tracking_lambda_arn != "" ? 1 : 0
  rule           = aws_cloudwatch_event_rule.impact_level_updated.name
  event_bus_name = aws_cloudwatch_event_bus.mission_events.name
  target_id      = "impact-level-updated-incident-lambda"
  arn            = var.incident_tracking_lambda_arn
}

resource "aws_lambda_permission" "allow_eventbridge_incident_impact_updated" {
  count         = var.incident_tracking_lambda_arn != "" ? 1 : 0
  statement_id  = "AllowEventBridgeIncidentImpactUpdated"
  action        = "lambda:InvokeFunction"
  function_name = var.incident_tracking_lambda_arn
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.impact_level_updated.arn
}

resource "aws_cloudwatch_event_target" "impact_level_updated_prioritization_sqs" {
  count          = var.prioritization_sqs_arn != "" ? 1 : 0
  rule           = aws_cloudwatch_event_rule.impact_level_updated.name
  event_bus_name = aws_cloudwatch_event_bus.mission_events.name
  target_id      = "impact-level-updated-prioritization-sqs"
  arn            = var.prioritization_sqs_arn
}

# ---------------------------------------------------
# NOTE: Lambda & SQS Resource Policies
# IncidentTracking: EventBridge invokes Lambda directly
#   → aws_lambda_permission managed above (AllowEventBridgeIncident*)
# Dispatch / Prioritization / RescueRequest: use SQS — teams must add EventBridge permission on their queue:
#   Principal: events.amazonaws.com
#   Action: sqs:SendMessage
#   Source ARN: aws_cloudwatch_event_rule.mission_status_changed.arn (for RescueRequest)
# ---------------------------------------------------
