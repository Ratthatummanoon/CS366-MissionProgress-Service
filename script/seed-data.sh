#!/bin/bash
set -e

REGION="${AWS_REGION:-us-east-1}"

# Flood scenario location: Hat Yai, Songkhla (~7.00°N, 100.47°E)
# Sub-areas used in timeline GPS entries:
#   ควนลัง (Khuan Lang):    7.0130,100.4370
#   ตลาดใหม่ (Talat Mai):   7.0066,100.4730
#   ม.หาดใหญ่ (Haad Yai):   6.9954,100.4802
#   สะพานป่าชาด:             7.0150,100.4610

echo "=== Seeding DynamoDB with sample data ==="

# ---------------------------------------------------
# MissionAssignment records
# ---------------------------------------------------
echo "--- Inserting MissionAssignment records ---"

# MSN-DEMO: DISPATCHED — ใช้สำหรับ live demo walkthrough (จุดเริ่มต้น)
aws dynamodb put-item --region "$REGION" --table-name MissionAssignment --item '{
  "mission_id":          {"S": "MSN-DEMO"},
  "dispatch_id":         {"S": "DSP-DEMO"},
  "request_id":          {"S": "REQ-DEMO"},
  "incident_id":         {"S": "INC-DEMO"},
  "rescue_team_id":      {"S": "TEAM-ALPHA"},
  "priority_level":      {"N": "3"},
  "current_status":      {"S": "DISPATCHED"},
  "latest_impact_level": {"N": "2"},
  "started_at":          {"S": "2025-05-18T08:00:00Z"},
  "last_updated_at":     {"S": "2025-05-18T08:00:00Z"}
}'
echo "  Inserted MSN-DEMO (DISPATCHED) — สำหรับ live demo"

# MSN-002: EN_ROUTE
aws dynamodb put-item --region "$REGION" --table-name MissionAssignment --item '{
  "mission_id":          {"S": "MSN-002"},
  "dispatch_id":         {"S": "DSP-00002"},
  "request_id":          {"S": "REQ-002"},
  "incident_id":         {"S": "INC-001"},
  "rescue_team_id":      {"S": "TEAM-BRAVO"},
  "priority_level":      {"N": "2"},
  "current_status":      {"S": "EN_ROUTE"},
  "latest_impact_level": {"N": "2"},
  "started_at":          {"S": "2025-05-18T07:00:00Z"},
  "last_updated_at":     {"S": "2025-05-18T07:10:00Z"}
}'
echo "  Inserted MSN-002 (EN_ROUTE)"

# MSN-003: ON_SITE
aws dynamodb put-item --region "$REGION" --table-name MissionAssignment --item '{
  "mission_id":          {"S": "MSN-003"},
  "dispatch_id":         {"S": "DSP-00003"},
  "request_id":          {"S": "REQ-003"},
  "incident_id":         {"S": "INC-002"},
  "rescue_team_id":      {"S": "TEAM-CHARLIE"},
  "priority_level":      {"N": "3"},
  "current_status":      {"S": "ON_SITE"},
  "latest_impact_level": {"N": "3"},
  "started_at":          {"S": "2025-05-18T06:30:00Z"},
  "last_updated_at":     {"S": "2025-05-18T07:45:00Z"}
}'
echo "  Inserted MSN-003 (ON_SITE)"

# MSN-004: NEED_BACKUP
aws dynamodb put-item --region "$REGION" --table-name MissionAssignment --item '{
  "mission_id":          {"S": "MSN-004"},
  "dispatch_id":         {"S": "DSP-00004"},
  "request_id":          {"S": "REQ-004"},
  "incident_id":         {"S": "INC-003"},
  "rescue_team_id":      {"S": "TEAM-DELTA"},
  "priority_level":      {"N": "4"},
  "current_status":      {"S": "NEED_BACKUP"},
  "latest_impact_level": {"N": "4"},
  "started_at":          {"S": "2025-05-18T05:00:00Z"},
  "last_updated_at":     {"S": "2025-05-18T07:30:00Z"}
}'
echo "  Inserted MSN-004 (NEED_BACKUP)"

# MSN-005: RESOLVED — full lifecycle ใช้แสดง timeline สมบูรณ์
aws dynamodb put-item --region "$REGION" --table-name MissionAssignment --item '{
  "mission_id":          {"S": "MSN-005"},
  "dispatch_id":         {"S": "DSP-00005"},
  "request_id":          {"S": "REQ-005"},
  "incident_id":         {"S": "INC-004"},
  "rescue_team_id":      {"S": "TEAM-ECHO"},
  "priority_level":      {"N": "3"},
  "current_status":      {"S": "RESOLVED"},
  "latest_impact_level": {"N": "1"},
  "started_at":          {"S": "2025-05-18T04:00:00Z"},
  "last_updated_at":     {"S": "2025-05-18T07:00:00Z"}
}'
echo "  Inserted MSN-005 (RESOLVED)"

# ---------------------------------------------------
# MissionTimeline records
# ---------------------------------------------------
echo ""
echo "--- Inserting MissionTimeline records ---"

# ---- MSN-DEMO: DISPATCHED only (starting point for demo) ----
aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id":   {"S": "MSN-DEMO"},
  "timestamp":    {"S": "2025-05-18T08:00:00Z"},
  "log_id":       {"S": "LOG-D01"},
  "action_type":  {"S": "STATUS_CHANGE"},
  "description":  {"S": "ภารกิจถูกส่งให้ TEAM-ALPHA"},
  "performed_by": {"S": "SYSTEM"},
  "old_status":   {"S": ""},
  "new_status":   {"S": "DISPATCHED"}
}'
echo "  Inserted timeline for MSN-DEMO"

# ---- MSN-002: DISPATCHED → EN_ROUTE ----
aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id":   {"S": "MSN-002"},
  "timestamp":    {"S": "2025-05-18T07:00:00Z"},
  "log_id":       {"S": "LOG-B01"},
  "action_type":  {"S": "STATUS_CHANGE"},
  "description":  {"S": "ภารกิจถูกส่งให้ TEAM-BRAVO"},
  "performed_by": {"S": "SYSTEM"},
  "old_status":   {"S": ""},
  "new_status":   {"S": "DISPATCHED"}
}'
aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id":   {"S": "MSN-002"},
  "timestamp":    {"S": "2025-05-18T07:10:00Z"},
  "log_id":       {"S": "LOG-B02"},
  "action_type":  {"S": "STATUS_CHANGE"},
  "description":  {"S": "ทีมออกเดินทางแล้ว"},
  "performed_by": {"S": "TEAM-BRAVO"},
  "old_status":   {"S": "DISPATCHED"},
  "new_status":   {"S": "EN_ROUTE"},
  "gps_location": {"S": "7.0130,100.4370"}
}'
echo "  Inserted timeline for MSN-002"

# ---- MSN-003: DISPATCHED → EN_ROUTE → ON_SITE ----
aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id":   {"S": "MSN-003"},
  "timestamp":    {"S": "2025-05-18T06:30:00Z"},
  "log_id":       {"S": "LOG-C01"},
  "action_type":  {"S": "STATUS_CHANGE"},
  "description":  {"S": "ภารกิจถูกส่งให้ TEAM-CHARLIE"},
  "performed_by": {"S": "SYSTEM"},
  "old_status":   {"S": ""},
  "new_status":   {"S": "DISPATCHED"}
}'
aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id":   {"S": "MSN-003"},
  "timestamp":    {"S": "2025-05-18T06:45:00Z"},
  "log_id":       {"S": "LOG-C02"},
  "action_type":  {"S": "STATUS_CHANGE"},
  "description":  {"S": "ทีมออกเดินทางแล้ว"},
  "performed_by": {"S": "TEAM-CHARLIE"},
  "old_status":   {"S": "DISPATCHED"},
  "new_status":   {"S": "EN_ROUTE"},
  "gps_location": {"S": "7.0150,100.4610"}
}'
aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id":   {"S": "MSN-003"},
  "timestamp":    {"S": "2025-05-18T07:45:00Z"},
  "log_id":       {"S": "LOG-C03"},
  "action_type":  {"S": "STATUS_CHANGE"},
  "description":  {"S": "ถึงจุดเกิดเหตุ น้ำระดับเอว ประเมินสถานการณ์"},
  "performed_by": {"S": "TEAM-CHARLIE"},
  "old_status":   {"S": "EN_ROUTE"},
  "new_status":   {"S": "ON_SITE"},
  "gps_location": {"S": "7.0066,100.4730"},
  "note":         {"S": "น้ำสูงประมาณ 80 ซม. ผู้ประสบภัย 3 คน ยังติดอยู่ในบ้าน"}
}'
echo "  Inserted timeline for MSN-003"

# ---- MSN-004: DISPATCHED → EN_ROUTE → ON_SITE → NEED_BACKUP ----
aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id":   {"S": "MSN-004"},
  "timestamp":    {"S": "2025-05-18T05:00:00Z"},
  "log_id":       {"S": "LOG-D01"},
  "action_type":  {"S": "STATUS_CHANGE"},
  "description":  {"S": "ภารกิจถูกส่งให้ TEAM-DELTA"},
  "performed_by": {"S": "SYSTEM"},
  "old_status":   {"S": ""},
  "new_status":   {"S": "DISPATCHED"}
}'
aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id":   {"S": "MSN-004"},
  "timestamp":    {"S": "2025-05-18T05:15:00Z"},
  "log_id":       {"S": "LOG-D02"},
  "action_type":  {"S": "STATUS_CHANGE"},
  "description":  {"S": "ทีมออกเดินทางแล้ว"},
  "performed_by": {"S": "TEAM-DELTA"},
  "old_status":   {"S": "DISPATCHED"},
  "new_status":   {"S": "EN_ROUTE"},
  "gps_location": {"S": "7.0130,100.4370"}
}'
aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id":   {"S": "MSN-004"},
  "timestamp":    {"S": "2025-05-18T06:10:00Z"},
  "log_id":       {"S": "LOG-D03"},
  "action_type":  {"S": "STATUS_CHANGE"},
  "description":  {"S": "ถึงจุดเกิดเหตุ"},
  "performed_by": {"S": "TEAM-DELTA"},
  "old_status":   {"S": "EN_ROUTE"},
  "new_status":   {"S": "ON_SITE"},
  "gps_location": {"S": "6.9954,100.4802"}
}'
aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id":   {"S": "MSN-004"},
  "timestamp":    {"S": "2025-05-18T07:30:00Z"},
  "log_id":       {"S": "LOG-D04"},
  "action_type":  {"S": "STATUS_CHANGE"},
  "description":  {"S": "สถานการณ์รุนแรงกว่าที่คาด ขอกำลังเสริม บ้านเริ่มพัง"},
  "performed_by": {"S": "TEAM-DELTA"},
  "old_status":   {"S": "ON_SITE"},
  "new_status":   {"S": "NEED_BACKUP"},
  "gps_location": {"S": "6.9954,100.4802"},
  "note":         {"S": "ต้องการเรือเพิ่มอีก 2 ลำ มีผู้ป่วยติดค้าง 1 ราย"}
}'
echo "  Inserted timeline for MSN-004"

# ---- MSN-005: full lifecycle DISPATCHED → EN_ROUTE → ON_SITE → NEED_BACKUP → ON_SITE → RESOLVED ----
aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id":   {"S": "MSN-005"},
  "timestamp":    {"S": "2025-05-18T04:00:00Z"},
  "log_id":       {"S": "LOG-E01"},
  "action_type":  {"S": "STATUS_CHANGE"},
  "description":  {"S": "ภารกิจถูกส่งให้ TEAM-ECHO"},
  "performed_by": {"S": "SYSTEM"},
  "old_status":   {"S": ""},
  "new_status":   {"S": "DISPATCHED"}
}'
aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id":   {"S": "MSN-005"},
  "timestamp":    {"S": "2025-05-18T04:10:00Z"},
  "log_id":       {"S": "LOG-E02"},
  "action_type":  {"S": "STATUS_CHANGE"},
  "description":  {"S": "ทีมออกเดินทางแล้ว"},
  "performed_by": {"S": "TEAM-ECHO"},
  "old_status":   {"S": "DISPATCHED"},
  "new_status":   {"S": "EN_ROUTE"},
  "gps_location": {"S": "7.0130,100.4370"}
}'
aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id":   {"S": "MSN-005"},
  "timestamp":    {"S": "2025-05-18T04:55:00Z"},
  "log_id":       {"S": "LOG-E03"},
  "action_type":  {"S": "STATUS_CHANGE"},
  "description":  {"S": "ถึงจุดเกิดเหตุ"},
  "performed_by": {"S": "TEAM-ECHO"},
  "old_status":   {"S": "EN_ROUTE"},
  "new_status":   {"S": "ON_SITE"},
  "gps_location": {"S": "7.0066,100.4730"}
}'
aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id":   {"S": "MSN-005"},
  "timestamp":    {"S": "2025-05-18T05:20:00Z"},
  "log_id":       {"S": "LOG-E04"},
  "action_type":  {"S": "STATUS_CHANGE"},
  "description":  {"S": "ขอกำลังเสริม พบผู้ป่วยโรคหัวใจต้องส่งโรงพยาบาล"},
  "performed_by": {"S": "TEAM-ECHO"},
  "old_status":   {"S": "ON_SITE"},
  "new_status":   {"S": "NEED_BACKUP"},
  "gps_location": {"S": "7.0066,100.4730"},
  "note":         {"S": "ผู้ป่วย 1 ราย อาการหนัก ต้องการรถพยาบาล"}
}'
aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id":   {"S": "MSN-005"},
  "timestamp":    {"S": "2025-05-18T05:50:00Z"},
  "log_id":       {"S": "LOG-E05"},
  "action_type":  {"S": "STATUS_CHANGE"},
  "description":  {"S": "กำลังเสริมมาถึง ส่งผู้ป่วยแล้ว กลับมาช่วยผู้ประสบภัยต่อ"},
  "performed_by": {"S": "TEAM-ECHO"},
  "old_status":   {"S": "NEED_BACKUP"},
  "new_status":   {"S": "ON_SITE"},
  "gps_location": {"S": "7.0066,100.4730"}
}'
aws dynamodb put-item --region "$REGION" --table-name MissionTimeline --item '{
  "mission_id":   {"S": "MSN-005"},
  "timestamp":    {"S": "2025-05-18T07:00:00Z"},
  "log_id":       {"S": "LOG-E06"},
  "action_type":  {"S": "STATUS_CHANGE"},
  "description":  {"S": "ภารกิจสำเร็จ อพยพผู้ประสบภัยครบทุกคนแล้ว"},
  "performed_by": {"S": "TEAM-ECHO"},
  "old_status":   {"S": "ON_SITE"},
  "new_status":   {"S": "RESOLVED"},
  "gps_location": {"S": "7.0066,100.4730"},
  "note":         {"S": "อพยพผู้ประสบภัย 4 คน ส่งศูนย์พักพิงโรงเรียนเทศบาล 1"}
}'
echo "  Inserted timeline for MSN-005"

# ---------------------------------------------------
# EventOutbox record — แสดง pending retry (Outbox Pattern demo)
# ---------------------------------------------------
echo ""
echo "--- Inserting EventOutbox record (pending retry demo) ---"

aws dynamodb put-item --region "$REGION" --table-name EventOutbox --item '{
  "outbox_id":      {"S": "OBX-001"},
  "created_at":     {"S": "2025-05-18T07:30:05Z"},
  "event_type":     {"S": "MissionBackupRequested"},
  "event_payload":  {"S": "{\"schema_version\":\"1.0\",\"mission_id\":\"MSN-004\",\"incident_id\":\"INC-003\",\"rescue_team_id\":\"TEAM-DELTA\",\"requested_at\":\"2025-05-18T07:30:00Z\",\"requested_by\":\"TEAM-DELTA\",\"location\":\"6.9954,100.4802\"}"},
  "status":         {"S": "PENDING"},
  "retry_count":    {"N": "2"},
  "last_error":     {"S": "EventBridge PutEvents: RequestError: send request failed"}
}'
echo "  Inserted OBX-001 (PENDING — MissionBackupRequested retry)"

echo ""
echo "=== Seed data complete ==="
echo ""
echo "Missions inserted:"
echo "  MSN-DEMO  DISPATCHED  TEAM-ALPHA  (REQ-DEMO)   ← ใช้สำหรับ live demo"
echo "  MSN-002   EN_ROUTE    TEAM-BRAVO  (REQ-002)"
echo "  MSN-003   ON_SITE     TEAM-CHARLIE(REQ-003)"
echo "  MSN-004   NEED_BACKUP TEAM-DELTA  (REQ-004)"
echo "  MSN-005   RESOLVED    TEAM-ECHO   (REQ-005)    ← timeline สมบูรณ์ 6 entries"
