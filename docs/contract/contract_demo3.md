# MissionProgress Service — API Contract (Demo 3: Full Cross-Service Integration)

**Service Owner:** นายรัฐธรรมนูญ โคสาแสง (6609612178)

**วัตถุประสงค์ของบริการ:**
MissionProgress Service คือบริการสำหรับทีมกู้ภัย (Rescue Team) เพื่อรายงานความคืบหน้าของภารกิจ อัปเดตสถานะเหตุการณ์ และบันทึกรายละเอียดการปฏิบัติงานหน้างาน เพื่อให้ศูนย์สั่งการได้รับข้อมูลที่ถูกต้องและเป็นปัจจุบัน

**ขอบเขต Demo 3 (Full Cross-Service Integration):**

- เป็นเวอร์ชันสมบูรณ์ที่สุด — เชื่อมต่อ **3 External Services แบบ HTTP พร้อมกัน** (RescueRequest + ManageDispatch + RescueTeam)
- `GET /missions/{request_id}` เรียก 3 Services แบบ Parallel → ตอบ Response ที่สมบูรณ์ที่สุด
- ทุก Terraform variable ใช้ค่าจริง (ไม่มี hardcoded tokens)
- S3 key ใช้ `mission_id` ที่ถูกต้อง (ไม่ใช่ `incident_id` ที่ deprecated)
- DynamoDB ถูก cleanup — ไม่มี GSI ที่ไม่ได้ใช้
- Demo scenario แบบ end-to-end ตั้งแต่ต้นจนจบ

**สิ่งที่เพิ่มจาก Demo 2:**

| ฟีเจอร์                                     | Demo 2                        | Demo 3                           |
| ------------------------------------------- | ----------------------------- | -------------------------------- |
| ManageDispatch Integration (HTTP)           | ❌                            | ✅ GET /v1/dispatches?teamId=... |
| RescueTeam Integration (HTTP)               | ❌                            | ✅ GET /v1/teams/{teamId}        |
| `dispatch_status` ใน GET response           | ❌                            | ✅                               |
| `priority_level` ใน GET response            | ❌                            | ✅                               |
| `team_name`, `team_type` ใน GET response    | ❌                            | ✅                               |
| `capabilities`, `equipment` ใน GET response | ❌                            | ✅                               |
| `team_location` ใน GET response             | ❌                            | ✅                               |
| Parallel HTTP calls (3 services at once)    | ❌ (sequential, 1 service)    | ✅ (parallel, ≤800ms worst-case) |
| Terraform tokens ไม่ hardcoded              | ❌ (มี mock defaults)         | ✅ (ค่าว่าง — ต้องตั้งค่าจริง)   |
| S3 key ใช้ `mission_id`                     | ❌ (ใช้ `incident_id` ที่ผิด) | ✅                               |
| `incident_type` ใน response                 | `requestType` (ชื่อเก่า)      | `incident_type` (ชื่อใหม่)       |
| DynamoDB cleanup                            | มี dead `incident-index`      | ✅ cleanup แล้ว                  |
| RescueRequest Integration                   | Full                          | Full (ยังคงมี)                   |
| Frontend (Next.js)                          | ✅                            | ✅                               |
| S3 Evidence Upload (presigned)              | ✅                            | ✅                               |
| List Missions (GET /missions)               | ✅                            | ✅                               |
| Inbound Event (MissionAssignedEvent)        | ✅                            | ✅                               |
| Outbox Processor (retry)                    | ✅                            | ✅                               |
| EventBridge → SQS Routing                   | ✅ SQS targets                | ✅ SQS targets                   |

---

## การพึ่งพาบริการภายนอก (Dependencies)

### บริการที่พึ่งพา — Sync HTTP (3 Services, เรียกแบบ Parallel)

| บริการ                     | เจ้าของ                | Endpoint ที่เรียก                      | ตัวแปร Env                                                     | ข้อมูลที่ได้รับ                                              |
| -------------------------- | ---------------------- | -------------------------------------- | -------------------------------------------------------------- | ------------------------------------------------------------ |
| **RescueRequest Service**  | Phattharaphum Kingchai | `GET /v1/rescue-requests/{request_id}` | `RESCUE_REQUEST_SERVICE_URL`, `RESCUE_REQUEST_SERVICE_TOKEN`   | description, location, requestType                           |
| **ManageDispatch Service** | Noppakron Songkroh     | `GET /v1/dispatches?teamId={teamId}`   | `MANAGE_DISPATCH_SERVICE_URL`, `MANAGE_DISPATCH_SERVICE_TOKEN` | dispatch_status, priority_level                              |
| **RescueTeam Service**     | กมลพันธ์ กันธายอด      | `GET /v1/teams/{teamId}`               | `RESCUE_TEAM_SERVICE_URL`, `RESCUE_TEAM_SERVICE_TOKEN`         | team_name, team_type, capabilities, equipment, team_location |

**Timeout:** 800ms per service + retry 2 ครั้ง สำหรับ 5xx  
**Degraded Mode:** ถ้า service ใดตอบไม่ได้ → skip ฟิลด์นั้น + `data_source: "partial"`

### API Contract ที่ใช้อ้างอิง — RescueRequest Service

```
GET {RESCUE_REQUEST_SERVICE_URL}/v1/rescue-requests/{requestId}
Authorization: Bearer <RESCUE_REQUEST_SERVICE_TOKEN>
```

**Response ที่คาดหวัง (HTTP 200):**

```json
{
  "request": {
    "requestId": "REQ-001",
    "description": "น้ำท่วมหนักบริเวณถนนพหลโยธิน",
    "location": {
      "latitude": 13.7563,
      "longitude": 100.5018
    },
    "requestType": "FLOOD"
  }
}
```

### API Contract ที่ใช้อ้างอิง — ManageDispatch Service

```
GET {MANAGE_DISPATCH_SERVICE_URL}/v1/dispatches?teamId={teamId}
Authorization: Bearer <MANAGE_DISPATCH_SERVICE_TOKEN>
```

**Response ที่คาดหวัง (HTTP 200):**

```json
{
  "items": [
    {
      "dispatch_id": "DSP-001",
      "status": "ACTIVE",
      "priority_level": 3
    }
  ]
}
```

### API Contract ที่ใช้อ้างอิง — RescueTeam Service

```
GET {RESCUE_TEAM_SERVICE_URL}/v1/teams/{teamId}
Authorization: Bearer <RESCUE_TEAM_SERVICE_TOKEN>
```

**Response ที่คาดหวัง (HTTP 200):**

```json
{
  "team_name": "Alpha Rescue Unit",
  "team_type": "FLOOD",
  "capabilities": ["boat_rescue", "swift_water"],
  "equipment": ["inflatable_boat", "life_vest"],
  "location": {
    "lat": 13.7563,
    "lng": 100.5018
  }
}
```

### Async Dependencies (EventBridge)

| บริการ              | เจ้าของ               | วิธีเชื่อมต่อ                       | ตัวแปร Terraform               | Degraded Mode                  |
| ------------------- | --------------------- | ----------------------------------- | ------------------------------ | ------------------------------ |
| IncidentTracking    | Krittamet Damthongkam | EventBridge → Lambda (direct)       | `incident_tracking_lambda_arn` | CloudWatch Logs fallback       |
| Dispatch (RESOLVED) | Noppakron Songkroh    | Sync PATCH fire-and-forget (no SQS) | `MANAGE_DISPATCH_SERVICE_URL`  | Log WARN + pass (non-blocking) |
| Prioritization SQS  | Nattasak Chonmanat    | EventBridge → SQS                   | `prioritization_sqs_arn`       | CloudWatch Logs fallback       |

### Inbound Async Event (from Manage Dispatch via SNS)

| รายการ         | ค่า                                                      |
| -------------- | -------------------------------------------------------- |
| Channel        | SNS Topic `rescue.mission.dispatch.v1`                   |
| Topic ARN      | `arn:aws:sns:us-east-1:460581038623:request-dispatch-v1` |
| messageType    | `DispatchOrderCreated` (in `header.messageType`)         |
| Handler Lambda | `mission-assigned-handler`                               |

---

## 1. Base URL & Environment

```
https://<api-id>.execute-api.us-east-1.amazonaws.com/v1
```

| รายการ       | ค่า                |
| ------------ | ------------------ |
| Region       | `us-east-1`        |
| Stage        | `v1`               |
| Protocol     | HTTPS              |
| Content-Type | `application/json` |

> **หมายเหตุ:** `<api-id>` จะได้จาก Terraform output หลัง deploy — จะแจ้งให้เพื่อนทราบก่อนวัน Demo  
> ถ้าใช้ Frontend: เปิด URL ของ S3 Static Website ที่ได้จาก Terraform output `frontend_url`

---

## 2. Authentication (จำเป็นสำหรับทุก Request)

ทุก Request ต้องส่ง **2 Headers** ดังนี้:

| Header             | ค่า                             | คำอธิบาย                                         |
| ------------------ | ------------------------------- | ------------------------------------------------ |
| `x-api-key`        | `<api-key ที่แจกให้ในวัน demo>` | API Key สำหรับยืนยันสิทธิ์การเรียกใช้บริการ      |
| `X-Rescue-Team-ID` | เช่น `TEAM-ALPHA`               | รหัสทีมกู้ภัยที่กำลังเรียก API (ห้ามเป็นค่าว่าง) |

### ตัวอย่างการส่ง Header (curl)

```bash
curl -X GET \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions/REQ-001" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA"
```

### กรณี Authentication ไม่ผ่าน

```
HTTP/1.1 403 Forbidden

{ "message": "User is not authorized to access this resource" }
```

---

## Demo Flow — ขั้นตอนที่ 0: ตั้งค่า Environment ก่อนเริ่ม

> **สำหรับผู้ที่ไม่รู้จัก Service นี้มาก่อน** — อ่านส่วนนี้ก่อนแล้วค่อยดำเนินการ

### วิธีได้รับ API Key และ Team ID

1. ถามเจ้าของ Service (รัฐธรรมนูญ) ให้แจก `api-key` และ `api-id` ในวันนั้น
2. `X-Rescue-Team-ID` ใช้ค่าใดก็ได้ เช่น `TEAM-ALPHA`, `TEAM-BRAVO` — ค่านี้จะเป็นตัวกรองข้อมูลภารกิจของทีมท่าน
3. ตั้ง Shell variables เพื่อสะดวกในการพิมพ์:

```bash
# ตั้งค่า environment variables สำหรับ Demo
export API_BASE="https://<api-id>.execute-api.us-east-1.amazonaws.com/v1"
export API_KEY="<your-api-key>"
export TEAM_ID="TEAM-ALPHA"
export REQ_ID="REQ-001"    # request_id ที่ได้จาก Manage Dispatch Service
```

หลังจากนั้นทุก curl ด้านล่างสามารถใช้ตัวแปรเหล่านี้ได้

---

## Demo Flow — ขั้นตอนที่ 1: สร้าง Mission โดยรับ Event จาก Manage Dispatch Service

### บริบท

เมื่อ Manage Dispatch Service มอบหมายภารกิจให้ทีมกู้ภัย → MissionProgress Service จะ**รับ event อัตโนมัติ** ผ่าน Amazon SNS Topic `rescue.mission.dispatch.v1` โดยไม่ต้องเรียก API ใดๆ

### ทดสอบด้วย seed-data.sh (ถ้ายังไม่ได้รับ event จาก Dispatch จริง)

ถ้า Manage Dispatch Service ยังไม่ได้ส่ง event มา → ใช้ script นี้เพื่อ simulate event:

```bash
# รันจาก root ของ project
chmod +x script/seed-data.sh
./script/seed-data.sh
```

Script นี้จะส่ง EventBridge event โดยตรง:

```json
{
  "source": "dispatch-management-service",
  "detail-type": "MissionAssignedEvent",
  "detail": {
    "mission_id": "MSN-001",
    "rescue_unit_id": "TEAM-ALPHA",
    "incident_id": "REQ-001",
    "assigned_at": "2025-01-15T08:00:00Z"
  }
}
```

### ผลลัพธ์ที่คาดหวัง

Lambda `mission-assigned-handler` จะทำงาน:

1. รับ payload จาก EventBridge
2. เรียก **RescueRequest Service** เพื่อแปลง `request_id` → `incident_id` (degraded ถ้าล้มเหลว)
3. สร้าง `MissionAssignment` ใน DynamoDB (status = `DISPATCHED`)
4. สร้าง Timeline entry แรก (`action_type = MISSION_ASSIGNED`)

> **Idempotency:** ถ้าส่ง event ซ้ำ → DynamoDB condition `attribute_not_exists(mission_id)` จะ skip โดยไม่ error

**ยืนยันว่า Mission สร้างสำเร็จ** โดยไปขั้นตอนที่ 2

---

## Demo Flow — ขั้นตอนที่ 2: ดึงข้อมูลภารกิจ (Full Integration)

### Endpoint

```
GET /missions/{request_id}
```

### curl

```bash
curl -X GET \
  "$API_BASE/missions/$REQ_ID" \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: $TEAM_ID"
```

### ผลลัพธ์ที่คาดหวัง — Full Mode (HTTP 200)

ใน Demo 3 response จะมีฟิลด์เพิ่มขึ้นจาก Demo 2 อย่างมาก เพราะระบบเรียก **3 Services แบบ Parallel**:

```json
{
  "request_id": "REQ-001",
  "incident_id": "REQ-001",
  "mission_id": "MSN-001",
  "dispatch_id": "DSP-001",
  "rescue_team_id": "TEAM-ALPHA",
  "team_name": "Alpha Rescue Unit",
  "team_type": "FLOOD",
  "capabilities": ["boat_rescue", "swift_water"],
  "equipment": ["inflatable_boat", "life_vest"],
  "team_location": {
    "lat": 13.7563,
    "lng": 100.5018
  },
  "priority_level": 3,
  "dispatch_status": "ACTIVE",
  "current_status": "DISPATCHED",
  "latest_impact_level": 0,
  "started_at": "2025-01-15T08:00:00Z",
  "last_updated_at": "2025-01-15T08:00:00Z",
  "description": "น้ำท่วมหนักบริเวณถนนพหลโยธิน",
  "location": "13.7563,100.5018",
  "incident_type": "FLOOD",
  "timeline": [
    {
      "mission_id": "MSN-001",
      "timestamp": "2025-01-15T08:00:00Z",
      "log_id": "LOG-uuid-001",
      "action_type": "MISSION_ASSIGNED",
      "description": "Mission assigned to TEAM-ALPHA",
      "performed_by": "SYSTEM"
    }
  ],
  "data_source": "full"
}
```

### คำอธิบายฟิลด์ใหม่ใน Demo 3

| ฟิลด์             | มาจาก                  | คำอธิบาย                                                            |
| ----------------- | ---------------------- | ------------------------------------------------------------------- |
| `dispatch_id`     | DynamoDB (local)       | รหัส Dispatch Order ที่เชื่อมโยง                                    |
| `team_name`       | RescueTeam Service     | ชื่อทีมกู้ภัย                                                       |
| `team_type`       | RescueTeam Service     | ประเภทความเชี่ยวชาญ เช่น `FLOOD`, `FIRE`                            |
| `capabilities`    | RescueTeam Service     | รายการความสามารถ เช่น `["boat_rescue", "swift_water"]`              |
| `equipment`       | RescueTeam Service     | รายการอุปกรณ์ เช่น `["inflatable_boat", "life_vest"]`               |
| `team_location`   | RescueTeam Service     | พิกัด GPS ปัจจุบันของทีม `{lat, lng}`                               |
| `priority_level`  | ManageDispatch Service | ระดับความสำคัญของ Dispatch (1–4)                                    |
| `dispatch_status` | ManageDispatch Service | สถานะ Dispatch Order เช่น `ACTIVE`, `COMPLETED`                     |
| `incident_type`   | RescueRequest Service  | ประเภทเหตุการณ์ (ชื่อใหม่ Demo 3 — เดิมคือ `requestType` ใน Demo 2) |

### Degraded Mode — เมื่อ Service ใด Service หนึ่งล่ม

ถ้า **ManageDispatch Service** ล่ม → `priority_level` และ `dispatch_status` จะไม่อยู่ใน response  
ถ้า **RescueTeam Service** ล่ม → `team_name`, `team_type`, `capabilities`, `equipment`, `team_location` จะไม่อยู่ใน response  
ถ้า **RescueRequest Service** ล่ม → `description`, `location`, `incident_type` จะไม่อยู่ใน response  
ทุกกรณี: `data_source` จะเป็น `"partial"` (แทนที่ `"full"`)

```json
{
  "request_id": "REQ-001",
  "incident_id": "REQ-001",
  "mission_id": "MSN-001",
  "rescue_team_id": "TEAM-ALPHA",
  "current_status": "DISPATCHED",
  "latest_impact_level": 0,
  "started_at": "2025-01-15T08:00:00Z",
  "last_updated_at": "2025-01-15T08:00:00Z",
  "timeline": [...],
  "data_source": "partial"
}
```

> **สำคัญ:** Degraded Mode ไม่ทำให้ Service หยุดทำงาน — ทีมกู้ภัยยังรายงานความคืบหน้าได้ตามปกติ

### Frontend (ทางเลือก)

เปิดหน้าเว็บที่ `<frontend_url>` → ค้นหาด้วย Request ID `REQ-001` → จะเห็น Panel แสดงข้อมูลทีมและ Dispatch status ด้านข้าง

---

## Demo Flow — ขั้นตอนที่ 3: รายงาน EN_ROUTE (ออกเดินทาง)

### Endpoint

```
POST /missions/{request_id}/progress
```

### curl

```bash
curl -X POST \
  "$API_BASE/missions/$REQ_ID/progress" \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: $TEAM_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "new_status": "EN_ROUTE",
    "note": "กำลังเดินทางไปจุดเกิดเหตุ",
    "current_location": "13.7563,100.5018"
  }'
```

### ผลลัพธ์ที่คาดหวัง (HTTP 200)

```json
{
  "message": "Progress reported successfully",
  "mission_id": "MSN-001",
  "request_id": "REQ-001",
  "incident_id": "REQ-001",
  "old_status": "DISPATCHED",
  "new_status": "EN_ROUTE",
  "updated_at": "2025-01-15T08:30:00Z"
}
```

**Events ที่ถูก publish ไป EventBridge:**

| Event                  | Consumer                             | Condition                            |
| ---------------------- | ------------------------------------ | ------------------------------------ |
| `MissionStatusChanged` | IncidentTracking SQS, Dispatch SQS\* | ทุกครั้ง (\*Dispatch เฉพาะ RESOLVED) |

### ตรวจสอบ event ที่ส่งออกไป (ไม่บังคับ)

```bash
# ดู CloudWatch Logs ของ report-progress Lambda
aws logs filter-log-events \
  --log-group-name "/aws/lambda/mission-progress-report-progress" \
  --filter-pattern "MissionStatusChanged" \
  --region us-east-1
```

---

## Demo Flow — ขั้นตอนที่ 4: รายงาน ON_SITE (ถึงจุดเกิดเหตุ)

### curl

```bash
curl -X POST \
  "$API_BASE/missions/$REQ_ID/progress" \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: $TEAM_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "new_status": "ON_SITE",
    "note": "ถึงจุดเกิดเหตุแล้ว กำลังประเมินสถานการณ์",
    "current_location": "13.7380,100.5230",
    "new_impact_level": 3
  }'
```

### ผลลัพธ์ที่คาดหวัง (HTTP 200)

```json
{
  "message": "Progress reported successfully",
  "mission_id": "MSN-001",
  "request_id": "REQ-001",
  "incident_id": "REQ-001",
  "old_status": "EN_ROUTE",
  "new_status": "ON_SITE",
  "updated_at": "2025-01-15T09:00:00Z"
}
```

**Events ที่ถูก publish:**

| Event                  | Consumer                                 |
| ---------------------- | ---------------------------------------- |
| `MissionStatusChanged` | IncidentTracking SQS                     |
| `ImpactLevelUpdated`   | IncidentTracking SQS, Prioritization SQS |

---

## Demo Flow — ขั้นตอนที่ 5: อัปโหลดภาพหลักฐาน

ขั้นตอนนี้มี 2 ส่วน: (1) ขอ Presigned URL, (2) อัปโหลดไฟล์

### 5a. ขอ Presigned URL

```bash
curl -X POST \
  "$API_BASE/missions/$REQ_ID/presigned-url" \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: $TEAM_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "file_name": "flood-evidence.jpg",
    "content_type": "image/jpeg"
  }'
```

### ผลลัพธ์ที่คาดหวัง (HTTP 200)

```json
{
  "upload_url": "https://s3.amazonaws.com/mission-progress-evidence-<account-id>/evidence/MSN-001/TEAM-ALPHA/1736935200-flood-evidence.jpg?X-Amz-...",
  "image_key": "evidence/MSN-001/TEAM-ALPHA/1736935200-flood-evidence.jpg",
  "expires_in": 300,
  "message": "Upload URL generated successfully. Use PUT method to upload."
}
```

> **สังเกต:** ใน Demo 3 `image_key` ใช้ `mission_id` (`MSN-001`) แทน `incident_id` — นี่คือการแก้ไขจาก Demo 2

### 5b. อัปโหลดภาพด้วย Presigned URL

```bash
# บันทึก image_key ก่อน
export IMAGE_KEY="evidence/MSN-001/TEAM-ALPHA/1736935200-flood-evidence.jpg"

# อัปโหลดไฟล์จริง (ต้องมีไฟล์ flood-evidence.jpg ในเครื่อง)
curl -X PUT \
  -T flood-evidence.jpg \
  -H "Content-Type: image/jpeg" \
  "https://s3.amazonaws.com/mission-progress-evidence-<account-id>/evidence/MSN-001/TEAM-ALPHA/1736935200-flood-evidence.jpg?X-Amz-..."
```

**หมายเหตุ:** URL ด้านบนคือตัวอย่าง — ต้องใช้ค่าจริงที่ได้จากขั้นตอน 5a

**ผลลัพธ์ที่คาดหวัง:** HTTP 200 (empty body) — S3 upload สำเร็จ

### Frontend (ทางเลือก)

ในหน้า Mission Detail มีปุ่ม "อัปโหลดภาพหลักฐาน" → เลือกไฟล์ → กด Upload → Frontend จะเรียก presigned-url API และ PUT อัตโนมัติ

---

## Demo Flow — ขั้นตอนที่ 6: รายงาน NEED_BACKUP (ขอกำลังเสริม) + แนบภาพหลักฐาน

> ใช้ `image_key` ที่ได้จากขั้นตอน 5 เพื่อแนบหลักฐาน

```bash
curl -X POST \
  "$API_BASE/missions/$REQ_ID/progress" \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: $TEAM_ID" \
  -H "Content-Type: application/json" \
  -d "{
    \"new_status\": \"NEED_BACKUP\",
    \"note\": \"น้ำท่วมสูงกว่าที่คาด ต้องการกำลังเสริม\",
    \"current_location\": \"13.7380,100.5230\",
    \"new_impact_level\": 4,
    \"image_key\": \"$IMAGE_KEY\"
  }"
```

### ผลลัพธ์ที่คาดหวัง (HTTP 200)

```json
{
  "message": "Progress reported successfully",
  "mission_id": "MSN-001",
  "request_id": "REQ-001",
  "incident_id": "REQ-001",
  "old_status": "ON_SITE",
  "new_status": "NEED_BACKUP",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

**Events ที่ถูก publish (3 events พร้อมกัน):**

| Event                    | Consumer                                 | เหตุที่ publish                       |
| ------------------------ | ---------------------------------------- | ------------------------------------- |
| `MissionStatusChanged`   | IncidentTracking SQS                     | สถานะเปลี่ยนทุกครั้ง                  |
| `MissionBackupRequested` | Prioritization SQS                       | เฉพาะเมื่อ `new_status = NEED_BACKUP` |
| `ImpactLevelUpdated`     | IncidentTracking SQS, Prioritization SQS | เฉพาะเมื่อมี `new_impact_level`       |

### ยืนยัน Timeline (GET ดูสถานะ)

```bash
curl -X GET \
  "$API_BASE/missions/$REQ_ID" \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: $TEAM_ID"
```

คาดหวัง: `current_status = "NEED_BACKUP"`, timeline มี 4 entries (ASSIGNED → EN_ROUTE → ON_SITE → NEED_BACKUP), timeline entry ล่าสุดมี `image_key`

---

## Demo Flow — ขั้นตอนที่ 7: กลับ ON_SITE หลังได้รับกำลังเสริม

```bash
curl -X POST \
  "$API_BASE/missions/$REQ_ID/progress" \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: $TEAM_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "new_status": "ON_SITE",
    "note": "ได้รับกำลังเสริมแล้ว กำลังดำเนินการต่อ",
    "current_location": "13.7380,100.5230"
  }'
```

### ผลลัพธ์ที่คาดหวัง (HTTP 200)

```json
{
  "message": "Progress reported successfully",
  "old_status": "NEED_BACKUP",
  "new_status": "ON_SITE",
  ...
}
```

---

## Demo Flow — ขั้นตอนที่ 8: รายงาน RESOLVED (ภารกิจสำเร็จ)

> สถานะนี้จะ trigger event ไปยัง **Dispatch SQS** เพื่อคืนสถานะทีม (Noppakron)

```bash
curl -X POST \
  "$API_BASE/missions/$REQ_ID/progress" \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: $TEAM_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "new_status": "RESOLVED",
    "note": "ควบคุมสถานการณ์ได้แล้ว ภารกิจสำเร็จ",
    "current_location": "13.7380,100.5230"
  }'
```

### ผลลัพธ์ที่คาดหวัง (HTTP 200)

```json
{
  "message": "Progress reported successfully",
  "mission_id": "MSN-001",
  "request_id": "REQ-001",
  "incident_id": "REQ-001",
  "old_status": "ON_SITE",
  "new_status": "RESOLVED",
  "updated_at": "2025-01-15T15:00:00Z"
}
```

**Events ที่ถูก publish:**

| Event                  | Consumer                                          | หมายเหตุ                                                |
| ---------------------- | ------------------------------------------------- | ------------------------------------------------------- |
| `MissionStatusChanged` | IncidentTracking SQS, **Dispatch SQS** (RESOLVED) | Dispatch SQS จะได้รับเฉพาะเมื่อ `new_status = RESOLVED` |

> **ยืนยันกับ Noppakron:** หลัง event นี้ถึง Dispatch SQS → ทีมควรกลับมาเป็น `AVAILABLE` ในระบบ Dispatch

### พยายาม RESOLVED อีกครั้ง (ต้องได้ Error)

```bash
curl -X POST \
  "$API_BASE/missions/$REQ_ID/progress" \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: $TEAM_ID" \
  -H "Content-Type: application/json" \
  -d '{"new_status": "EN_ROUTE"}'
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

## Demo Flow — ขั้นตอนที่ 9: ดูรายการภารกิจทั้งหมดของทีม

```bash
curl -X GET \
  "$API_BASE/missions" \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: $TEAM_ID"
```

### ผลลัพธ์ที่คาดหวัง (HTTP 200)

```json
{
  "team_id": "TEAM-ALPHA",
  "total_missions": 1,
  "missions": [
    {
      "mission_id": "MSN-001",
      "incident_id": "REQ-001",
      "rescue_team_id": "TEAM-ALPHA",
      "current_status": "RESOLVED",
      "latest_impact_level": 4,
      "started_at": "2025-01-15T08:00:00Z",
      "last_updated_at": "2025-01-15T15:00:00Z"
    }
  ]
}
```

### กรองเฉพาะสถานะ

```bash
# ดูเฉพาะภารกิจที่ยัง active อยู่
curl -X GET \
  "$API_BASE/missions?status=ON_SITE" \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: $TEAM_ID"
```

---

## 3. Synchronous API: Get Mission Details (ฉบับ Demo 3)

### Endpoint

```
GET /missions/{request_id}
```

### Request

| พารามิเตอร์  | ประเภท | จำเป็น | คำอธิบาย                                              |
| ------------ | ------ | ------ | ----------------------------------------------------- |
| `request_id` | String | ✅     | รหัส request จาก RescueRequest Service เช่น `REQ-001` |

### Response — Success (200 OK) — Full Mode

```json
{
  "request_id": "REQ-001",
  "incident_id": "REQ-001",
  "mission_id": "MSN-001",
  "dispatch_id": "DSP-001",
  "rescue_team_id": "TEAM-ALPHA",
  "team_name": "Alpha Rescue Unit",
  "team_type": "FLOOD",
  "capabilities": ["boat_rescue", "swift_water"],
  "equipment": ["inflatable_boat", "life_vest"],
  "team_location": {
    "lat": 13.7563,
    "lng": 100.5018
  },
  "priority_level": 3,
  "dispatch_status": "ACTIVE",
  "current_status": "ON_SITE",
  "latest_impact_level": 3,
  "started_at": "2025-01-15T08:00:00Z",
  "last_updated_at": "2025-01-15T10:00:00Z",
  "description": "น้ำท่วมหนักบริเวณถนนพหลโยธิน",
  "location": "13.7563,100.5018",
  "incident_type": "FLOOD",
  "timeline": [
    {
      "mission_id": "MSN-001",
      "timestamp": "2025-01-15T08:00:00Z",
      "log_id": "LOG-uuid-001",
      "action_type": "MISSION_ASSIGNED",
      "description": "Mission assigned to TEAM-ALPHA",
      "performed_by": "SYSTEM"
    },
    {
      "mission_id": "MSN-001",
      "timestamp": "2025-01-15T08:30:00Z",
      "log_id": "LOG-uuid-002",
      "action_type": "STATUS_CHANGE",
      "description": "Team en route to incident",
      "performed_by": "TEAM-ALPHA",
      "old_status": "DISPATCHED",
      "new_status": "EN_ROUTE",
      "location": "13.7563,100.5018",
      "note": "กำลังเดินทางไปจุดเกิดเหตุ"
    },
    {
      "mission_id": "MSN-001",
      "timestamp": "2025-01-15T09:00:00Z",
      "log_id": "LOG-uuid-003",
      "action_type": "STATUS_CHANGE",
      "description": "Arrived on site",
      "performed_by": "TEAM-ALPHA",
      "old_status": "EN_ROUTE",
      "new_status": "ON_SITE",
      "location": "13.7380,100.5230",
      "image_key": "evidence/MSN-001/TEAM-ALPHA/1736942400-flood-evidence.jpg"
    }
  ],
  "data_source": "full"
}
```

### คำอธิบายฟิลด์ทั้งหมดใน Response

| ฟิลด์                 | ประเภท           | มาจาก                  | คำอธิบาย                                                        |
| --------------------- | ---------------- | ---------------------- | --------------------------------------------------------------- |
| `request_id`          | String           | DynamoDB               | รหัส Request (จาก RescueRequest Service)                        |
| `incident_id`         | String           | DynamoDB               | รหัส Incident (เดิมคือ `incident_id`)                           |
| `mission_id`          | String           | DynamoDB               | รหัสภารกิจ (Primary Key)                                        |
| `dispatch_id`         | String           | DynamoDB               | รหัส Dispatch Order                                             |
| `rescue_team_id`      | String           | DynamoDB               | รหัสทีมกู้ภัยที่รับผิดชอบ                                       |
| `team_name`           | String           | RescueTeam Service     | ชื่อทีม (ไม่มีใน Degraded Mode)                                 |
| `team_type`           | String           | RescueTeam Service     | ประเภทความเชี่ยวชาญ (ไม่มีใน Degraded Mode)                     |
| `capabilities`        | Array of String  | RescueTeam Service     | รายการความสามารถ (ไม่มีใน Degraded Mode)                        |
| `equipment`           | Array of String  | RescueTeam Service     | รายการอุปกรณ์ (ไม่มีใน Degraded Mode)                           |
| `team_location`       | Object {lat,lng} | RescueTeam Service     | พิกัด GPS ของทีม (ไม่มีใน Degraded Mode)                        |
| `priority_level`      | Integer          | ManageDispatch Service | ระดับความสำคัญ 1–4 (ไม่มีใน Degraded Mode)                      |
| `dispatch_status`     | String           | ManageDispatch Service | สถานะ Dispatch Order (ไม่มีใน Degraded Mode)                    |
| `current_status`      | String           | DynamoDB               | สถานะปัจจุบัน: DISPATCHED/EN_ROUTE/ON_SITE/NEED_BACKUP/RESOLVED |
| `latest_impact_level` | Integer          | DynamoDB               | ระดับความรุนแรงล่าสุด (1–4, 0=ยังไม่ประเมิน)                    |
| `started_at`          | DateTime         | DynamoDB               | เวลาที่รับภารกิจ (ISO 8601)                                     |
| `last_updated_at`     | DateTime         | DynamoDB               | เวลาอัปเดตล่าสุด (ISO 8601)                                     |
| `description`         | String           | RescueRequest Service  | คำอธิบาย Request (ไม่มีใน Degraded Mode)                        |
| `location`            | String           | RescueRequest Service  | พิกัด Request เช่น `"13.7563,100.5018"` (ไม่มีใน Degraded Mode) |
| `incident_type`       | String           | RescueRequest Service  | ประเภทเหตุการณ์ เช่น `FLOOD`, `FIRE` (ไม่มีใน Degraded Mode)    |
| `timeline`            | Array            | DynamoDB               | ประวัติการปฏิบัติงาน (เรียงตามเวลา)                             |
| `data_source`         | String           | computed               | `"full"` = ครบทุก service, `"partial"` = บาง service ล่ม        |

### Response — Degraded Mode (200 OK)

```json
{
  "request_id": "REQ-001",
  "mission_id": "MSN-001",
  "rescue_team_id": "TEAM-ALPHA",
  "current_status": "ON_SITE",
  "latest_impact_level": 3,
  "started_at": "2025-01-15T08:00:00Z",
  "last_updated_at": "2025-01-15T10:00:00Z",
  "timeline": [...],
  "data_source": "partial"
}
```

### Response — Error (404)

```json
{
  "error": "REQUEST_NOT_FOUND",
  "code": "REQUEST_NOT_FOUND",
  "message": "No mission found for request: REQ-99999",
  "traceId": "a1b2c3d4-e5f6-..."
}
```

### ตัวอย่าง Error Cases

#### ❌ ล้มเหลว — request_id ไม่มีในระบบ

```bash
curl -X GET \
  "$API_BASE/missions/REQ-99999" \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: $TEAM_ID"
```

ผลลัพธ์ที่คาดหวัง (HTTP 404):

```json
{
  "error": "REQUEST_NOT_FOUND",
  "code": "REQUEST_NOT_FOUND",
  "message": "No mission found for request: REQ-99999"
}
```

#### ❌ ล้มเหลว — ไม่ส่ง x-api-key

```bash
curl -X GET "$API_BASE/missions/REQ-001" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA"
```

ผลลัพธ์ที่คาดหวัง (HTTP 403):

```json
{ "message": "Forbidden" }
```

#### ❌ ล้มเหลว — ไม่ส่ง X-Rescue-Team-ID

```bash
curl -X GET "$API_BASE/missions/REQ-001" \
  -H "x-api-key: $API_KEY"
```

ผลลัพธ์ที่คาดหวัง (HTTP 403):

```json
{ "message": "User is not authorized to access this resource" }
```

---

## 4. Sync + Async API: Report Progress

### Endpoint

```
POST /missions/{request_id}/progress
```

### การทำงานภายใน

1. **(Sync)** ตรวจสอบ State Machine → อัปเดต DynamoDB → สร้าง Timeline entry → ตอบ 200
2. **(Async)** Publish events → EventBridge → SQS ของแต่ละ Service
3. **(Fallback)** ถ้า EventBridge ล่ม → บันทึกลง EventOutbox table → Outbox Processor retry ทุก 5 นาที

### Body Parameters

```json
{
  "new_status": "EN_ROUTE",
  "note": "กำลังเดินทางไปจุดเกิดเหตุ",
  "current_location": "13.7563,100.5018",
  "new_impact_level": 3,
  "image_key": "evidence/MSN-001/TEAM-ALPHA/1736935200-flood-evidence.jpg"
}
```

| ฟิลด์              | ประเภท  | จำเป็น | คำอธิบาย                                                    |
| ------------------ | ------- | ------ | ----------------------------------------------------------- |
| `new_status`       | String  | ✅     | สถานะใหม่ (ตาม State Machine)                               |
| `note`             | String  | ❌     | หมายเหตุการปฏิบัติงาน                                       |
| `current_location` | String  | ❌     | พิกัด GPS ปัจจุบัน `"lat,lng"`                              |
| `new_impact_level` | Integer | ❌     | ระดับความรุนแรงใหม่ 1–4 (trigger ImpactLevelUpdated event)  |
| `image_key`        | String  | ❌     | S3 key จาก presigned-url endpoint (บันทึกลง Timeline entry) |

### State Machine

| สถานะปัจจุบัน | สถานะที่เปลี่ยนไปได้                |
| ------------- | ----------------------------------- |
| `DISPATCHED`  | `EN_ROUTE`                          |
| `EN_ROUTE`    | `ON_SITE`                           |
| `ON_SITE`     | `NEED_BACKUP`, `RESOLVED`           |
| `NEED_BACKUP` | `ON_SITE`, `RESOLVED`               |
| `RESOLVED`    | _(สถานะสุดท้าย — เปลี่ยนต่อไม่ได้)_ |

```
DISPATCHED → EN_ROUTE → ON_SITE ──→ RESOLVED
                            ↓            ↑
                       NEED_BACKUP ──────┘
                            ↑
                   (กลับ ON_SITE ได้)
```

### Response — Success (200)

```json
{
  "message": "Progress reported successfully",
  "mission_id": "MSN-001",
  "request_id": "REQ-001",
  "incident_id": "REQ-001",
  "old_status": "DISPATCHED",
  "new_status": "EN_ROUTE",
  "updated_at": "2025-01-15T08:30:00Z"
}
```

### Error Responses

| HTTP | Error Code                 | เงื่อนไข                                   |
| ---- | -------------------------- | ------------------------------------------ |
| 400  | `MISSING_PARAMETER`        | ไม่ส่ง `new_status`                        |
| 400  | `INVALID_BODY`             | JSON body ไม่ถูกต้อง                       |
| 400  | `INVALID_STATUS`           | ค่า `new_status` ไม่มีในระบบ               |
| 400  | `INVALID_STATE_TRANSITION` | เปลี่ยนสถานะไม่ตรงตาม State Machine        |
| 403  | —                          | ไม่ส่ง `x-api-key` หรือ `X-Rescue-Team-ID` |
| 404  | `REQUEST_NOT_FOUND`        | ไม่พบภารกิจสำหรับ `request_id` ที่ระบุ     |
| 500  | `INTERNAL_ERROR`           | เกิดข้อผิดพลาดภายใน                        |

### ตัวอย่าง Error Cases — curl

#### ❌ State Transition ผิดกฎ (DISPATCHED → RESOLVED)

```bash
curl -X POST \
  "$API_BASE/missions/$REQ_ID/progress" \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: $TEAM_ID" \
  -H "Content-Type: application/json" \
  -d '{"new_status": "RESOLVED"}'
```

ผลลัพธ์ (HTTP 400):

```json
{
  "error": "INVALID_STATE_TRANSITION",
  "code": "INVALID_STATE_TRANSITION",
  "message": "Cannot transition from DISPATCHED to RESOLVED"
}
```

#### ❌ ไม่ส่ง new_status

```bash
curl -X POST \
  "$API_BASE/missions/$REQ_ID/progress" \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: $TEAM_ID" \
  -H "Content-Type: application/json" \
  -d '{"note": "ทดสอบ"}'
```

ผลลัพธ์ (HTTP 400):

```json
{
  "error": "MISSING_PARAMETER",
  "code": "MISSING_PARAMETER",
  "message": "new_status is required"
}
```

#### ❌ ส่ง status ที่ไม่มีในระบบ

```bash
curl -X POST \
  "$API_BASE/missions/$REQ_ID/progress" \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: $TEAM_ID" \
  -H "Content-Type: application/json" \
  -d '{"new_status": "CANCELLED"}'
```

ผลลัพธ์ (HTTP 400):

```json
{
  "error": "INVALID_STATUS",
  "code": "INVALID_STATUS",
  "message": "Invalid status value: CANCELLED"
}
```

---

## 5. Sync API: Generate Presigned URL

### Endpoint

```
POST /missions/{request_id}/presigned-url
```

### Body

```json
{
  "file_name": "flood-evidence.jpg",
  "content_type": "image/jpeg"
}
```

| ฟิลด์          | ประเภท | จำเป็น | คำอธิบาย                                           |
| -------------- | ------ | ------ | -------------------------------------------------- |
| `file_name`    | String | ✅     | ชื่อไฟล์ภาพ                                        |
| `content_type` | String | ✅     | MIME type: `image/jpeg`, `image/png`, `image/webp` |

### Response — Success (200)

```json
{
  "upload_url": "https://s3.amazonaws.com/...",
  "image_key": "evidence/MSN-001/TEAM-ALPHA/1736935200-flood-evidence.jpg",
  "expires_in": 300,
  "message": "Upload URL generated successfully. Use PUT method to upload."
}
```

> **Demo 3:** `image_key` ใช้ `mission_id` (`MSN-001`) เป็น prefix — ถูกต้องแล้ว (Demo 2 เคยใช้ `incident_id` ที่ผิด)

### ขั้นตอนอัปโหลดไฟล์จริง

```bash
curl -X PUT \
  -T <path-to-local-file.jpg> \
  -H "Content-Type: image/jpeg" \
  "<upload_url จาก response>"
```

### Error Responses

| HTTP | Error Code             | เงื่อนไข                               |
| ---- | ---------------------- | -------------------------------------- |
| 400  | `MISSING_PARAMETER`    | ไม่ส่ง `file_name` หรือ `content_type` |
| 400  | `INVALID_CONTENT_TYPE` | content_type ไม่อยู่ใน whitelist       |
| 404  | `REQUEST_NOT_FOUND`    | ไม่พบภารกิจ                            |
| 500  | `PRESIGN_FAILED`       | S3 ไม่สามารถ generate URL ได้          |

### ตัวอย่าง

#### ✅ ขอ URL สำหรับ PNG

```bash
curl -X POST \
  "$API_BASE/missions/$REQ_ID/presigned-url" \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: $TEAM_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "file_name": "site-photo.png",
    "content_type": "image/png"
  }'
```

#### ❌ content_type ไม่รองรับ (PDF)

```bash
curl -X POST \
  "$API_BASE/missions/$REQ_ID/presigned-url" \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: $TEAM_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "file_name": "report.pdf",
    "content_type": "application/pdf"
  }'
```

ผลลัพธ์ (HTTP 400):

```json
{
  "error": "INVALID_CONTENT_TYPE",
  "code": "INVALID_CONTENT_TYPE",
  "message": "content_type must be one of: image/jpeg, image/png, image/webp"
}
```

---

## 6. Sync API: List Missions by Team

### Endpoint

```
GET /missions
```

### Query Parameters

| พารามิเตอร์ | ประเภท | จำเป็น | คำอธิบาย                                                                     |
| ----------- | ------ | ------ | ---------------------------------------------------------------------------- |
| `status`    | String | ❌     | กรองตามสถานะ: `DISPATCHED`, `EN_ROUTE`, `ON_SITE`, `NEED_BACKUP`, `RESOLVED` |

### Response — Success (200)

```json
{
  "team_id": "TEAM-ALPHA",
  "total_missions": 2,
  "missions": [
    {
      "mission_id": "MSN-001",
      "incident_id": "REQ-001",
      "rescue_team_id": "TEAM-ALPHA",
      "current_status": "RESOLVED",
      "latest_impact_level": 4,
      "started_at": "2025-01-15T08:00:00Z",
      "last_updated_at": "2025-01-15T15:00:00Z"
    },
    {
      "mission_id": "MSN-007",
      "incident_id": "REQ-007",
      "rescue_team_id": "TEAM-ALPHA",
      "current_status": "DISPATCHED",
      "latest_impact_level": 0,
      "started_at": "2025-01-16T07:00:00Z",
      "last_updated_at": "2025-01-16T07:00:00Z"
    }
  ]
}
```

### ตัวอย่างการใช้งาน

#### ✅ ดึงภารกิจทั้งหมดของทีม

```bash
curl -X GET \
  "$API_BASE/missions" \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: $TEAM_ID"
```

#### ✅ กรองเฉพาะ NEED_BACKUP

```bash
curl -X GET \
  "$API_BASE/missions?status=NEED_BACKUP" \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: $TEAM_ID"
```

#### ✅ ทีมที่ไม่มีภารกิจ (return array ว่าง ไม่ใช่ 404)

```bash
curl -X GET \
  "$API_BASE/missions" \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-NEWBIE"
```

ผลลัพธ์ (HTTP 200):

```json
{
  "team_id": "TEAM-NEWBIE",
  "total_missions": 0,
  "missions": []
}
```

#### ❌ status filter ไม่ถูกต้อง

```bash
curl -X GET \
  "$API_BASE/missions?status=UNKNOWN" \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: $TEAM_ID"
```

ผลลัพธ์ (HTTP 400):

```json
{
  "error": "INVALID_STATUS",
  "code": "INVALID_STATUS",
  "message": "Invalid status filter: UNKNOWN"
}
```

---

## 7. Asynchronous Events Contract (EventBridge → SQS)

### Event Bus

| รายการ    | ค่า                       |
| --------- | ------------------------- |
| Event Bus | `mission-progress-events` |
| Source    | `MissionProgressService`  |
| Region    | `us-east-1`               |

### 7.1 MissionStatusChanged

**เงื่อนไข:** publish ทุกครั้งที่ POST progress สำเร็จ  
**Consumers:** IncidentTracking SQS, Dispatch SQS (เฉพาะ `new_status = RESOLVED`)

```json
{
  "source": "MissionProgressService",
  "detail-type": "MissionStatusChanged",
  "detail": {
    "schema_version": "1.0",
    "mission_id": "MSN-001",
    "incident_id": "REQ-001",
    "rescue_team_id": "TEAM-ALPHA",
    "old_status": "DISPATCHED",
    "new_status": "EN_ROUTE",
    "changed_at": "2025-01-15T08:30:00Z",
    "changed_by": "TEAM-ALPHA"
  }
}
```

### 7.2 MissionBackupRequested

**เงื่อนไข:** เฉพาะเมื่อ `new_status = NEED_BACKUP`  
**Consumer:** Prioritization SQS (Nattasak)

```json
{
  "source": "MissionProgressService",
  "detail-type": "MissionBackupRequested",
  "detail": {
    "schema_version": "1.0",
    "mission_id": "MSN-001",
    "incident_id": "REQ-001",
    "rescue_team_id": "TEAM-ALPHA",
    "requested_at": "2025-01-15T10:30:00Z",
    "requested_by": "TEAM-ALPHA",
    "location": "13.7380,100.5230"
  }
}
```

### 7.3 ImpactLevelUpdated

**เงื่อนไข:** เฉพาะเมื่อ request body มีฟิลด์ `new_impact_level`  
**Consumers:** IncidentTracking SQS (Krittamet), Prioritization SQS (Nattasak)

```json
{
  "source": "MissionProgressService",
  "detail-type": "ImpactLevelUpdated",
  "detail": {
    "schema_version": "1.0",
    "mission_id": "MSN-001",
    "incident_id": "REQ-001",
    "rescue_team_id": "TEAM-ALPHA",
    "old_level": 3,
    "new_level": 4,
    "updated_at": "2025-01-15T10:30:00Z",
    "updated_by": "TEAM-ALPHA"
  }
}
```

### Event Routing Summary

| Event                    | IncidentTracking SQS | Dispatch SQS       | Prioritization SQS | CloudWatch Logs |
| ------------------------ | -------------------- | ------------------ | ------------------ | --------------- |
| `MissionStatusChanged`   | ✅ (ทุก status)      | ✅ (RESOLVED only) | ❌                 | ✅              |
| `MissionBackupRequested` | ❌                   | ❌                 | ✅                 | ✅              |
| `ImpactLevelUpdated`     | ✅                   | ❌                 | ✅                 | ✅              |

### Outbox Pattern (Fallback)

ถ้า EventBridge publish ล้มเหลว → บันทึก event ลง **EventOutbox** table ใน DynamoDB → Outbox Processor Lambda ทำงาน **ทุก 5 นาที** เพื่อ retry event ที่ค้างอยู่

**ยืนยัน Outbox Pattern ใน Demo:**

```bash
# ดู DynamoDB Outbox table (ถ้ามี event ค้างอยู่จะแสดงที่นี่)
aws dynamodb scan \
  --table-name MissionProgressEventOutbox \
  --region us-east-1 \
  --output table
```

ถ้าระบบทำงานปกติ table จะว่างหรือมีแค่ record ชั่วคราว

---

## 8. Inbound Async Event (from Manage Dispatch)

### MissionAssignedEvent

| รายการ         | ค่า                           |
| -------------- | ----------------------------- |
| Source         | `dispatch-management-service` |
| Detail-type    | `MissionAssignedEvent`        |
| Handler Lambda | `mission-assigned-handler`    |

**Expected Payload:**

```json
{
  "source": "dispatch-management-service",
  "detail-type": "MissionAssignedEvent",
  "detail": {
    "mission_id": "MSN-001",
    "rescue_unit_id": "TEAM-ALPHA",
    "incident_id": "REQ-001",
    "assigned_at": "2025-01-15T08:00:00Z"
  }
}
```

**การทำงานภายใน (Demo 3):**

1. Parse payload จาก EventBridge
2. **เรียก RescueRequest Service** เพื่อแปลง `request_id` → `incident_id` (ใหม่ใน Demo 3)
3. สร้าง `MissionAssignment` ใน DynamoDB (`status = DISPATCHED`)
4. สร้าง Timeline entry แรก (`MISSION_ASSIGNED`)
5. Degraded: ถ้า RescueRequest ล่ม → ใช้ `incident_id` ว่างและ log warning — ไม่ fail

**Idempotency:** ถ้า event ซ้ำ → `attribute_not_exists(mission_id)` → skip ไม่ error

---

## สรุปภาพรวม Endpoints

| Method | Path                                   | คำอธิบาย                                | ประเภท       |
| ------ | -------------------------------------- | --------------------------------------- | ------------ |
| GET    | `/missions/{request_id}`               | ดึงข้อมูลภารกิจ + 3 services + Timeline | Synchronous  |
| POST   | `/missions/{request_id}/progress`      | อัปเดตสถานะ + publish events            | Sync + Async |
| POST   | `/missions/{request_id}/presigned-url` | ขอ URL อัปโหลดภาพหลักฐาน (S3)           | Synchronous  |
| GET    | `/missions`                            | ดึงรายการภารกิจทั้งหมดของทีม            | Synchronous  |

---

## สรุป Error Codes ทั้งหมด

| HTTP | Error Code                 | สาเหตุ                                     | Endpoint ที่เกี่ยว                             |
| ---- | -------------------------- | ------------------------------------------ | ---------------------------------------------- |
| 400  | `MISSING_PARAMETER`        | ไม่ส่ง parameter ที่จำเป็น                 | ทั้งหมด                                        |
| 400  | `INVALID_BODY`             | JSON body ไม่ถูกต้อง                       | POST progress, POST presigned-url              |
| 400  | `INVALID_STATUS`           | new_status หรือ filter ไม่ใช่สถานะที่กำหนด | POST progress, GET /missions                   |
| 400  | `INVALID_STATE_TRANSITION` | เปลี่ยนสถานะไม่ตรงตามกฎ State Machine      | POST progress                                  |
| 400  | `INVALID_CONTENT_TYPE`     | content_type ไม่อยู่ใน whitelist           | POST presigned-url                             |
| 403  | —                          | ไม่ส่ง `x-api-key` หรือ `X-Rescue-Team-ID` | ทั้งหมด                                        |
| 404  | `REQUEST_NOT_FOUND`        | ไม่พบภารกิจสำหรับ `request_id` ที่ระบุ     | GET mission, POST progress, POST presigned-url |
| 500  | `INTERNAL_ERROR`           | เกิดข้อผิดพลาดภายในระบบ                    | ทั้งหมด                                        |
| 500  | `PRESIGN_FAILED`           | S3 ไม่สามารถ generate presigned URL ได้    | POST presigned-url                             |

---

## สรุปผลการทดสอบ (Test Matrix)

| #   | กรณีทดสอบ                                              | Endpoint           | ประเภท     | HTTP | ผลที่คาดหวัง                                                      |
| --- | ------------------------------------------------------ | ------------------ | ---------- | ---- | ----------------------------------------------------------------- |
| 1   | ไม่ส่ง API Key                                         | ทุก endpoint       | ❌ ล้มเหลว | 403  | `Forbidden`                                                       |
| 2   | ไม่ส่ง X-Rescue-Team-ID                                | ทุก endpoint       | ❌ ล้มเหลว | 403  | `User is not authorized...`                                       |
| 3   | GET mission ที่มีอยู่ — Full Mode (3 services ตอบปกติ) | GET /missions/{id} | ✅ สำเร็จ  | 200  | ข้อมูลครบ + `data_source: "full"`                                 |
| 4   | GET mission — Degraded Mode (RescueRequest ล่ม)        | GET /missions/{id} | ✅ สำเร็จ  | 200  | ไม่มี description/location + `data_source: "partial"`             |
| 5   | GET mission — Degraded Mode (ManageDispatch ล่ม)       | GET /missions/{id} | ✅ สำเร็จ  | 200  | ไม่มี dispatch_status/priority_level + `data_source: "partial"`   |
| 6   | GET mission — Degraded Mode (RescueTeam ล่ม)           | GET /missions/{id} | ✅ สำเร็จ  | 200  | ไม่มี team_name/capabilities/equipment + `data_source: "partial"` |
| 7   | GET mission ที่ไม่มีในระบบ                             | GET /missions/{id} | ❌ ล้มเหลว | 404  | `REQUEST_NOT_FOUND`                                               |
| 8   | POST progress: DISPATCHED → EN_ROUTE                   | POST progress      | ✅ สำเร็จ  | 200  | `Progress reported successfully`                                  |
| 9   | POST progress: EN_ROUTE → ON_SITE + impact_level       | POST progress      | ✅ สำเร็จ  | 200  | ImpactLevelUpdated event ถูก publish                              |
| 10  | POST progress: ON_SITE → NEED_BACKUP + image_key       | POST progress      | ✅ สำเร็จ  | 200  | 3 events ถูก publish                                              |
| 11  | POST progress: NEED_BACKUP → ON_SITE                   | POST progress      | ✅ สำเร็จ  | 200  | MissionStatusChanged event                                        |
| 12  | POST progress: ON_SITE → RESOLVED                      | POST progress      | ✅ สำเร็จ  | 200  | MissionStatusChanged + Dispatch SQS ได้รับ                        |
| 13  | POST progress: RESOLVED → EN_ROUTE (terminal state)    | POST progress      | ❌ ล้มเหลว | 400  | `INVALID_STATE_TRANSITION`                                        |
| 14  | POST progress: DISPATCHED → RESOLVED (ข้ามขั้น)        | POST progress      | ❌ ล้มเหลว | 400  | `INVALID_STATE_TRANSITION`                                        |
| 15  | POST progress: ไม่ส่ง new_status                       | POST progress      | ❌ ล้มเหลว | 400  | `MISSING_PARAMETER`                                               |
| 16  | POST progress: status = CANCELLED (ไม่มีในระบบ)        | POST progress      | ❌ ล้มเหลว | 400  | `INVALID_STATUS`                                                  |
| 17  | POST progress: JSON body ไม่ถูกต้อง                    | POST progress      | ❌ ล้มเหลว | 400  | `INVALID_BODY`                                                    |
| 18  | POST progress: request_id ไม่มีในระบบ                  | POST progress      | ❌ ล้มเหลว | 404  | `REQUEST_NOT_FOUND`                                               |
| 19  | POST presigned-url: JPEG                               | POST presigned-url | ✅ สำเร็จ  | 200  | `upload_url` ใช้ `mission_id` เป็น prefix                         |
| 20  | POST presigned-url: PNG                                | POST presigned-url | ✅ สำเร็จ  | 200  | `upload_url` + `image_key`                                        |
| 21  | POST presigned-url: content_type = PDF                 | POST presigned-url | ❌ ล้มเหลว | 400  | `INVALID_CONTENT_TYPE`                                            |
| 22  | POST presigned-url: ไม่ส่ง file_name                   | POST presigned-url | ❌ ล้มเหลว | 400  | `MISSING_PARAMETER`                                               |
| 23  | PUT ภาพจริงด้วย presigned URL                          | S3 (direct)        | ✅ สำเร็จ  | 200  | ภาพถูกเก็บใน S3 bucket                                            |
| 24  | GET /missions: ดูภารกิจทั้งหมดของทีม                   | GET /missions      | ✅ สำเร็จ  | 200  | รายการภารกิจ                                                      |
| 25  | GET /missions?status=ON_SITE                           | GET /missions      | ✅ สำเร็จ  | 200  | เฉพาะ ON_SITE missions                                            |
| 26  | GET /missions: ทีมที่ไม่มีภารกิจ                       | GET /missions      | ✅ สำเร็จ  | 200  | `missions: []`, `total: 0`                                        |
| 27  | GET /missions?status=UNKNOWN                           | GET /missions      | ❌ ล้มเหลว | 400  | `INVALID_STATUS`                                                  |
| 28  | MissionAssignedEvent — สร้างภารกิจใหม่                 | Inbound Event      | ✅ สำเร็จ  | —    | Mission + Timeline entry สร้างใน DynamoDB                         |
| 29  | MissionAssignedEvent — event ซ้ำ (idempotent)          | Inbound Event      | ✅ สำเร็จ  | —    | skip ไม่ error                                                    |
| 30  | Outbox pattern: EventBridge ล่ม → retry สำเร็จ         | Outbox Processor   | ✅ สำเร็จ  | —    | event ถูกส่งออกไปหลัง retry                                       |

---

## CORS

| Header                         | ค่า                                       |
| ------------------------------ | ----------------------------------------- |
| `Access-Control-Allow-Origin`  | `*`                                       |
| `Access-Control-Allow-Methods` | `GET,POST,OPTIONS`                        |
| `Access-Control-Allow-Headers` | `Content-Type,x-api-key,X-Rescue-Team-ID` |

ทุก response มี header `X-Trace-Id` (UUID) สำหรับ debug/tracing

---

## Frontend

| รายการ    | ค่า                                    |
| --------- | -------------------------------------- |
| Framework | Next.js (App Router, Static Export)    |
| Hosting   | S3 Static Website Hosting              |
| URL       | ได้จาก Terraform output `frontend_url` |

**Flow บน Frontend (ไม่ใช้ curl):**

1. เปิด `<frontend_url>` → กรอก API Key + Team ID
2. หน้า Mission List แสดงภารกิจทั้งหมดของทีม (GET /missions)
3. กดเข้าดูรายละเอียด Mission — เห็น Team Info, Dispatch Status, Timeline ครบ (GET /missions/{id})
4. กดปุ่ม "อัปเดตสถานะ" → เลือก EN_ROUTE/ON_SITE/... → กด Submit (POST progress)
5. กดปุ่ม "อัปโหลดภาพหลักฐาน" → เลือกไฟล์ → Frontend เรียก presigned-url แล้ว PUT อัตโนมัติ
6. Timeline อัปเดตทันทีพร้อม thumbnail ภาพ
