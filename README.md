# MissionProgress Service — Demo 1 (MVP)

> CS366 — รัฐธรรมนูญ โคสาแสง (6609612178)

## ภาพรวม

MissionProgress Service คือบริการสำหรับทีมกู้ภัย (Rescue Team) ใช้รายงานความคืบหน้าของภารกิจที่ได้รับมอบหมาย อัปเดตสถานะเหตุการณ์ และบันทึกรายละเอียดการปฏิบัติงานหน้างาน เพื่อให้ศูนย์สั่งการได้รับข้อมูลที่ถูกต้องและเป็นปัจจุบัน

**Demo 1** สาธิต 2 ฟังก์ชันหลัก:

1. **GET /incidents/{id}** (Synchronous) — ดึงข้อมูลภารกิจ + Timeline จาก DynamoDB + เรียก IncidentTracking Service (degraded mode เมื่อเรียกไม่สำเร็จ)
2. **POST /incidents/{id}/progress** (Sync + Async) — อัปเดตสถานะใน DynamoDB (sync) + publish events ไป EventBridge (async) พร้อม Outbox fallback

---

## สถาปัตยกรรม

```
Client (curl/Postman)
        │
        ▼
  API Gateway (REST)
   ├── Lambda Authorizer (ตรวจ API Key + Team ID)
   │
   ├── GET  /incidents/{id}          → get-mission Lambda
   │       ├── DynamoDB (MissionAssignment, MissionTimeline)
   │       └── HTTP → IncidentTracking Service (degraded mode)
   │
   └── POST /incidents/{id}/progress → report-progress Lambda
           ├── DynamoDB (อัปเดตสถานะ + เพิ่ม Timeline)
           ├── EventBridge (publish events)
           │       └── CloudWatch Logs (3 rules)
           └── Outbox Table (fallback เมื่อ EventBridge ล้มเหลว)
```

---

## Tech Stack

| Layer           | เทคโนโลยี                                |
| --------------- | ---------------------------------------- |
| Backend         | Go 1.21 (AWS Lambda, `provided.al2023`)  |
| API Gateway     | Amazon API Gateway (REST) + API Key Auth |
| Database        | Amazon DynamoDB (PAY_PER_REQUEST)        |
| Async Messaging | Amazon EventBridge (pub/sub)             |
| IaC             | Terraform (~5.0)                         |
| Auth            | API Key + Lambda Authorizer (REQUEST)    |

---

## โครงสร้างโปรเจค

```
CS366-MissionProgress-Service/
├── src/backend/
│   ├── cmd/
│   │   ├── report-progress/main.go   # Lambda: POST /incidents/{id}/progress
│   │   ├── get-mission/main.go       # Lambda: GET /incidents/{id}
│   │   └── authorizer/main.go        # Lambda Authorizer
│   ├── internal/
│   │   ├── models/                   # Structs: Mission, Timeline, Events, Outbox
│   │   ├── statemachine/             # State transition validation
│   │   ├── repository/               # DynamoDB CRUD
│   │   ├── client/                   # HTTP Client (IncidentTracking)
│   │   ├── events/                   # EventBridge publisher + Outbox fallback
│   │   └── response/                 # HTTP response helpers
│   ├── go.mod
│   └── go.sum
├── terraform/
│   ├── main.tf                       # AWS provider + backend
│   ├── variables.tf                  # ตัวแปร Terraform
│   ├── outputs.tf                    # Output: API URL, API Key
│   ├── iam.tf                        # LabRole reference
│   ├── dynamodb.tf                   # 3 DynamoDB tables
│   ├── lambda.tf                     # 3 Lambda functions
│   ├── api_gateway.tf                # REST API + Authorizer + CORS
│   └── eventbridge.tf                # Event bus + rules + CloudWatch Logs
├── script/
│   ├── build.sh                      # Cross-compile Go → zip
│   ├── deploy.sh                     # Build + terraform apply
│   ├── seed-data.sh                  # Insert ข้อมูลตัวอย่างใน DynamoDB
│   └── destroy.sh                    # terraform destroy
├── plan/
│   ├── implement_plan.md
│   └── checkpoint/demo1.md
└── docs/
    └── Service Proposal.md
```

---

## DynamoDB Tables

| Table               | PK           | SK           | GSIs                           |
| ------------------- | ------------ | ------------ | ------------------------------ |
| `MissionAssignment` | `mission_id` | —            | `incident-index`, `team-index` |
| `MissionTimeline`   | `mission_id` | `timestamp`  | `log-id-index`                 |
| `EventOutbox`       | `outbox_id`  | `created_at` | `status-index` + TTL on `ttl`  |

---

## API Endpoints

### Authentication

ทุก request ต้องส่ง headers:

```
x-api-key: <API_KEY>
X-Rescue-Team-ID: <TEAM_ID>
```

ถ้าไม่ส่ง หรือ API key ไม่ถูกต้อง → `403` พร้อม `{"message": "Unauthorized"}`

### GET /incidents/{incident_id}

ดึงข้อมูลภารกิจตาม incident_id พร้อม timeline

**Response 200:**

```json
{
  "incident_id": "INC-001",
  "mission_id": "MSN-001",
  "rescue_team_id": "TEAM-ALPHA",
  "current_status": "DISPATCHED",
  "latest_impact_level": 2,
  "started_at": "2024-12-01T08:00:00Z",
  "last_updated_at": "2024-12-01T08:00:00Z",
  "timeline": [
    {
      "mission_id": "MSN-001",
      "timestamp": "2024-12-01T08:00:00Z",
      "log_id": "LOG-001",
      "action_type": "STATUS_CHANGE",
      "description": "Mission dispatched to TEAM-ALPHA",
      "performed_by": "SYSTEM"
    }
  ],
  "data_source": "partial"
}
```

- `data_source: "full"` — เรียก IncidentTracking สำเร็จ
- `data_source: "partial"` — เรียกไม่สำเร็จ (degraded mode)

### POST /incidents/{incident_id}/progress

อัปเดตสถานะภารกิจ

**Request Body:**

```json
{
  "new_status": "EN_ROUTE",
  "note": "กำลังเดินทางไปจุดเกิดเหตุ",
  "new_impact_level": "HIGH"
}
```

| Field              | จำเป็น | คำอธิบาย                                        |
| ------------------ | ------ | ----------------------------------------------- |
| `new_status`       | ✅     | สถานะใหม่ (ต้องเป็น transition ที่ถูกต้อง)      |
| `note`             | ❌     | หมายเหตุเพิ่มเติม                               |
| `new_impact_level` | ❌     | ระดับผลกระทบใหม่ → publish `ImpactLevelUpdated` |

**Response 200:**

```json
{
  "message": "Progress reported successfully",
  "mission_id": "MSN-001",
  "incident_id": "INC-001",
  "old_status": "DISPATCHED",
  "new_status": "EN_ROUTE",
  "updated_at": "2025-..."
}
```

---

## State Machine

```
DISPATCHED ──→ EN_ROUTE ──→ ON_SITE ──→ RESOLVED
                                │
                                ▼
                          NEED_BACKUP ──→ RESOLVED
                                │
                                └──→ ON_SITE
```

| จาก         | ไปได้                 |
| ----------- | --------------------- |
| DISPATCHED  | EN_ROUTE              |
| EN_ROUTE    | ON_SITE               |
| ON_SITE     | NEED_BACKUP, RESOLVED |
| NEED_BACKUP | ON_SITE, RESOLVED     |

การ transition ที่ไม่ถูกต้อง → `400 INVALID_STATE_TRANSITION`

---

## EventBridge Events

เมื่อ POST สำเร็จ จะ publish events ไปยัง custom event bus `mission-progress-events`:

| Event                    | เงื่อนไข                      | Target          |
| ------------------------ | ----------------------------- | --------------- |
| `MissionStatusChanged`   | ทุกครั้งที่สถานะเปลี่ยน       | CloudWatch Logs |
| `MissionBackupRequested` | สถานะใหม่ = `NEED_BACKUP`     | CloudWatch Logs |
| `ImpactLevelUpdated`     | มี `new_impact_level` ใน body | CloudWatch Logs |

หาก publish ไป EventBridge ล้มเหลว → บันทึกลง Outbox table (ไม่ fail request หลัก)

---

## วิธีใช้งาน

### ข้อกำหนดเบื้องต้น

- AWS Learner Lab (รองรับ LabRole)
- Terraform ติดตั้งบน environment
- Go 1.21+ (สำหรับ build)
- AWS CLI (สำหรับ seed data)

### เตรียม AWS Credentials

ตั้งค่า AWS credentials (ผ่าน environment variables หรือ AWS CLI config) ให้สามารถ deploy resources ใน AWS Learner Lab ได้

```
aws configure
```

จะแสดงผลให้คุณกรอก Access Key ID, Secret Access Key, region (เช่น us-east-1) และ output format (เช่น json)
คุณสามารถยืนยันว่า credentials ถูกตั้งค่าเรียบร้อยแล้วได้ผ่านการ run

```
aws sts get-caller-identity
```

### 1. Build

```bash
bash script/build.sh
```

Cross-compile Go binaries 3 ตัว (authorizer, get-mission, report-progress) เป็น Linux amd64 → zip ใน `terraform/build/`

### 2. Deploy

```bash
bash script/deploy.sh
```

รัน build → `terraform init` → `terraform apply` → แสดง API Gateway URL และ API Key

### 3. Seed Data

```bash
bash script/seed-data.sh
```

Insert ข้อมูลตัวอย่าง 5 ภารกิจ + timeline entries ลง DynamoDB

### 4. ทดสอบ

```bash
# ตั้งค่าตัวแปร
export API_URL="<API_GATEWAY_URL>"
export API_KEY="<API_KEY_VALUE>"
```

#### 4.1 ทดสอบ Unauthorized (ไม่มี API key)

```bash
curl -s "$API_URL/incidents/INC-001" | jq .
```

Expected output:

```json
{
  "message": "Unauthorized"
}
```

#### 4.2 ทดสอบ GET ภารกิจ

```bash
curl -s -H "x-api-key: $API_KEY" -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  "$API_URL/incidents/INC-001" | jq .
```

Expected output (degraded mode — IncidentTracking Service ไม่พร้อมใช้งาน):

```json
{
  "incident_id": "INC-001",
  "mission_id": "MSN-001",
  "rescue_team_id": "TEAM-ALPHA",
  "current_status": "DISPATCHED",
  "latest_impact_level": 2,
  "started_at": "2024-12-01T08:00:00Z",
  "last_updated_at": "2024-12-01T08:00:00Z",
  "timeline": [
    {
      "mission_id": "MSN-001",
      "timestamp": "2024-12-01T08:00:00Z",
      "log_id": "LOG-001",
      "action_type": "STATUS_CHANGE",
      "description": "Mission dispatched to TEAM-ALPHA",
      "performed_by": "SYSTEM"
    }
  ],
  "data_source": "partial"
}
```

#### 4.3 ทดสอบ POST อัปเดตสถานะ (DISPATCHED → EN_ROUTE)

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"new_status":"EN_ROUTE","note":"กำลังเดินทาง"}' \
  "$API_URL/incidents/INC-001/progress" | jq .
```

Expected output:

```json
{
  "message": "Progress reported successfully",
  "mission_id": "MSN-001",
  "incident_id": "INC-001",
  "old_status": "DISPATCHED",
  "new_status": "EN_ROUTE",
  "updated_at": "2025-..."
}
```

#### 4.4 ทดสอบ transition ไม่ถูกต้อง (EN_ROUTE → RESOLVED)

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-BRAVO" \
  -H "Content-Type: application/json" \
  -d '{"new_status":"RESOLVED"}' \
  "$API_URL/incidents/INC-002/progress" | jq .
```

Expected output:

```json
{
  "error": "INVALID_STATE_TRANSITION",
  "code": "INVALID_STATE_TRANSITION",
  "message": "Cannot transition from EN_ROUTE to RESOLVED"
}
```

### 5. Destroy

```bash
bash script/destroy.sh
```

ลบ resources ทั้งหมดออกจาก AWS

---

## Design Patterns

| Pattern           | คำอธิบาย                                                                                             |
| ----------------- | ---------------------------------------------------------------------------------------------------- |
| State Machine     | ตรวจสอบ transition ของสถานะภารกิจตาม business rules ที่กำหนดไว้                                      |
| Outbox Pattern    | เมื่อ EventBridge publish ล้มเหลว → บันทึกลง Outbox table แทน (retry processor จะทำใน demo 2+)       |
| Degraded Mode     | เมื่อเรียก IncidentTracking Service ไม่สำเร็จ → ส่งข้อมูลเฉพาะที่ตัวเองมี (`data_source: "partial"`) |
| Lambda Authorizer | ตรวจสอบ API Key + Rescue Team ID ก่อนเข้าถึง API (แทน Cognito เนื่องจากข้อจำกัดของ Learner Lab)      |

---

## สิ่งที่ยังไม่ได้ทำใน Demo 1 (Defer ไป Demo 2+)

- Frontend (Next.js + Tailwind CSS)
- presigned-url Lambda + S3 สำหรับอัปโหลดรูปหลักฐาน
- outbox-processor Lambda (cron retry)
- Offline mode (localStorage queue)
- Automated unit tests / integration tests

---

## Demo 2
