# MissionProgress Service — API Contract (Demo 1 MVP)

**Service Owner:** นายรัฐธรรมนูญ โคสาแสง (6609612178)

**วัตถุประสงค์ของบริการ:**
MissionProgress Service คือบริการสำหรับทีมกู้ภัย (Rescue Team) เพื่อรายงานความคืบหน้าของภารกิจ อัปเดตสถานะเหตุการณ์ และบันทึกรายละเอียดการปฏิบัติงานหน้างาน เพื่อให้ศูนย์สั่งการได้รับข้อมูลที่ถูกต้องและเป็นปัจจุบัน

**ขอบเขต Demo 1 (MVP):**

- เป็น MVP สำหรับสาธิตในคาบเรียน Demo 1 เท่านั้น — **ยังไม่มี Frontend** ให้ทดสอบผ่าน `curl` หรือ Postman
- Deploy บน AWS (API Gateway + Lambda + DynamoDB + EventBridge) ผ่าน Terraform
- มี 2 Endpoints หลัก:
  1. **GET** `/incidents/{incident_id}` — ดึงข้อมูลภารกิจ + Timeline (Synchronous)
  2. **POST** `/incidents/{incident_id}/progress` — อัปเดตสถานะ + publish events (Sync + Async)
- การเรียก IncidentTracking Service ของเพื่อนจะ timeout โดยตั้งใจ (Degraded Mode) → ตอบข้อมูลเฉพาะที่ตัวเองมี
- ยังไม่มี: Frontend, S3 Upload, Outbox Processor (cron retry), Cognito

---

## การพึ่งพาบริการภายนอก (Dependencies)

MissionProgress Service พึ่งพา **IncidentTracking Service** ของเพื่อนสำหรับดึงรายละเอียดเหตุการณ์ รายละเอียดจุดเชื่อมต่อและส่วนที่จำลอง (mock) มีดังนี้:

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

เมื่อเรียกสำเร็จ ฟิลด์เหล่านี้จะปรากฏใน response ของ `GET /incidents/{incident_id}` และ `data_source` จะเป็น `"full"`

### ส่วนที่ Mock ใน Demo 1

> **สำคัญ:** ใน Demo 1 นี้ **IncidentTracking Service ยังไม่พร้อมใช้งานจริง** จึงมีการ mock/จำลองการทำงานดังนี้:

| ส่วน                     | สิ่งที่จำลอง (Mock)                                                                | วิธีการจำลอง                                                                   |
| ------------------------ | ---------------------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| **URL ของบริการ**        | ตัวแปร `INCIDENT_SERVICE_URL` ถูกตั้งค่าเป็น `http://localhost:9999` (placeholder) | กำหนดใน `terraform/variables.tf` → ส่งเป็น environment variable ของ Lambda     |
| **พฤติกรรมการเรียก**     | HTTP Client จะ **timeout ภายใน 3 วินาที** เสมอ เพราะ URL ปลายทางไม่มีอยู่จริง      | ตั้ง timeout ใน `incident_client.go` → `http.Client{Timeout: 3 * time.Second}` |
| **การตอบกลับ**           | ฟังก์ชัน `GetIncidentDetail()` คืนค่า `nil` (แทน response จริง)                    | เมื่อ HTTP call ล้มเหลว → return `nil` → ระบบเข้า Degraded Mode                |
| **ข้อมูลที่หายไป**       | ฟิลด์ `description`, `location`, `incident_type` จะ**ไม่ปรากฏ**ใน response         | ฟิลด์เหล่านี้มี `omitempty` จึงไม่แสดงเมื่อเป็นค่าว่าง                         |
| **ตัวบ่งชี้ใน response** | `data_source` จะเป็น `"partial"` เสมอ (แทนที่จะเป็น `"full"`)                      | โค้ดตั้ง `dataSource = "partial"` เมื่อ `GetIncidentDetail()` คืนค่า `nil`     |

### ผลกระทบต่อ Response ของ GET /incidents/{incident_id}

| โหมด                       | `data_source` | ฟิลด์เพิ่มเติมจาก IncidentTracking                     |
| -------------------------- | ------------- | ------------------------------------------------------ |
| **Degraded Mode** (Demo 1) | `"partial"`   | ไม่มี — ขาด `description`, `location`, `incident_type` |
| **Full Mode** (Demo 2+)    | `"full"`      | มีครบ — `description`, `location`, `incident_type`     |

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

1. แก้ไขค่า `incident_service_url` ใน Terraform variables ให้ชี้ไปยัง URL จริง
2. รัน `terraform apply` ใหม่

ระบบจะเปลี่ยนจาก Degraded Mode เป็น Full Mode โดยอัตโนมัติ — `data_source` จะเป็น `"full"` และมีฟิลด์ `description`, `location`, `incident_type` ครบ

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
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/incidents/INC-001" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA"
```

---

## 3. Synchronous API: Get Mission Details

### Endpoint

```
GET /incidents/{incident_id}
```

### คำอธิบาย

ดึงข้อมูลภารกิจ (Mission Assignment) และ Timeline การปฏิบัติงานจาก DynamoDB โดยระบบจะพยายามเรียก IncidentTracking Service เพื่อดึงรายละเอียดเหตุการณ์ (description, location, incident_type) มาแสดงด้วย

### หมายเหตุ Demo 1 — Degraded Mode

> ใน Demo 1 ระบบตั้ง `INCIDENT_SERVICE_URL` เป็น placeholder URL → **จะ timeout ภายใน 3 วินาที** แล้ว fallback เป็น Degraded Mode โดยอัตโนมัติ
>
> ผลที่ได้:
>
> - `data_source` จะเป็น `"partial"` เสมอ
> - ฟิลด์ `description`, `location`, `incident_type` จะ**ไม่ปรากฏ**ใน response (เพราะข้อมูลมาจาก IncidentTracking ซึ่งเรียกไม่สำเร็จ)
> - ข้อมูลที่แสดงจะเป็นเฉพาะข้อมูลที่บริการนี้ถือครองเอง (mission + timeline)
>
> เมื่อ IncidentTracking Service พร้อมใช้งานจริง → `data_source` จะเป็น `"full"` และมีฟิลด์จาก IncidentTracking ครบ

### Request

**Path Parameters:**

| พารามิเตอร์   | ประเภท | จำเป็น | คำอธิบาย                     |
| ------------- | ------ | ------ | ---------------------------- |
| `incident_id` | UUID   | ✅     | รหัสเหตุการณ์ เช่น `INC-001` |

**Headers:** ดูหัวข้อ [2. Authentication](#2-authentication-จำเป็นสำหรับทุก-request)

**Body:** ไม่มี

### Response — Success (200 OK)

```json
{
  "incident_id": "INC-003",
  "mission_id": "MSN-003",
  "rescue_team_id": "TEAM-CHARLIE",
  "current_status": "ON_SITE",
  "latest_impact_level": 4,
  "started_at": "2024-12-01T07:30:00Z",
  "last_updated_at": "2024-12-01T10:00:00Z",
  "timeline": [
    {
      "mission_id": "MSN-003",
      "timestamp": "2024-12-01T07:30:00Z",
      "log_id": "LOG-004",
      "action_type": "STATUS_CHANGE",
      "description": "Mission dispatched to TEAM-CHARLIE",
      "performed_by": "SYSTEM"
    },
    {
      "mission_id": "MSN-003",
      "timestamp": "2024-12-01T08:30:00Z",
      "log_id": "LOG-005",
      "action_type": "STATUS_CHANGE",
      "description": "Team en route",
      "performed_by": "TEAM-CHARLIE",
      "gps_location": "13.7400,100.5200"
    },
    {
      "mission_id": "MSN-003",
      "timestamp": "2024-12-01T10:00:00Z",
      "log_id": "LOG-006",
      "action_type": "STATUS_CHANGE",
      "description": "Arrived on site, assessing situation",
      "performed_by": "TEAM-CHARLIE",
      "gps_location": "13.7380,100.5230"
    }
  ],
  "data_source": "partial"
}
```

> **สังเกต:** ไม่มีฟิลด์ `description`, `location`, `incident_type` เพราะเป็น Degraded Mode (`data_source: "partial"`)

### คำอธิบายฟิลด์ Response

| ฟิลด์                 | ประเภท   | คำอธิบาย                                                                          |
| --------------------- | -------- | --------------------------------------------------------------------------------- |
| `incident_id`         | UUID     | รหัสเหตุการณ์                                                                     |
| `mission_id`          | UUID     | รหัสภารกิจ                                                                        |
| `rescue_team_id`      | String   | รหัสทีมกู้ภัยที่รับผิดชอบ                                                         |
| `current_status`      | String   | สถานะปัจจุบันของภารกิจ                                                            |
| `latest_impact_level` | Integer  | ระดับความรุนแรงล่าสุดที่ประเมิน (1–4)                                             |
| `started_at`          | DateTime | เวลาที่เริ่มรับภารกิจ                                                             |
| `last_updated_at`     | DateTime | เวลาอัปเดตล่าสุด                                                                  |
| `description`         | String   | คำอธิบายเหตุการณ์ _(จาก IncidentTracking — ไม่แสดงใน Degraded Mode)_              |
| `location`            | String   | พิกัดเหตุการณ์ _(จาก IncidentTracking — ไม่แสดงใน Degraded Mode)_                 |
| `incident_type`       | String   | ประเภทเหตุการณ์ _(จาก IncidentTracking — ไม่แสดงใน Degraded Mode)_                |
| `timeline`            | Array    | รายการ Timeline การปฏิบัติงาน (เรียงตามเวลา)                                      |
| `data_source`         | String   | `"full"` = ข้อมูลครบ, `"partial"` = Degraded Mode (ขาดข้อมูลจาก IncidentTracking) |

**ฟิลด์ใน Timeline Entry:**

| ฟิลด์          | ประเภท   | คำอธิบาย                                   |
| -------------- | -------- | ------------------------------------------ |
| `mission_id`   | UUID     | รหัสภารกิจ                                 |
| `timestamp`    | DateTime | เวลาที่เกิดเหตุการณ์                       |
| `log_id`       | UUID     | รหัส Log                                   |
| `action_type`  | String   | ประเภทการกระทำ เช่น `STATUS_CHANGE`        |
| `description`  | String   | รายละเอียดการปฏิบัติงาน                    |
| `performed_by` | String   | ผู้ดำเนินการ (รหัสทีม หรือ `SYSTEM`)       |
| `gps_location` | String   | พิกัด GPS _(ไม่บังคับ — อาจไม่มีฟิลด์นี้)_ |

### Response — Error (404 Not Found)

เมื่อไม่พบ incident_id ในระบบ:

```json
{
  "error": "INCIDENT_NOT_FOUND",
  "code": "INCIDENT_NOT_FOUND",
  "message": "No mission found for incident: INC-99999"
}
```

### Response — Error (400 Bad Request)

เมื่อไม่ส่ง incident_id:

```json
{
  "error": "MISSING_PARAMETER",
  "code": "MISSING_PARAMETER",
  "message": "incident_id is required"
}
```

### ตัวอย่างการทดสอบ — กรณีสำเร็จและล้มเหลว

#### ✅ สำเร็จ — ดึงข้อมูลภารกิจที่มีอยู่ในระบบ

```bash
curl -X GET \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/incidents/INC-003" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-CHARLIE"
```

ผลลัพธ์ที่คาดหวัง (HTTP 200):

```json
{
  "incident_id": "INC-003",
  "mission_id": "MSN-003",
  "rescue_team_id": "TEAM-CHARLIE",
  "current_status": "ON_SITE",
  "latest_impact_level": 4,
  "started_at": "2024-12-01T07:30:00Z",
  "last_updated_at": "2024-12-01T10:00:00Z",
  "timeline": [...],
  "data_source": "partial"
}
```

#### ❌ ล้มเหลว — ดึงข้อมูล incident_id ที่ไม่มีในระบบ

```bash
curl -X GET \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/incidents/INC-99999" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA"
```

ผลลัพธ์ที่คาดหวัง (HTTP 404):

```json
{
  "error": "INCIDENT_NOT_FOUND",
  "code": "INCIDENT_NOT_FOUND",
  "message": "No mission found for incident: INC-99999"
}
```

#### ❌ ล้มเหลว — ไม่ส่ง API Key

```bash
curl -X GET \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/incidents/INC-001"
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
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/incidents/INC-001" \
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
POST /incidents/{incident_id}/progress
```

### คำอธิบาย

ใช้สำหรับทีมกู้ภัยเพื่อ **อัปเดตสถานะภารกิจ** และ **บันทึก Timeline** การปฏิบัติงาน

**การทำงานภายใน:**

1. **(Sync)** ตรวจสอบความถูกต้องของสถานะ → อัปเดต DynamoDB → เพิ่ม Timeline entry → ตอบกลับ 200
2. **(Async)** Publish events ไป Amazon EventBridge เพื่อแจ้ง Service อื่น ๆ (ดูหัวข้อ 5)
3. **(Fallback)** หากส่ง event ไป EventBridge ไม่สำเร็จ → บันทึกลง Outbox table แทน (ไม่ fail request หลัก)

### Request

**Path Parameters:**

| พารามิเตอร์   | ประเภท | จำเป็น | คำอธิบาย                     |
| ------------- | ------ | ------ | ---------------------------- |
| `incident_id` | UUID   | ✅     | รหัสเหตุการณ์ เช่น `INC-001` |

**Headers:** ดูหัวข้อ [2. Authentication](#2-authentication-จำเป็นสำหรับทุก-request)

**Body (JSON):**

```json
{
  "new_status": "EN_ROUTE",
  "note": "กำลังเดินทางไปจุดเกิดเหตุ",
  "current_location": "13.7563,100.5018",
  "new_impact_level": 3
}
```

| ฟิลด์              | ประเภท  | จำเป็น | คำอธิบาย                                                             |
| ------------------ | ------- | ------ | -------------------------------------------------------------------- |
| `new_status`       | String  | ✅     | สถานะใหม่ (ดูตาราง State Machine ด้านล่าง)                           |
| `note`             | String  | ❌     | หมายเหตุ / รายละเอียดการปฏิบัติงาน                                   |
| `current_location` | String  | ❌     | พิกัด GPS ปัจจุบัน เช่น `"13.7563,100.5018"`                         |
| `new_impact_level` | Integer | ❌     | ระดับความรุนแรงใหม่ที่ประเมินจากหน้างาน (1–4) ส่งเมื่อต้องการปรับค่า |

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
  "incident_id": "INC-001",
  "old_status": "DISPATCHED",
  "new_status": "EN_ROUTE",
  "updated_at": "2024-12-01T12:00:00Z"
}
```

| ฟิลด์         | ประเภท   | คำอธิบาย            |
| ------------- | -------- | ------------------- |
| `message`     | String   | ข้อความยืนยันสำเร็จ |
| `mission_id`  | UUID     | รหัสภารกิจที่อัปเดต |
| `incident_id` | UUID     | รหัสเหตุการณ์       |
| `old_status`  | String   | สถานะเดิมก่อนอัปเดต |
| `new_status`  | String   | สถานะใหม่หลังอัปเดต |
| `updated_at`  | DateTime | เวลาที่อัปเดตสำเร็จ |

### Response — Error (400 Bad Request): Invalid State Transition

เมื่อเปลี่ยนสถานะไม่ตรงตามกฎ:

```json
{
  "error": "INVALID_STATE_TRANSITION",
  "code": "INVALID_STATE_TRANSITION",
  "message": "Cannot transition from DISPATCHED to RESOLVED"
}
```

### Response — Error (400 Bad Request): Missing / Invalid Fields

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
  "message": "Invalid status: UNKNOWN_STATUS"
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
  "error": "INCIDENT_NOT_FOUND",
  "code": "INCIDENT_NOT_FOUND",
  "message": "No mission found for incident: INC-99999"
}
```

### ตัวอย่างการทดสอบ — กรณีสำเร็จและล้มเหลว

#### ✅ สำเร็จ — เปลี่ยนสถานะ DISPATCHED → EN_ROUTE

```bash
curl -X POST \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/incidents/INC-001/progress" \
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
  "incident_id": "INC-001",
  "old_status": "DISPATCHED",
  "new_status": "EN_ROUTE",
  "updated_at": "2025-..."
}
```

#### ✅ สำเร็จ — เปลี่ยนสถานะเป็น NEED_BACKUP (จะ trigger MissionBackupRequested event)

```bash
curl -X POST \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/incidents/INC-003/progress" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-CHARLIE" \
  -H "Content-Type: application/json" \
  -d '{
    "new_status": "NEED_BACKUP",
    "note": "น้ำท่วมสูงกว่าที่คาด ต้องการกำลังเสริม",
    "current_location": "13.7380,100.5230",
    "new_impact_level": 4
  }'
```

ผลลัพธ์ที่คาดหวัง (HTTP 200):

```json
{
  "message": "Progress reported successfully",
  "mission_id": "MSN-003",
  "incident_id": "INC-003",
  "old_status": "ON_SITE",
  "new_status": "NEED_BACKUP",
  "updated_at": "2025-..."
}
```

#### ❌ ล้มเหลว — Transition สถานะไม่ถูกต้อง (EN_ROUTE → RESOLVED)

```bash
curl -X POST \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/incidents/INC-002/progress" \
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
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/incidents/INC-001/progress" \
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
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/incidents/INC-001/progress" \
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
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/incidents/INC-99999/progress" \
  -H "x-api-key: <your-api-key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"new_status": "EN_ROUTE"}'
```

ผลลัพธ์ที่คาดหวัง (HTTP 404):

```json
{
  "error": "INCIDENT_NOT_FOUND",
  "code": "INCIDENT_NOT_FOUND",
  "message": "No mission found for incident: INC-99999"
}
```

#### ❌ ล้มเหลว — ส่ง JSON body ไม่ถูกต้อง

```bash
curl -X POST \
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/incidents/INC-001/progress" \
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
  "https://<api-id>.execute-api.us-east-1.amazonaws.com/v1/incidents/INC-005/progress" \
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

## 5. Asynchronous Events Contract (EventBridge)

เมื่อ POST `/incidents/{incident_id}/progress` สำเร็จ ระบบจะ publish events ไปยัง Amazon EventBridge **แบบ asynchronous** (ไม่กระทบ response ที่ตอบกลับผู้เรียก)

### ข้อมูล Event Bus

| รายการ    | ค่า                       |
| --------- | ------------------------- |
| Event Bus | `mission-progress-events` |
| Source    | `MissionProgressService`  |
| Region    | `us-east-1`               |

### หมายเหตุ Demo 1

- Events ทั้งหมดจะถูกส่งไปยัง CloudWatch Log Groups เพื่อให้ตรวจสอบได้ใน AWS Console
- หากทีมเพื่อนต้องการ subscribe events เหล่านี้ สามารถสร้าง EventBridge Rule ที่ match `source` = `MissionProgressService` และ `detail-type` ที่ต้องการ บน event bus `mission-progress-events`
- หากการ publish ไม่สำเร็จ → ระบบจะบันทึก event ลง Outbox table ใน DynamoDB แทน (Outbox Pattern) — ตัว processor สำหรับ retry จะทำใน Demo 2+

---

### 5.1 MissionStatusChanged

**เงื่อนไขการ publish:** ทุกครั้งที่สถานะภารกิจเปลี่ยน (publish ทุก POST ที่สำเร็จ)

**Detail Type:** `MissionStatusChanged`

**Payload:**

```
{
  "mission_id": "MSN-001",
  "incident_id": "INC-001",
  "rescue_team_id": "TEAM-ALPHA",
  "old_status": "DISPATCHED",
  "new_status": "EN_ROUTE",
  "changed_at": "2024-12-01T12:00:00Z",
  "changed_by": "TEAM-ALPHA"
}
```

| ฟิลด์            | ประเภท   | คำอธิบาย                           |
| ---------------- | -------- | ---------------------------------- |
| `mission_id`     | UUID     | รหัสภารกิจ                         |
| `incident_id`    | UUID     | รหัสเหตุการณ์                      |
| `rescue_team_id` | String   | รหัสทีมกู้ภัย                      |
| `old_status`     | String   | สถานะก่อนเปลี่ยน                   |
| `new_status`     | String   | สถานะใหม่                          |
| `changed_at`     | DateTime | เวลาที่เปลี่ยนสถานะ                |
| `changed_by`     | String   | ผู้ดำเนินการเปลี่ยนสถานะ (รหัสทีม) |

**ตัวอย่าง EventBridge Envelope ที่จะปรากฏใน CloudWatch Logs:**

```json
{
  "source": "MissionProgressService",
  "detail-type": "MissionStatusChanged",
  "detail": {
    "mission_id": "MSN-001",
    "incident_id": "INC-001",
    "rescue_team_id": "TEAM-ALPHA",
    "old_status": "DISPATCHED",
    "new_status": "EN_ROUTE",
    "changed_at": "2024-12-01T12:00:00Z",
    "changed_by": "TEAM-ALPHA"
  }
}
```

---

### 5.2 MissionBackupRequested

**เงื่อนไขการ publish:** เฉพาะเมื่อ `new_status` = `NEED_BACKUP` เท่านั้น

**Detail Type:** `MissionBackupRequested`

**Payload:**

```json
{
  "mission_id": "MSN-003",
  "incident_id": "INC-003",
  "rescue_team_id": "TEAM-CHARLIE",
  "requested_at": "2024-12-01T10:30:00Z",
  "requested_by": "TEAM-CHARLIE",
  "location": "13.7380,100.5230"
}
```

| ฟิลด์            | ประเภท   | คำอธิบาย                                   |
| ---------------- | -------- | ------------------------------------------ |
| `mission_id`     | UUID     | รหัสภารกิจ                                 |
| `incident_id`    | UUID     | รหัสเหตุการณ์                              |
| `rescue_team_id` | String   | รหัสทีมกู้ภัยที่ขอกำลังเสริม               |
| `requested_at`   | DateTime | เวลาที่ขอ backup                           |
| `requested_by`   | String   | ผู้ร้องขอ (รหัสทีม)                        |
| `location`       | String   | พิกัด GPS _(ไม่บังคับ — อาจไม่มีฟิลด์นี้)_ |

> **สำหรับ Rescue Prioritization Service:** event นี้มีไว้เพื่อแจ้งว่าทีมกู้ภัยหน้างานต้องการกำลังเสริม สามารถนำไปคำนวณ Priority Score ใหม่ได้

---

### 5.3 ImpactLevelUpdated

**เงื่อนไขการ publish:** เฉพาะเมื่อ request body มีฟิลด์ `new_impact_level`

**Detail Type:** `ImpactLevelUpdated`

**Payload:**

```json
{
  "mission_id": "MSN-003",
  "incident_id": "INC-003",
  "rescue_team_id": "TEAM-CHARLIE",
  "old_level": 3,
  "new_level": 4,
  "updated_at": "2024-12-01T10:30:00Z",
  "updated_by": "TEAM-CHARLIE"
}
```

| ฟิลด์            | ประเภท   | คำอธิบาย                              |
| ---------------- | -------- | ------------------------------------- |
| `mission_id`     | UUID     | รหัสภารกิจ                            |
| `incident_id`    | UUID     | รหัสเหตุการณ์                         |
| `rescue_team_id` | String   | รหัสทีมกู้ภัยที่ประเมินความรุนแรงใหม่ |
| `old_level`      | Integer  | ระดับความรุนแรงเดิม                   |
| `new_level`      | Integer  | ระดับความรุนแรงใหม่ (1–4)             |
| `updated_at`     | DateTime | เวลาที่อัปเดต                         |
| `updated_by`     | String   | ผู้ดำเนินการ (รหัสทีม)                |

> **สำหรับ Rescue Prioritization Service / IncidentTracking Service:** event นี้แจ้งว่าระดับความรุนแรงของเหตุการณ์ถูกปรับโดยทีมหน้างาน สามารถนำ `new_level` ไปอัปเดตฐานข้อมูลหรือคำนวณลำดับความสำคัญใหม่ได้

---

## สรุปภาพรวม Endpoints

| Method | Path                                | คำอธิบาย                     | ประเภท       |
| ------ | ----------------------------------- | ---------------------------- | ------------ |
| GET    | `/incidents/{incident_id}`          | ดึงข้อมูลภารกิจ + Timeline   | Synchronous  |
| POST   | `/incidents/{incident_id}/progress` | อัปเดตสถานะ + publish events | Sync + Async |

## สรุป Error Codes

| HTTP Status | Error Code                 | สาเหตุ                                 |
| ----------- | -------------------------- | -------------------------------------- |
| 400         | `MISSING_PARAMETER`        | ไม่ส่ง incident_id หรือ new_status     |
| 400         | `INVALID_BODY`             | JSON body ไม่ถูกต้อง                   |
| 400         | `INVALID_STATUS`           | new_status ไม่ใช่สถานะที่กำหนด         |
| 400         | `INVALID_STATE_TRANSITION` | เปลี่ยนสถานะไม่ตรงตามกฎ State Machine  |
| 403         | —                          | ไม่ส่ง x-api-key หรือ X-Rescue-Team-ID |
| 404         | `INCIDENT_NOT_FOUND`       | ไม่พบภารกิจสำหรับ incident_id ที่ระบุ  |
| 500         | `INTERNAL_ERROR`           | เกิดข้อผิดพลาดภายในระบบ                |

---

## สรุปผลการทดสอบ

| #   | กรณีทดสอบ                                     | ประเภท     | HTTP Status | ผลที่คาดหวัง                     |
| --- | --------------------------------------------- | ---------- | ----------- | -------------------------------- |
| 1   | ส่ง API Key + Team ID ถูกต้อง                 | ✅ สำเร็จ  | 200         | ได้รับข้อมูลภารกิจ               |
| 2   | ไม่ส่ง API Key                                | ❌ ล้มเหลว | 403         | `Forbidden`                      |
| 3   | ไม่ส่ง X-Rescue-Team-ID                       | ❌ ล้มเหลว | 403         | `User is not authorized...`      |
| 4   | GET ภารกิจที่มีอยู่ (INC-003)                 | ✅ สำเร็จ  | 200         | ข้อมูลภารกิจ + Timeline          |
| 5   | GET ภารกิจที่ไม่มี (INC-99999)                | ❌ ล้มเหลว | 404         | `INCIDENT_NOT_FOUND`             |
| 6   | POST transition ถูกต้อง (DISPATCHED→EN_ROUTE) | ✅ สำเร็จ  | 200         | `Progress reported successfully` |
| 7   | POST NEED_BACKUP + ImpactLevel                | ✅ สำเร็จ  | 200         | `Progress reported successfully` |
| 8   | POST transition ผิดกฎ (EN_ROUTE→RESOLVED)     | ❌ ล้มเหลว | 400         | `INVALID_STATE_TRANSITION`       |
| 9   | POST ไม่ส่ง new_status                        | ❌ ล้มเหลว | 400         | `MISSING_PARAMETER`              |
| 10  | POST ส่ง status ที่ไม่มีในระบบ                | ❌ ล้มเหลว | 400         | `INVALID_STATUS`                 |
| 11  | POST incident_id ไม่มีในระบบ                  | ❌ ล้มเหลว | 404         | `INCIDENT_NOT_FOUND`             |
| 12  | POST ส่ง JSON ไม่ถูกต้อง                      | ❌ ล้มเหลว | 400         | `INVALID_BODY`                   |
| 13  | POST อัปเดตภารกิจที่ RESOLVED แล้ว            | ❌ ล้มเหลว | 400         | `INVALID_STATE_TRANSITION`       |

---

## CORS

ระบบรองรับ Cross-Origin requests:

| Header                         | ค่า                                       |
| ------------------------------ | ----------------------------------------- |
| `Access-Control-Allow-Origin`  | `*`                                       |
| `Access-Control-Allow-Methods` | `GET,POST,OPTIONS`                        |
| `Access-Control-Allow-Headers` | `Content-Type,x-api-key,X-Rescue-Team-ID` |

---

> **คำถามหรือปัญหา:** ติดต่อ นายรัฐธรรมนูญ โคสาแสง (6609612178)
