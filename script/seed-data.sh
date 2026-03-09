#!/bin/bash
set -e

REGION="${AWS_REGION:-us-east-1}"

echo "=== Seeding DynamoDB with sample data ==="

# ---------------------------------------------------
# MissionAssignment records
# ---------------------------------------------------
echo "--- Inserting MissionAssignment records ---"

# Mission 1: DISPATCHED
aws dynamodb put-item --region "$REGION" --table-name MissionAssignment --item '{
  "mission_id": {"S": "MSN-001"},
  "incident_id": {"S": "INC-001"},
  "rescue_team_id": {"S": "TEAM-ALPHA"},
  "current_status": {"S": "DISPATCHED"},
  "latest_impact_level": {"N": "2"},
  "started_at": {"S": "2024-12-01T08:00:00Z"},
  "last_updated_at": {"S": "2024-12-01T08:00:00Z"}
}'
echo "  Inserted MSN-001 (DISPATCHED)"

# Mission 2: EN_ROUTE
aws dynamodb put-item --region "$REGION" --table-name MissionAssignment --item '{
  "mission_id": {"S": "MSN-002"},
  "incident_id": {"S": "INC-002"},
  "rescue_team_id": {"S": "TEAM-BRAVO"},
  "current_status": {"S": "EN_ROUTE"},
  "latest_impact_level": {"N": "3"},
  "started_at": {"S": "2024-12-01T09:00:00Z"},
  "last_updated_at": {"S": "2024-12-01T09:15:00Z"}
}'
echo "  Inserted MSN-002 (EN_ROUTE)"

# Mission 3: ON_SITE
aws dynamodb put-item --region "$REGION" --table-name MissionAssignment --item '{
  "mission_id": {"S": "MSN-003"},
  "incident_id": {"S": "INC-003"},
  "rescue_team_id": {"S": "TEAM-CHARLIE"},
  "current_status": {"S": "ON_SITE"},
  "latest_impact_level": {"N": "4"},
  "started_at": {"S": "2024-12-01T07:30:00Z"},
  "last_updated_at": {"S": "2024-12-01T10:00:00Z"}
}'
echo "  Inserted MSN-003 (ON_SITE)"

# Mission 4: NEED_BACKUP
aws dynamodb put-item --region "$REGION" --table-name MissionAssignment --item '{
  "mission_id": {"S": "MSN-004"},
  "incident_id": {"S": "INC-004"},
  "rescue_team_id": {"S": "TEAM-DELTA"},
  "current_status": {"S": "NEED_BACKUP"},
  "latest_impact_level": {"N": "4"},
  "started_at": {"S": "2024-12-01T06:00:00Z"},
  "last_updated_at": {"S": "2024-12-01T11:00:00Z"}
}'
echo "  Inserted MSN-004 (NEED_BACKUP)"

# Mission 5: RESOLVED
aws dynamodb put-item --region "$REGION" --table-name MissionAssignment --item '{
  "mission_id": {"S": "MSN-005"},
  "incident_id": {"S": "INC-005"},
  "rescue_team_id": {"S": "TEAM-ECHO"},
  "current_status": {"S": "RESOLVED"},
  "latest_impact_level": {"N": "1"},
  "started_at": {"S": "2024-12-01T05:00:00Z"},
  "last_updated_at": {"S": "2024-12-01T08:30:00Z"}
}'
echo "  Inserted MSN-005 (RESOLVED)"

# ---------------------------------------------------
# MissionTimeline records
# ---------------------------------------------------
echo ""
echo "--- Inserting MissionTimeline records ---"

# Timeline for MSN-001 (DISPATCHED)
aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id": {"S": "MSN-001"},
  "timestamp": {"S": "2024-12-01T08:00:00Z"},
  "log_id": {"S": "LOG-001"},
  "action_type": {"S": "STATUS_CHANGE"},
  "description": {"S": "Mission dispatched to TEAM-ALPHA"},
  "performed_by": {"S": "SYSTEM"}
}'
echo "  Inserted timeline for MSN-001"

# Timeline for MSN-002 (DISPATCHED → EN_ROUTE)
aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id": {"S": "MSN-002"},
  "timestamp": {"S": "2024-12-01T09:00:00Z"},
  "log_id": {"S": "LOG-002"},
  "action_type": {"S": "STATUS_CHANGE"},
  "description": {"S": "Mission dispatched to TEAM-BRAVO"},
  "performed_by": {"S": "SYSTEM"}
}'

aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id": {"S": "MSN-002"},
  "timestamp": {"S": "2024-12-01T09:15:00Z"},
  "log_id": {"S": "LOG-003"},
  "action_type": {"S": "STATUS_CHANGE"},
  "description": {"S": "Team en route to incident location"},
  "performed_by": {"S": "TEAM-BRAVO"},
  "gps_location": {"S": "13.7563,100.5018"}
}'
echo "  Inserted timeline for MSN-002"

# Timeline for MSN-003 (DISPATCHED → EN_ROUTE → ON_SITE)
aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id": {"S": "MSN-003"},
  "timestamp": {"S": "2024-12-01T07:30:00Z"},
  "log_id": {"S": "LOG-004"},
  "action_type": {"S": "STATUS_CHANGE"},
  "description": {"S": "Mission dispatched to TEAM-CHARLIE"},
  "performed_by": {"S": "SYSTEM"}
}'

aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id": {"S": "MSN-003"},
  "timestamp": {"S": "2024-12-01T08:30:00Z"},
  "log_id": {"S": "LOG-005"},
  "action_type": {"S": "STATUS_CHANGE"},
  "description": {"S": "Team en route"},
  "performed_by": {"S": "TEAM-CHARLIE"},
  "gps_location": {"S": "13.7400,100.5200"}
}'

aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id": {"S": "MSN-003"},
  "timestamp": {"S": "2024-12-01T10:00:00Z"},
  "log_id": {"S": "LOG-006"},
  "action_type": {"S": "STATUS_CHANGE"},
  "description": {"S": "Arrived on site, assessing situation"},
  "performed_by": {"S": "TEAM-CHARLIE"},
  "gps_location": {"S": "13.7380,100.5230"}
}'
echo "  Inserted timeline for MSN-003"

# Timeline for MSN-004 (multiple states)
aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id": {"S": "MSN-004"},
  "timestamp": {"S": "2024-12-01T06:00:00Z"},
  "log_id": {"S": "LOG-007"},
  "action_type": {"S": "STATUS_CHANGE"},
  "description": {"S": "Mission dispatched to TEAM-DELTA"},
  "performed_by": {"S": "SYSTEM"}
}'

aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id": {"S": "MSN-004"},
  "timestamp": {"S": "2024-12-01T11:00:00Z"},
  "log_id": {"S": "LOG-008"},
  "action_type": {"S": "STATUS_CHANGE"},
  "description": {"S": "Requesting backup - situation escalated, multiple casualties"},
  "performed_by": {"S": "TEAM-DELTA"},
  "gps_location": {"S": "13.7200,100.4800"}
}'
echo "  Inserted timeline for MSN-004"

# Timeline for MSN-005 (RESOLVED)
aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id": {"S": "MSN-005"},
  "timestamp": {"S": "2024-12-01T05:00:00Z"},
  "log_id": {"S": "LOG-009"},
  "action_type": {"S": "STATUS_CHANGE"},
  "description": {"S": "Mission dispatched to TEAM-ECHO"},
  "performed_by": {"S": "SYSTEM"}
}'

aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id": {"S": "MSN-005"},
  "timestamp": {"S": "2024-12-01T08:30:00Z"},
  "log_id": {"S": "LOG-010"},
  "action_type": {"S": "STATUS_CHANGE"},
  "description": {"S": "Incident resolved, all victims evacuated safely"},
  "performed_by": {"S": "TEAM-ECHO"},
  "gps_location": {"S": "13.7100,100.5100"}
}'
echo "  Inserted timeline for MSN-005"

echo ""
echo "=== Seed data complete ==="
