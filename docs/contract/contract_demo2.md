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

### 2.1 GET /incidents/{incident_id} — ดึงข้อมูลภารกิจ

**Request:**

```bash
curl -X GET \
  "{BASE_URL}/incidents/INC-001" \
  -H "x-api-key: <key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA"
```

**Response 200 (Full Mode):**

```json
{
  "incident_id": "INC-001",
  "mission_id": "MSN-001",
  "rescue_team_id": "TEAM-ALPHA",
  "current_status": "DISPATCHED",
  "latest_impact_level": 2,
  "started_at": "2024-12-01T08:00:00Z",
  "last_updated_at": "2024-12-01T08:00:00Z",
  "description": "น้ำท่วมหนักบริเวณถนนพหลโยธิน",
  "location": "13.7563,100.5018",
  "incident_type": "FLOOD",
  "timeline": [...],
  "data_source": "full"
}
```

- `data_source: "full"` — IncidentTracking ตอบสำเร็จ
- `data_source: "partial"` — degraded mode (ไม่มี description, location, incident_type)

**Errors:** `404 INCIDENT_NOT_FOUND`, `400 MISSING_PARAMETER`

---

### 2.2 POST /incidents/{incident_id}/progress — อัปเดตสถานะ

**Request:**

```bash
curl -X POST \
  "{BASE_URL}/incidents/INC-001/progress" \
  -H "x-api-key: <key>" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{
    "new_status": "EN_ROUTE",
    "note": "กำลังเดินทางไปจุดเกิดเหตุ",
    "current_location": "13.7563,100.5018",
    "new_impact_level": 3,
    "image_key": "evidence/INC-001/TEAM-ALPHA/1718353500-photo.jpg"
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
  "incident_id": "INC-001",
  "old_status": "DISPATCHED",
  "new_status": "EN_ROUTE",
  "updated_at": "2025-..."
}
```

**Errors:** `400 INVALID_STATE_TRANSITION`, `400 INVALID_STATUS`, `404 INCIDENT_NOT_FOUND`

---

### 2.3 POST /incidents/{incident_id}/presigned-url — ขอ URL อัปโหลดภาพหลักฐาน (ใหม่ Demo 2)

**Request:**

```bash
curl -X POST \
  "{BASE_URL}/incidents/INC-001/presigned-url" \
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
  "image_key": "evidence/INC-001/TEAM-ALPHA/1718353500-flood-evidence.jpg",
  "expires_in": 300,
  "message": "Upload URL generated successfully. Use PUT method to upload."
}
```

**อัปโหลดภาพ:**

```bash
curl -X PUT -T photo.jpg -H "Content-Type: image/jpeg" "{upload_url}"
```

**Errors:** `400 INVALID_CONTENT_TYPE`, `400 MISSING_PARAMETER`, `404 INCIDENT_NOT_FOUND`, `500 PRESIGN_FAILED`

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
    "incident_id": "INC-001",
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
    "incident_id": "INC-001",
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
    "incident_id": "INC-001",
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
    "incident_id": "INC-001",
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

| บริการ               | วิธีเชื่อมต่อ                     | ตัวแปร Terraform            | Degraded Mode            |
| -------------------- | --------------------------------- | --------------------------- | ------------------------ |
| IncidentTracking     | HTTP GET `/incidents/{id}`        | `incident_service_url`      | `data_source: "partial"` |
| IncidentTracking SQS | EventBridge → SQS                 | `incident_tracking_sqs_arn` | CloudWatch Logs เดิม     |
| Dispatch SQS         | EventBridge → SQS (RESOLVED only) | `dispatch_sqs_arn`          | CloudWatch Logs เดิม     |
| Prioritization SQS   | EventBridge → SQS                 | `prioritization_sqs_arn`    | CloudWatch Logs เดิม     |

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
