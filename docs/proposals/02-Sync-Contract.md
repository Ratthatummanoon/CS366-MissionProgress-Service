# **Synchronous Function Contract**

> **Base URL:** `https://api.disaster-management.net/mission-progress/v1`

---

## Response Header Convention (ใช้กับทุก API)

ทุก Response จาก MissionProgress Service จะมี Header ดังนี้:

| Header                          | คำอธิบาย                                                                 |
| :------------------------------ | :----------------------------------------------------------------------- |
| `Content-Type`                  | `application/json`                                                       |
| `X-Trace-Id`                    | Unique Trace ID (UUID v4) สำหรับติดตาม request — ใช้อ้างอิงเมื่อแจ้งปัญหา |
| `Access-Control-Allow-Origin`   | `*`                                                                      |
| `Access-Control-Allow-Methods`  | `GET,POST,OPTIONS`                                                       |
| `Access-Control-Allow-Headers`  | `x-api-key,X-Rescue-Team-ID,Content-Type`                               |

> ⚠️ **ยกเว้น** `403 Forbidden` ที่ return จาก API Gateway Authorizer โดยตรง — จะไม่มี `X-Trace-Id` เนื่องจากไม่ผ่าน application layer

---

## Error Response Convention (ใช้กับทุก API)

ทุก Error Response (ยกเว้น 403) จะมีโครงสร้างเดียวกัน:

```json
{
  "error": "ERROR_CODE",
  "code": "ERROR_CODE",
  "message": "Human-readable error description",
  "traceId": "550e8400-e29b-41d4-a716-446655440000"
}
```

| Field     | Type   | คำอธิบาย                                                  |
| :-------- | :----- | :-------------------------------------------------------- |
| `error`   | String | Error code สำหรับ programmatic handling                   |
| `code`    | String | Error code (เหมือน `error`)                               |
| `message` | String | ข้อความอธิบายปัญหาที่อ่านเข้าใจง่าย                       |
| `traceId` | String | Trace ID ตัวเดียวกับ `X-Trace-Id` ใน header — ใช้แจ้ง debug |

---

---

# API Contract #1: Report Progress & Update Status

## ข้อมูลทั่วไป

| รายการ     | ค่า                                 |
| :--------- | :---------------------------------- |
| **Name**   | `reportMissionProgress`             |
| **Method** | `POST`                              |
| **Path**   | `/incidents/{incident_id}/progress` |
| **Type**   | Synchronous                         |
| **Lambda** | `report-progress` (Go)              |

## คำอธิบาย

ใช้สำหรับทีมกู้ภัยเพื่อ "บันทึกการปฏิบัติงาน" (Create new Timeline entry) และ "อัปเดตสถานะ" (Update mission status) ของภารกิจในครั้งเดียว เช่น แจ้งว่าถึงจุดเกิดเหตุแล้ว, ขอกำลังเสริม, หรือแจ้งปิดงาน พร้อมทั้ง Publish Events ไป EventBridge เพื่อแจ้ง Service อื่นๆ

---

## Request

### Path Parameters

| Parameter     | Type   | Required | คำอธิบาย                                            |
| :------------ | :----- | :------: | :-------------------------------------------------- |
| `incident_id` | String |    ✅    | รหัสเหตุการณ์ที่กำลังปฏิบัติภารกิจ (เช่น `INC-001`) |

### Headers

| Header             | Required | คำอธิบาย                                      |
| :----------------- | :------: | :-------------------------------------------- |
| `Content-Type`     |    ✅    | `application/json`                            |
| `x-api-key`        |    ✅    | API Key สำหรับ Authentication                 |
| `X-Rescue-Team-ID` |    ✅    | รหัสทีมกู้ภัยที่ส่งข้อมูล (เช่น `TEAM-ALPHA`) |

### Body

```json
{
  "new_status": "ON_SITE",
  "note": "ถึงจุดเกิดเหตุแล้ว กำลังเริ่มปฐมพยาบาลเบื้องต้น",
  "new_impact_level": "HIGH"
}
```

| Field              | Type          | Required | คำอธิบาย                                                                     |
| :----------------- | :------------ | :------: | :--------------------------------------------------------------------------- |
| `new_status`       | String (Enum) |    ✅    | สถานะใหม่: `EN_ROUTE`, `ON_SITE`, `NEED_BACKUP`, `RESOLVED`                  |
| `note`             | String        |    ❌    | รายละเอียดการปฏิบัติงาน / หมายเหตุ                                           |
| `new_impact_level` | String        |    ❌    | ระดับความรุนแรงใหม่จากการประเมินหน้างาน → publish `ImpactLevelUpdated` event |

---

## Response

### Success `200 OK`

**Response Headers:**

```
Content-Type: application/json
X-Trace-Id: 550e8400-e29b-41d4-a716-446655440000
```

**Response Body:**

```json
{
  "message": "Progress reported successfully",
  "mission_id": "MSN-001",
  "incident_id": "INC-001",
  "old_status": "EN_ROUTE",
  "new_status": "ON_SITE",
  "updated_at": "2025-06-14T09:32:15Z"
}
```

### Error `400 Bad Request` — State Transition ผิดกฎ

```json
{
  "error": "INVALID_STATE_TRANSITION",
  "code": "INVALID_STATE_TRANSITION",
  "message": "Cannot transition from EN_ROUTE to RESOLVED",
  "traceId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
}
```

### Error `400 Bad Request` — ไม่ส่ง new_status

```json
{
  "error": "MISSING_PARAMETER",
  "code": "MISSING_PARAMETER",
  "message": "new_status is required",
  "traceId": "b2c3d4e5-f6a7-8901-bcde-f12345678901"
}
```

### Error `400 Bad Request` — สถานะไม่รู้จัก

```json
{
  "error": "INVALID_STATUS",
  "code": "INVALID_STATUS",
  "message": "Invalid status value: UNKNOWN_STATUS",
  "traceId": "c3d4e5f6-a7b8-9012-cdef-123456789012"
}
```

### Error `400 Bad Request` — JSON body ไม่ถูกต้อง

```json
{
  "error": "INVALID_BODY",
  "code": "INVALID_BODY",
  "message": "Invalid request body",
  "traceId": "d4e5f6a7-b8c9-0123-defa-234567890123"
}
```

### Error `404 Not Found` — ไม่พบ incident_id

```json
{
  "error": "INCIDENT_NOT_FOUND",
  "code": "INCIDENT_NOT_FOUND",
  "message": "No mission found for incident: INC-99999",
  "traceId": "e5f6a7b8-c9d0-1234-efab-345678901234"
}
```

### Error `403 Forbidden` — Auth ไม่ผ่าน

> ⚠️ Response นี้มาจาก API Gateway โดยตรง — ไม่มี `X-Trace-Id` header และไม่มี `traceId` ใน body

```json
{
  "message": "Forbidden"
}
```

---

## State Machine (Validation Rules)

```
DISPATCHED ──→ EN_ROUTE ──→ ON_SITE ──→ RESOLVED
                                │
                                ▼
                          NEED_BACKUP ──→ RESOLVED
                                │
                                └──→ ON_SITE
```

| จาก           | ไปได้                     | ไปไม่ได้                     |
| :------------ | :------------------------ | :--------------------------- |
| `DISPATCHED`  | `EN_ROUTE`                | ❌ ข้ามขั้นตอนไม่ได้         |
| `EN_ROUTE`    | `ON_SITE`                 | ❌ `RESOLVED`, `NEED_BACKUP` |
| `ON_SITE`     | `NEED_BACKUP`, `RESOLVED` | ❌ `EN_ROUTE`, `DISPATCHED`  |
| `NEED_BACKUP` | `ON_SITE`, `RESOLVED`     | ❌ `EN_ROUTE`, `DISPATCHED`  |
| `RESOLVED`    | ❌ (Final State)          | ❌ ทุกสถานะ                  |

---

## Dependency / Reliability

| ประเภท                    | รายละเอียด                                                                                                                                                |
| :------------------------ | :-------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Internal Data (Write)** | อัปเดต MissionAssignment table (`current_status`, `last_updated_at`) + Insert entry ใน MissionTimeline table                                              |
| **EventBridge (Async)**   | Publish สูงสุด 3 Events: `MissionStatusChanged` (ทุกครั้ง), `MissionBackupRequested` (ถ้า `NEED_BACKUP`), `ImpactLevelUpdated` (ถ้ามี `new_impact_level`) |
| **Outbox Fallback**       | หาก EventBridge Publish ล้มเหลว → บันทึกลง EventOutbox table → POST request ไม่ fail (ข้อมูลสถานะ safe ใน DynamoDB แล้ว)                                  |
| **Validation**            | ตรวจสอบ State Transition ตาม Business Rules ก่อนบันทึก — reject ถ้าไม่ถูกกฎ                                                                               |

---

---

# API Contract #2: View Incident Mission Details

> _(Action B: ดูข้อมูลเหตุการณ์)_

## ข้อมูลทั่วไป

| รายการ     | ค่า                        |
| :--------- | :------------------------- |
| **Name**   | `getMissionDetails`        |
| **Method** | `GET`                      |
| **Path**   | `/incidents/{incident_id}` |
| **Type**   | Synchronous                |
| **Lambda** | `get-mission` (Go)         |
| **Demo 1** | ✅ Implemented             |

## คำอธิบาย

ใช้ดึงข้อมูลรายละเอียดของภารกิจ รวมถึง "ประวัติการทำงานทั้งหมด (Timeline)" และ "ข้อมูลเหตุการณ์จาก IncidentTracking Service" (Degraded Mode เมื่อเรียกไม่สำเร็จ) เพื่อให้ทีมกู้ภัยดูย้อนหลังหรือ Dispatcher ตรวจสอบความคืบหน้า

---

## Request

### Path Parameters

| Parameter     | Type   | Required | คำอธิบาย                                   |
| :------------ | :----- | :------: | :----------------------------------------- |
| `incident_id` | String |    ✅    | รหัสเหตุการณ์ที่ต้องการดู (เช่น `INC-001`) |

### Headers

| Header             | Required | คำอธิบาย                      |
| :----------------- | :------: | :---------------------------- |
| `x-api-key`        |    ✅    | API Key สำหรับ Authentication |
| `X-Rescue-Team-ID` |    ✅    | รหัสทีมกู้ภัย                 |

### Body

> ไม่มี

---

## Response

### Success `200 OK` — Degraded Mode (Demo 1 เสมอ)

**Response Headers:**

```
Content-Type: application/json
X-Trace-Id: f6a7b8c9-d0e1-2345-fab0-456789012345
```

**Response Body:**

```json
{
  "incident_id": "INC-001",
  "mission_id": "MSN-001",
  "rescue_team_id": "TEAM-ALPHA",
  "current_status": "ON_SITE",
  "latest_impact_level": 3,
  "started_at": "2024-12-01T08:00:00Z",
  "last_updated_at": "2025-06-14T09:32:15Z",
  "timeline": [
    {
      "mission_id": "MSN-001",
      "timestamp": "2024-12-01T08:00:00Z",
      "log_id": "LOG-001",
      "action_type": "STATUS_CHANGE",
      "description": "Mission dispatched to TEAM-ALPHA",
      "performed_by": "SYSTEM"
    },
    {
      "mission_id": "MSN-001",
      "timestamp": "2025-06-14T09:32:15Z",
      "log_id": "LOG-002",
      "action_type": "STATUS_CHANGE",
      "description": "ถึงจุดเกิดเหตุแล้ว",
      "old_status": "EN_ROUTE",
      "new_status": "ON_SITE",
      "performed_by": "TEAM-ALPHA"
    }
  ],
  "data_source": "partial"
}
```

### Success `200 OK` — Full Mode (Demo 2+ เมื่อ IncidentTracking พร้อม)

```json
{
  "incident_id": "INC-001",
  "mission_id": "MSN-001",
  "rescue_team_id": "TEAM-ALPHA",
  "current_status": "ON_SITE",
  "latest_impact_level": 3,
  "started_at": "2024-12-01T08:00:00Z",
  "last_updated_at": "2025-06-14T09:32:15Z",
  "description": "น้ำท่วมหนักบริเวณถนนพหลโยธิน",
  "location": "13.7563,100.5018",
  "incident_type": "FLOOD",
  "timeline": [ "..." ],
  "data_source": "full"
}
```

### Field Availability by Mode

| Field                                         | มี Degraded? | มี Full? | Source                   |
| :-------------------------------------------- | :----------: | :------: | :----------------------- |
| `incident_id`, `mission_id`, `rescue_team_id` |      ✅      |    ✅    | MissionAssignment table  |
| `current_status`, `latest_impact_level`       |      ✅      |    ✅    | MissionAssignment table  |
| `started_at`, `last_updated_at`               |      ✅      |    ✅    | MissionAssignment table  |
| `timeline`                                    |      ✅      |    ✅    | MissionTimeline table    |
| `description`, `location`, `incident_type`    |      ❌      |    ✅    | IncidentTracking Service |
| `data_source`                                 | `"partial"`  | `"full"` | —                        |

### Error `404 Not Found`

```json
{
  "error": "INCIDENT_NOT_FOUND",
  "code": "INCIDENT_NOT_FOUND",
  "message": "No mission found for incident: INC-99999",
  "traceId": "11112222-3333-4444-5555-666677778888"
}
```

### Error `403 Forbidden`

> ⚠️ Response นี้มาจาก API Gateway โดยตรง — ไม่มี `X-Trace-Id` header และไม่มี `traceId` ใน body

```json
{
  "message": "Forbidden"
}
```

---

## Dependency / Reliability

| ประเภท                   | รายละเอียด                                                                                            |
| :----------------------- | :---------------------------------------------------------------------------------------------------- |
| **Internal Data (Read)** | อ่าน MissionAssignment table (state) + MissionTimeline table (timeline เรียงตาม timestamp)            |
| **External Dependency**  | HTTP GET → IncidentTracking Service เพื่อดึง description, location, incident_type (timeout: 3 วินาที) |
| **Degraded Mode**        | IncidentTracking ล่ม/timeout → ส่งข้อมูลเฉพาะที่มี → `data_source: "partial"`                         |

---

---

# API Contract #3: Request Presigned URL for Evidence Upload

## ข้อมูลทั่วไป

| รายการ     | ค่า                                      |
| :--------- | :--------------------------------------- |
| **Name**   | `requestEvidenceUploadURL`               |
| **Method** | `POST`                                   |
| **Path**   | `/incidents/{incident_id}/presigned-url` |
| **Type**   | Synchronous                              |
| **Lambda** | `presigned-url` (Go)                     |

## คำอธิบาย

ใช้สำหรับทีมกู้ภัยเพื่อ "ขอ Presigned URL" จากระบบ ก่อนอัปโหลดรูปภาพหลักฐานจากหน้างาน (Evidence Image) ตรงไปยัง Amazon S3 โดยไม่ต้องส่งไฟล์ผ่าน Lambda (ลดภาระ Server + หลีกเลี่ยง Lambda payload limit 6MB)

---

## Flow การใช้งาน

```
1. Frontend เรียก POST /incidents/{id}/presigned-url → ส่ง file_name + content_type
2. Lambda สร้าง Presigned URL (PUT) อายุ 5 นาที → ส่งกลับ upload_url + image_key
3. Frontend ใช้ upload_url อัปโหลดไฟล์ตรงไป S3 (HTTP PUT)
4. Frontend ส่ง image_key ไปพร้อม POST /incidents/{id}/progress (เพื่อเชื่อม Evidence กับ Timeline entry)
```

---

## Request

### Path Parameters

| Parameter     | Type   | Required | คำอธิบาย                                                  |
| :------------ | :----- | :------: | :-------------------------------------------------------- |
| `incident_id` | String |    ✅    | รหัสเหตุการณ์ที่ต้องการอัปโหลดรูปหลักฐาน (เช่น `INC-001`) |

### Headers

| Header             | Required | คำอธิบาย                                      |
| :----------------- | :------: | :-------------------------------------------- |
| `Content-Type`     |    ✅    | `application/json`                            |
| `x-api-key`        |    ✅    | API Key สำหรับ Authentication                 |
| `X-Rescue-Team-ID` |    ✅    | รหัสทีมกู้ภัยที่ส่งข้อมูล (เช่น `TEAM-ALPHA`) |

### Body

```json
{
  "file_name": "flood-evidence-001.jpg",
  "content_type": "image/jpeg"
}
```

| Field          | Type   | Required | คำอธิบาย                                           |
| :------------- | :----- | :------: | :------------------------------------------------- |
| `file_name`    | String |    ✅    | ชื่อไฟล์ที่ต้องการอัปโหลด (เช่น `photo-001.jpg`)   |
| `content_type` | String |    ✅    | MIME type ของไฟล์ (เช่น `image/jpeg`, `image/png`) |

---

## Response

### Success `200 OK`

**Response Headers:**

```
Content-Type: application/json
X-Trace-Id: aabb1122-ccdd-3344-eeff-556677889900
```

**Response Body:**

```json
{
  "upload_url": "https://s3.amazonaws.com/mission-evidence-bucket/evidence/INC-001/TEAM-ALPHA/1718352735-flood-evidence-001.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&...",
  "image_key": "evidence/INC-001/TEAM-ALPHA/1718352735-flood-evidence-001.jpg",
  "expires_in": 300,
  "message": "Presigned URL generated successfully"
}
```

| Field        | Type    | คำอธิบาย                                                              |
| :----------- | :------ | :-------------------------------------------------------------------- |
| `upload_url` | String  | Presigned URL สำหรับ HTTP PUT อัปโหลดไฟล์ตรงไป S3                     |
| `image_key`  | String  | S3 Key ของไฟล์ — ใช้แนบไปกับ `POST /progress` เพื่อเชื่อมกับ Timeline |
| `expires_in` | Integer | อายุของ URL เป็นวินาที (300 = 5 นาที)                                 |
| `message`    | String  | ข้อความยืนยัน                                                         |

### S3 Key Format

```
evidence/{incident_id}/{team_id}/{unix_timestamp}-{file_name}
```

> ตัวอย่าง: `evidence/INC-001/TEAM-ALPHA/1718352735-flood-evidence-001.jpg`
>
> ใช้ `{unix_timestamp}` เพื่อป้องกันชื่อไฟล์ซ้ำ

### Error `400 Bad Request` — ไม่ส่ง field ที่จำเป็น

```json
{
  "error": "MISSING_PARAMETER",
  "code": "MISSING_PARAMETER",
  "message": "file_name and content_type are required",
  "traceId": "11aa22bb-33cc-44dd-55ee-66ff77008811"
}
```

### Error `400 Bad Request` — content_type ไม่รองรับ

```json
{
  "error": "INVALID_CONTENT_TYPE",
  "code": "INVALID_CONTENT_TYPE",
  "message": "Supported content types: image/jpeg, image/png, image/webp",
  "traceId": "22bb33cc-44dd-55ee-66ff-778800991122"
}
```

### Error `404 Not Found` — ไม่พบ incident_id

```json
{
  "error": "INCIDENT_NOT_FOUND",
  "code": "INCIDENT_NOT_FOUND",
  "message": "No mission found for incident: INC-99999",
  "traceId": "33cc44dd-55ee-66ff-7788-009911223344"
}
```

### Error `403 Forbidden`

> ⚠️ Response นี้มาจาก API Gateway โดยตรง — ไม่มี `X-Trace-Id` header และไม่มี `traceId` ใน body

```json
{
  "message": "Forbidden"
}
```

---

## Frontend Upload Flow (หลังได้ Presigned URL)

```bash
# Frontend ใช้ upload_url ที่ได้มา อัปโหลดไฟล์ตรงไป S3
curl -X PUT \
  -H "Content-Type: image/jpeg" \
  --data-binary @flood-evidence-001.jpg \
  "https://s3.amazonaws.com/mission-evidence-bucket/evidence/INC-001/...?X-Amz-Algorithm=..."

# HTTP 200 = อัปโหลดสำเร็จ

# จากนั้นส่ง image_key ไปพร้อม POST /progress
curl -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{
    "new_status": "ON_SITE",
    "note": "ถึงจุดเกิดเหตุ น้ำสูง 1.2m",
    "image_key": "evidence/INC-001/TEAM-ALPHA/1718352735-flood-evidence-001.jpg"
  }' \
  "$API_URL/incidents/INC-001/progress"
```

---

## Dependency / Reliability

| ประเภท                   | รายละเอียด                                                                                   |
| :----------------------- | :------------------------------------------------------------------------------------------- |
| **Internal Data (Read)** | ตรวจสอบว่า incident_id มี Mission อยู่ใน MissionAssignment table                             |
| **AWS S3**               | สร้าง Presigned URL (PUT) ด้วย AWS SDK — ไม่ได้ upload ไฟล์จริงใน step นี้                   |
| **Validation**           | ตรวจสอบ file_name ไม่ว่าง, content_type เป็นรูปภาพที่รองรับ (`jpeg`/`png`/`webp`)            |
| **Security**             | Presigned URL อายุ 5 นาที → หมดอายุแล้วใช้ไม่ได้ ต้องขอใหม่                                  |
| **Upload Failure**       | ถ้า Frontend อัปโหลดไป S3 ไม่สำเร็จ → อนุญาตให้ "ข้าม" (Skip) → ส่งเฉพาะ Text Status ก่อนได้ |

---

---

# API Contract #4: List Team Missions

## ข้อมูลทั่วไป

| รายการ     | ค่า                                                               |
| :--------- | :---------------------------------------------------------------- |
| **Name**   | `listTeamMissions`                                                |
| **Method** | `GET`                                                             |
| **Path**   | `/incidents`                                                      |
| **Type**   | Synchronous                                                       |
| **Lambda** | `get-mission` (Go) — เพิ่ม handler path ใหม่ หรือ แยก Lambda ใหม่ |

## คำอธิบาย

ใช้ดึง รายการภารกิจทั้งหมด ของทีมกู้ภัยที่ระบุ เพื่อให้ทีมกู้ภัยเห็นภาพรวมว่าตัวเองมีกี่ภารกิจ แต่ละภารกิจอยู่สถานะอะไร → กดเข้าไปดู Timeline ละเอียดด้วย `GET /incidents/{incident_id}` ต่อได้

## Use Cases

- ทีมกู้ภัยเปิดแอปมา → เห็นรายการภารกิจทั้งหมดของตัวเอง
- Dispatcher ดูว่าทีมหนึ่งๆ รับผิดชอบภารกิจอะไรบ้าง
- กรอง Active missions (ยังไม่ `RESOLVED`) เพื่อโฟกัสงานที่ต้องทำ

---

## Request

### Query Parameters

| Parameter | Type   | Required | คำอธิบาย                                                               |
| :-------- | :----- | :------: | :--------------------------------------------------------------------- |
| `team_id` | String |    ✅    | รหัสทีมกู้ภัย (เช่น `TEAM-ALPHA`)                                      |
| `status`  | String |    ❌    | กรองเฉพาะสถานะ (เช่น `ON_SITE`, `NEED_BACKUP`) ถ้าไม่ส่ง = ดึงทุกสถานะ |

### Headers

| Header             | Required | คำอธิบาย                      |
| :----------------- | :------: | :---------------------------- |
| `x-api-key`        |    ✅    | API Key สำหรับ Authentication |
| `X-Rescue-Team-ID` |    ✅    | รหัสทีมกู้ภัย                 |

### Body

> ไม่มี

### ตัวอย่าง Request

```bash
# ดึงภารกิจทั้งหมดของ TEAM-ALPHA
curl -s -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  "$API_URL/incidents?team_id=TEAM-ALPHA" | jq .

# ดึงเฉพาะภารกิจที่สถานะ ON_SITE
curl -s -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  "$API_URL/incidents?team_id=TEAM-ALPHA&status=ON_SITE" | jq .
```

---

## Response

### Success `200 OK`

**Response Headers:**

```
Content-Type: application/json
X-Trace-Id: 99887766-5544-3322-1100-aabbccddeeff
```

**Response Body:**

```json
{
  "team_id": "TEAM-ALPHA",
  "total_missions": 3,
  "missions": [
    {
      "mission_id": "MSN-001",
      "incident_id": "INC-001",
      "current_status": "ON_SITE",
      "latest_impact_level": 3,
      "started_at": "2024-12-01T08:00:00Z",
      "last_updated_at": "2025-06-14T09:32:15Z"
    },
    {
      "mission_id": "MSN-003",
      "incident_id": "INC-003",
      "current_status": "NEED_BACKUP",
      "latest_impact_level": 5,
      "started_at": "2025-06-14T07:00:00Z",
      "last_updated_at": "2025-06-14T10:15:00Z"
    },
    {
      "mission_id": "MSN-005",
      "incident_id": "INC-005",
      "current_status": "RESOLVED",
      "latest_impact_level": 2,
      "started_at": "2025-06-13T14:00:00Z",
      "last_updated_at": "2025-06-13T18:30:00Z"
    }
  ]
}
```

| Field                            | Type              | คำอธิบาย                            |
| :------------------------------- | :---------------- | :---------------------------------- |
| `team_id`                        | String            | รหัสทีมกู้ภัยที่ค้นหา               |
| `total_missions`                 | Integer           | จำนวนภารกิจทั้งหมดที่พบ             |
| `missions`                       | Array             | รายการภารกิจ (สรุป ไม่รวม Timeline) |
| `missions[].mission_id`          | String            | รหัสภารกิจ                          |
| `missions[].incident_id`         | String            | รหัสเหตุการณ์                       |
| `missions[].current_status`      | String            | สถานะปัจจุบัน                       |
| `missions[].latest_impact_level` | Integer           | ระดับความรุนแรงล่าสุด               |
| `missions[].started_at`          | String (ISO 8601) | เวลาเริ่มภารกิจ                     |
| `missions[].last_updated_at`     | String (ISO 8601) | เวลาอัปเดตล่าสุด                    |

### Success `200 OK` — ไม่พบภารกิจ

```json
{
  "team_id": "TEAM-UNKNOWN",
  "total_missions": 0,
  "missions": []
}
```

### Success `200 OK` — กรองด้วย status

```json
{
  "team_id": "TEAM-ALPHA",
  "total_missions": 1,
  "missions": [
    {
      "mission_id": "MSN-001",
      "incident_id": "INC-001",
      "current_status": "ON_SITE",
      "latest_impact_level": 3,
      "started_at": "2024-12-01T08:00:00Z",
      "last_updated_at": "2025-06-14T09:32:15Z"
    }
  ]
}
```

### Error `400 Bad Request` — ไม่ส่ง team_id

```json
{
  "error": "MISSING_PARAMETER",
  "code": "MISSING_PARAMETER",
  "message": "team_id query parameter is required",
  "traceId": "44556677-8899-aabb-ccdd-eeff00112233"
}
```

### Error `400 Bad Request` — status ไม่ถูกต้อง

```json
{
  "error": "INVALID_STATUS",
  "code": "INVALID_STATUS",
  "message": "Invalid status filter: UNKNOWN_STATUS. Valid values: DISPATCHED, EN_ROUTE, ON_SITE, NEED_BACKUP, RESOLVED",
  "traceId": "55667788-99aa-bbcc-ddee-ff0011223344"
}
```

### Error `403 Forbidden`

> ⚠️ Response นี้มาจาก API Gateway โดยตรง — ไม่มี `X-Trace-Id` header และไม่มี `traceId` ใน body

```json
{
  "message": "Forbidden"
}
```

---

## DynamoDB Query Strategy

ใช้ GSI `team-index` ของ MissionAssignment table:

```
GSI: team-index
  Partition Key: rescue_team_id = "TEAM-ALPHA"

→ ได้ภารกิจทั้งหมดของทีม

ถ้ามี status filter:
→ Filter Expression: current_status = :status
→ DynamoDB filter หลัง query (ไม่ใช่ Key Condition)
```

---

## Dependency / Reliability

| ประเภท                     | รายละเอียด                                                                             |
| :------------------------- | :------------------------------------------------------------------------------------- |
| **Internal Data (Read)**   | Query MissionAssignment table ผ่าน GSI `team-index` (Partition Key = `rescue_team_id`) |
| **No External Dependency** | ไม่เรียก Service อื่น — อ่านจาก DynamoDB ของตัวเองเท่านั้น                             |
| **Validation**             | ตรวจสอบ team_id ไม่ว่าง, status (ถ้ามี) เป็นค่าที่ถูกต้อง                              |
| **Performance**            | GSI query = single-digit ms / Filter Expression ทำที่ DynamoDB → ไม่กระทบ Lambda       |
| **Empty Result**           | ไม่พบภารกิจ → return `200 OK` พร้อม `missions: []` (ไม่ใช่ 404)                        |


---
