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
