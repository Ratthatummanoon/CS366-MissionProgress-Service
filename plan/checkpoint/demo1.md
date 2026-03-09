# Demo 1 Plan — MissionProgress Service MVP

## Overview

สร้าง MissionProgress Service MVP บน AWS Learner Lab ที่สาธิตได้ 2 ฟังก์ชันหลัก:

1. **GET /incidents/{id}** (Synchronous) — ดึงข้อมูลภารกิจ + Timeline จาก DynamoDB + mock call ไป IncidentTracking Service (degraded mode เมื่อเรียกไม่สำเร็จ)
2. **POST /incidents/{id}/progress** (Sync + Async) — อัปเดตสถานะใน DynamoDB (sync) + publish `MissionStatusChanged`, `MissionBackupRequested`, `ImpactLevelUpdated` events ไป EventBridge (async) พร้อม Outbox fallback

Deploy ด้วย Terraform ทั้งหมด, demo ผ่าน curl/Postman ในคาบเรียน

---

## ฟังก์ชันที่จะสาธิต

| #   | Function                          | Type         | Demo จุดเด่น                                                                     |
| --- | --------------------------------- | ------------ | -------------------------------------------------------------------------------- |
| 1   | **GET /incidents/{id}**           | Synchronous  | ดึง mission + timeline จาก DynamoDB + mock call IncidentTracking (degraded mode) |
| 2   | **POST /incidents/{id}/progress** | Sync + Async | อัปเดต DynamoDB (sync) + publish EventBridge events (async) + Outbox fallback    |

**Mock data/function:** GET /incidents/{id} เรียก IncidentTracking Service ของเพื่อนผ่าน HTTP Client — ใน demo 1 ตั้ง `INCIDENT_SERVICE_URL` เป็น placeholder → timeout → fallback แสดงเฉพาะข้อมูลที่ตัวเองมี (`data_source: "partial"`)

---

## Implementation Steps

### Phase A: Project Foundation

> ไม่มี dependency ระหว่างกัน ทำขนานได้

**A1. Initialize Go Module** _(อ้างอิง implement_plan Step 1)_

- สร้าง `src/backend/go.mod`
- Module: `github.com/Ratthatummanoon/CS366-MissionProgress-Service`
- Dependencies: `aws-lambda-go`, `aws-sdk-go-v2` (dynamodb, eventbridge, config), `google/uuid`

**A2. Terraform Base Configuration** _(อ้างอิง implement_plan Steps 3, 10)_

- `terraform/main.tf` — AWS provider `us-east-1` + `data "aws_iam_role" "lab_role"` reference LabRole
- `terraform/variables.tf` — ตัวแปร: `aws_region`, `project_name`, `lab_role_arn`, `api_key_value`, `incident_service_url`
- `terraform/outputs.tf` — Output: API Gateway invoke URL, API Key value
- `terraform/iam.tf` — LabRole reference

---

### Phase B: Database Layer

> depends on A2

**B1. DynamoDB Tables** _(อ้างอิง implement_plan Step 4)_

- `terraform/dynamodb.tf` — สร้าง 3 tables:

| Table               | PK           | SK           | GSIs                                                          | Billing         |
| ------------------- | ------------ | ------------ | ------------------------------------------------------------- | --------------- |
| `MissionAssignment` | `mission_id` | —            | `incident-index` (incident_id), `team-index` (rescue_team_id) | PAY_PER_REQUEST |
| `MissionTimeline`   | `mission_id` | `timestamp`  | `log-id-index` (log_id)                                       | PAY_PER_REQUEST |
| `EventOutbox`       | `outbox_id`  | `created_at` | `status-index` (status) + TTL on `ttl`                        | PAY_PER_REQUEST |

---

### Phase C: Go Backend Code

> parallel with Phase B, depends on A1

**C1. Models** _(อ้างอิง implement_plan Step 5)_

- `src/backend/internal/models/` — 6 files:
  - `mission.go` — struct `MissionAssignment` ตาม DynamoDB schema
  - `timeline.go` — struct `TimelineEntry`
  - `events.go` — structs: `MissionStatusChangedEvent`, `MissionBackupRequestedEvent`, `ImpactLevelUpdatedEvent`
  - `outbox.go` — struct `OutboxEntry`
  - `requests.go` — structs: `ReportProgressRequest`, `ReportProgressResponse`, `GetMissionResponse`, `ErrorResponse`
  - `incident.go` — struct `IncidentDetail` (ข้อมูลจาก IncidentTracking Service)

**C2. State Machine** _(อ้างอิง implement_plan Step 6)_

- `src/backend/internal/statemachine/statemachine.go`
- Functions:
  - `IsValidTransition(from, to string) bool`
  - `ValidateStatus(status string) bool`
- Valid Transitions:
  ```
  DISPATCHED  → [EN_ROUTE]
  EN_ROUTE    → [ON_SITE]
  ON_SITE     → [NEED_BACKUP, RESOLVED]
  NEED_BACKUP → [ON_SITE, RESOLVED]
  ```

**C3. Repository Layer** _(อ้างอิง implement_plan Step 7)_

- `src/backend/internal/repository/` — 3 files:
  - `mission_repo.go` — `GetMissionByIncidentID()`, `UpdateMissionStatus()`, `CreateMission()`
  - `timeline_repo.go` — `AddTimelineEntry()`, `GetTimelineByMissionID()`
  - `outbox_repo.go` — `SaveOutboxEntry()` (save only, cron processor defer ไป demo 2+)

**C4. IncidentTracking HTTP Client (Mock-ready)** _(อ้างอิง implement_plan Step 7b)_

- `src/backend/internal/client/incident_client.go`
- `GetIncidentDetail(incidentID string) (*IncidentDetail, error)`
- Timeout: 3 วินาที, failure → return nil (degraded mode)
- Demo 1: ตั้ง `INCIDENT_SERVICE_URL` เป็น placeholder → timeout → auto-degrade

**C5. EventBridge Publisher + Outbox Fallback** _(อ้างอิง implement_plan Step 8)_

- `src/backend/internal/events/publisher.go`
- Event bus: `mission-progress-events`, Source: `MissionProgressService`
- Functions:
  - `PublishMissionStatusChanged()` — ทุกครั้งที่สถานะเปลี่ยน
  - `PublishBackupRequested()` — เมื่อ status = `NEED_BACKUP`
  - `PublishImpactLevelUpdated()` — เมื่อมี `new_impact_level`
- Failure → save to Outbox table (ไม่ fail request หลัก)

**C6. Response Helpers**

- `src/backend/internal/response/response.go` — JSON response + error response helpers

**C7. Lambda Handlers** _(อ้างอิง implement_plan Step 9a, 9b, 9d)_

- `src/backend/cmd/report-progress/main.go` — **POST handler:**
  1. Parse `incident_id` จาก path + `X-Rescue-Team-ID` จาก header
  2. Parse & validate request body
  3. Query mission by incident_id → 404 ถ้าไม่พบ
  4. Validate state transition → 400 ถ้าผิด
  5. Update mission status ใน DynamoDB
  6. เพิ่ม Timeline entry
  7. Publish events (+ Outbox fallback)
  8. Return 200

- `src/backend/cmd/get-mission/main.go` — **GET handler:**
  1. Parse `incident_id` จาก path
  2. Query mission → 404 ถ้าไม่พบ
  3. เรียก IncidentTracking Service (mock/degraded)
  4. Query timeline entries sorted by timestamp
  5. Return combined response (full/partial data_source)

- `src/backend/cmd/authorizer/main.go` — **Lambda Authorizer:**
  1. ตรวจ `x-api-key` ตรงกับ `VALID_API_KEY` env var
  2. ตรวจ `X-Rescue-Team-ID` ไม่ว่าง
  3. ผ่าน → IAM Allow + `principalId` = rescue_team_id
  4. ไม่ผ่าน → IAM Deny → 403

---

### Phase D: Terraform Infrastructure

> depends on Phase B + Phase C complete

**D1. Lambda Functions** _(อ้างอิง implement_plan Step 11)_

- `terraform/lambda.tf` — 3 functions:

| Function          | Runtime           | Memory | Timeout | Trigger                |
| ----------------- | ----------------- | ------ | ------- | ---------------------- |
| `report-progress` | `provided.al2023` | 256 MB | 30s     | API Gateway            |
| `get-mission`     | `provided.al2023` | 256 MB | 30s     | API Gateway            |
| `authorizer`      | `provided.al2023` | 256 MB | 30s     | API Gateway Authorizer |

- Environment variables: `TABLE_MISSION`, `TABLE_TIMELINE`, `TABLE_OUTBOX`, `EVENT_BUS_NAME`, `INCIDENT_SERVICE_URL`, `VALID_API_KEY`
- Role: LabRole ARN
- Source: zip files จาก `terraform/build/`

**D2. API Gateway + Authorizer** _(อ้างอิง implement_plan Step 12)_

- `terraform/api_gateway.tf` — REST API `mission-progress-api`
- Resources:

| Method | Path                                | Lambda Target     |
| ------ | ----------------------------------- | ----------------- |
| GET    | `/incidents/{incident_id}`          | `get-mission`     |
| POST   | `/incidents/{incident_id}/progress` | `report-progress` |

- Lambda Authorizer (REQUEST type) — identity sources: `x-api-key`, `X-Rescue-Team-ID`
- API Key + Usage Plan
- CORS: `*` origins, GET/POST/OPTIONS methods
- Stage: `v1`

**D3. EventBridge** _(อ้างอิง implement_plan Step 14)_

- `terraform/eventbridge.tf` — custom bus `mission-progress-events`
- 3 rules → CloudWatch Log Groups:

| Rule                          | Detail Type              | Target               |
| ----------------------------- | ------------------------ | -------------------- |
| `mission-status-changed-rule` | `MissionStatusChanged`   | CloudWatch Log Group |
| `backup-requested-rule`       | `MissionBackupRequested` | CloudWatch Log Group |
| `impact-level-updated-rule`   | `ImpactLevelUpdated`     | CloudWatch Log Group |

---

### Phase E: Build & Deploy Scripts

> depends on Phase C, D

**E1. Build Script** _(อ้างอิง implement_plan Step 19 — เฉพาะ Go)_

- `script/build.sh`
- Cross-compile 3 Go binaries (`GOOS=linux GOARCH=amd64`) → ตั้งชื่อ `bootstrap` → zip → `terraform/build/`

**E2. Deploy Script** _(อ้างอิง implement_plan Step 20 — เฉพาะ backend)_

- `script/deploy.sh`
- เรียก build.sh → `terraform init` → `terraform apply -auto-approve` → แสดง output URLs

**E3. Seed Data Script** _(อ้างอิง implement_plan Step 21)_

- `script/seed-data.sh`
- ใช้ AWS CLI `aws dynamodb put-item` insert:
  - 3-5 MissionAssignment records (ครอบคลุม status: DISPATCHED, EN_ROUTE, ON_SITE, NEED_BACKUP, RESOLVED)
  - Timeline entries ที่สอดคล้อง

**E4. Destroy Script** _(อ้างอิง implement_plan Step 24)_

- `script/destroy.sh`
- `terraform destroy -auto-approve`

---

### Phase F: Testing & Demo Preparation

> depends on deploy สำเร็จ

**F1. Manual Verification**

| #   | Test                                                 | Expected                                             |
| --- | ---------------------------------------------------- | ---------------------------------------------------- |
| 1   | curl ไม่มี API key                                   | 403 Forbidden                                        |
| 2   | curl GET incident ที่ไม่มี                           | 404 INCIDENT_NOT_FOUND                               |
| 3   | Run seed-data.sh                                     | Insert สำเร็จ                                        |
| 4   | curl GET incident ที่มี                              | 200 + mission + timeline + `data_source: "partial"`  |
| 5   | curl POST valid transition (DISPATCHED → EN_ROUTE)   | 200 success                                          |
| 6   | curl POST invalid transition (DISPATCHED → RESOLVED) | 400 INVALID_STATE_TRANSITION                         |
| 7   | curl POST NEED_BACKUP                                | 200 + ตรวจ CW Logs มี `MissionBackupRequested` event |
| 8   | curl POST พร้อม new_impact_level                     | 200 + ตรวจ CW Logs มี `ImpactLevelUpdated` event     |

**F2. Demo Script (ลำดับสาธิต)**

1. แสดง architecture overview สั้นๆ
2. **Demo Sync (GET):** GET /incidents/{id} → แสดง mission details + timeline + degraded mode (mock IncidentTracking)
3. **Demo Sync + Async (POST):** POST /incidents/{id}/progress → เปลี่ยนสถานะ + แสดง EventBridge events ใน CloudWatch Logs
4. **Demo State Machine:** แสดง 400 error เมื่อ transition ผิดลำดับ
5. **Demo Auth:** แสดง 403 เมื่อไม่มี API key
6. เปิดรับ feedback จากเพื่อนร่วมชั้น

---

## Files to Create

สร้างใหม่ทั้งหมด (~25 files):

**Go Backend:**

- `src/backend/go.mod`, `src/backend/go.sum`
- `src/backend/internal/models/` — mission.go, timeline.go, events.go, outbox.go, requests.go, incident.go
- `src/backend/internal/statemachine/statemachine.go`
- `src/backend/internal/repository/` — mission_repo.go, timeline_repo.go, outbox_repo.go
- `src/backend/internal/client/incident_client.go`
- `src/backend/internal/events/publisher.go`
- `src/backend/internal/response/response.go`
- `src/backend/cmd/report-progress/main.go`
- `src/backend/cmd/get-mission/main.go`
- `src/backend/cmd/authorizer/main.go`

**Terraform:**

- `terraform/main.tf`, `terraform/variables.tf`, `terraform/outputs.tf`, `terraform/iam.tf`
- `terraform/dynamodb.tf`, `terraform/lambda.tf`, `terraform/api_gateway.tf`, `terraform/eventbridge.tf`

**Scripts:**

- `script/build.sh`, `script/deploy.sh`, `script/seed-data.sh`, `script/destroy.sh`

---

## Scope Decisions

| Decision                                 | Reason                                                      |
| ---------------------------------------- | ----------------------------------------------------------- |
| ไม่ทำ Frontend                           | Demo ผ่าน curl/Postman — ลด scope ให้ focus backend + infra |
| ไม่ทำ presigned-url Lambda + S3          | ไม่จำเป็นสำหรับ 2 ฟังก์ชันหลัก → defer demo 2+              |
| ไม่ทำ outbox-processor Lambda            | Outbox save logic มี แต่ cron retry defer → demo 2+         |
| Mock IncidentTracking ด้วย degraded mode | URL placeholder → timeout → auto-fallback แสดงข้อมูลตัวเอง  |
| ใช้ API Key + Lambda Authorizer          | Learner Lab ไม่มี Cognito → ตาม implement_plan              |
| Demo via curl/Postman                    | เร็ว ตรงจุด แสดง request/response ชัดเจน                    |

---

## Excluded from Demo 1 (Defer)

- Frontend (Next.js + Tailwind CSS) → demo 2+
- presigned-url Lambda + S3 buckets → demo 2+
- outbox-processor Lambda (cron retry) → demo 2+
- Offline mode (localStorage queue) → demo 2+
- Automated unit tests / integration tests → demo 2+
