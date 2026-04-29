# MissionProgress Service

> CS366 — รัฐธรรมนูญ โคสาแสง (6609612178)

---

## ภาพรวม

**MissionProgress Service** คือบริการสำหรับทีมกู้ภัย (Rescue Team) ใช้รายงานความคืบหน้าของภารกิจที่ได้รับมอบหมาย อัปเดตสถานะ และบันทึกรายละเอียดการปฏิบัติงานหน้างาน (Timeline) เพื่อให้ศูนย์สั่งการได้รับข้อมูลที่ถูกต้องและเป็นปัจจุบัน

**ขอบเขตความรับผิดชอบของ Service นี้:**

| รับผิดชอบ                                 | ไม่รับผิดชอบ                                                      |
| ----------------------------------------- | ----------------------------------------------------------------- |
| การเปลี่ยนสถานะภารกิจตาม State Machine    | การสั่งการ / มอบหมายงาน (→ Manage Dispatch Service)               |
| การบันทึก Timeline / Action Log           | การเป็น Source of Truth ของ Incident (→ IncidentTracking Service) |
| การรับและจัดเก็บหลักฐานภาพถ่าย (Evidence) | การค้นหาเส้นทาง (→ SafeRoute Service)                             |
| การ publish Domain Events ไป EventBridge  | การจัดการทรัพยากรโรงพยาบาล (→ HospitalResourceStatus Service)     |
| Degraded Mode เมื่อ Service อื่นไม่ตอบ    | การจัดการ Profile / ความสามารถของทีมกู้ภัย (→ RescueTeam Service) |

---

## สถาปัตยกรรม

```
Client (Web App / Postman)
        │
        ▼
  API Gateway (REST)
        │
        ▼
  Lambda Authorizer ─── ตรวจสอบ x-api-key + X-Rescue-Team-ID
        │
        ├── GET  /missions/{request_id}
        │         ├── DynamoDB: MissionAssignment (อ่านสถานะ + ข้อมูลภารกิจ)
        │         ├── DynamoDB: MissionTimeline   (อ่าน Timeline ทั้งหมด)
        │         └── HTTP GET → RescueRequest Service (degraded เมื่อล้มเหลว)
        │
        ├── GET  /missions?team_id={team_id}
        │         └── DynamoDB: MissionAssignment (query GSI team-index)
        │
        ├── POST /missions/{request_id}/progress
        │         ├── DynamoDB: MissionAssignment (อัปเดตสถานะ)
        │         ├── DynamoDB: MissionTimeline   (เพิ่ม entry ใหม่)
        │         ├── EventBridge: publish MissionStatusChanged
        │         ├── EventBridge: publish MissionBackupRequested  (เมื่อ NEED_BACKUP)
        │         ├── EventBridge: publish ImpactLevelUpdated      (เมื่อมี new_impact_level)
        │         └── DynamoDB: EventOutbox (Outbox fallback เมื่อ EventBridge ล้มเหลว)
        │
        └── POST /missions/{request_id}/presigned-url
                  ├── DynamoDB: MissionAssignment (ตรวจสอบว่าภารกิจมีอยู่)
                  └── Amazon S3: สร้าง Presigned PUT URL สำหรับอัปโหลดรูป

EventBridge (mission-progress-events)
        ├── Rule: MissionStatusChanged → IncidentTracking, Manage Dispatch (RESOLVED)
        ├── Rule: MissionBackupRequested → Rescue Prioritization
        └── Rule: ImpactLevelUpdated → IncidentTracking, Rescue Prioritization

EventBridge (ขาเข้า)
        └── DispatchOrderCreated ← Manage Dispatch Service
                  └── mission-assigned-handler Lambda → สร้าง MissionAssignment
```

---

## Tech Stack

| Layer           | เทคโนโลยี                                     |
| --------------- | --------------------------------------------- |
| Backend         | Go 1.21 (AWS Lambda, `provided.al2023`)       |
| API Gateway     | Amazon API Gateway (REST) + Lambda Authorizer |
| Database        | Amazon DynamoDB (PAY_PER_REQUEST)             |
| Async Messaging | Amazon EventBridge                            |
| Object Storage  | Amazon S3 (Evidence Images)                   |
| IaC             | Terraform ~5.0                                |
| Frontend        | Next.js (Static Export บน S3)                 |

---

## โครงสร้างโปรเจค

```
CS366-MissionProgress-Service/
├── src/
│   ├── backend/
│   │   ├── cmd/
│   │   │   ├── authorizer/              # Lambda Authorizer (ตรวจ API Key + Team ID)
│   │   │   ├── get-mission/             # GET /missions/{request_id}
│   │   │   ├── list-missions/           # GET /missions?team_id={team_id}
│   │   │   ├── report-progress/         # POST /missions/{request_id}/progress
│   │   │   ├── presigned-url/           # POST /missions/{request_id}/presigned-url
│   │   │   ├── mission-assigned-handler/ # EventBridge: DispatchOrderCreated → สร้าง Mission
│   │   │   └── outbox-processor/        # Scheduled: retry failed EventBridge publishes
│   │   ├── internal/
│   │   │   ├── models/                  # Structs: MissionAssignment, TimelineEntry, Events
│   │   │   ├── statemachine/            # State transition validation
│   │   │   ├── repository/              # DynamoDB CRUD (MissionRepo, TimelineRepo, OutboxRepo)
│   │   │   ├── client/                  # HTTP Clients (RescueRequest, ManageDispatch, RescueTeam)
│   │   │   ├── events/                  # EventBridge publisher + Outbox fallback
│   │   │   └── response/                # HTTP response helpers
│   │   ├── go.mod
│   │   └── go.sum
│   └── frontend/                        # Next.js Web App
├── terraform/
│   ├── main.tf                          # AWS provider + S3 backend
│   ├── variables.tf                     # Terraform variables
│   ├── outputs.tf                       # Output: API URL, API Key
│   ├── iam.tf                           # LabRole reference
│   ├── dynamodb.tf                      # 3 DynamoDB tables
│   ├── lambda.tf                        # 7 Lambda functions
│   ├── api_gateway.tf                   # REST API + Authorizer + CORS
│   ├── eventbridge.tf                   # Event bus + rules
│   └── s3.tf                            # Evidence bucket
├── script/
│   ├── build.sh                         # Cross-compile Go → zip
│   ├── deploy.sh                        # Build + terraform apply
│   ├── seed-data.sh                     # Insert ข้อมูลตัวอย่างใน DynamoDB
│   └── destroy.sh                       # terraform destroy
├── docs/
│   └── proposals/                       # Service Proposal (Overview, Contracts, Architecture)
└── plan/
    └── refactor/                        # Refactor notes ตาม phase ต่างๆ
```

---

## DynamoDB Tables

### MissionAssignment

เก็บข้อมูลหลักของภารกิจแต่ละชิ้น

| Attribute             | ชนิด        | คำอธิบาย                                       |
| --------------------- | ----------- | ---------------------------------------------- |
| `mission_id`          | String (PK) | รหัสภารกิจ (สร้างโดย Service นี้)              |
| `dispatch_id`         | String      | รหัส Dispatch Order จาก Manage Dispatch        |
| `request_id`          | String      | รหัส Rescue Request ที่ผูกกับภารกิจนี้         |
| `incident_id`         | String      | รหัส Incident (ดึงมาจาก RescueRequest Service) |
| `rescue_team_id`      | String      | ทีมที่รับผิดชอบ                                |
| `current_status`      | String      | สถานะปัจจุบัน                                  |
| `latest_impact_level` | Number      | Impact Level ล่าสุดจากหน้างาน                  |
| `started_at`          | String      | เวลาเริ่มภารกิจ                                |
| `last_updated_at`     | String      | เวลาอัปเดตล่าสุด                               |

**GSIs:**

| Index            | Hash Key         | ใช้สำหรับ                         |
| ---------------- | ---------------- | --------------------------------- |
| `team-index`     | `rescue_team_id` | ดึงภารกิจทั้งหมดของทีม            |
| `request-index`  | `request_id`     | ค้นหาภารกิจด้วย request_id        |
| `dispatch-index` | `dispatch_id`    | Idempotency check ตาม dispatch_id |

### MissionTimeline

เก็บ Log การปฏิบัติงานทุกรายการของแต่ละภารกิจ

| Attribute      | ชนิด        | คำอธิบาย                                                |
| -------------- | ----------- | ------------------------------------------------------- |
| `mission_id`   | String (PK) | รหัสภารกิจ                                              |
| `timestamp`    | String (SK) | เวลาบันทึก (ISO 8601)                                   |
| `log_id`       | String      | รหัส Log (UUID)                                         |
| `action_type`  | String      | ประเภทการกระทำ เช่น `STATUS_CHANGE`, `MISSION_ASSIGNED` |
| `description`  | String      | คำอธิบาย                                                |
| `performed_by` | String      | ผู้กระทำ                                                |

### EventOutbox

เก็บ Events ที่ publish ไป EventBridge ไม่สำเร็จ สำหรับ retry ภายหลัง

| Attribute    | ชนิด        | คำอธิบาย                        |
| ------------ | ----------- | ------------------------------- |
| `outbox_id`  | String (PK) | UUID                            |
| `created_at` | String (SK) | เวลาสร้าง                       |
| `status`     | String      | `PENDING` หรือ `SENT`           |
| `ttl`        | Number      | TTL สำหรับ DynamoDB auto-delete |

**GSI:** `status-index` (hash: `status`) — ใช้สำหรับ outbox-processor query events ที่ยังค้างอยู่

---

## Authentication

ทุก Request ไป API Gateway ต้องส่ง headers:

| Header             | คำอธิบาย                                     |
| ------------------ | -------------------------------------------- |
| `x-api-key`        | API Key สำหรับ authenticate กับ API Gateway  |
| `X-Rescue-Team-ID` | รหัสทีมกู้ภัย (ใช้ระบุตัวตนและ scope สิทธิ์) |

Lambda Authorizer ทำหน้าที่ตรวจสอบทั้งสอง header ก่อน route ไป Lambda จริง  
หาก header ขาดหรือ API Key ไม่ถูกต้อง → `403 Unauthorized` ทันที (ไม่ผ่าน Lambda)

---

## API Endpoints

Base URL: `https://<api-id>.execute-api.<region>.amazonaws.com/prod`

---

### GET /missions/{request_id}

ดึงข้อมูลภารกิจที่ผูกกับ `request_id` นี้ พร้อม Timeline ทั้งหมด

**Path Parameter:**

| Parameter    | คำอธิบาย                           |
| ------------ | ---------------------------------- |
| `request_id` | รหัส Rescue Request เช่น `REQ-001` |

**ขั้นตอนการทำงานภายใน:**

1. ค้นหา MissionAssignment ด้วย `request-index` GSI
2. ดึง MissionTimeline ทั้งหมดของภารกิจนั้น
3. เรียก RescueRequest Service เพื่อดึงข้อมูล Request (description, location, type)
   - สำเร็จ → `data_source: "full"`
   - ล้มเหลว / timeout → ส่งข้อมูลเฉพาะส่วนที่มีใน DynamoDB → `data_source: "partial"` (Degraded Mode)

---

### GET /missions?team_id={team_id}

ดึงรายการภารกิจทั้งหมดของทีมที่ระบุ

**Query Parameter:**

| Parameter | คำอธิบาย                        |
| --------- | ------------------------------- |
| `team_id` | รหัสทีมกู้ภัย เช่น `TEAM-ALPHA` |

**ขั้นตอนการทำงานภายใน:**

1. Query `team-index` GSI บน MissionAssignment table ด้วย `rescue_team_id`
2. Return รายการภารกิจทั้งหมดของทีม

---

### POST /missions/{request_id}/progress

อัปเดตสถานะภารกิจและบันทึก Timeline entry ใหม่

**Path Parameter:**

| Parameter    | คำอธิบาย            |
| ------------ | ------------------- |
| `request_id` | รหัส Rescue Request |

**Request Body:**

| Field              | ชนิด          | Required | คำอธิบาย                                                           |
| ------------------ | ------------- | -------- | ------------------------------------------------------------------ |
| `new_status`       | String (Enum) | ✅       | สถานะใหม่: `EN_ROUTE`, `ON_SITE`, `NEED_BACKUP`, `RESOLVED`        |
| `note`             | String        | ❌       | รายละเอียด / หมายเหตุการปฏิบัติงาน                                 |
| `new_impact_level` | String        | ❌       | ระดับความรุนแรงใหม่จากหน้างาน — trigger `ImpactLevelUpdated` event |

**ขั้นตอนการทำงานภายใน:**

1. ค้นหาภารกิจด้วย `request-index` GSI
2. ตรวจสอบ State Transition ว่าถูกต้องตาม State Machine
3. อัปเดต MissionAssignment (สถานะ, impact level, last_updated_at)
4. เพิ่ม TimelineEntry ใหม่ใน MissionTimeline
5. Publish Events ไป EventBridge:
   - `MissionStatusChanged` — ทุกครั้งที่สถานะเปลี่ยน
   - `MissionBackupRequested` — เมื่อ `new_status = NEED_BACKUP`
   - `ImpactLevelUpdated` — เมื่อมี `new_impact_level` ใน request
6. หาก EventBridge ล้มเหลว → บันทึกลง EventOutbox (Outbox Pattern) — Response ยังคง `200`

---

### POST /missions/{request_id}/presigned-url

ขอ S3 Presigned PUT URL สำหรับอัปโหลดรูปภาพหลักฐาน

**Path Parameter:**

| Parameter    | คำอธิบาย            |
| ------------ | ------------------- |
| `request_id` | รหัส Rescue Request |

**Request Body:**

| Field          | ชนิด   | Required | คำอธิบาย                     |
| -------------- | ------ | -------- | ---------------------------- |
| `file_name`    | String | ✅       | ชื่อไฟล์รูป เช่น `scene.jpg` |
| `content_type` | String | ✅       | MIME type เช่น `image/jpeg`  |

**ขั้นตอนการทำงานภายใน:**

1. ค้นหาภารกิจด้วย `request-index` GSI เพื่อตรวจสอบว่ามีอยู่
2. สร้าง S3 key ในรูปแบบ `evidence/{mission_id}/{rescue_team_id}/{timestamp}-{file_name}`
3. สร้าง Presigned PUT URL (อายุ 5 นาที) สำหรับ upload ตรงไปยัง S3
4. Client ใช้ URL ที่ได้ PUT ไฟล์โดยตรงโดยไม่ผ่าน API Gateway

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

| สถานะปัจจุบัน | เปลี่ยนไปได้              |
| ------------- | ------------------------- |
| `DISPATCHED`  | `EN_ROUTE`                |
| `EN_ROUTE`    | `ON_SITE`                 |
| `ON_SITE`     | `NEED_BACKUP`, `RESOLVED` |
| `NEED_BACKUP` | `ON_SITE`, `RESOLVED`     |

การ transition ที่ไม่ถูกต้อง → `400 INVALID_STATE_TRANSITION`  
สถานะ `RESOLVED` เป็น terminal state — ไม่สามารถเปลี่ยนต่อได้

---

## การเชื่อมต่อกับ Services อื่น

### Services ที่ Service นี้เรียกออก (Outbound)

#### 1. RescueRequest Service (Synchronous)

| รายละเอียด   | ค่า                                                    |
| ------------ | ------------------------------------------------------ |
| เจ้าของ      | Phattharaphum Kingchai                                 |
| Criticality  | High                                                   |
| Endpoint     | `GET /v1/rescue-requests/{requestId}`                  |
| Auth         | `Authorization: Bearer <RESCUE_REQUEST_SERVICE_TOKEN>` |
| ใช้ใน Lambda | `get-mission`, `mission-assigned-handler`              |

**วัตถุประสงค์:**

- `get-mission` — ดึงข้อมูล Request (description, location, type, peopleCount) เพื่อแนบกับ response
- `mission-assigned-handler` — ดึง `incident_id` ที่ผูกกับ request นั้น เพื่อบันทึกลง MissionAssignment

**Failure Handling:**

- `get-mission`: หาก timeout/error → Degraded Mode (`data_source: "partial"`) — ไม่ fail
- `mission-assigned-handler`: หากเรียกไม่ได้ → บันทึก `incident_id = ""` พร้อม log warning — ไม่ fail

---

#### 2. Manage Dispatch Service (Synchronous)

| รายละเอียด   | ค่า                                                     |
| ------------ | ------------------------------------------------------- |
| เจ้าของ      | Noppakron Songkroh                                      |
| Criticality  | Medium                                                  |
| Auth         | `Authorization: Bearer <MANAGE_DISPATCH_SERVICE_TOKEN>` |
| ใช้ใน Lambda | `get-mission`                                           |

**วัตถุประสงค์:**

- ดึงข้อมูล Dispatch Order เพื่อเสริมข้อมูลใน response

**Failure Handling:** Degraded Mode — ไม่ fail request หลัก

---

#### 3. RescueTeam Service (Synchronous)

| รายละเอียด   | ค่า                                                 |
| ------------ | --------------------------------------------------- |
| เจ้าของ      | กมลพันธ์ กันธายอด                                   |
| Auth         | `Authorization: Bearer <RESCUE_TEAM_SERVICE_TOKEN>` |
| ใช้ใน Lambda | `get-mission`                                       |

**วัตถุประสงค์:**

- ดึงข้อมูลทีมกู้ภัย (ชื่อ, ความสามารถ)

**Failure Handling:** Degraded Mode — ไม่ fail request หลัก

---

### Events ที่ Service นี้ Publish ออก (Outbound Async)

ทุก Event ถูก publish ผ่าน **Amazon EventBridge** (event bus: `mission-progress-events`)  
หาก publish ล้มเหลว → บันทึกลง EventOutbox และ retry โดย `outbox-processor` Lambda

| Event                    | Trigger                               | Consumers                                                                           |
| ------------------------ | ------------------------------------- | ----------------------------------------------------------------------------------- |
| `MissionStatusChanged`   | ทุกครั้งที่สถานะเปลี่ยน               | IncidentTracking (อัปเดต Incident), Manage Dispatch (เมื่อ `RESOLVED` → ปลดล็อกทีม) |
| `MissionBackupRequested` | เมื่อ `new_status = NEED_BACKUP`      | Rescue Prioritization (คำนวณ Priority ใหม่)                                         |
| `ImpactLevelUpdated`     | เมื่อมี `new_impact_level` ใน request | IncidentTracking (อัปเดต Impact Level), Rescue Prioritization                       |

**โครงสร้าง Event (EventBridge envelope):**

```json
{
  "source": "mission-progress-service",
  "detail-type": "<EventName>",
  "detail": {
    "schema_version": "1.0",
    "mission_id": "...",
    "incident_id": "...",
    "rescue_team_id": "...",
    ...
  }
}
```

---

### Events ที่ Service นี้รับเข้า (Inbound Async)

| Event                  | Producer                | Lambda ที่รับ              | การกระทำ                                                                |
| ---------------------- | ----------------------- | -------------------------- | ----------------------------------------------------------------------- |
| `DispatchOrderCreated` | Manage Dispatch Service | `mission-assigned-handler` | สร้าง MissionAssignment ใหม่ พร้อม idempotency check ด้วย `dispatch_id` |

**ขั้นตอนของ `mission-assigned-handler`:**

1. Parse `DispatchOrderCreated` payload (dispatchId, requestId, teamId, priorityLevel)
2. Idempotency check — query `dispatch-index` GSI เพื่อป้องกัน duplicate
3. เรียก RescueRequest Service เพื่อดึง `incident_id` ที่ผูกกับ `requestId`
4. สร้าง MissionAssignment record (missionId = `MISS-<uuid[:8]>`)
5. สร้าง TimelineEntry แรก (`MISSION_ASSIGNED`)

---

## Outbox Pattern

เมื่อ EventBridge publish ล้มเหลว บริการจะ **ไม่ fail** request แต่บันทึก Event ลง EventOutbox table แทน

`outbox-processor` Lambda ทำงานแบบ Scheduled (ตั้งเวลา) เพื่อ retry Events ที่ค้างอยู่โดยอัตโนมัติ  
EventBridge เองมี built-in retry นาน 24 ชั่วโมง

---

## การ Deploy

### Prerequisites

- AWS CLI ตั้งค่า credentials แล้ว (ใช้ LabRole)
- Terraform >= 1.0
- Go >= 1.21
- Bash

### ขั้นตอน

**1. Build Lambda binaries ทั้งหมด**

```bash
cd src/backend
./../../script/build.sh
```

Script จะ cross-compile Go → `linux/arm64` และบรรจุเป็น zip ไว้ที่ `terraform/build/`

**2. ตั้งค่า Terraform variables**

สร้างไฟล์ `terraform/terraform.tfvars`:

```hcl
aws_region                    = "us-east-1"
lab_role_arn                  = "arn:aws:iam::<account-id>:role/LabRole"
api_key_value                 = "mission-progress-token-default"
rescue_request_service_url    = "<url>"
rescue_request_service_token  = "<token>"
manage_dispatch_service_url   = "<url>"
manage_dispatch_service_token = "<token>"
rescue_team_service_url       = "<url>"
rescue_team_service_token     = "<token>"
```

**3. Deploy ด้วย Terraform**

```bash
cd terraform
terraform init
terraform apply
```

หรือใช้ script รวม:

```bash
./script/deploy.sh
```

**4. ดู Outputs**

หลัง apply สำเร็จ Terraform จะแสดง:

- `api_gateway_url` — Base URL ของ API
- `api_key_value` — API Key สำหรับ authentication

**5. Seed ข้อมูลตัวอย่าง (สำหรับ dev/demo)**

```bash
./script/seed-data.sh
```

### Destroy

```bash
./script/destroy.sh
```

---

## Error Codes

| Code                       | HTTP Status | ความหมาย                                    |
| -------------------------- | ----------- | ------------------------------------------- |
| `UNAUTHORIZED`             | 403         | API Key หรือ Team ID ไม่ถูกต้อง             |
| `MISSION_NOT_FOUND`        | 404         | ไม่พบภารกิจสำหรับ request_id นี้            |
| `INVALID_STATE_TRANSITION` | 400         | การเปลี่ยนสถานะไม่ถูกต้องตาม State Machine  |
| `INVALID_STATUS`           | 400         | ค่า `new_status` ไม่อยู่ในรายการที่รองรับ   |
| `INVALID_REQUEST_BODY`     | 400         | Request Body ไม่ถูกต้องหรือขาด field จำเป็น |
| `PRESIGN_FAILED`           | 500         | สร้าง Presigned URL ไม่สำเร็จ               |
| `INTERNAL_ERROR`           | 500         | ข้อผิดพลาดภายใน                             |
