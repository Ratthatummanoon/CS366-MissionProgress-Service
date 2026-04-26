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
# Rule: MissionAssignedEvent from Dispatch → mission-assigned-handler
# ---------------------------------------------------
resource "aws_cloudwatch_event_rule" "mission_assigned" {
  name        = "mission-assigned-rule"
  description = "Capture DispatchOrderCreated from Manage Dispatch Service"

  event_pattern = jsonencode({
    source      = ["ManageDispatchService"]
    detail-type = ["DispatchOrderCreated"]
  })

  tags = {
    Project = var.project_name
  }
}

resource "aws_cloudwatch_event_target" "mission_assigned_handler" {
  rule      = aws_cloudwatch_event_rule.mission_assigned.name
  target_id = "mission-assigned-handler-lambda"
  arn       = aws_lambda_function.mission_assigned_handler.arn
}

# ---------------------------------------------------
# SQS Targets: MissionStatusChanged → IncidentTracking
# ---------------------------------------------------
resource "aws_cloudwatch_event_target" "mission_status_changed_incident_sqs" {
  count          = var.incident_tracking_sqs_arn != "" ? 1 : 0
  rule           = aws_cloudwatch_event_rule.mission_status_changed.name
  event_bus_name = aws_cloudwatch_event_bus.mission_events.name
  target_id      = "mission-status-changed-incident-sqs"
  arn            = var.incident_tracking_sqs_arn
}

# ---------------------------------------------------
# Rule: MissionStatusChanged (RESOLVED only) → Dispatch SQS
# ---------------------------------------------------
resource "aws_cloudwatch_event_rule" "mission_resolved_dispatch" {
  count          = var.dispatch_sqs_arn != "" ? 1 : 0
  name           = "mission-resolved-dispatch-rule"
  event_bus_name = aws_cloudwatch_event_bus.mission_events.name
  description    = "Route RESOLVED status to Dispatch service"

  event_pattern = jsonencode({
    source      = ["MissionProgressService"]
    detail-type = ["MissionStatusChanged"]
    detail = {
      new_status = ["RESOLVED"]
    }
  })

  tags = {
    Project = var.project_name
  }
}

resource "aws_cloudwatch_event_target" "mission_resolved_dispatch_sqs" {
  count          = var.dispatch_sqs_arn != "" ? 1 : 0
  rule           = aws_cloudwatch_event_rule.mission_resolved_dispatch[0].name
  event_bus_name = aws_cloudwatch_event_bus.mission_events.name
  target_id      = "mission-resolved-dispatch-sqs"
  arn            = var.dispatch_sqs_arn
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
# SQS Targets: ImpactLevelUpdated → IncidentTracking + Prioritization
# ---------------------------------------------------
resource "aws_cloudwatch_event_target" "impact_level_updated_incident_sqs" {
  count          = var.incident_tracking_sqs_arn != "" ? 1 : 0
  rule           = aws_cloudwatch_event_rule.impact_level_updated.name
  event_bus_name = aws_cloudwatch_event_bus.mission_events.name
  target_id      = "impact-level-updated-incident-sqs"
  arn            = var.incident_tracking_sqs_arn
}

resource "aws_cloudwatch_event_target" "impact_level_updated_prioritization_sqs" {
  count          = var.prioritization_sqs_arn != "" ? 1 : 0
  rule           = aws_cloudwatch_event_rule.impact_level_updated.name
  event_bus_name = aws_cloudwatch_event_bus.mission_events.name
  target_id      = "impact-level-updated-prioritization-sqs"
  arn            = var.prioritization_sqs_arn
}

# ---------------------------------------------------
# NOTE: SQS Resource Policies
# Consumer teams must add EventBridge permission on their SQS queues:
#   Principal: events.amazonaws.com
#   Action: sqs:SendMessage
# ---------------------------------------------------
