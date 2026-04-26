# MissionProgress Service — API Contract (Demo 2: Full Integration)

**Service Owner:** นายรัฐธรรมนูญ โคสาแสง (6609612178)

**วัตถุประสงค์ของบริการ:**
MissionProgress Service คือบริการสำหรับทีมกู้ภัย (Rescue Team) เพื่อรายงานความคืบหน้าของภารกิจ อัปเดตสถานะเหตุการณ์ และบันทึกรายละเอียดการปฏิบัติงานหน้างาน เพื่อให้ศูนย์สั่งการได้รับข้อมูลที่ถูกต้องและเป็นปัจจุบัน

**ขอบเขต Demo 2 (Full Integration):**

- เป็นเวอร์ชันสมบูรณ์สำหรับ Demo 2 — มีทั้ง **Frontend** (Next.js บน S3) และ Backend (API Gateway + Lambda + DynamoDB + EventBridge)
- Deploy บน AWS Learner Lab ผ่าน Terraform
- มี 4 Synchronous Endpoints:
  1. **GET** `/missions/{request_id}` — ดึงข้อมูลภารกิจ + Timeline (มาตั้งแต่ Demo 1)
  2. **POST** `/missions/{request_id}/progress` — อัปเดตสถานะ + publish events (มาตั้งแต่ Demo 1, เพิ่ม `image_key`)
  3. **POST** `/missions/{request_id}/presigned-url` — ขอ URL อัปโหลดภาพหลักฐาน **(ใหม่ Demo 2)**
  4. **GET** `/incidents` — ดึงรายการภารกิจทั้งหมดของทีม **(ใหม่ Demo 2)**
- มี 3 Outbound Events ผ่าน EventBridge → SQS (MissionStatusChanged, MissionBackupRequested, ImpactLevelUpdated)
- มี 1 Inbound Event จาก Dispatch Service (MissionAssignedEvent) **(ใหม่ Demo 2)**
- **Outbox Processor** ทำงานเป็น scheduled Lambda (CloudWatch Events) retry events ที่ส่งไม่สำเร็จ **(ใหม่ Demo 2)**
- RescueRequest Service เชื่อมต่อได้จริง → `data_source: "full"` (ไม่ใช่ Degraded Mode แบบ Demo 1)

**สิ่งที่เพิ่มจาก Demo 1:**

| ฟีเจอร์                         | Demo 1          | Demo 2         |
| ------------------------------- | --------------- | -------------- |
| Frontend (Next.js)              | ❌              | ✅             |
| S3 Evidence Upload (presigned)  | ❌              | ✅             |
| List Missions (GET /missions)   | ❌              | ✅             |
| Inbound Event (MissionAssigned) | ❌              | ✅             |
| Outbox Processor (retry)        | ❌              | ✅             |
| RescueRequest Integration       | Mock (timeout)  | Full           |
| EventBridge → SQS Routing       | CloudWatch only | ✅ SQS targets |
| `image_key` in progress report  | ❌              | ✅             |

---

## การพึ่งพาบริการภายนอก (Dependencies)

MissionProgress Service พึ่งพาบริการอื่นทั้งแบบ Synchronous (HTTP) และ Asynchronous (EventBridge → SQS):

### บริการที่พึ่งพา (Sync HTTP)

| รายการ            | รายละเอียด                                                                                   |
| ----------------- | -------------------------------------------------------------------------------------------- |
| ชื่อบริการ        | **RescueRequest Service**                                                                    |
| เจ้าของบริการ     | **Phattharaphum Kingchai**                                                                   |
| ใช้ใน Endpoint    | `GET /missions/{request_id}`                                                                 |
| วิธีเรียก         | HTTP GET ไปยัง `{RESCUE_REQUEST_SERVICE_URL}/v1/rescue-requests/{request_id}` + Bearer token |
| Timeout           | 800 มิลลิวินาที + retry 2 ครั้ง                                                              |
| ไฟล์ที่เกี่ยวข้อง | `src/backend/internal/client/rescue_request_client.go`                                       |

### API Contract ที่ใช้อ้างอิง

```
GET {RESCUE_REQUEST_SERVICE_URL}/v1/rescue-requests/{requestId}
Authorization: Bearer <RESCUE_REQUEST_SERVICE_TOKEN>
```

**Response ที่คาดหวังจาก RescueRequest Service (HTTP 200):**

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

| ฟิลด์                 | ประเภท | คำอธิบาย                            |
| --------------------- | ------ | ----------------------------------- |
| `request.requestId`   | String | รหัส Request                        |
| `request.description` | String | คำอธิบาย Request                    |
| `request.location`    | Object | พิกัด GPS (`latitude`, `longitude`) |
| `request.requestType` | String | ประเภท เช่น `FLOOD`, `FIRE` เป็นต้น |

### Degraded Mode

เมื่อ RescueRequest Service ไม่ตอบภายใน 800ms (หลัง retry 2x) หรือ return error → ระบบจะ fallback เป็น Degraded Mode:

| โหมด              | `data_source` | ฟิลด์เพิ่มเติมจาก RescueRequest Service              |
| ----------------- | ------------- | ---------------------------------------------------- |
| **Full Mode**     | `"full"`      | มีครบ — `description`, `location`, `requestType`     |
| **Degraded Mode** | `"partial"`   | ไม่มี — ขาด `description`, `location`, `requestType` |

> **Demo 2:** RescueRequest Service เชื่อมต่อได้จริง → `data_source` จะเป็น `"full"` ในกรณีปกติ ถ้า RescueRequest ล่ม → fallback เป็น `"partial"` อัตโนมัติ

### Async Dependencies (EventBridge → SQS)

| บริการ               | เจ้าของ                | วิธีเชื่อมต่อ                                | ตัวแปร Terraform                                             | Degraded Mode            |
| -------------------- | ---------------------- | -------------------------------------------- | ------------------------------------------------------------ | ------------------------ |
| RescueRequest (Sync) | Phattharaphum Kingchai | HTTP GET `/v1/rescue-requests/{id}` (Bearer) | `rescue_request_service_url`, `rescue_request_service_token` | `data_source: "partial"` |
| IncidentTracking SQS | Krittamet Damthongkam  | EventBridge → SQS                            | `incident_tracking_sqs_arn`                                  | CloudWatch Logs fallback |
| Dispatch SQS         | Noppakron Songkroh     | EventBridge → SQS (RESOLVED only)            | `dispatch_sqs_arn`                                           | CloudWatch Logs fallback |
| Prioritization SQS   | Nattasak Chonmanat     | EventBridge → SQS                            | `prioritization_sqs_arn`                                     | CloudWatch Logs fallback |

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

---

## 2. Authentication (จำเป็นสำหรับทุก Request)

ทุก Request ต้องส่ง **2 Headers** ดังนี้:

| Header             | ค่า                             | คำอธิบาย                                         |
| ------------------ | ------------------------------- | ------------------------------------------------ |
| `x-api-key`        | `<api-key ที่แจกให้ในวัน demo>` | API Key สำหรับยืนยันสิทธิ์การเรียกใช้บริการ      |
| `X-Rescue-Team-ID` | เช่น `TEAM-ALPHA`               | รหัสทีมกู้ภัยที่กำลังเรียก API (ห้ามเป็นค่าว่าง) |

### กรณี Authentication ไม่ผ่าน

หาก **ไม่ส่ง** หรือส่ง Header ไม่ถูกต้อง จะได้รับ **403 Forbidden** ทันที:

```
HTTP/1.1 403 Forbidden

{ "message": "User is not authorized to access this resource" }
```

### ตัวอย่างการส่ง Header (curl)

```bash
curl -X GET \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions/REQ-001" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA"
```

---

## 3. Synchronous API: Get Mission Details

### Endpoint

```
GET /missions/{request_id}
```

### คำอธิบาย

ดึงข้อมูลภารกิจ (Mission Assignment) และ Timeline การปฏิบัติงานจาก DynamoDB โดยระบบจะเรียก RescueRequest Service เพื่อดึงรายละเอียด Request (description, location, requestType) มาแสดงด้วย

### Request

**Path Parameters:**

| พารามิเตอร์  | ประเภท | จำเป็น | คำอธิบาย                                              |
| ------------ | ------ | ------ | ----------------------------------------------------- |
| `request_id` | String | ✅     | รหัส request จาก RescueRequest Service เช่น `REQ-003` |

**Headers:** ดูหัวข้อ [2. Authentication](#2-authentication-จำเป็นสำหรับทุก-request)

**Body:** ไม่มี

### Response — Success (200 OK) — Full Mode

```json
{
  "request_id": "REQ-001",
  "incident_id": "REQ-001",
  "mission_id": "MSN-001",
  "rescue_team_id": "TEAM-ALPHA",
  "current_status": "ON_SITE",
  "latest_impact_level": 3,
  "started_at": "2024-12-01T08:00:00Z",
  "last_updated_at": "2024-12-01T10:00:00Z",
  "description": "น้ำท่วมหนักบริเวณถนนพหลโยธิน",
  "location": "13.7563,100.5018",
  "requestType": "FLOOD",
  "timeline": [
    {
      "mission_id": "MSN-001",
      "timestamp": "2024-12-01T08:00:00Z",
      "log_id": "LOG-001",
      "action_type": "MISSION_ASSIGNED",
      "description": "Mission assigned to TEAM-ALPHA",
      "performed_by": "SYSTEM"
    },
    {
      "mission_id": "MSN-001",
      "timestamp": "2024-12-01T09:00:00Z",
      "log_id": "LOG-002",
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
      "timestamp": "2024-12-01T10:00:00Z",
      "log_id": "LOG-003",
      "action_type": "STATUS_CHANGE",
      "description": "Arrived on site",
      "performed_by": "TEAM-ALPHA",
      "old_status": "EN_ROUTE",
      "new_status": "ON_SITE",
      "location": "13.7380,100.5230",
      "image_key": "evidence/REQ-001/TEAM-ALPHA/1718353500-photo.jpg"
    }
  ],
  "data_source": "full"
}
```

### Response — Success (200 OK) — Degraded Mode

เมื่อ RescueRequest Service ไม่ตอบ → `data_source: "partial"`, ไม่มี `description`, `location`, `requestType`:

```json
{
  "incident_id": "REQ-001",
  "mission_id": "MSN-001",
  "rescue_team_id": "TEAM-ALPHA",
  "current_status": "ON_SITE",
  "latest_impact_level": 3,
  "started_at": "2024-12-01T08:00:00Z",
  "last_updated_at": "2024-12-01T10:00:00Z",
  "timeline": [...],
  "data_source": "partial"
}
```

### คำอธิบายฟิลด์ Response

| ฟิลด์                 | ประเภท   | คำอธิบาย                                                                       |
| --------------------- | -------- | ------------------------------------------------------------------------------ |
| `incident_id`         | String   | รหัสเหตุการณ์                                                                  |
| `mission_id`          | String   | รหัสภารกิจ                                                                     |
| `rescue_team_id`      | String   | รหัสทีมกู้ภัยที่รับผิดชอบ                                                      |
| `current_status`      | String   | สถานะปัจจุบันของภารกิจ                                                         |
| `latest_impact_level` | Integer  | ระดับความรุนแรงล่าสุดที่ประเมิน (1–4, 0=ยังไม่ประเมิน)                         |
| `started_at`          | DateTime | เวลาที่เริ่มรับภารกิจ (ISO 8601)                                               |
| `last_updated_at`     | DateTime | เวลาอัปเดตล่าสุด (ISO 8601)                                                    |
| `description`         | String   | คำอธิบาย Request _(จาก RescueRequest — ไม่แสดงใน Degraded Mode)_               |
| `location`            | String   | พิกัด Request _(จาก RescueRequest — ไม่แสดงใน Degraded Mode)_                  |
| `requestType`         | String   | ประเภท Request _(จาก RescueRequest — ไม่แสดงใน Degraded Mode)_                 |
| `timeline`            | Array    | รายการ Timeline การปฏิบัติงาน (เรียงตามเวลา)                                   |
| `data_source`         | String   | `"full"` = ข้อมูลครบ, `"partial"` = Degraded Mode (ขาดข้อมูลจาก RescueRequest) |

**ฟิลด์ใน Timeline Entry:**

| ฟิลด์          | ประเภท   | คำอธิบาย                                        |
| -------------- | -------- | ----------------------------------------------- |
| `mission_id`   | String   | รหัสภารกิจ                                      |
| `timestamp`    | DateTime | เวลาที่เกิดเหตุการณ์ (ISO 8601)                 |
| `log_id`       | String   | รหัส Log (UUID)                                 |
| `action_type`  | String   | ประเภท: `MISSION_ASSIGNED`, `STATUS_CHANGE`     |
| `description`  | String   | รายละเอียดการปฏิบัติงาน                         |
| `performed_by` | String   | ผู้ดำเนินการ (รหัสทีม หรือ `SYSTEM`)            |
| `old_status`   | String   | สถานะก่อนเปลี่ยน _(ไม่บังคับ)_                  |
| `new_status`   | String   | สถานะหลังเปลี่ยน _(ไม่บังคับ)_                  |
| `location`     | String   | พิกัด GPS _(ไม่บังคับ — อาจไม่มีฟิลด์นี้)_      |
| `note`         | String   | หมายเหตุ _(ไม่บังคับ)_                          |
| `image_key`    | String   | S3 key ของภาพหลักฐาน _(ไม่บังคับ, ใหม่ Demo 2)_ |

### Response — Error (404 Not Found)

```json
{
  "error": "REQUEST_NOT_FOUND",
  "code": "REQUEST_NOT_FOUND",
  "message": "No mission found for request: REQ-99999",
  "traceId": "a1b2c3d4-..."
}
```

### Response — Error (400 Bad Request)

```json
{
  "error": "MISSING_PARAMETER",
  "code": "MISSING_PARAMETER",
  "message": "request_id is required",
  "traceId": "a1b2c3d4-..."
}
```

### ตัวอย่างการทดสอบ — กรณีสำเร็จและล้มเหลว

#### ✅ สำเร็จ — ดึงข้อมูลภารกิจที่มีอยู่ (Full Mode)

```bash
curl -X GET \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions/REQ-001" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA"
```

ผลลัพธ์ที่คาดหวัง (HTTP 200):

```json
{
  "incident_id": "REQ-001",
  "mission_id": "MSN-001",
  "rescue_team_id": "TEAM-ALPHA",
  "current_status": "ON_SITE",
  "latest_impact_level": 3,
  "started_at": "2024-12-01T08:00:00Z",
  "last_updated_at": "2024-12-01T10:00:00Z",
  "description": "น้ำท่วมหนักบริเวณถนนพหลโยธิน",
  "location": "13.7563,100.5018",
  "requestType": "FLOOD",
  "timeline": [...],
  "data_source": "full"
}
```

#### ❌ ล้มเหลว — incident_id ที่ไม่มีในระบบ

```bash
curl -X GET \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions/REQ-99999" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA"
```

ผลลัพธ์ที่คาดหวัง (HTTP 404):

```json
{
  "error": "REQUEST_NOT_FOUND",
  "code": "REQUEST_NOT_FOUND",
  "message": "No mission found for request: REQ-99999"
}
```

#### ❌ ล้มเหลว — ไม่ส่ง API Key

```bash
curl -X GET \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions/REQ-001"
```

ผลลัพธ์ที่คาดหวัง (HTTP 403):

```json
{
  "message": "Forbidden"
}
```

#### ❌ ล้มเหลว — ไม่ส่ง X-Rescue-Team-ID

```bash
curl -X GET \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions/REQ-001" \
  -H "x-api-key: <your-api-key>"
```

ผลลัพธ์ที่คาดหวัง (HTTP 403):

```json
{
  "message": "User is not authorized to access this resource"
}
```

---

## 4. Sync + Async API: Report Progress

### Endpoint

```
POST /missions/{request_id}/progress
```

### คำอธิบาย

ใช้สำหรับทีมกู้ภัยเพื่อ **อัปเดตสถานะภารกิจ** และ **บันทึก Timeline** การปฏิบัติงาน

**การทำงานภายใน:**

1. **(Sync)** ตรวจสอบความถูกต้องของสถานะ → อัปเดต DynamoDB → เพิ่ม Timeline entry → ตอบกลับ 200
2. **(Async)** Publish events ไป Amazon EventBridge → SQS เพื่อแจ้ง Service อื่น ๆ (ดูหัวข้อ 7)
3. **(Fallback)** หากส่ง event ไป EventBridge ไม่สำเร็จ → บันทึกลง Outbox table แทน (Outbox Pattern) → Outbox Processor จะ retry อัตโนมัติ

### Request

**Path Parameters:**

| พารามิเตอร์  | ประเภท | จำเป็น | คำอธิบาย                                              |
| ------------ | ------ | ------ | ----------------------------------------------------- |
| `request_id` | String | ✅     | รหัส request จาก RescueRequest Service เช่น `REQ-003` |

**Headers:** ดูหัวข้อ [2. Authentication](#2-authentication-จำเป็นสำหรับทุก-request)

**Body (JSON):**

```json
{
  "new_status": "EN_ROUTE",
  "note": "กำลังเดินทางไปจุดเกิดเหตุ",
  "current_location": "13.7563,100.5018",
  "new_impact_level": 3,
  "image_key": "evidence/REQ-001/TEAM-ALPHA/1718353500-photo.jpg"
}
```

| ฟิลด์              | ประเภท  | จำเป็น | คำอธิบาย                                                               |
| ------------------ | ------- | ------ | ---------------------------------------------------------------------- |
| `new_status`       | String  | ✅     | สถานะใหม่ (ดูตาราง State Machine ด้านล่าง)                             |
| `note`             | String  | ❌     | หมายเหตุ / รายละเอียดการปฏิบัติงาน                                     |
| `current_location` | String  | ❌     | พิกัด GPS ปัจจุบัน เช่น `"13.7563,100.5018"`                           |
| `new_impact_level` | Integer | ❌     | ระดับความรุนแรงใหม่ที่ประเมินจากหน้างาน (1–4) ส่งเมื่อต้องการปรับค่า   |
| `image_key`        | String  | ❌     | S3 key ของภาพหลักฐาน (ได้จาก presigned-url endpoint) **(ใหม่ Demo 2)** |

### State Machine — กฎการเปลี่ยนสถานะ

สถานะที่ใช้ในระบบ: `DISPATCHED`, `EN_ROUTE`, `ON_SITE`, `NEED_BACKUP`, `RESOLVED`

| สถานะปัจจุบัน (From) | สถานะที่เปลี่ยนไปได้ (To)           |
| -------------------- | ----------------------------------- |
| `DISPATCHED`         | `EN_ROUTE`                          |
| `EN_ROUTE`           | `ON_SITE`                           |
| `ON_SITE`            | `NEED_BACKUP`, `RESOLVED`           |
| `NEED_BACKUP`        | `ON_SITE`, `RESOLVED`               |
| `RESOLVED`           | _(สถานะสุดท้าย — เปลี่ยนต่อไม่ได้)_ |

**ภาพรวม Flow:**

```
DISPATCHED → EN_ROUTE → ON_SITE → RESOLVED
                            ↓          ↑
                       NEED_BACKUP ─────┘
                            ↑          │
                            └──────────┘  (กลับไป ON_SITE ได้)
```

> **สำคัญ:** ถ้าส่งสถานะที่ไม่ตรงตามตาราง จะได้ Error `INVALID_STATE_TRANSITION` (400)

### Response — Success (200 OK)

```json
{
  "message": "Progress reported successfully",
  "mission_id": "MSN-001",
  "incident_id": "REQ-001",
  "old_status": "DISPATCHED",
  "new_status": "EN_ROUTE",
  "updated_at": "2025-01-15T12:00:00Z"
}
```

| ฟิลด์        | ประเภท   | คำอธิบาย            |
| ------------ | -------- | ------------------- |
| `message`    | String   | ข้อความยืนยันสำเร็จ |
| `mission_id` | String   | รหัสภารกิจที่อัปเดต |
| `request_id` | String   | รหัสเหตุการณ์       |
| `old_status` | String   | สถานะเดิมก่อนอัปเดต |
| `new_status` | String   | สถานะใหม่หลังอัปเดต |
| `updated_at` | DateTime | เวลาที่อัปเดตสำเร็จ |

### Response — Error (400): Invalid State Transition

```json
{
  "error": "INVALID_STATE_TRANSITION",
  "code": "INVALID_STATE_TRANSITION",
  "message": "Cannot transition from DISPATCHED to RESOLVED",
  "traceId": "a1b2c3d4-..."
}
```

### Response — Error (400): Missing / Invalid Fields

```json
{
  "error": "MISSING_PARAMETER",
  "code": "MISSING_PARAMETER",
  "message": "new_status is required"
}
```

```json
{
  "error": "INVALID_STATUS",
  "code": "INVALID_STATUS",
  "message": "Invalid status value: UNKNOWN_STATUS"
}
```

```json
{
  "error": "INVALID_BODY",
  "code": "INVALID_BODY",
  "message": "Invalid request body"
}
```

### Response — Error (404 Not Found)

```json
{
  "error": "REQUEST_NOT_FOUND",
  "code": "REQUEST_NOT_FOUND",
  "message": "No mission found for request: REQ-99999"
}
```

### ตัวอย่างการทดสอบ — กรณีสำเร็จและล้มเหลว

#### ✅ สำเร็จ — เปลี่ยนสถานะ DISPATCHED → EN_ROUTE

```bash
curl -X POST \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions/REQ-001/progress" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{
    "new_status": "EN_ROUTE",
    "note": "กำลังเดินทางไปจุดเกิดเหตุ",
    "current_location": "13.7563,100.5018"
  }'
```

ผลลัพธ์ที่คาดหวัง (HTTP 200):

```json
{
  "message": "Progress reported successfully",
  "mission_id": "MSN-001",
  "incident_id": "REQ-001",
  "old_status": "DISPATCHED",
  "new_status": "EN_ROUTE",
  "updated_at": "2025-..."
}
```

#### ✅ สำเร็จ — เปลี่ยนสถานะเป็น NEED_BACKUP + ImpactLevel + ภาพหลักฐาน

```bash
curl -X POST \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions/REQ-003/progress" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-CHARLIE" \
  -H "Content-Type: application/json" \
  -d '{
    "new_status": "NEED_BACKUP",
    "note": "น้ำท่วมสูงกว่าที่คาด ต้องการกำลังเสริม",
    "current_location": "13.7380,100.5230",
    "new_impact_level": 4,
    "image_key": "evidence/REQ-003/TEAM-CHARLIE/1718353500-flood.jpg"
  }'
```

ผลลัพธ์ที่คาดหวัง (HTTP 200):

```json
{
  "message": "Progress reported successfully",
  "mission_id": "MSN-003",
  "incident_id": "REQ-003",
  "old_status": "ON_SITE",
  "new_status": "NEED_BACKUP",
  "updated_at": "2025-..."
}
```

> **Events ที่ถูก publish:** MissionStatusChanged + MissionBackupRequested + ImpactLevelUpdated (ทั้ง 3 events)

#### ❌ ล้มเหลว — Transition สถานะไม่ถูกต้อง (EN_ROUTE → RESOLVED)

```bash
curl -X POST \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions/REQ-002/progress" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-BRAVO" \
  -H "Content-Type: application/json" \
  -d '{"new_status": "RESOLVED"}'
```

ผลลัพธ์ที่คาดหวัง (HTTP 400):

```json
{
  "error": "INVALID_STATE_TRANSITION",
  "code": "INVALID_STATE_TRANSITION",
  "message": "Cannot transition from EN_ROUTE to RESOLVED"
}
```

#### ❌ ล้มเหลว — ไม่ส่ง new_status ใน body

```bash
curl -X POST \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions/REQ-001/progress" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"note": "ทดสอบ"}'
```

ผลลัพธ์ที่คาดหวัง (HTTP 400):

```json
{
  "error": "MISSING_PARAMETER",
  "code": "MISSING_PARAMETER",
  "message": "new_status is required"
}
```

#### ❌ ล้มเหลว — ส่งค่า status ที่ไม่มีในระบบ

```bash
curl -X POST \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions/REQ-001/progress" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"new_status": "UNKNOWN_STATUS"}'
```

ผลลัพธ์ที่คาดหวัง (HTTP 400):

```json
{
  "error": "INVALID_STATUS",
  "code": "INVALID_STATUS",
  "message": "Invalid status value: UNKNOWN_STATUS"
}
```

#### ❌ ล้มเหลว — incident_id ไม่มีในระบบ

```bash
curl -X POST \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions/REQ-99999/progress" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"new_status": "EN_ROUTE"}'
```

ผลลัพธ์ที่คาดหวัง (HTTP 404):

```json
{
  "error": "REQUEST_NOT_FOUND",
  "code": "REQUEST_NOT_FOUND",
  "message": "No mission found for request: REQ-99999"
}
```

#### ❌ ล้มเหลว — ส่ง JSON body ไม่ถูกต้อง

```bash
curl -X POST \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions/REQ-001/progress" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d 'invalid-json'
```

ผลลัพธ์ที่คาดหวัง (HTTP 400):

```json
{
  "error": "INVALID_BODY",
  "code": "INVALID_BODY",
  "message": "Invalid request body"
}
```

#### ❌ ล้มเหลว — อัปเดตภารกิจที่ RESOLVED แล้ว (สถานะสุดท้าย)

```bash
curl -X POST \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions/REQ-005/progress" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-ECHO" \
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

## 5. Sync API: Generate Presigned URL (ใหม่ Demo 2)

### Endpoint

```
POST /missions/{request_id}/presigned-url
```

### คำอธิบาย

ขอ Presigned URL สำหรับอัปโหลดภาพหลักฐานไปยัง S3 Evidence Bucket โดยทีมกู้ภัยจะได้ URL ที่ใช้ได้ 300 วินาที (5 นาที) สำหรับ PUT ไฟล์ภาพ

### Request

**Path Parameters:**

| พารามิเตอร์  | ประเภท | จำเป็น | คำอธิบาย                                              |
| ------------ | ------ | ------ | ----------------------------------------------------- |
| `request_id` | String | ✅     | รหัส request จาก RescueRequest Service เช่น `REQ-003` |

**Headers:** ดูหัวข้อ [2. Authentication](#2-authentication-จำเป็นสำหรับทุก-request)

**Body (JSON):**

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

### Response — Success (200 OK)

```json
{
  "upload_url": "https://s3.amazonaws.com/mission-progress-evidence-123456789/evidence/REQ-001/TEAM-ALPHA/1718353500-flood-evidence.jpg?...",
  "image_key": "evidence/REQ-001/TEAM-ALPHA/1718353500-flood-evidence.jpg",
  "expires_in": 300,
  "message": "Upload URL generated successfully. Use PUT method to upload."
}
```

| ฟิลด์        | ประเภท  | คำอธิบาย                                       |
| ------------ | ------- | ---------------------------------------------- |
| `upload_url` | String  | Presigned URL สำหรับ PUT upload (หมดอายุ 300s) |
| `image_key`  | String  | S3 key ของไฟล์ — ใช้ส่งใน `report-progress`    |
| `expires_in` | Integer | จำนวนวินาทีก่อน URL หมดอายุ (300)              |
| `message`    | String  | คำแนะนำการใช้งาน                               |

### การอัปโหลดภาพ (ขั้นตอนที่ 2)

หลังได้ `upload_url` แล้ว ให้ใช้ HTTP PUT เพื่ออัปโหลดภาพ:

```bash
curl -X PUT \
  -T flood-evidence.jpg \
  -H "Content-Type: image/jpeg" \
  "https://s3.amazonaws.com/...presigned-url..."
```

> **สำคัญ:** `Content-Type` ใน PUT request ต้องตรงกับ `content_type` ที่ส่งตอนขอ presigned URL

### Response — Error (400): Missing Parameters

```json
{
  "error": "MISSING_PARAMETER",
  "code": "MISSING_PARAMETER",
  "message": "file_name is required"
}
```

```json
{
  "error": "MISSING_PARAMETER",
  "code": "MISSING_PARAMETER",
  "message": "content_type is required"
}
```

### Response — Error (400): Invalid Content Type

```json
{
  "error": "INVALID_CONTENT_TYPE",
  "code": "INVALID_CONTENT_TYPE",
  "message": "content_type must be one of: image/jpeg, image/png, image/webp"
}
```

### Response — Error (404): Mission Not Found

```json
{
  "error": "REQUEST_NOT_FOUND",
  "code": "REQUEST_NOT_FOUND",
  "message": "No mission found for request: REQ-99999"
}
```

### Response — Error (500): Presign Failed

```json
{
  "error": "PRESIGN_FAILED",
  "code": "PRESIGN_FAILED",
  "message": "Failed to generate upload URL. Mission can still operate in text-only mode."
}
```

### ตัวอย่างการทดสอบ — กรณีสำเร็จและล้มเหลว

#### ✅ สำเร็จ — ขอ presigned URL สำหรับ JPEG

```bash
curl -X POST \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions/REQ-001/presigned-url" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{
    "file_name": "flood-evidence.jpg",
    "content_type": "image/jpeg"
  }'
```

ผลลัพธ์ที่คาดหวัง (HTTP 200):

```json
{
  "upload_url": "https://s3.amazonaws.com/...",
  "image_key": "evidence/REQ-001/TEAM-ALPHA/1718353500-flood-evidence.jpg",
  "expires_in": 300,
  "message": "Upload URL generated successfully. Use PUT method to upload."
}
```

#### ✅ สำเร็จ — ขอ presigned URL สำหรับ PNG

```bash
curl -X POST \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions/REQ-001/presigned-url" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{
    "file_name": "screenshot.png",
    "content_type": "image/png"
  }'
```

ผลลัพธ์ที่คาดหวัง (HTTP 200): เหมือนด้านบน แต่ `image_key` จะลงท้ายด้วย `screenshot.png`

#### ❌ ล้มเหลว — content_type ไม่รองรับ (PDF)

```bash
curl -X POST \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions/REQ-001/presigned-url" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{
    "file_name": "document.pdf",
    "content_type": "application/pdf"
  }'
```

ผลลัพธ์ที่คาดหวัง (HTTP 400):

```json
{
  "error": "INVALID_CONTENT_TYPE",
  "code": "INVALID_CONTENT_TYPE",
  "message": "content_type must be one of: image/jpeg, image/png, image/webp"
}
```

#### ❌ ล้มเหลว — ไม่ส่ง file_name

```bash
curl -X POST \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions/REQ-001/presigned-url" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"content_type": "image/jpeg"}'
```

ผลลัพธ์ที่คาดหวัง (HTTP 400):

```json
{
  "error": "MISSING_PARAMETER",
  "code": "MISSING_PARAMETER",
  "message": "file_name is required"
}
```

#### ❌ ล้มเหลว — incident_id ไม่มีในระบบ

```bash
curl -X POST \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions/REQ-99999/presigned-url" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{
    "file_name": "photo.jpg",
    "content_type": "image/jpeg"
  }'
```

ผลลัพธ์ที่คาดหวัง (HTTP 404):

```json
{
  "error": "REQUEST_NOT_FOUND",
  "code": "REQUEST_NOT_FOUND",
  "message": "No mission found for request: REQ-99999"
}
```

---

## 6. Sync API: List Missions by Team (ใหม่ Demo 2)

### Endpoint

```
GET /missions
```

### คำอธิบาย

ดึงรายการภารกิจทั้งหมดของทีมกู้ภัย โดยใช้ `X-Rescue-Team-ID` จาก Header เป็นตัวกรอง (ทีมอื่นจะดูข้อมูลของทีมอื่นไม่ได้) สามารถกรองตามสถานะได้ด้วย query parameter `status`

### Request

**Path Parameters:** ไม่มี

**Query Parameters:**

| พารามิเตอร์ | ประเภท | จำเป็น | คำอธิบาย                                                                     |
| ----------- | ------ | ------ | ---------------------------------------------------------------------------- |
| `status`    | String | ❌     | กรองตามสถานะ: `DISPATCHED`, `EN_ROUTE`, `ON_SITE`, `NEED_BACKUP`, `RESOLVED` |

**Headers:** ดูหัวข้อ [2. Authentication](#2-authentication-จำเป็นสำหรับทุก-request)

**Body:** ไม่มี

### Response — Success (200 OK)

```json
{
  "team_id": "TEAM-ALPHA",
  "total_missions": 2,
  "missions": [
    {
      "mission_id": "MSN-001",
      "incident_id": "REQ-001",
      "rescue_team_id": "TEAM-ALPHA",
      "current_status": "ON_SITE",
      "latest_impact_level": 3,
      "started_at": "2024-12-01T08:00:00Z",
      "last_updated_at": "2024-12-01T10:00:00Z"
    },
    {
      "mission_id": "MSN-004",
      "incident_id": "REQ-004",
      "rescue_team_id": "TEAM-ALPHA",
      "current_status": "DISPATCHED",
      "latest_impact_level": 1,
      "started_at": "2024-12-02T06:00:00Z",
      "last_updated_at": "2024-12-02T06:00:00Z"
    }
  ]
}
```

| ฟิลด์            | ประเภท  | คำอธิบาย                       |
| ---------------- | ------- | ------------------------------ |
| `team_id`        | String  | รหัสทีมกู้ภัย (จาก Header)     |
| `total_missions` | Integer | จำนวนภารกิจทั้งหมด             |
| `missions`       | Array   | รายการภารกิจ (อาจเป็น [] ว่าง) |

**ฟิลด์ในแต่ละ Mission:**

| ฟิลด์                 | ประเภท   | คำอธิบาย                               |
| --------------------- | -------- | -------------------------------------- |
| `mission_id`          | String   | รหัสภารกิจ                             |
| `incident_id`         | String   | รหัสเหตุการณ์                          |
| `rescue_team_id`      | String   | รหัสทีมกู้ภัย                          |
| `current_status`      | String   | สถานะปัจจุบัน                          |
| `latest_impact_level` | Integer  | ระดับความรุนแรง (1–4, 0=ยังไม่ประเมิน) |
| `started_at`          | DateTime | เวลาเริ่มภารกิจ                        |
| `last_updated_at`     | DateTime | เวลาอัปเดตล่าสุด                       |

> **หมายเหตุ:** ถ้าไม่มี missions → return `200` กับ `missions: []` (ไม่ใช่ 404)

### Response — Error (400): Invalid Status Filter

```json
{
  "error": "INVALID_STATUS",
  "code": "INVALID_STATUS",
  "message": "Invalid status filter: UNKNOWN_STATUS"
}
```

### ตัวอย่างการทดสอบ — กรณีสำเร็จและล้มเหลว

#### ✅ สำเร็จ — ดึงภารกิจทั้งหมดของทีม

```bash
curl -X GET \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA"
```

ผลลัพธ์ที่คาดหวัง (HTTP 200):

```json
{
  "team_id": "TEAM-ALPHA",
  "total_missions": 2,
  "missions": [...]
}
```

#### ✅ สำเร็จ — กรองเฉพาะสถานะ ON_SITE

```bash
curl -X GET \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions?status=ON_SITE" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA"
```

ผลลัพธ์ที่คาดหวัง (HTTP 200):

```json
{
  "team_id": "TEAM-ALPHA",
  "total_missions": 1,
  "missions": [
    {
      "mission_id": "MSN-001",
      "incident_id": "REQ-001",
      "rescue_team_id": "TEAM-ALPHA",
      "current_status": "ON_SITE",
      ...
    }
  ]
}
```

#### ✅ สำเร็จ — ทีมที่ไม่มีภารกิจ (return array ว่าง)

```bash
curl -X GET \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-NEWBIE"
```

ผลลัพธ์ที่คาดหวัง (HTTP 200):

```json
{
  "team_id": "TEAM-NEWBIE",
  "total_missions": 0,
  "missions": []
}
```

#### ❌ ล้มเหลว — status filter ไม่ถูกต้อง

```bash
curl -X GET \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/missions?status=UNKNOWN" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA"
```

ผลลัพธ์ที่คาดหวัง (HTTP 400):

```json
{
  "error": "INVALID_STATUS",
  "code": "INVALID_STATUS",
  "message": "Invalid status filter: UNKNOWN"
}
```

---

## 7. Asynchronous Events Contract (EventBridge → SQS)

เมื่อ POST `/missions/{request_id}/progress` สำเร็จ ระบบจะ publish events ไปยัง Amazon EventBridge **แบบ asynchronous** (ไม่กระทบ response ที่ตอบกลับผู้เรียก) แล้ว EventBridge จะ route ไปยัง SQS queues ของ Service ปลายทาง

### ข้อมูล Event Bus

| รายการ    | ค่า                       |
| --------- | ------------------------- |
| Event Bus | `mission-progress-events` |
| Source    | `MissionProgressService`  |
| Region    | `us-east-1`               |

### Outbox Pattern (Fallback)

หากการ publish ไม่สำเร็จ → ระบบจะบันทึก event ลง Outbox table ใน DynamoDB → **Outbox Processor** Lambda (scheduled ทุก 5 นาที) จะ retry ส่ง event ที่ค้างอยู่

---

### 7.1 MissionStatusChanged

**เงื่อนไขการ publish:** ทุกครั้งที่สถานะภารกิจเปลี่ยน (publish ทุก POST ที่สำเร็จ)

**Detail Type:** `MissionStatusChanged`

**Consumers:** IncidentTracking (SQS), Dispatch (SQS, เฉพาะ RESOLVED)

**Payload:**

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
    "changed_at": "2025-01-15T12:00:00Z",
    "changed_by": "TEAM-ALPHA"
  }
}
```

| ฟิลด์            | ประเภท   | คำอธิบาย                           |
| ---------------- | -------- | ---------------------------------- |
| `schema_version` | String   | เวอร์ชัน schema (`"1.0"`)          |
| `mission_id`     | String   | รหัสภารกิจ                         |
| `incident_id`    | String   | รหัสเหตุการณ์                      |
| `rescue_team_id` | String   | รหัสทีมกู้ภัย                      |
| `old_status`     | String   | สถานะก่อนเปลี่ยน                   |
| `new_status`     | String   | สถานะใหม่                          |
| `changed_at`     | DateTime | เวลาที่เปลี่ยนสถานะ                |
| `changed_by`     | String   | ผู้ดำเนินการเปลี่ยนสถานะ (รหัสทีม) |

---

### 7.2 MissionBackupRequested

**เงื่อนไขการ publish:** เฉพาะเมื่อ `new_status` = `NEED_BACKUP` เท่านั้น

**Detail Type:** `MissionBackupRequested`

**Consumer:** Prioritization (SQS)

**Payload:**

```json
{
  "source": "MissionProgressService",
  "detail-type": "MissionBackupRequested",
  "detail": {
    "schema_version": "1.0",
    "mission_id": "MSN-003",
    "incident_id": "REQ-003",
    "rescue_team_id": "TEAM-CHARLIE",
    "requested_at": "2025-01-15T10:30:00Z",
    "requested_by": "TEAM-CHARLIE",
    "location": "13.7380,100.5230"
  }
}
```

| ฟิลด์            | ประเภท   | คำอธิบาย                                   |
| ---------------- | -------- | ------------------------------------------ |
| `schema_version` | String   | เวอร์ชัน schema (`"1.0"`)                  |
| `mission_id`     | String   | รหัสภารกิจ                                 |
| `incident_id`    | String   | รหัสเหตุการณ์                              |
| `rescue_team_id` | String   | รหัสทีมกู้ภัยที่ขอกำลังเสริม               |
| `requested_at`   | DateTime | เวลาที่ขอ backup                           |
| `requested_by`   | String   | ผู้ร้องขอ (รหัสทีม)                        |
| `location`       | String   | พิกัด GPS _(ไม่บังคับ — อาจไม่มีฟิลด์นี้)_ |

> **สำหรับ Rescue Prioritization Service:** event นี้มีไว้เพื่อแจ้งว่าทีมกู้ภัยหน้างานต้องการกำลังเสริม สามารถนำไปคำนวณ Priority Score ใหม่ได้

---

### 7.3 ImpactLevelUpdated

**เงื่อนไขการ publish:** เฉพาะเมื่อ request body มีฟิลด์ `new_impact_level`

**Detail Type:** `ImpactLevelUpdated`

**Consumers:** IncidentTracking (SQS), Prioritization (SQS)

**Payload:**

```json
{
  "source": "MissionProgressService",
  "detail-type": "ImpactLevelUpdated",
  "detail": {
    "schema_version": "1.0",
    "mission_id": "MSN-003",
    "incident_id": "REQ-003",
    "rescue_team_id": "TEAM-CHARLIE",
    "old_level": 3,
    "new_level": 4,
    "updated_at": "2025-01-15T10:30:00Z",
    "updated_by": "TEAM-CHARLIE"
  }
}
```

| ฟิลด์            | ประเภท   | คำอธิบาย                              |
| ---------------- | -------- | ------------------------------------- |
| `schema_version` | String   | เวอร์ชัน schema (`"1.0"`)             |
| `mission_id`     | String   | รหัสภารกิจ                            |
| `incident_id`    | String   | รหัสเหตุการณ์                         |
| `rescue_team_id` | String   | รหัสทีมกู้ภัยที่ประเมินความรุนแรงใหม่ |
| `old_level`      | Integer  | ระดับความรุนแรงเดิม                   |
| `new_level`      | Integer  | ระดับความรุนแรงใหม่ (1–4)             |
| `updated_at`     | DateTime | เวลาที่อัปเดต                         |
| `updated_by`     | String   | ผู้ดำเนินการ (รหัสทีม)                |

> **สำหรับ Rescue Prioritization Service / IncidentTracking Service:** event นี้แจ้งว่าระดับความรุนแรงของเหตุการณ์ถูกปรับโดยทีมหน้างาน สามารถนำ `new_level` ไปอัปเดตฐานข้อมูลหรือคำนวณลำดับความสำคัญใหม่ได้

---

### Event Routing Summary

| Event                  | IncidentTracking SQS | Dispatch SQS (RESOLVED only) | Prioritization SQS | CloudWatch Logs |
| ---------------------- | -------------------- | ---------------------------- | ------------------ | --------------- |
| MissionStatusChanged   | ✅                   | ✅ (filtered)                | ❌                 | ✅              |
| MissionBackupRequested | ❌                   | ❌                           | ✅                 | ✅              |
| ImpactLevelUpdated     | ✅                   | ❌                           | ✅                 | ✅              |

### Subscribe Events ของ MissionProgress Service

หากต้องการ subscribe events เหล่านี้ สามารถสร้าง EventBridge Rule ที่ match:

- `source` = `MissionProgressService`
- `detail-type` = `MissionStatusChanged` / `MissionBackupRequested` / `ImpactLevelUpdated`
- บน event bus `mission-progress-events`

---

## 8. Inbound Async Event (from Dispatch) — ใหม่ Demo 2

### MissionAssignedEvent

**Source:** `dispatch-management-service`
**Detail-type:** `MissionAssignedEvent`
**Handler:** `mission-assigned-handler` Lambda

เมื่อ Dispatch Service มอบหมายภารกิจใหม่ → MissionProgress จะสร้าง mission record อัตโนมัติ พร้อมตั้งสถานะเริ่มต้นเป็น `DISPATCHED` และสร้าง Timeline entry `MISSION_ASSIGNED`

**Expected Payload:**

```json
{
  "source": "dispatch-management-service",
  "detail-type": "MissionAssignedEvent",
  "detail": {
    "mission_id": "MSN-001",
    "rescue_unit_id": "TEAM-ALPHA",
    "incident_id": "REQ-001",
    "assigned_at": "2025-06-14T08:45:00Z"
  }
}
```

| ฟิลด์            | ประเภท   | จำเป็น | คำอธิบาย                                    |
| ---------------- | -------- | ------ | ------------------------------------------- |
| `mission_id`     | String   | ✅     | รหัสภารกิจ                                  |
| `rescue_unit_id` | String   | ✅     | รหัสทีมกู้ภัย (map เป็น `rescue_team_id`)   |
| `incident_id`    | String   | ✅     | รหัสเหตุการณ์                               |
| `assigned_at`    | DateTime | ❌     | เวลาที่มอบหมาย (ถ้าไม่ส่งจะใช้ `"unknown"`) |

### การทำงานภายใน

1. Parse event payload จาก EventBridge
2. Validate required fields (`mission_id`, `incident_id`, `rescue_unit_id`)
3. สร้าง MissionAssignment record ใน DynamoDB (status = `DISPATCHED`)
4. สร้าง Timeline entry `MISSION_ASSIGNED`

### Idempotency

ใช้ DynamoDB condition expression `attribute_not_exists(mission_id)`:

- ถ้า mission **ยังไม่มี** → สร้างใหม่ ✅
- ถ้า mission **มีอยู่แล้ว** → skip โดยไม่ error (idempotent) ✅

> **หมายเหตุ:** Dispatch Service สามารถส่ง event ซ้ำได้โดยไม่ต้องกังวลว่าจะเกิด duplicate

---

## สรุปภาพรวม Endpoints

| Method | Path                                   | คำอธิบาย                     | ประเภท       | Demo |
| ------ | -------------------------------------- | ---------------------------- | ------------ | ---- |
| GET    | `/missions/{request_id}`               | ดึงข้อมูลภารกิจ + Timeline   | Synchronous  | 1+2  |
| POST   | `/missions/{request_id}/progress`      | อัปเดตสถานะ + publish events | Sync + Async | 1+2  |
| POST   | `/missions/{request_id}/presigned-url` | ขอ URL อัปโหลดภาพหลักฐาน     | Synchronous  | 2    |
| GET    | `/incidents`                           | ดึงรายการภารกิจทั้งหมดของทีม | Synchronous  | 2    |

## สรุป Error Codes

| HTTP Status | Error Code                 | สาเหตุ                                         | Endpoints ที่เกี่ยว                            |
| ----------- | -------------------------- | ---------------------------------------------- | ---------------------------------------------- |
| 400         | `MISSING_PARAMETER`        | ไม่ส่ง parameter ที่จำเป็น                     | ทั้งหมด                                        |
| 400         | `INVALID_BODY`             | JSON body ไม่ถูกต้อง                           | POST progress, POST presigned-url              |
| 400         | `INVALID_STATUS`           | new_status / status filter ไม่ใช่สถานะที่กำหนด | POST progress, GET /incidents                  |
| 400         | `INVALID_STATE_TRANSITION` | เปลี่ยนสถานะไม่ตรงตามกฎ State Machine          | POST progress                                  |
| 400         | `INVALID_CONTENT_TYPE`     | content_type ไม่รองรับ                         | POST presigned-url                             |
| 403         | —                          | ไม่ส่ง x-api-key หรือ X-Rescue-Team-ID         | ทั้งหมด                                        |
| 404         | `REQUEST_NOT_FOUND`        | ไม่พบภารกิจสำหรับ incident_id ที่ระบุ          | GET mission, POST progress, POST presigned-url |
| 500         | `INTERNAL_ERROR`           | เกิดข้อผิดพลาดภายในระบบ                        | ทั้งหมด                                        |
| 500         | `PRESIGN_FAILED`           | ไม่สามารถสร้าง presigned URL ได้               | POST presigned-url                             |

---

## สรุปผลการทดสอบ

| #   | กรณีทดสอบ                                     | Endpoint            | ประเภท     | HTTP Status | ผลที่คาดหวัง                      |
| --- | --------------------------------------------- | ------------------- | ---------- | ----------- | --------------------------------- |
| 1   | ส่ง API Key + Team ID ถูกต้อง                 | ทุก endpoint        | ✅ สำเร็จ  | 200         | ได้รับข้อมูลตามปกติ               |
| 2   | ไม่ส่ง API Key                                | ทุก endpoint        | ❌ ล้มเหลว | 403         | `Forbidden`                       |
| 3   | ไม่ส่ง X-Rescue-Team-ID                       | ทุก endpoint        | ❌ ล้มเหลว | 403         | `User is not authorized...`       |
| 4   | GET ภารกิจที่มีอยู่ (Full Mode)               | GET /incidents/{id} | ✅ สำเร็จ  | 200         | ข้อมูลครบ + `data_source: "full"` |
| 5   | GET ภารกิจที่ไม่มี (REQ-99999)                | GET /incidents/{id} | ❌ ล้มเหลว | 404         | `REQUEST_NOT_FOUND`               |
| 6   | POST transition ถูกต้อง (DISPATCHED→EN_ROUTE) | POST progress       | ✅ สำเร็จ  | 200         | `Progress reported successfully`  |
| 7   | POST NEED_BACKUP + ImpactLevel + image_key    | POST progress       | ✅ สำเร็จ  | 200         | `Progress reported successfully`  |
| 8   | POST transition ผิดกฎ (EN_ROUTE→RESOLVED)     | POST progress       | ❌ ล้มเหลว | 400         | `INVALID_STATE_TRANSITION`        |
| 9   | POST ไม่ส่ง new_status                        | POST progress       | ❌ ล้มเหลว | 400         | `MISSING_PARAMETER`               |
| 10  | POST ส่ง status ที่ไม่มีในระบบ                | POST progress       | ❌ ล้มเหลว | 400         | `INVALID_STATUS`                  |
| 11  | POST incident_id ไม่มีในระบบ                  | POST progress       | ❌ ล้มเหลว | 404         | `REQUEST_NOT_FOUND`               |
| 12  | POST ส่ง JSON ไม่ถูกต้อง                      | POST progress       | ❌ ล้มเหลว | 400         | `INVALID_BODY`                    |
| 13  | POST อัปเดตภารกิจที่ RESOLVED แล้ว            | POST progress       | ❌ ล้มเหลว | 400         | `INVALID_STATE_TRANSITION`        |
| 14  | POST presigned-url สำเร็จ (JPEG)              | POST presigned-url  | ✅ สำเร็จ  | 200         | ได้ upload_url + image_key        |
| 15  | POST presigned-url สำเร็จ (PNG)               | POST presigned-url  | ✅ สำเร็จ  | 200         | ได้ upload_url + image_key        |
| 16  | POST presigned-url content_type ไม่รองรับ     | POST presigned-url  | ❌ ล้มเหลว | 400         | `INVALID_CONTENT_TYPE`            |
| 17  | POST presigned-url ไม่ส่ง file_name           | POST presigned-url  | ❌ ล้มเหลว | 400         | `MISSING_PARAMETER`               |
| 18  | POST presigned-url incident_id ไม่มี          | POST presigned-url  | ❌ ล้มเหลว | 404         | `REQUEST_NOT_FOUND`               |
| 19  | GET /incidents ดึงภารกิจทั้งหมดของทีม         | GET /incidents      | ✅ สำเร็จ  | 200         | ได้รายการภารกิจ                   |
| 20  | GET /incidents กรองตาม status=ON_SITE         | GET /incidents      | ✅ สำเร็จ  | 200         | ได้เฉพาะภารกิจ ON_SITE            |
| 21  | GET /incidents ทีมที่ไม่มีภารกิจ              | GET /incidents      | ✅ สำเร็จ  | 200         | `missions: []`, `total: 0`        |
| 22  | GET /incidents status filter ไม่ถูกต้อง       | GET /incidents      | ❌ ล้มเหลว | 400         | `INVALID_STATUS`                  |
| 23  | MissionAssignedEvent — สร้างภารกิจใหม่        | Inbound Event       | ✅ สำเร็จ  | —           | สร้าง mission + timeline          |
| 24  | MissionAssignedEvent — ซ้ำ (idempotent)       | Inbound Event       | ✅ สำเร็จ  | —           | skip ไม่ error                    |

---

## CORS

ระบบรองรับ Cross-Origin requests (จำเป็นสำหรับ Frontend ที่ deploy บน S3):

| Header                         | ค่า                                       |
| ------------------------------ | ----------------------------------------- |
| `Access-Control-Allow-Origin`  | `*`                                       |
| `Access-Control-Allow-Methods` | `GET,POST,OPTIONS`                        |
| `Access-Control-Allow-Headers` | `Content-Type,x-api-key,X-Rescue-Team-ID` |

ทุก response จะมี header `X-Trace-Id` (UUID) สำหรับ debug/tracing

---

## Frontend

Frontend ถูก deploy เป็น Next.js static export บน S3 Static Website Hosting:

| รายการ    | ค่า                                                  |
| --------- | ---------------------------------------------------- |
| Framework | Next.js 16 (App Router, Static Export)               |
| Hosting   | S3 Static Website                                    |
| URL       | `http://<bucket>.s3-website-us-east-1.amazonaws.com` |

Frontend ไม่ต้อง authenticate ตัวเอง — ผู้ใช้ต้องกรอก API URL, API Key, และ Team ID ในหน้า Login

---

> **คำถามหรือปัญหา:** ติดต่อ นายรัฐธรรมนูญ โคสาแสง (6609612178)

# MissionProgress Service — API Contract (Demo 2: Full Integration)

**Service Owner:** นายรัฐธรรมนูญ โคสาแสง (6609612178)

---

## 1. Base URL & Authentication

```
https://<api-id>.execute-api.us-east-1.amazonaws.com/v1
```

**ทุก request ต้องส่ง 2 headers:**

| Header             | ค่า                           | คำอธิบาย                   |
| ------------------ | ----------------------------- | -------------------------- |
| `x-api-key`        | `<api-key ที่แจกให้วัน demo>` | API Key สำหรับยืนยันสิทธิ์ |
| `X-Rescue-Team-ID` | เช่น `TEAM-ALPHA`             | รหัสทีมกู้ภัยที่เรียก API  |

---

## 2. Synchronous Endpoints

### 2.1 GET /missions/{request_id} — ดึงข้อมูลภารกิจ

**Request:**

```bash
curl -X GET \
  "{BASE_URL}/incidents/REQ-001" \
  -H "x-api-key: <key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA"
```

**Response 200 (Full Mode):**

```json
{
  "incident_id": "REQ-001",
  "mission_id": "MSN-001",
  "rescue_team_id": "TEAM-ALPHA",
  "current_status": "DISPATCHED",
  "latest_impact_level": 2,
  "started_at": "2024-12-01T08:00:00Z",
  "last_updated_at": "2024-12-01T08:00:00Z",
  "description": "น้ำท่วมหนักบริเวณถนนพหลโยธิน",
  "location": "13.7563,100.5018",
  "requestType": "FLOOD",
  "timeline": [...],
  "data_source": "full"
}
```

- `data_source: "full"` — IncidentTracking ตอบสำเร็จ
- `data_source: "partial"` — degraded mode (ไม่มี description, location, requestType)

**Errors:** `404 REQUEST_NOT_FOUND`, `400 MISSING_PARAMETER`

---

### 2.2 POST /missions/{request_id}/progress — อัปเดตสถานะ

**Request:**

```bash
curl -X POST \
  "{BASE_URL}/incidents/REQ-001/progress" \
  -H "x-api-key: <key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{
    "new_status": "EN_ROUTE",
    "note": "กำลังเดินทางไปจุดเกิดเหตุ",
    "current_location": "13.7563,100.5018",
    "new_impact_level": 3,
    "image_key": "evidence/REQ-001/TEAM-ALPHA/1718353500-photo.jpg"
  }'
```

| Field              | จำเป็น | คำอธิบาย                                             |
| ------------------ | ------ | ---------------------------------------------------- |
| `new_status`       | ✅     | สถานะใหม่ (ตาม State Machine)                        |
| `note`             | ❌     | หมายเหตุ                                             |
| `current_location` | ❌     | พิกัด GPS                                            |
| `new_impact_level` | ❌     | ระดับผลกระทบ (1-4) → publish ImpactLevelUpdated      |
| `image_key`        | ❌     | S3 key ของภาพหลักฐาน (ได้จาก presigned-url endpoint) |

**Response 200:**

```json
{
  "message": "Progress reported successfully",
  "mission_id": "MSN-001",
  "incident_id": "REQ-001",
  "old_status": "DISPATCHED",
  "new_status": "EN_ROUTE",
  "updated_at": "2025-..."
}
```

**Errors:** `400 INVALID_STATE_TRANSITION`, `400 INVALID_STATUS`, `404 REQUEST_NOT_FOUND`

---

### 2.3 POST /missions/{request_id}/presigned-url — ขอ URL อัปโหลดภาพหลักฐาน (ใหม่ Demo 2)

**Request:**

```bash
curl -X POST \
  "{BASE_URL}/incidents/REQ-001/presigned-url" \
  -H "x-api-key: <key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{
    "file_name": "flood-evidence.jpg",
    "content_type": "image/jpeg"
  }'
```

| Field          | จำเป็น | คำอธิบาย                                           |
| -------------- | ------ | -------------------------------------------------- |
| `file_name`    | ✅     | ชื่อไฟล์ภาพ                                        |
| `content_type` | ✅     | MIME type: `image/jpeg`, `image/png`, `image/webp` |

**Response 200:**

```json
{
  "upload_url": "https://s3.amazonaws.com/...",
  "image_key": "evidence/REQ-001/TEAM-ALPHA/1718353500-flood-evidence.jpg",
  "expires_in": 300,
  "message": "Upload URL generated successfully. Use PUT method to upload."
}
```

**อัปโหลดภาพ:**

```bash
curl -X PUT -T photo.jpg -H "Content-Type: image/jpeg" "{upload_url}"
```

**Errors:** `400 INVALID_CONTENT_TYPE`, `400 MISSING_PARAMETER`, `404 REQUEST_NOT_FOUND`, `500 PRESIGN_FAILED`

---

### 2.4 GET /incidents — ดึงรายการภารกิจทั้งหมดของทีม (ใหม่ Demo 2)

**Request:**

```bash
curl -X GET \
  "{BASE_URL}/incidents?status=ON_SITE" \
  -H "x-api-key: <key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA"
```

| Query Param | จำเป็น | คำอธิบาย                                                            |
| ----------- | ------ | ------------------------------------------------------------------- |
| `status`    | ❌     | กรองตามสถานะ (DISPATCHED, EN_ROUTE, ON_SITE, NEED_BACKUP, RESOLVED) |

**Response 200:**

```json
{
  "team_id": "TEAM-ALPHA",
  "total_missions": 2,
  "missions": [
    {
      "mission_id": "MSN-001",
      "incident_id": "REQ-001",
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
- ใช้ `X-Rescue-Team-ID` เป็น team_id → ป้องกันทีมอื่นดึงข้อมูลทีมอื่น

**Errors:** `400 INVALID_STATUS`

---

## 3. Outbound Async Events (EventBridge → SQS)

เมื่อ POST progress สำเร็จ ระบบ publish events ไป EventBridge bus `mission-progress-events`:

### 3.1 MissionStatusChanged

**Trigger:** ทุกครั้งที่เปลี่ยนสถานะ  
**Consumers:** IncidentTracking (SQS), Dispatch (SQS, เฉพาะ RESOLVED)

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
    "changed_at": "2025-...",
    "changed_by": "TEAM-ALPHA"
  }
}
```

### 3.2 MissionBackupRequested

**Trigger:** เมื่อเปลี่ยนสถานะเป็น `NEED_BACKUP`  
**Consumer:** Prioritization (SQS)

```json
{
  "source": "MissionProgressService",
  "detail-type": "MissionBackupRequested",
  "detail": {
    "schema_version": "1.0",
    "mission_id": "MSN-001",
    "incident_id": "REQ-001",
    "rescue_team_id": "TEAM-ALPHA",
    "requested_at": "2025-...",
    "requested_by": "TEAM-ALPHA",
    "location": "13.7563,100.5018"
  }
}
```

### 3.3 ImpactLevelUpdated

**Trigger:** เมื่อส่ง `new_impact_level` ใน request  
**Consumers:** IncidentTracking (SQS), Prioritization (SQS)

```json
{
  "source": "MissionProgressService",
  "detail-type": "ImpactLevelUpdated",
  "detail": {
    "schema_version": "1.0",
    "mission_id": "MSN-001",
    "incident_id": "REQ-001",
    "rescue_team_id": "TEAM-ALPHA",
    "old_level": 2,
    "new_level": 4,
    "updated_at": "2025-...",
    "updated_by": "TEAM-ALPHA"
  }
}
```

### Event Routing Summary

| Event                  | IncidentTracking SQS | Dispatch SQS (RESOLVED only) | Prioritization SQS | CloudWatch Logs |
| ---------------------- | -------------------- | ---------------------------- | ------------------ | --------------- |
| MissionStatusChanged   | ✅                   | ✅ (filtered)                | ❌                 | ✅              |
| MissionBackupRequested | ❌                   | ❌                           | ✅                 | ✅              |
| ImpactLevelUpdated     | ✅                   | ❌                           | ✅                 | ✅              |

---

## 4. Inbound Async Event (from Dispatch)

### MissionAssignedEvent

**Source:** `dispatch-management-service`  
**Detail-type:** `MissionAssignedEvent`  
**Handler:** `mission-assigned-handler` Lambda

เมื่อ Dispatch service มอบหมายภารกิจใหม่ → MissionProgress จะสร้าง mission record อัตโนมัติ

**Expected Payload:**

```json
{
  "source": "dispatch-management-service",
  "detail-type": "MissionAssignedEvent",
  "detail": {
    "mission_id": "MSN-001",
    "rescue_unit_id": "TEAM-ALPHA",
    "incident_id": "REQ-001",
    "assigned_at": "2025-06-14T08:45:00Z"
  }
}
```

| Field            | คำอธิบาย                       |
| ---------------- | ------------------------------ |
| `mission_id`     | รหัสภารกิจ                     |
| `rescue_unit_id` | รหัสทีมกู้ภัย → rescue_team_id |
| `incident_id`    | รหัสเหตุการณ์                  |
| `assigned_at`    | เวลาที่มอบหมาย                 |

**Idempotency:** ใช้ `attribute_not_exists(mission_id)` — ถ้ามีอยู่แล้วจะ skip (ไม่ error)

---

## 5. Dependencies

| บริการ               | วิธีเชื่อมต่อ                                | ตัวแปร Terraform                                             | Degraded Mode            |
| -------------------- | -------------------------------------------- | ------------------------------------------------------------ | ------------------------ |
| RescueRequest (Sync) | HTTP GET `/v1/rescue-requests/{id}` (Bearer) | `rescue_request_service_url`, `rescue_request_service_token` | `data_source: "partial"` |
| IncidentTracking SQS | EventBridge → SQS                            | `incident_tracking_sqs_arn`                                  | CloudWatch Logs เดิม     |
| Dispatch SQS         | EventBridge → SQS (RESOLVED only)            | `dispatch_sqs_arn`                                           | CloudWatch Logs เดิม     |
| Prioritization SQS   | EventBridge → SQS                            | `prioritization_sqs_arn`                                     | CloudWatch Logs เดิม     |

---

## 6. State Machine

```
DISPATCHED → EN_ROUTE → ON_SITE → RESOLVED
                            ↓          ↑
                       NEED_BACKUP ─────┘
                            ↑          │
                            └──────────┘
```

| From        | To                    |
| ----------- | --------------------- |
| DISPATCHED  | EN_ROUTE              |
| EN_ROUTE    | ON_SITE               |
| ON_SITE     | NEED_BACKUP, RESOLVED |
| NEED_BACKUP | ON_SITE, RESOLVED     |
| RESOLVED    | _(terminal state)_    |
