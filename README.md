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

## การพึ่งพาบริการภายนอก (Dependencies)

บริการนี้พึ่งพา **IncidentTracking Service** ของเพื่อนสำหรับดึงรายละเอียดเหตุการณ์ โดยมีรายละเอียดดังนี้:

### บริการที่พึ่งพา

| รายการ            | รายละเอียด                                                      |
| ----------------- | --------------------------------------------------------------- |
| ชื่อบริการ        | **IncidentTracking Service**                                    |
| ใช้ใน Endpoint    | `GET /incidents/{incident_id}`                                  |
| วิธีเรียก         | HTTP GET ไปยัง `{INCIDENT_SERVICE_URL}/incidents/{incident_id}` |
| Timeout           | 3 วินาที                                                        |
| ไฟล์ที่เกี่ยวข้อง | `src/backend/internal/client/incident_client.go`                |

### API Contract ที่ใช้อ้างอิง

MissionProgress Service เรียก IncidentTracking Service ผ่าน HTTP GET เพื่อดึงข้อมูลเหตุการณ์:

```
GET {INCIDENT_SERVICE_URL}/incidents/{incident_id}
```

**Response ที่คาดหวังจาก IncidentTracking Service (HTTP 200):**

```json
{
  "incident_id": "INC-001",
  "description": "น้ำท่วมหนักบริเวณถนนพหลโยธิน",
  "location": "13.7563,100.5018",
  "incident_type": "FLOOD"
}
```

| ฟิลด์           | ประเภท | คำอธิบาย                                     |
| --------------- | ------ | -------------------------------------------- |
| `incident_id`   | String | รหัสเหตุการณ์                                |
| `description`   | String | คำอธิบายเหตุการณ์                            |
| `location`      | String | พิกัด GPS ของเหตุการณ์                       |
| `incident_type` | String | ประเภทเหตุการณ์ เช่น `FLOOD`, `FIRE` เป็นต้น |

เมื่อเรียกสำเร็จ ข้อมูลเหล่านี้จะถูกรวมเข้ากับ response ของ `GET /incidents/{incident_id}` และ `data_source` จะเป็น `"full"`

### ส่วนที่ Mock ใน Demo 1

> **สำคัญ:** ใน Demo 1 นี้ **IncidentTracking Service ยังไม่พร้อมใช้งานจริง** จึงมีการ mock/จำลองการทำงานดังนี้:

| ส่วน                     | สิ่งที่จำลอง (Mock)                                                                | วิธีการจำลอง                                                                   |
| ------------------------ | ---------------------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| **URL ของบริการ**        | ตัวแปร `INCIDENT_SERVICE_URL` ถูกตั้งค่าเป็น `http://localhost:9999` (placeholder) | กำหนดใน `terraform/variables.tf` → ส่งเป็น environment variable ของ Lambda     |
| **พฤติกรรมการเรียก**     | HTTP Client จะ **timeout ภายใน 3 วินาที** เสมอ เพราะ URL ปลายทางไม่มีอยู่จริง      | ตั้ง timeout ใน `incident_client.go` → `http.Client{Timeout: 3 * time.Second}` |
| **การตอบกลับ**           | ฟังก์ชัน `GetIncidentDetail()` คืนค่า `nil` (แทน response จริง)                    | เมื่อ HTTP call ล้มเหลว → return `nil` → ระบบเข้า Degraded Mode                |
| **ข้อมูลที่หายไป**       | ฟิลด์ `description`, `location`, `incident_type` จะ**ไม่ปรากฏ**ใน response         | ฟิลด์เหล่านี้มี `omitempty` จึงไม่แสดงเมื่อเป็นค่าว่าง                         |
| **ตัวบ่งชี้ใน response** | `data_source` จะเป็น `"partial"` เสมอ (แทนที่จะเป็น `"full"`)                      | โค้ดตั้ง `dataSource = "partial"` เมื่อ `GetIncidentDetail()` คืนค่า `nil`     |

### Flow การทำงานใน Demo 1 (Degraded Mode)

```
GET /incidents/{incident_id}
        │
        ▼
  get-mission Lambda
        │
        ├── 1. ดึงข้อมูลภารกิจจาก DynamoDB (MissionAssignment) ✅
        ├── 2. ดึง Timeline จาก DynamoDB (MissionTimeline) ✅
        ├── 3. เรียก IncidentTracking Service ❌ (timeout 3 วินาที)
        │       └── return nil → Degraded Mode
        └── 4. ตอบกลับข้อมูลเฉพาะที่ตัวเองมี
                └── data_source: "partial" (ขาดข้อมูลจาก IncidentTracking)
```

### เมื่อ IncidentTracking Service พร้อมใช้งาน (Demo 2+)

เมื่อเพื่อน deploy IncidentTracking Service แล้ว สามารถเปลี่ยน URL ได้โดย:

1. แก้ไขค่า `incident_service_url` ใน Terraform variables
2. รัน `terraform apply` ใหม่

ระบบจะเปลี่ยนจาก Degraded Mode เป็น Full Mode โดยอัตโนมัติ — `data_source` จะเป็น `"full"` และมีฟิลด์ `description`, `location`, `incident_type` ครบ

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

> **หมายเหตุ:** ข้อมูลตัวอย่างจาก `seed-data.sh` ประกอบด้วย 5 ภารกิจ:
> | Mission | Incident | Team | สถานะเริ่มต้น |
> | --------- | -------- | ------------- | ------------- |
> | MSN-001 | INC-001 | TEAM-ALPHA | DISPATCHED |
> | MSN-002 | INC-002 | TEAM-BRAVO | EN_ROUTE |
> | MSN-003 | INC-003 | TEAM-CHARLIE | ON_SITE |
> | MSN-004 | INC-004 | TEAM-DELTA | NEED_BACKUP |
> | MSN-005 | INC-005 | TEAM-ECHO | RESOLVED |

---

#### 4.1 ทดสอบ Authentication

##### ✅ สำเร็จ — ส่ง API Key + Team ID ถูกต้อง

```bash
curl -s -H "x-api-key: $API_KEY" -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  "$API_URL/incidents/INC-001" | jq .
```

ผลลัพธ์ที่คาดหวัง: ได้รับข้อมูลภารกิจ (HTTP 200) — ดูตัวอย่าง response ในหัวข้อ 4.2

##### ❌ ล้มเหลว — ไม่ส่ง API Key

```bash
curl -s "$API_URL/incidents/INC-001" | jq .
```

ผลลัพธ์ที่คาดหวัง (HTTP 403):

```json
{
  "message": "Forbidden"
}
```

##### ❌ ล้มเหลว — ไม่ส่ง X-Rescue-Team-ID

```bash
curl -s -H "x-api-key: $API_KEY" \
  "$API_URL/incidents/INC-001" | jq .
```

ผลลัพธ์ที่คาดหวัง (HTTP 403):

```json
{
  "message": "User is not authorized to access this resource"
}
```

---

#### 4.2 ทดสอบ GET /incidents/{incident_id}

##### ✅ สำเร็จ — ดึงข้อมูลภารกิจที่มีอยู่ในระบบ

```bash
curl -s -H "x-api-key: $API_KEY" -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  "$API_URL/incidents/INC-001" | jq .
```

ผลลัพธ์ที่คาดหวัง (HTTP 200 — Degraded Mode เพราะ IncidentTracking ยังไม่พร้อม):

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

##### ❌ ล้มเหลว — ดึงข้อมูล incident_id ที่ไม่มีในระบบ

```bash
curl -s -H "x-api-key: $API_KEY" -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  "$API_URL/incidents/INC-99999" | jq .
```

ผลลัพธ์ที่คาดหวัง (HTTP 404):

```json
{
  "error": "INCIDENT_NOT_FOUND",
  "code": "INCIDENT_NOT_FOUND",
  "message": "No mission found for incident: INC-99999"
}
```

---

#### 4.3 ทดสอบ POST /incidents/{incident_id}/progress

##### ✅ สำเร็จ — อัปเดตสถานะถูกต้อง (DISPATCHED → EN_ROUTE)

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"new_status":"EN_ROUTE","note":"กำลังเดินทางไปจุดเกิดเหตุ"}' \
  "$API_URL/incidents/INC-001/progress" | jq .
```

ผลลัพธ์ที่คาดหวัง (HTTP 200):

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

##### ❌ ล้มเหลว — Transition สถานะไม่ถูกต้อง (EN_ROUTE → RESOLVED)

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-BRAVO" \
  -H "Content-Type: application/json" \
  -d '{"new_status":"RESOLVED"}' \
  "$API_URL/incidents/INC-002/progress" | jq .
```

ผลลัพธ์ที่คาดหวัง (HTTP 400):

```json
{
  "error": "INVALID_STATE_TRANSITION",
  "code": "INVALID_STATE_TRANSITION",
  "message": "Cannot transition from EN_ROUTE to RESOLVED"
}
```

##### ❌ ล้มเหลว — ไม่ส่ง new_status ใน body

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"note":"ทดสอบ"}' \
  "$API_URL/incidents/INC-001/progress" | jq .
```

ผลลัพธ์ที่คาดหวัง (HTTP 400):

```json
{
  "error": "MISSING_PARAMETER",
  "code": "MISSING_PARAMETER",
  "message": "new_status is required"
}
```

##### ❌ ล้มเหลว — ส่งค่า status ที่ไม่มีในระบบ

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"new_status":"UNKNOWN_STATUS"}' \
  "$API_URL/incidents/INC-001/progress" | jq .
```

ผลลัพธ์ที่คาดหวัง (HTTP 400):

```json
{
  "error": "INVALID_STATUS",
  "code": "INVALID_STATUS",
  "message": "Invalid status value: UNKNOWN_STATUS"
}
```

##### ❌ ล้มเหลว — incident_id ไม่มีในระบบ

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"new_status":"EN_ROUTE"}' \
  "$API_URL/incidents/INC-99999/progress" | jq .
```

ผลลัพธ์ที่คาดหวัง (HTTP 404):

```json
{
  "error": "INCIDENT_NOT_FOUND",
  "code": "INCIDENT_NOT_FOUND",
  "message": "No mission found for incident: INC-99999"
}
```

##### ❌ ล้มเหลว — ส่ง JSON body ไม่ถูกต้อง

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d 'invalid-json' \
  "$API_URL/incidents/INC-001/progress" | jq .
```

ผลลัพธ์ที่คาดหวัง (HTTP 400):

```json
{
  "error": "INVALID_BODY",
  "code": "INVALID_BODY",
  "message": "Invalid request body"
}
```

##### ❌ ล้มเหลว — อัปเดตภารกิจที่ RESOLVED แล้ว (สถานะสุดท้าย)

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ECHO" \
  -H "Content-Type: application/json" \
  -d '{"new_status":"EN_ROUTE"}' \
  "$API_URL/incidents/INC-005/progress" | jq .
```

ผลลัพธ์ที่คาดหวัง (HTTP 400):

```json
{
  "error": "INVALID_STATE_TRANSITION",
  "code": "INVALID_STATE_TRANSITION",
  "message": "Cannot transition from RESOLVED to EN_ROUTE"
}
```

---

#### 4.4 สรุปผลการทดสอบ

| #   | กรณีทดสอบ                                     | ประเภท     | HTTP Status | ผลที่คาดหวัง                     |
| --- | --------------------------------------------- | ---------- | ----------- | -------------------------------- |
| 1   | ส่ง API Key + Team ID ถูกต้อง                 | ✅ สำเร็จ  | 200         | ได้รับข้อมูลภารกิจ               |
| 2   | ไม่ส่ง API Key                                | ❌ ล้มเหลว | 403         | `Forbidden`                      |
| 3   | ไม่ส่ง X-Rescue-Team-ID                       | ❌ ล้มเหลว | 403         | `User is not authorized...`      |
| 4   | GET ภารกิจที่มีอยู่ (INC-001)                 | ✅ สำเร็จ  | 200         | ข้อมูลภารกิจ + Timeline          |
| 5   | GET ภารกิจที่ไม่มี (INC-99999)                | ❌ ล้มเหลว | 404         | `INCIDENT_NOT_FOUND`             |
| 6   | POST transition ถูกต้อง (DISPATCHED→EN_ROUTE) | ✅ สำเร็จ  | 200         | `Progress reported successfully` |
| 7   | POST transition ผิดกฎ (EN_ROUTE→RESOLVED)     | ❌ ล้มเหลว | 400         | `INVALID_STATE_TRANSITION`       |
| 8   | POST ไม่ส่ง new_status                        | ❌ ล้มเหลว | 400         | `MISSING_PARAMETER`              |
| 9   | POST ส่ง status ที่ไม่มีในระบบ                | ❌ ล้มเหลว | 400         | `INVALID_STATUS`                 |
| 10  | POST incident_id ไม่มีในระบบ                  | ❌ ล้มเหลว | 404         | `INCIDENT_NOT_FOUND`             |
| 11  | POST ส่ง JSON ไม่ถูกต้อง                      | ❌ ล้มเหลว | 400         | `INVALID_BODY`                   |
| 12  | POST อัปเดตภารกิจที่ RESOLVED แล้ว            | ❌ ล้มเหลว | 400         | `INVALID_STATE_TRANSITION`       |

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

## Demo 2 — Full Integration

### สิ่งที่เพิ่มจาก Demo 1

| หมวด                | รายการ                               | คำอธิบาย                                                         |
| ------------------- | ------------------------------------ | ---------------------------------------------------------------- |
| Lambda (ใหม่)       | `presigned-url`                      | ขอ S3 Presigned URL สำหรับอัปโหลดภาพหลักฐาน                      |
| Lambda (ใหม่)       | `list-missions`                      | ดึงรายการภารกิจทั้งหมดของทีม (GET /incidents)                    |
| Lambda (ใหม่)       | `mission-assigned-handler`           | รับ MissionAssignedEvent จาก Dispatch → สร้าง mission อัตโนมัติ  |
| Infrastructure      | S3 Evidence Bucket                   | เก็บภาพหลักฐานการปฏิบัติงาน + CORS สำหรับ PUT                    |
| API Endpoint (ใหม่) | `POST /incidents/{id}/presigned-url` | ขอ URL อัปโหลดภาพ (image/jpeg, image/png, image/webp)            |
| API Endpoint (ใหม่) | `GET /incidents`                     | ดึงรายการภารกิจของทีม กรองตามสถานะได้                            |
| Report Progress     | `image_key` field                    | รองรับแนบ S3 key ของภาพหลักฐานใน Timeline                        |
| EventBridge         | SQS Targets                          | ส่ง events ไป SQS ของ IncidentTracking, Dispatch, Prioritization |
| EventBridge         | Inbound MissionAssignedEvent         | รับ event จาก `dispatch-management-service`                      |
| IncidentTracking    | Real connection                      | เชื่อมต่อ HTTP จริง (ไม่ mock แล้ว) → `data_source: "full"`      |

---

### สถาปัตยกรรม Demo 2

```
Client (curl/Postman)
        │
        ▼
  API Gateway (REST)
   ├── Lambda Authorizer (ตรวจ API Key + Team ID)
   │
   ├── GET  /incidents/{id}              → get-mission Lambda
   │       ├── DynamoDB (MissionAssignment, MissionTimeline)
   │       └── HTTP → IncidentTracking Service (degraded mode)
   │
   ├── POST /incidents/{id}/progress     → report-progress Lambda
   │       ├── DynamoDB (อัปเดตสถานะ + เพิ่ม Timeline + image_key)
   │       ├── EventBridge (publish events)
   │       │       ├── CloudWatch Logs (3 rules)
   │       │       ├── IncidentTracking SQS
   │       │       ├── Dispatch SQS (RESOLVED only)
   │       │       └── Prioritization SQS
   │       └── Outbox Table (fallback เมื่อ EventBridge ล้มเหลว)
   │
   ├── POST /incidents/{id}/presigned-url → presigned-url Lambda [ใหม่]
   │       ├── DynamoDB (ตรวจว่า mission มีอยู่)
   │       └── S3 (สร้าง Presigned PUT URL)
   │
   └── GET  /incidents                    → list-missions Lambda [ใหม่]
           └── DynamoDB (query team-index GSI)

  EventBridge (Inbound)
   └── MissionAssignedEvent (from Dispatch)
           → mission-assigned-handler Lambda [ใหม่]
               ├── DynamoDB (สร้าง MissionAssignment)
               └── DynamoDB (สร้าง Timeline entry)
```

---

### โครงสร้างโปรเจค (อัปเดต)

```
CS366-MissionProgress-Service/
├── src/backend/
│   ├── cmd/
│   │   ├── report-progress/main.go        # Lambda: POST /incidents/{id}/progress
│   │   ├── get-mission/main.go            # Lambda: GET /incidents/{id}
│   │   ├── authorizer/main.go             # Lambda Authorizer
│   │   ├── outbox-processor/main.go       # Lambda: Outbox retry processor
│   │   ├── presigned-url/main.go          # Lambda: POST /incidents/{id}/presigned-url [ใหม่]
│   │   ├── list-missions/main.go          # Lambda: GET /incidents [ใหม่]
│   │   └── mission-assigned-handler/main.go # Lambda: EventBridge MissionAssigned [ใหม่]
│   ├── internal/
│   │   ├── models/                        # Structs: Mission, Timeline, Events, Outbox, Requests
│   │   ├── statemachine/                  # State transition validation
│   │   ├── repository/                    # DynamoDB CRUD (+ CreateMissionIdempotent, GetMissionsByTeamID)
│   │   ├── client/                        # HTTP Client (IncidentTracking)
│   │   ├── events/                        # EventBridge publisher + Outbox fallback
│   │   └── response/                      # HTTP response helpers
│   ├── go.mod
│   └── go.sum
├── terraform/
│   ├── main.tf                            # AWS provider + backend
│   ├── variables.tf                       # ตัวแปร Terraform (+ SQS ARNs)
│   ├── outputs.tf                         # Output: API URL, API Key
│   ├── iam.tf                             # LabRole reference
│   ├── dynamodb.tf                        # 3 DynamoDB tables
│   ├── lambda.tf                          # 7 Lambda functions (เดิม 4 + ใหม่ 3)
│   ├── api_gateway.tf                     # REST API + Authorizer + CORS (+ presigned-url, list-missions)
│   ├── eventbridge.tf                     # Event bus + rules + SQS targets + MissionAssigned rule
│   └── s3.tf                              # S3 Evidence bucket [ใหม่]
├── script/
│   ├── build.sh                           # Cross-compile Go → zip (7 functions)
│   ├── deploy.sh                          # Build + terraform apply
│   ├── seed-data.sh                       # Insert ข้อมูลตัวอย่างใน DynamoDB
│   └── destroy.sh                         # terraform destroy
├── plan/
│   ├── implement_plan.md
│   └── checkpoint/
│       ├── demo1.md
│       └── demo2.md
└── docs/
    ├── Service_Proposal.md
    └── contract/
        ├── contract_demo1.md
        └── contract_demo2.md
```

---

### API Endpoints ใหม่

#### POST /incidents/{incident_id}/presigned-url

ขอ S3 Presigned URL สำหรับอัปโหลดภาพหลักฐาน

**Request Body:**

```json
{
  "file_name": "flood-evidence.jpg",
  "content_type": "image/jpeg"
}
```

| Field          | จำเป็น | คำอธิบาย                                           |
| -------------- | ------ | -------------------------------------------------- |
| `file_name`    | ✅     | ชื่อไฟล์ภาพ                                        |
| `content_type` | ✅     | MIME type: `image/jpeg`, `image/png`, `image/webp` |

**Response 200:**

```json
{
  "upload_url": "https://s3.amazonaws.com/...",
  "image_key": "evidence/INC-001/TEAM-ALPHA/1718353500-flood-evidence.jpg",
  "expires_in": 300,
  "message": "Upload URL generated successfully. Use PUT method to upload."
}
```

**อัปโหลดภาพ:**

```bash
curl -X PUT -T photo.jpg -H "Content-Type: image/jpeg" "<upload_url>"
```

**Errors:** `400 INVALID_CONTENT_TYPE`, `400 MISSING_PARAMETER`, `404 INCIDENT_NOT_FOUND`, `500 PRESIGN_FAILED`

---

#### GET /incidents

ดึงรายการภารกิจทั้งหมดของทีม (ใช้ `X-Rescue-Team-ID` จาก header)

**Query Parameters:**

| Param    | จำเป็น | คำอธิบาย                                  |
| -------- | ------ | ----------------------------------------- |
| `status` | ❌     | กรองตามสถานะ เช่น `DISPATCHED`, `ON_SITE` |

**Response 200:**

```json
{
  "team_id": "TEAM-ALPHA",
  "total_missions": 2,
  "missions": [
    {
      "mission_id": "MSN-001",
      "incident_id": "INC-001",
      "rescue_team_id": "TEAM-ALPHA",
      "current_status": "ON_SITE",
      "latest_impact_level": 3,
      "started_at": "2024-12-01T08:00:00Z",
      "last_updated_at": "2024-12-01T10:00:00Z"
    }
  ]
}
```

- ถ้าไม่มี missions → return `200` กับ `missions: []` (ไม่ใช่ 404)
- ป้องกันทีมอื่นดึงข้อมูลข้ามทีม (ใช้ Team ID จาก Authorizer)

**Errors:** `400 INVALID_STATUS`

---

### POST /incidents/{incident_id}/progress — อัปเดต Demo 2

เพิ่มฟิลด์ `image_key` ใน request body:

```json
{
  "new_status": "ON_SITE",
  "note": "ถึงจุดเกิดเหตุแล้ว",
  "current_location": "13.7380,100.5230",
  "new_impact_level": 3,
  "image_key": "evidence/INC-001/TEAM-ALPHA/1718353500-flood-photo.jpg"
}
```

| Field ใหม่  | จำเป็น | คำอธิบาย                                             |
| ----------- | ------ | ---------------------------------------------------- |
| `image_key` | ❌     | S3 key ของภาพหลักฐาน (ได้จาก presigned-url endpoint) |

`image_key` จะถูกบันทึกใน Timeline entry

---

### EventBridge — อัปเดต Demo 2

#### Outbound Events → SQS Targets

| Event                    | เงื่อนไข                      | Targets                                                             |
| ------------------------ | ----------------------------- | ------------------------------------------------------------------- |
| `MissionStatusChanged`   | ทุกครั้งที่สถานะเปลี่ยน       | CloudWatch Logs, IncidentTracking SQS, Dispatch SQS (RESOLVED only) |
| `MissionBackupRequested` | สถานะใหม่ = `NEED_BACKUP`     | CloudWatch Logs, Prioritization SQS                                 |
| `ImpactLevelUpdated`     | มี `new_impact_level` ใน body | CloudWatch Logs, IncidentTracking SQS, Prioritization SQS           |

#### SQS Consumer Targets (ตั้งค่าผ่าน Terraform variables)

| Variable                    | Consumer Service | Events ที่ได้รับ                           |
| --------------------------- | ---------------- | ------------------------------------------ |
| `incident_tracking_sqs_arn` | IncidentTracking | MissionStatusChanged, ImpactLevelUpdated   |
| `dispatch_sqs_arn`          | Dispatch         | MissionStatusChanged (RESOLVED only)       |
| `prioritization_sqs_arn`    | Prioritization   | MissionBackupRequested, ImpactLevelUpdated |

> **หมายเหตุ:** ถ้าเพื่อนสร้าง SQS เอง → ขอให้เพื่อนเพิ่ม resource policy อนุญาต EventBridge ส่ง message ด้วย

#### Inbound Event (รับจาก Dispatch Service)

| Event                  | Source                        | Handler                           | การทำงาน                                    |
| ---------------------- | ----------------------------- | --------------------------------- | ------------------------------------------- |
| `MissionAssignedEvent` | `dispatch-management-service` | `mission-assigned-handler` Lambda | สร้าง mission (DISPATCHED) + Timeline entry |

**Expected Payload:**

```json
{
  "source": "dispatch-management-service",
  "detail-type": "MissionAssignedEvent",
  "detail": {
    "mission_id": "MSN-001",
    "rescue_unit_id": "TEAM-ALPHA",
    "incident_id": "INC-001",
    "assigned_at": "2025-06-14T08:45:00Z"
  }
}
```

- Idempotent: ใช้ `attribute_not_exists(mission_id)` — ถ้า mission_id ซ้ำจะ skip (ไม่ error)

---

### Dependencies — อัปเดต Demo 2

| บริการ               | วิธีเชื่อมต่อ                     | ตัวแปร Terraform            | Degraded Mode            |
| -------------------- | --------------------------------- | --------------------------- | ------------------------ |
| IncidentTracking     | HTTP GET `/incidents/{id}`        | `incident_service_url`      | `data_source: "partial"` |
| IncidentTracking SQS | EventBridge → SQS                 | `incident_tracking_sqs_arn` | CloudWatch Logs เดิม     |
| Dispatch SQS         | EventBridge → SQS (RESOLVED only) | `dispatch_sqs_arn`          | CloudWatch Logs เดิม     |
| Prioritization SQS   | EventBridge → SQS                 | `prioritization_sqs_arn`    | CloudWatch Logs เดิม     |

เมื่อ IncidentTracking Service พร้อมใช้งาน → ตั้งค่า `incident_service_url` ใน Terraform → `terraform apply` → ระบบจะเปลี่ยนจาก `data_source: "partial"` เป็น `"full"` อัตโนมัติ

---

### ทดสอบ API ใหม่ Demo 2

```bash
export API_URL="<API_GATEWAY_URL>"
export API_KEY="<API_KEY_VALUE>"
```

#### ✅ ขอ Presigned URL

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"file_name":"flood-photo.jpg","content_type":"image/jpeg"}' \
  "$API_URL/incidents/INC-001/presigned-url" | jq .
```

#### ✅ อัปโหลดภาพ + แนบใน Progress

```bash
# 1. ขอ presigned URL (ดึง upload_url และ image_key จาก response)
# 2. อัปโหลดภาพ
curl -X PUT -T photo.jpg -H "Content-Type: image/jpeg" "<upload_url>"
# 3. แนบ image_key ใน progress
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"new_status":"EN_ROUTE","note":"เดินทาง","image_key":"<image_key>"}' \
  "$API_URL/incidents/INC-001/progress" | jq .
```

#### ✅ ดึงรายการภารกิจทั้งหมดของทีม

```bash
curl -s -H "x-api-key: $API_KEY" -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  "$API_URL/incidents" | jq .
```

#### ✅ กรองตามสถานะ

```bash
curl -s -H "x-api-key: $API_KEY" -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  "$API_URL/incidents?status=DISPATCHED" | jq .
```

#### ❌ Presigned URL — content_type ไม่รองรับ

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"file_name":"doc.pdf","content_type":"application/pdf"}' \
  "$API_URL/incidents/INC-001/presigned-url" | jq .
```

ผลลัพธ์ที่คาดหวัง (HTTP 400): `INVALID_CONTENT_TYPE`

---

### Design Patterns — เพิ่มใน Demo 2

| Pattern          | คำอธิบาย                                                                     |
| ---------------- | ---------------------------------------------------------------------------- |
| Idempotent Write | MissionAssigned handler ใช้ conditional write ป้องกัน duplicate              |
| Presigned URL    | ให้ client อัปโหลดไฟล์ตรงไป S3 โดยไม่ผ่าน Lambda (ลด payload size + latency) |

---

### สรุปผลการทดสอบ Demo 2

| #   | กรณีทดสอบ                                 | ประเภท     | HTTP Status | ผลที่คาดหวัง                |
| --- | ----------------------------------------- | ---------- | ----------- | --------------------------- |
| 1   | POST presigned-url สำเร็จ                 | ✅ สำเร็จ  | 200         | Presigned URL + image_key   |
| 2   | POST presigned-url content_type ไม่รองรับ | ❌ ล้มเหลว | 400         | `INVALID_CONTENT_TYPE`      |
| 3   | GET /incidents ดึงรายการภารกิจ            | ✅ สำเร็จ  | 200         | รายการภารกิจของทีม          |
| 4   | GET /incidents กรองตามสถานะ               | ✅ สำเร็จ  | 200         | รายการภารกิจตามสถานะ        |
| 5   | GET /incidents ทีมไม่มีภารกิจ             | ✅ สำเร็จ  | 200         | `missions: []`              |
| 6   | POST progress พร้อม image_key             | ✅ สำเร็จ  | 200         | Timeline มี image_key       |
| 7   | MissionAssigned event → mission ถูกสร้าง  | ✅ สำเร็จ  | —           | DISPATCHED + Timeline entry |
| 8   | MissionAssigned event ซ้ำ → skip          | ✅ สำเร็จ  | —           | ไม่ error (idempotent)      |
