# Implementation Plan — MissionProgress Service

## Full-Stack Serverless on AWS Learner Lab

---

## Overview

สร้าง MissionProgress Service แบบ Full-stack Serverless บน AWS Learner Lab ประกอบด้วย **Backend** (Go 1.26 Lambda + API Gateway + DynamoDB + EventBridge + S3) และ **Frontend** (Next.js + Tailwind CSS) deploy ด้วย Terraform ทั้งหมด ใช้ API Key แทน Cognito เพื่อหลีกเลี่ยงข้อจำกัดของ Learner Lab ระบบรองรับ 2 API หลัก (POST report progress, GET mission details) พร้อม State Machine validation, EventBridge async event publishing, และ S3 presigned URL สำหรับอัปโหลดรูปหลักฐาน

---

## Tech Stack

| Layer               | Technology                               |
| ------------------- | ---------------------------------------- |
| **Frontend**        | Next.js (App Router) + Tailwind CSS      |
| **Backend**         | Go 1.26 (AWS Lambda, `provided.al2023`)  |
| **API Gateway**     | Amazon API Gateway (REST) + API Key Auth |
| **Database**        | Amazon DynamoDB (PAY_PER_REQUEST)        |
| **File Storage**    | Amazon S3 (presigned URL upload)         |
| **Async Messaging** | Amazon EventBridge (pub/sub)             |
| **IaC**             | Terraform                                |
| **Auth**            | API Key + Usage Plan (แทน Cognito)       |

---

## Project Structure

```
CS366-MissionProgress-Service/
├── src/
│   ├── backend/
│   │   ├── cmd/
│   │   │   ├── report-progress/main.go      # Lambda: POST /incidents/{id}/progress
│   │   │   ├── get-mission/main.go           # Lambda: GET /incidents/{id}
│   │   │   ├── presigned-url/main.go         # Lambda: GET /upload-url
│   │   │   ├── authorizer/main.go            # Lambda Authorizer (API Key + Team validation)
│   │   │   └── outbox-processor/main.go      # Lambda: Cron Job สำหรับ Outbox Pattern retry
│   │   ├── internal/
│   │   │   ├── models/                       # Structs: Mission, Timeline, Events
│   │   │   ├── statemachine/                 # Status transition validation
│   │   │   ├── repository/                   # DynamoDB CRUD (mission, timeline, outbox)
│   │   │   ├── client/                       # HTTP Client สำหรับเรียก External Services
│   │   │   ├── events/                       # EventBridge publisher (all event types)
│   │   │   └── response/                     # HTTP response helpers
│   │   ├── go.mod
│   │   └── go.sum
│   └── frontend/
│       ├── app/                              # Next.js App Router pages
│       ├── components/                       # UI components (Tailwind)
│       ├── lib/                              # API client, types
│       ├── hooks/                            # Custom hooks (useOfflineQueue, useGeolocation)
│       ├── package.json
│       ├── tailwind.config.ts
│       └── next.config.ts
├── terraform/
│   ├── main.tf                               # Provider, backend config
│   ├── variables.tf                          # Input variables
│   ├── outputs.tf                            # API URL, S3 bucket, etc.
│   ├── lambda.tf                             # Lambda functions (5 ตัว)
│   ├── api_gateway.tf                        # REST API + Lambda Authorizer + API key
│   ├── dynamodb.tf                           # 3 tables (Mission, Timeline, Outbox)
│   ├── s3.tf                                 # Evidence bucket + frontend bucket
│   ├── eventbridge.tf                        # Event bus + rules + scheduler (outbox cron)
│   └── iam.tf                                # LabRole reference
├── test/
│   ├── unit/                                 # Go unit tests
│   └── integration/                          # API integration tests
├── script/
│   ├── build.sh                              # Build Go binaries for Lambda
│   ├── deploy.sh                             # terraform init + apply
│   ├── seed-data.sh                          # Insert sample data into DynamoDB
│   └── destroy.sh                            # terraform destroy
├── docs/
│   └── Service Proposal.md
└── plan/
    └── implement_plan.md
```

---

## Implementation Steps

### Phase 1: Project Initialization & Foundation

#### Step 1 — Initialize Go Module

- **Directory:** `src/backend/`
- **Action:** สร้าง `go.mod` สำหรับ Go module
- **Module Name:** `github.com/Ratthatummanoon/CS366-MissionProgress-Service`
- **Dependencies:**
  - `github.com/aws/aws-lambda-go` — Lambda handler
  - `github.com/aws/aws-sdk-go-v2` — DynamoDB, EventBridge, S3 client
  - `github.com/google/uuid` — generate `log_id`

#### Step 2 — Initialize Next.js Frontend

- **Directory:** `src/frontend/`
- **Action:** สร้างโปรเจกต์ Next.js ด้วย `npx create-next-app@latest`
- **Options:** App Router, TypeScript, Tailwind CSS, ESLint
- **Additional packages:** `axios` (สำหรับ API client)

#### Step 3 — สร้าง Terraform Base Configuration

- **Directory:** `terraform/`
- **Files:**
  - `main.tf` — AWS provider (region `us-east-1`), reference `LabRole` ARN ผ่าน `data "aws_iam_role"`
  - `variables.tf` — ตัวแปร: `aws_region`, `project_name`, `lab_role_arn` (default LabRole), `api_key_value`
  - `outputs.tf` — Output: API Gateway invoke URL, S3 bucket names, API Key

---

### Phase 2: Database Layer

#### Step 4 — สร้าง DynamoDB Tables

- **File:** `terraform/dynamodb.tf`

**Table 1: `MissionAssignment`** — สถานะปัจจุบันของภารกิจ

| Attribute             | Type     | Key    | Description                           |
| --------------------- | -------- | ------ | ------------------------------------- |
| `mission_id`          | UUID     | PK     | รหัสภารกิจ (Primary Key)              |
| `incident_id`         | UUID     | GSI-PK | รหัสเหตุการณ์ (GSI: `incident-index`) |
| `rescue_team_id`      | String   | GSI-PK | รหัสทีมกู้ภัย (GSI: `team-index`)     |
| `current_status`      | String   | —      | สถานะปัจจุบัน (Enum)                  |
| `latest_impact_level` | Integer  | —      | ระดับความรุนแรงล่าสุด (1-4)           |
| `started_at`          | DateTime | —      | เวลาที่เริ่มรับภารกิจ (ISO 8601)      |
| `last_updated_at`     | DateTime | —      | เวลาที่อัปเดตล่าสุด (ISO 8601)        |

- Billing Mode: `PAY_PER_REQUEST`
- GSI `incident-index`: Partition Key = `incident_id`
- GSI `team-index`: Partition Key = `rescue_team_id`

**Table 2: `MissionTimeline`** — ประวัติ Timeline / Action Logs

| Attribute      | Type     | Key    | Description                                     |
| -------------- | -------- | ------ | ----------------------------------------------- |
| `mission_id`   | UUID     | PK     | รหัสภารกิจ (Partition Key)                      |
| `timestamp`    | DateTime | SK     | เวลาที่เกิดเหตุการณ์ (Sort Key, ISO 8601)       |
| `log_id`       | UUID     | GSI-PK | รหัส Log (GSI: `log-id-index`)                  |
| `action_type`  | String   | —      | ประเภทการกระทำ (STATUS_CHANGE, COMMENT, UPLOAD) |
| `description`  | String   | —      | รายละเอียดของสิ่งที่ทำ                          |
| `performed_by` | String   | —      | ID ของผู้ที่ทำรายการ                            |
| `gps_location` | String   | —      | พิกัด GPS (Lat, Long) — Optional                |

- Billing Mode: `PAY_PER_REQUEST`
- GSI `log-id-index`: Partition Key = `log_id`

**Table 3: `EventOutbox`** — Outbox Pattern สำหรับ EventBridge failure handling

| Attribute       | Type     | Key    | Description                                                 |
| --------------- | -------- | ------ | ----------------------------------------------------------- |
| `outbox_id`     | UUID     | PK     | รหัส Outbox entry (UUID)                                    |
| `created_at`    | DateTime | SK     | เวลาที่สร้าง (ISO 8601) — ใช้ sort เพื่อ process เรียงลำดับ |
| `event_type`    | String   | —      | ชนิดของ Event (e.g. `MissionStatusChanged`)                 |
| `event_payload` | String   | —      | JSON payload ของ Event ที่ต้องส่ง                           |
| `status`        | String   | GSI-PK | สถานะ: `PENDING`, `SENT`, `FAILED` (GSI: `status-index`)    |
| `retry_count`   | Integer  | —      | จำนวนครั้งที่ retry แล้ว                                    |
| `last_error`    | S        | —      | Error message ล่าสุด (ถ้ามี) — Optional                     |

- Billing Mode: `PAY_PER_REQUEST`
- GSI `status-index`: Partition Key = `status` — ใช้ query เฉพาะ `PENDING` items
- TTL Attribute: `ttl` — auto-delete records ที่ส่งสำเร็จแล้วหลัง 7 วัน

---

### Phase 3: Backend Lambda Functions (Go)

#### Step 5 — สร้าง Models

- **Directory:** `src/backend/internal/models/`
- **Files:**
  - `mission.go` — struct `MissionAssignment` ตรงตาม DynamoDB schema + JSON tags
  - `timeline.go` — struct `TimelineEntry` ตรงตาม DynamoDB schema
  - `events.go` — Event structs ทั้งหมด:
    - `MissionStatusChangedEvent` — เมื่อสถานะภารกิจเปลี่ยน (ตรงตาม Async Contract)
    - `MissionBackupRequestedEvent` — เมื่อทีมกู้ภัยร้องขอกำลังเสริม (status = `NEED_BACKUP`)
    - `ImpactLevelUpdatedEvent` — เมื่อทีมกู้ภัยปรับ Impact Level ใหม่จากหน้างาน
  - `outbox.go` — struct `OutboxEntry` สำหรับ Outbox Pattern
  - `requests.go` — struct `ReportProgressRequest` (body ของ POST), `ReportProgressResponse`, `GetMissionResponse`, `ErrorResponse`
  - `incident.go` — struct `IncidentDetail` สำหรับข้อมูลที่ดึงจาก IncidentTracking Service (description, location, type)

#### Step 6 — สร้าง State Machine

- **File:** `src/backend/internal/statemachine/statemachine.go`
- **Valid Transitions:**

```
DISPATCHED  → [EN_ROUTE]
EN_ROUTE    → [ON_SITE]
ON_SITE     → [NEED_BACKUP, RESOLVED]
NEED_BACKUP → [ON_SITE, RESOLVED]
```

- **Functions:**
  - `IsValidTransition(from, to string) bool` — ตรวจสอบว่า transition ถูกต้องตาม flow หรือไม่
  - `ValidateStatus(status string) bool` — ตรวจว่าค่าอยู่ใน enum ที่กำหนด
- **หมายเหตุ:** Unit testable — ไม่พึ่ง AWS SDK

#### Step 7 — สร้าง Repository Layer (DynamoDB CRUD)

- **Directory:** `src/backend/internal/repository/`
- **Files:**

  **`mission_repo.go`:**
  - `GetMissionByIncidentID(incidentID string) (*MissionAssignment, error)` — ใช้ GSI `incident-index` query
  - `UpdateMissionStatus(mission *MissionAssignment) error` — ใช้ conditional update (ป้องกัน race condition)
  - `CreateMission(mission *MissionAssignment) error`

  **`timeline_repo.go`:**
  - `AddTimelineEntry(entry *TimelineEntry) error` — เพิ่มรายการ Timeline ใหม่
  - `GetTimelineByMissionID(missionID string) ([]TimelineEntry, error)` — query by PK sorted by SK (timestamp)

  **`outbox_repo.go`:**
  - `SaveOutboxEntry(entry *OutboxEntry) error` — บันทึก event ที่ส่งไม่สำเร็จลง Outbox table
  - `GetPendingEntries(limit int) ([]OutboxEntry, error)` — ดึง entries ที่ status = `PENDING` (ใช้ GSI `status-index`)
  - `MarkAsSent(outboxID string) error` — อัปเดต status เป็น `SENT` + ตั้ง TTL
  - `MarkAsFailed(outboxID string, errMsg string) error` — อัปเดต status เป็น `FAILED` + เพิ่ม retry_count

#### Step 7b — สร้าง HTTP Client สำหรับ IncidentTracking Service

- **File:** `src/backend/internal/client/incident_client.go`
- **Purpose:** ดึงข้อมูล Master Data (description, location, incident_type) จาก IncidentTracking Service
- **Functions:**
  - `GetIncidentDetail(incidentID string) (*IncidentDetail, error)` — เรียก GET `/incidents/{incident_id}` ของ IncidentTracking Service
- **Configuration:**
  - Base URL อ่านจาก environment variable `INCIDENT_SERVICE_URL`
  - Timeout: 3 วินาที (ป้องกัน slow dependency)
  - Retry: 1 ครั้ง ด้วย exponential backoff
- **Failure Handling (Degraded Mode):**
  - หากเรียกไม่สำเร็จ (timeout, 5xx, connection error) → return `nil` แทนที่จะ error
  - Lambda handler จะแสดงเฉพาะข้อมูลที่ตัวเองถืออยู่ (Timeline, Status) โดยไม่แสดง description/location จาก IncidentTracking
  - Log warning เพื่อ monitoring

#### Step 8 — สร้าง EventBridge Publisher + Outbox Fallback

- **File:** `src/backend/internal/events/publisher.go`
- **Event Bus:** `mission-progress-events`
- **Source:** `MissionProgressService`

**Functions & Event Types:**

| Function                        | Detail Type              | เมื่อไหร่ถูกส่ง                                      | Consumer                   |
| ------------------------------- | ------------------------ | ---------------------------------------------------- | -------------------------- |
| `PublishMissionStatusChanged()` | `MissionStatusChanged`   | ทุกครั้งที่สถานะภารกิจเปลี่ยน                        | IncidentTracking, Dispatch |
| `PublishBackupRequested()`      | `MissionBackupRequested` | เมื่อ status เปลี่ยนเป็น `NEED_BACKUP`               | Rescue Prioritization      |
| `PublishImpactLevelUpdated()`   | `ImpactLevelUpdated`     | เมื่อทีมกู้ภัยส่ง `new_impact_level` มาพร้อม request | Rescue Prioritization      |

**Outbox Pattern (Failure Handling):**

1. เมื่อ Publish event สำเร็จ → จบ
2. เมื่อ Publish event ล้มเหลว (EventBridge error/timeout):
   - บันทึก event ลง `EventOutbox` table ด้วย status = `PENDING`
   - Log error แต่ **ไม่ fail request หลัก** (core DynamoDB write สำเร็จแล้ว)
   - Outbox Processor Lambda (Cron) จะมาดึง PENDING entries ไปส่งซ้ำภายหลัง

#### Step 9 — สร้าง Lambda Handlers

- **Directory:** `src/backend/cmd/`

**9a. `report-progress/main.go`** — POST `/incidents/{incident_id}/progress`

1. Parse `incident_id` จาก path parameters
2. Parse `X-Rescue-Team-ID` จาก headers
3. Parse & validate request body (`status`, `action_description`, `current_location`, `new_impact_level`)
4. เรียก repository หา mission by `incident_id` → ถ้าไม่พบ return **404** `INCIDENT_NOT_FOUND`
5. Validate state transition ด้วย statemachine → ถ้าผิด return **400** `INVALID_STATE_TRANSITION`
6. Update mission status ใน DynamoDB
7. เพิ่ม Timeline entry ใหม่
8. **Publish Events (ตามเงื่อนไข ผ่าน Outbox Pattern):**
   - ทุกกรณี: Publish `MissionStatusChangedEvent` → IncidentTracking, Dispatch
   - ถ้า status = `NEED_BACKUP`: Publish `MissionBackupRequestedEvent` เพิ่มเติม → Rescue Prioritization
   - ถ้า `new_impact_level` มีค่า: Publish `ImpactLevelUpdatedEvent` เพิ่มเติม → Rescue Prioritization
   - หาก EventBridge ล้มเหลว → เขียนลง Outbox table (ไม่ fail request หลัก)
9. Return **200** success response

**9b. `get-mission/main.go`** — GET `/incidents/{incident_id}`

1. Parse `incident_id` จาก path parameters
2. Query mission by `incident_id` → ถ้าไม่พบ return **404**
3. **เรียก IncidentTracking Service (Sync):** ดึงข้อมูล `description`, `location`, `incident_type` ผ่าน HTTP Client
   - **สำเร็จ:** รวมข้อมูลเข้า response พร้อม timeline
   - **ล้มเหลว (Degraded Mode):** แสดงเฉพาะข้อมูลที่ตัวเองมี (Timeline, Status) + field `description` = `"ไม่สามารถโหลดรายละเอียดเหตุการณ์ได้"` + field `data_source` = `"partial"` เพื่อแจ้ง client
4. Query timeline entries by `mission_id` (sorted by timestamp)
5. Return JSON response ตรงตาม contract (`incident_id`, `description`, `current_status`, `exact_location`, `timeline[]`, `data_source`)

**9c. `presigned-url/main.go`** — GET `/upload-url`

1. รับ query parameter `filename` และ `content_type`
2. Generate S3 presigned PUT URL (expire 15 นาที)
3. Return `{ "upload_url": "...", "file_key": "..." }`

**9d. `authorizer/main.go`** — Lambda Authorizer สำหรับ API Gateway

1. รับ API Key จาก header (`x-api-key`) และ `X-Rescue-Team-ID`
2. ตรวจสอบว่า API Key ตรงกับค่าที่กำหนด (อ่านจาก environment variable `VALID_API_KEY`)
3. ตรวจว่า `X-Rescue-Team-ID` ไม่ว่างเปล่า
4. **ผ่าน:** Return IAM Policy Allow + แนบ `principalId` = rescue_team_id (ส่งต่อไป Lambda ปลายทางผ่าน `requestContext.authorizer`)
5. **ไม่ผ่าน:** Return IAM Policy Deny → API Gateway return 403 Forbidden

**9e. `outbox-processor/main.go`** — Cron Job Lambda (EventBridge Scheduler)

1. ถูกเรียกตาม schedule (ทุก 1 นาที)
2. Query `EventOutbox` table ดึง entries ที่ status = `PENDING` (จำกัด batch ละ 25 records)
3. สำหรับแต่ละ entry:
   - พยายามส่ง event ไป EventBridge ตาม `event_type` และ `event_payload`
   - **สำเร็จ:** เรียก `MarkAsSent()` + ตั้ง TTL 7 วัน
   - **ล้มเหลวอีก:** เรียก `MarkAsFailed()` + เพิ่ม retry_count
   - ถ้า `retry_count >= 5` → ค้างไว้ ไม่ retry อีก (แจ้งเตือนผ่าน CloudWatch Alarm)
4. Log สรุปจำนวน processed / success / failed

---

### Phase 4: Infrastructure as Code (Terraform)

#### Step 10 — IAM Configuration

- **File:** `terraform/iam.tf`
- ใช้ `data "aws_iam_role" "lab_role"` เพื่อ reference `LabRole` ที่มีอยู่แล้วใน Learner Lab
- **ไม่สร้าง IAM role ใหม่** เพราะ Learner Lab มีข้อจำกัด — ใช้ `LabRole` สำหรับทุก Lambda function

#### Step 11 — Lambda Functions

- **File:** `terraform/lambda.tf`
- **Functions (5 ตัว):**

| Function           | Description                              | Trigger                |
| ------------------ | ---------------------------------------- | ---------------------- |
| `report-progress`  | POST report progress & update status     | API Gateway            |
| `get-mission`      | GET mission details + timeline           | API Gateway            |
| `presigned-url`    | GET S3 presigned upload URL              | API Gateway            |
| `authorizer`       | Lambda Authorizer (API Key + Team ID)    | API Gateway Authorizer |
| `outbox-processor` | Cron Job retry failed EventBridge events | EventBridge Scheduler  |

- **Configuration (ทั้ง 5 ตัว):**
  - Runtime: `provided.al2023` (custom runtime สำหรับ Go compiled binary)
  - Handler: `bootstrap` (Go binary ชื่อ `bootstrap`)
  - Memory: 256 MB (outbox-processor: 128 MB)
  - Timeout: 30 seconds (outbox-processor: 60 seconds)
  - Environment Variables: `TABLE_MISSION`, `TABLE_TIMELINE`, `TABLE_OUTBOX`, `S3_BUCKET`, `EVENT_BUS_NAME`, `INCIDENT_SERVICE_URL`, `VALID_API_KEY`
  - Role: reference `LabRole` ARN
  - Source Code: reference zip file จาก build output

#### Step 12 — API Gateway + Lambda Authorizer

- **File:** `terraform/api_gateway.tf`
- **REST API:** `mission-progress-api`
- **Resources:**

| Method | Path                                | Lambda Target     |
| ------ | ----------------------------------- | ----------------- |
| GET    | `/incidents/{incident_id}`          | `get-mission`     |
| POST   | `/incidents/{incident_id}/progress` | `report-progress` |
| GET    | `/upload-url`                       | `presigned-url`   |

- **Authentication (2 ชั้น):**
  1. **API Key + Usage Plan:** API Gateway ตรวจสอบ `x-api-key` header เบื้องต้น (ป้องกัน anonymous access)
  2. **Lambda Authorizer (REQUEST type):** ตรวจสอบ `x-api-key` และ `X-Rescue-Team-ID` ใน Go code — ส่ง `rescue_team_id` ต่อไป Lambda ปลายทางผ่าน `requestContext.authorizer.principalId`
  - **Authorizer Configuration:**
    - Type: `REQUEST` (header-based)
    - Identity Sources: `method.request.header.x-api-key`, `method.request.header.X-Rescue-Team-ID`
    - Result TTL: 300 seconds (cache ผล auth ลดการเรียก Lambda ซ้ำ)
- **CORS:** allow origin `*`, methods `GET,POST,OPTIONS`, headers `x-api-key,X-Rescue-Team-ID,Content-Type`
- **Stage:** `v1`
- **Lambda Permissions:** อนุญาตให้ API Gateway invoke แต่ละ Lambda function (รวม authorizer)

#### Step 13 — S3 Buckets

- **File:** `terraform/s3.tf`

**Bucket 1: Evidence Bucket** (`mission-progress-evidence-{random}`)

- CORS configuration (allow PUT from frontend origin)
- ใช้เก็บรูปภาพ/วิดีโอหลักฐานจากหน้างาน

**Bucket 2: Frontend Hosting** (`mission-progress-frontend-{random}`)

- Static website hosting enabled
- Public access (bucket policy allow `GetObject`)
- ใช้ host Next.js static export

#### Step 14 — EventBridge + Outbox Scheduler

- **File:** `terraform/eventbridge.tf`
- **Custom Event Bus:** `mission-progress-events`

**Event Rules:**

| Rule Name                     | Detail Type              | Target               | Description                                  |
| ----------------------------- | ------------------------ | -------------------- | -------------------------------------------- |
| `mission-status-changed-rule` | `MissionStatusChanged`   | CloudWatch Log Group | บันทึกทุกการเปลี่ยนสถานะ (demo target)       |
| `backup-requested-rule`       | `MissionBackupRequested` | CloudWatch Log Group | บันทึกเมื่อทีมร้องขอกำลังเสริม (demo target) |
| `impact-level-updated-rule`   | `ImpactLevelUpdated`     | CloudWatch Log Group | บันทึกเมื่อปรับ Impact Level (demo target)   |

**Outbox Processor Scheduler:**

- **EventBridge Scheduler Rule:** `outbox-processor-schedule`
  - Schedule: `rate(1 minute)` — เรียก outbox-processor Lambda ทุก 1 นาที
  - Target: `outbox-processor` Lambda function
  - ใช้ default EventBridge bus (ไม่ใช่ custom bus)

- **Dead Letter Queue:** SQS queue สำหรับ failed events

---

### Phase 5: Frontend (Next.js + Tailwind CSS)

#### Step 15 — API Client Layer

- **Directory:** `src/frontend/lib/`
- **Files:**
  - `api.ts` — Axios/fetch wrapper ที่แนบ API Key header (`x-api-key`) และ `X-Rescue-Team-ID` ทุก request
  - `types.ts` — TypeScript interfaces ตรงกับ API response:
    - `MissionDetail` — ข้อมูลภารกิจ + timeline
    - `TimelineEntry` — แต่ละรายการใน timeline
    - `ReportProgressRequest` — body สำหรับ POST
    - `ReportProgressResponse` — response จาก POST
    - `ErrorResponse` — error response

#### Step 16 — หน้า Dashboard

- **File:** `src/frontend/app/page.tsx`
- **Features:**
  - แสดงรายการ incidents ทั้งหมดที่ทีมกู้ภัยรับผิดชอบ
  - แสดง Status Badge (สี) ตามสถานะ:
    - `DISPATCHED` → เทา
    - `EN_ROUTE` → น้ำเงิน
    - `ON_SITE` → ส้ม
    - `NEED_BACKUP` → แดง
    - `RESOLVED` → เขียว
  - คลิกเพื่อเข้าดูรายละเอียดภารกิจ

#### Step 17 — หน้า Mission Detail

- **File:** `src/frontend/app/incidents/[id]/page.tsx`
- **Features:**
  - แสดงข้อมูลเหตุการณ์ (description, location, status ปัจจุบัน)
  - แสดง Timeline เรียงตามเวลา (vertical timeline component)
  - ปุ่ม "Update Status" → เปิด modal/form สำหรับ report progress
  - ปุ่ม "Upload Evidence" → ขอ presigned URL แล้ว upload ตรงไป S3

#### Step 18 — Report Progress Form Component

- **Features:**
  - Dropdown เลือก status ใหม่ (แสดงเฉพาะ status ที่ valid ตาม state machine)
  - Text area สำหรับ `action_description`
  - Optional: impact level selector (1-4)
  - GPS location auto-detect ด้วย Browser Geolocation API
  - Submit → POST `/incidents/{id}/progress`

#### Step 18b — Offline Mode & Resilience (Frontend)

- **Directory:** `src/frontend/hooks/` และ `src/frontend/lib/`
- **Purpose:** รองรับการทำงานเมื่ออินเทอร์เน็ตของทีมกู้ภัยหลุดชั่วคราว (ตาม NFR: Resilience)

**ไฟล์และรายละเอียด:**

1. **`src/frontend/lib/offlineQueue.ts`** — Offline Queue Manager:
   - ใช้ `localStorage` เก็บ pending requests (เฉพาะ POST report progress) ที่ส่งไม่สำเร็จเพราะไม่มีเน็ต
   - แต่ละ entry เก็บ: `{ id, url, method, body, headers, created_at, retry_count }`
   - Functions: `enqueue(request)`, `dequeueAll()`, `removeById(id)`, `getPendingCount()`

2. **`src/frontend/hooks/useOfflineQueue.ts`** — React Hook:
   - ตรวจสถานะ online/offline ด้วย `navigator.onLine` + event listeners (`online`, `offline`)
   - เมื่อกลับมา online → ดึง pending requests จาก localStorage ส่งซ้ำอัตโนมัติ (retry)
   - แสดง UI indicator: "ออฟไลน์ — ข้อมูลจะถูกส่งเมื่อกลับมาออนไลน์" พร้อมจำนวน pending items

3. **`src/frontend/lib/api.ts`** — ปรับปรุง API Client:
   - Wrap POST requests: ถ้าส่งล้มเหลว (network error / timeout) → เก็บลง offline queue แทน + แจ้ง user ว่า "บันทึกไว้แล้ว จะส่งอีกครั้งเมื่อมีสัญญาณ"
   - GET requests: ถ้า offline แสดงข้อมูลล่าสุดที่ cache ไว้ใน localStorage (เก็บทุกครั้งที่ GET สำเร็จ)

4. **`src/frontend/components/OfflineBanner.tsx`** — UI Component:
   - แถบสีเหลืองด้านบน เมื่อ offline: "⚠️ ไม่มีสัญญาณอินเทอร์เน็ต — ข้อมูลจะถูกส่งอัตโนมัติเมื่อกลับมาออนไลน์ (N รายการรอส่ง)"
   - แถบสีเขียว เมื่อกลับมา online + syncing: "✅ กำลังส่งข้อมูลที่ค้างอยู่..."
   - ใช้ใน Root Layout (`app/layout.tsx`)

**ข้อควรระวัง (Idempotency):**

- ทุก POST request ที่สร้างจะแนบ header `X-Idempotency-Key` (ใช้ UUID v4) เพื่อให้ Backend ตรวจสอบว่า request ซ้ำจาก offline queue ไม่ถูกประมวลผลซ้ำ

---

### Phase 6: Build & Deploy Scripts

#### Step 19 — Build Script

- **File:** `script/build.sh`
- **Actions:**
  1. Cross-compile Go binaries สำหรับ Lambda (`GOOS=linux GOARCH=amd64`)
  2. ตั้งชื่อ binary เป็น `bootstrap` (required by `provided.al2023` runtime)
  3. สร้าง zip file สำหรับแต่ละ function (5 ตัว) → output ไป `terraform/build/`
  4. Build Next.js frontend (`next build` สำหรับ static export)

#### Step 20 — Deploy Script

- **File:** `script/deploy.sh`
- **Actions:**
  1. เรียก `script/build.sh` ก่อน
  2. `cd terraform && terraform init && terraform apply -auto-approve`
  3. Upload frontend static files ไป S3 frontend bucket
  4. แสดง output: API URL, Frontend URL, API Key

#### Step 21 — Seed Data Script

- **File:** `script/seed-data.sh`
- **Actions:**
  - Insert sample mission assignments (3-5 records ที่มี status ต่างกัน)
  - Insert sample timeline entries
  - ใช้ AWS CLI `aws dynamodb put-item`

---

### Phase 7: Testing

#### Step 22 — Unit Tests

- **Directory:** `test/unit/`
- **Files:**
  - `statemachine_test.go` — ทดสอบทุก valid/invalid transition combinations
  - `models_test.go` — ทดสอบ JSON marshal/unmarshal
  - `handler_test.go` — ทดสอบ Lambda handler logic ด้วย mock repository
  - `publisher_test.go` — ทดสอบ EventBridge publisher + Outbox fallback
  - `incident_client_test.go` — ทดสอบ HTTP client + degraded mode

#### Step 23 — Integration Tests

- **Directory:** `test/integration/`
- **Test Scenarios:**

| #   | Scenario                                         | Expected           |
| --- | ------------------------------------------------ | ------------------ |
| 1   | GET mission ที่ไม่มีอยู่                         | 404                |
| 2   | POST report progress (DISPATCHED → EN_ROUTE)     | 200 success        |
| 3   | POST invalid transition (DISPATCHED → RESOLVED)  | 400 error          |
| 4   | GET mission details → ดู timeline ที่เพิ่งเพิ่ม  | 200 + timeline     |
| 5   | GET presigned URL → upload file สำเร็จ           | 200 + upload OK    |
| 6   | POST NEED_BACKUP → ตรวจ BackupRequested event    | 200 + event in CW  |
| 7   | POST พร้อม new_impact_level → ตรวจ event         | 200 + event in CW  |
| 8   | เรียกโดยไม่มี API Key → Lambda Authorizer ปฏิเสธ | 403 Forbidden      |
| 9   | GET mission เมื่อ IncidentTracking ล่ม (mock)    | 200 + partial data |

#### Step 24 — Destroy Script

- **File:** `script/destroy.sh`
- **Actions:**
  1. Empty S3 buckets ก่อน (Terraform ลบ non-empty bucket ไม่ได้)
  2. `cd terraform && terraform destroy -auto-approve`

---

## Verification Checklist

| #   | Check                     | Command / Action                                      | Expected Result                           |
| --- | ------------------------- | ----------------------------------------------------- | ----------------------------------------- |
| 1   | Unit Test ผ่าน            | `cd src/backend && go test ./internal/... -v`         | All tests PASS                            |
| 2   | Build สำเร็จ              | `bash script/build.sh`                                | 5 zip files สร้างใน `terraform/build/`    |
| 3   | Deploy สำเร็จ             | `bash script/deploy.sh`                               | Terraform apply สำเร็จ + output URLs      |
| 4   | Auth ปฏิเสธ (no API key)  | `curl <API_URL>/v1/incidents/INC-12345`               | 403 Forbidden                             |
| 5   | GET 404 (ไม่มี data)      | `curl -H "x-api-key: <key>" ... /incidents/INC-12345` | 404 INCIDENT_NOT_FOUND                    |
| 6   | Seed Data                 | `bash script/seed-data.sh`                            | Insert สำเร็จ                             |
| 7   | GET 200 (มี data)         | `curl -H "x-api-key: <key>" ... /incidents/INC-12345` | 200 + mission detail + timeline           |
| 8   | POST report progress      | `curl -X POST ... /progress`                          | 200 + status updated                      |
| 9   | POST invalid transition   | `curl -X POST ... (DISPATCHED→RESOLVED)`              | 400 INVALID_STATE_TRANSITION              |
| 10  | POST NEED_BACKUP event    | `curl -X POST ... {"status":"NEED_BACKUP"}`           | 200 + BackupRequested event ใน CW Logs    |
| 11  | EventBridge events ถูกส่ง | ตรวจ CloudWatch Logs                                  | มี events ทั้ง 3 ชนิด                     |
| 12  | Outbox Processor ทำงาน    | ตรวจ CloudWatch Logs ของ outbox-processor             | มี log แสดงการ process ทุก 1 นาที         |
| 13  | Frontend แสดงผลได้        | เปิด S3 static website URL                            | Dashboard แสดง missions                   |
| 14  | Frontend offline banner   | ตัดเน็ตใน DevTools → กดอัปเดตสถานะ                    | แสดง offline banner + เก็บใน localStorage |

---

## Key Decisions

| Decision                                     | Reason                                                                                        |
| -------------------------------------------- | --------------------------------------------------------------------------------------------- |
| **Go แทน Node.js**                           | ตาม user เลือก — ใช้ Go 1.26 กับ `provided.al2023` runtime                                    |
| **Lambda Authorizer + API Key**              | ใช้ทั้ง API Key (Usage Plan) และ Lambda Authorizer เพื่อ validate + ส่ง team_id ต่อไป handler |
| **LabRole ไม่สร้าง IAM ใหม่**                | Learner Lab ไม่อนุญาตสร้าง IAM role/policy — ใช้ `LabRole` ที่มีอยู่                          |
| **PAY_PER_REQUEST DynamoDB**                 | เหมาะกับ Learner Lab ไม่ต้อง provision capacity ไม่กระทบ budget                               |
| **Static Export Next.js → S3**               | ไม่ต้องใช้ server (เหมาะกับ Learner Lab ไม่ต้อง ECS/EC2)                                      |
| **EventBridge target → CloudWatch Log**      | ไม่มี consumer service จริงใน scope — ใช้ log เพื่อ demo event ถูกส่ง                         |
| **S3 Frontend Hosting แทน Amplify**          | Learner Lab ไม่มี Amplify — ใช้ S3 static website hosting ตรงๆ                                |
| **Outbox Pattern + Cron Lambda**             | ตาม Proposal ระบุ — ป้องกัน event สูญหายเมื่อ EventBridge ล่มด้วย DynamoDB Outbox table       |
| **Sync call to IncidentTracking + Fallback** | ตาม Proposal ระบุ — ดึงข้อมูล + degraded mode เมื่อระบบล่ม                                    |
| **3 Event Types (ไม่ใช่แค่ StatusChanged)**  | ตาม Proposal ระบุ — ส่ง BackupRequested, ImpactUpdated ไป Rescue Prioritization               |
| **localStorage Offline Queue**               | ตาม NFR Resilience — เก็บ request ในเครื่องเมื่อออฟไลน์ ส่งใหม่เมื่อ online                   |

---

## Dependencies Summary

```
Frontend (Next.js + Offline Queue)
  │
  ├── localStorage (offline cache + pending queue)
  │
  └── calls → API Gateway (REST, x-api-key)
                 │
                 ├── Lambda Authorizer → Validate API Key + X-Rescue-Team-ID
                 │
                 ├── GET  /incidents/{id}         → Lambda: get-mission
                 │                                   ├── DynamoDB (Mission + Timeline)
                 │                                   └── HTTP → IncidentTracking Service (Sync, fallback: degraded)
                 │
                 ├── POST /incidents/{id}/progress → Lambda: report-progress
                 │                                    ├── DynamoDB (Update Mission + Add Timeline)
                 │                                    ├── EventBridge (publish events)
                 │                                    │    ├── MissionStatusChanged    → CW Logs
                 │                                    │    ├── MissionBackupRequested  → CW Logs
                 │                                    │    └── ImpactLevelUpdated      → CW Logs
                 │                                    └── Outbox (DynamoDB) ≤─ fallback เมื่อ EventBridge ล้ม
                 │
                 └── GET  /upload-url              → Lambda: presigned-url → S3

EventBridge Scheduler (rate: 1 min)
  └── Lambda: outbox-processor → DynamoDB Outbox → EventBridge (retry)
```

---

## Timeline Estimate

| Phase     | Description                         | Estimated Time    |
| --------- | ----------------------------------- | ----------------- |
| 1         | Project Initialization & Foundation | 1-2 ชั่วโมง       |
| 2         | Database Layer (DynamoDB)           | 1-2 ชั่วโมง       |
| 3         | Backend Lambda Functions (Go)       | 6-8 ชั่วโมง       |
| 4         | Infrastructure as Code (Terraform)  | 3-4 ชั่วโมง       |
| 5         | Frontend (Next.js + Tailwind)       | 5-7 ชั่วโมง       |
| 6         | Build & Deploy Scripts              | 1-2 ชั่วโมง       |
| 7         | Testing                             | 3-4 ชั่วโมง       |
| **Total** |                                     | **20-29 ชั่วโมง** |
