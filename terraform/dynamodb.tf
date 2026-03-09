# ---------------------------------------------------
# DynamoDB Table: MissionAssignment
# ---------------------------------------------------
resource "aws_dynamodb_table" "mission_assignment" {
  name         = "MissionAssignment"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "mission_id"

  attribute {
    name = "mission_id"
    type = "S"
  }

  attribute {
    name = "incident_id"
    type = "S"
  }

  attribute {
    name = "rescue_team_id"
    type = "S"
  }

  global_secondary_index {
    name            = "incident-index"
    hash_key        = "incident_id"
    projection_type = "ALL"
  }

  global_secondary_index {
    name            = "team-index"
    hash_key        = "rescue_team_id"
    projection_type = "ALL"
  }

  tags = {
    Project = var.project_name
  }
}

# ---------------------------------------------------
# DynamoDB Table: MissionTimeline
# ---------------------------------------------------
resource "aws_dynamodb_table" "mission_timeline" {
  name         = "MissionTimeline"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "mission_id"
  range_key    = "timestamp"

  attribute {
    name = "mission_id"
    type = "S"
  }

  attribute {
    name = "timestamp"
    type = "S"
  }

  attribute {
    name = "log_id"
    type = "S"
  }

  global_secondary_index {
    name            = "log-id-index"
    hash_key        = "log_id"
    projection_type = "ALL"
  }

  tags = {
    Project = var.project_name
  }
}

# ---------------------------------------------------
# DynamoDB Table: EventOutbox
# ---------------------------------------------------
resource "aws_dynamodb_table" "event_outbox" {
  name         = "EventOutbox"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "outbox_id"
  range_key    = "created_at"

  attribute {
    name = "outbox_id"
    type = "S"
  }

  attribute {
    name = "created_at"
    type = "S"
  }

  attribute {
    name = "status"
    type = "S"
  }

  global_secondary_index {
    name            = "status-index"
    hash_key        = "status"
    projection_type = "ALL"
  }

  ttl {
    attribute_name = "ttl"
    enabled        = true
  }

  tags = {
    Project = var.project_name
  }
}
