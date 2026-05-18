# **Asynchronous Function Contract**

---

## **Message Contract #1: MissionStatusChanged**

### ข้อมูลทั่วไป

| Field             | Value                                                                        |
| ----------------- | ---------------------------------------------------------------------------- |
| Message Name      | MissionStatusChangedEvent                                                    |
| Interaction Style | Asynchronous (Publish/Subscribe)                                             |
| Producer          | MissionProgress Service (report-progress Lambda — Go)                        |
| Consumers         | IncidentTracking, Dispatch Management                                        |
| Channel           | EventBridge (mission-progress-events)                                        |
| Demo 1            | ✅ CloudWatch Logs                                                           |
| Demo 2+           | ✅ EventBridge configured — routing ไปยัง real services เมื่อ endpoint พร้อม |

---

### คำอธิบาย

Event ถูก publish เมื่อมีการเปลี่ยนสถานะภารกิจสำเร็จ (ผ่าน validation)

**ใช้สำหรับ:**

- IncidentTracking → อัปเดตสถานะ Incident
- Dispatch → ปลดล็อกทีม (เฉพาะ `RESOLVED`)

**Trigger:** `POST /incidents/{id}/progress`

---

### Message Format

```json
{
  "source": "MissionProgressService",
  "detail-type": "MissionStatusChanged",
  "detail": {
    "schema_version": "1.0",
    "mission_id": "MSN-001",
    "requestId": "REQ-8812-9901",
    "incident_id": "INC-001",
    "rescue_team_id": "TEAM-ALPHA",
    "old_status": "EN_ROUTE",
    "new_status": "ON_SITE",
    "changed_at": "2025-06-14T09:32:15Z",
    "changed_by": "TEAM-ALPHA"
  }
}
```

---

### Field Definition

| Field                 | Type            | Required | Description                     |
| --------------------- | --------------- | -------- | ------------------------------- |
| source                | String          | ✅       | MissionProgressService          |
| detail-type           | String          | ✅       | MissionStatusChanged            |
| detail.schema_version | String          | ✅       | Schema version (current: "1.0") |
| detail.mission_id     | String          | ✅       | รหัสภารกิจ                      |
| detail.requestId      | String          | ✅       | รหัส request จาก RescueRequest  |
| detail.incident_id    | String          | ✅       | รหัสเหตุการณ์                   |
| detail.rescue_team_id | String          | ✅       | ทีมกู้ภัย                       |
| detail.old_status     | String          | ✅       | สถานะเดิม                       |
| detail.new_status     | String          | ✅       | สถานะใหม่                       |
| detail.changed_at     | ISO 8601 String | ✅       | เวลาที่เปลี่ยนสถานะ             |
| detail.changed_by     | String          | ✅       | ผู้กระทำ (Rescue Team ID)       |

---

### Validation Rules

- `new_status` ต้องถูกต้องตาม State Machine
- `changed_at` ต้องเป็น ISO 8601
- `rescue_team_id` ห้ามว่าง

---

### Consumer Routing

| Consumer         | Rule Filter                        | สถานะปัจจุบัน                   |
| ---------------- | ---------------------------------- | ------------------------------- |
| IncidentTracking | detail-type = MissionStatusChanged | ✅ EventBridge Direct (Lambda)  |
| Dispatch         | (sync PATCH on RESOLVED)           | ✅ Sync PATCH (fire-and-forget) |

---

### Failure Handling

- Publish ล้มเหลว → **Outbox Pattern**
- POST **ไม่ fail**
- EventBridge retry อัตโนมัติ (24 ชม.)

---

## **Message Contract #2: MissionBackupRequested**

### ข้อมูลทั่วไป

| Field        | Value                       |
| ------------ | --------------------------- |
| Message Name | MissionBackupRequestedEvent |
| Producer     | MissionProgress Service     |
| Consumers    | Prioritization Service      |
| Channel      | EventBridge                 |

---

### คำอธิบาย

Event เมื่อสถานะเป็น `NEED_BACKUP`

**ใช้สำหรับ:**

- Prioritization → คำนวณ Priority ใหม่

---

### Message Format

```json
{
  "source": "MissionProgressService",
  "detail-type": "MissionBackupRequested",
  "detail": {
    "schema_version": "1.0",
    "mission_id": "MSN-001",
    "incident_id": "INC-001",
    "rescue_team_id": "TEAM-ALPHA",
    "requested_at": "2025-06-14T10:15:00Z",
    "requested_by": "TEAM-ALPHA",
    "location": "13.7563,100.5018"
  }
}
```

---

### Field Definition

| Field                 | Required | Description                     |
| --------------------- | -------- | ------------------------------- |
| detail.schema_version | ✅       | Schema version (current: "1.0") |
| detail.mission_id     | ✅       | รหัสภารกิจ                      |
| detail.incident_id    | ✅       | รหัสเหตุการณ์                   |
| detail.rescue_team_id | ✅       | ทีมกู้ภัย                       |
| detail.requested_at   | ✅       | ISO 8601 เวลาที่ขอ backup       |
| detail.requested_by   | ✅       | ผู้ขอ backup (Rescue Team ID)   |
| detail.location       | ❌       | GPS string หน้างาน (optional)   |

---

### Consumer Routing

| Consumer       | Rule Filter            | สถานะปัจจุบัน       |
| -------------- | ---------------------- | ------------------- |
| Prioritization | MissionBackupRequested | ✅ SQS Route Active |

---

### Failure Handling

- Outbox Pattern
- Non-blocking

---

## **Message Contract #3: ImpactLevelUpdated**

### ข้อมูลทั่วไป

| Field        | Value                     |
| ------------ | ------------------------- |
| Message Name | ImpactLevelUpdatedEvent   |
| Consumers    | Incident + Prioritization |

---

### คำอธิบาย

Event เมื่อมีการปรับ Impact Level จากหน้างาน

---

### Message Format

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
    "updated_at": "2025-06-14T09:35:00Z",
    "updated_by": "TEAM-ALPHA"
  }
}
```

---

### Field Definition

| Field             | Type     | Required | Description                               |
| ----------------- | -------- | -------- | ----------------------------------------- |
| detail.old_level  | Integer  | ✅       | ระดับความรุนแรงเดิม (capture ก่อน update) |
| detail.new_level  | Integer  | ✅       | ระดับความรุนแรงใหม่จากหน้างาน             |
| detail.updated_at | ISO 8601 | ✅       | เวลา                                      |
| detail.updated_by | String   | ✅       | ผู้ปรับ (Rescue Team ID)                  |

> ⚠️ **หมายเหตุ:** Event นี้จะถูก publish **เฉพาะเมื่อ `new_level != old_level`** — ใส่ค่าเดิมซ้ำจะไม่ถูกส่ง

---

### Consumer Routing

| Consumer         | Rule Filter        | สถานะปัจจุบัน                  |
| ---------------- | ------------------ | ------------------------------ |
| IncidentTracking | ImpactLevelUpdated | ✅ EventBridge Direct (Lambda) |
| Prioritization   | ImpactLevelUpdated | ✅ SQS Route Active            |

---

### Failure Handling

- Outbox Pattern
- EventBridge retry
- Non-blocking - ทีมกู้ภัยยังทำงานได้แม้ Event ส่งไม่ถึง

---

## **Message Contract #4: DispatchOrderCreated (Inbound)**

### ข้อมูลทั่วไป

| Field             | Value                                                                                         |
| ----------------- | --------------------------------------------------------------------------------------------- |
| Message Name      | DispatchOrderCreated                                                                          |
| Interaction Style | Asynchronous (Subscribe)                                                                      |
| Producer          | Manage Dispatch Service                                                                       |
| Consumer          | MissionProgress Service (mission-assigned-handler Lambda — Go)                                |
| Channel           | SNS (`rescue.mission.dispatch.v1` — `arn:aws:sns:us-east-1:460581038623:request-dispatch-v1`) |
| Demo 2            | ✅ Implemented — `mission-assigned-handler` Lambda                                            |

---

### คำอธิบาย

MissionProgress ฟัง Event นี้จาก Manage Dispatch Service เมื่อ Dispatcher มอบหมายงานให้ทีมกู้ภัย Lambda `mission-assigned-handler` จะสร้าง MissionAssignment record และ Timeline entry แรกอัตโนมัติ

**ผลลัพธ์:**

1. สร้าง MissionAssignment (`status = DISPATCHED`)
2. สร้าง MissionTimeline entry แรก (`action_type = MISSION_ASSIGNED`)
3. ดึง `incident_id` จาก RescueRequest Service (degraded: empty string ถ้าล้มเหลว)
4. Idempotency check ด้วย `dispatch_id` — ถ้า mission มีอยู่แล้ว → skip

---

### Expected Payload (จาก Manage Dispatch Service via SNS)

Manage Dispatch ส่ง message มาผ่าน SNS Topic `rescue.mission.dispatch.v1` ด้วย envelope ดังนี้:

```json
{
  "header": {
    "messageType": "DispatchOrderCreated",
    "traceId": "trace-uuid"
  },
  "body": {
    "dispatchId": "DSP-001",
    "requestId": "REQ-001",
    "teamId": "TEAM-ALPHA",
    "priorityLevel": "HIGH",
    "status": "DISPATCHED",
    "dispatchedAt": "2025-06-14T08:45:00Z",
    "timestamp": "2025-06-14T08:45:00Z"
  }
}
```

### Field Definition

| Field              | Type   | Required | Description                                           |
| ------------------ | ------ | -------- | ----------------------------------------------------- |
| header.messageType | String | ✅       | ต้องเป็น `"DispatchOrderCreated"` (filter ใน handler) |
| header.traceId     | String | ❌       | สำหรับ distributed tracing                            |
| body.dispatchId    | String | ✅       | รหัส Dispatch Order (idempotency key)                 |
| body.requestId     | String | ✅       | รหัส request จาก RescueRequest                        |
| body.teamId        | String | ✅       | รหัสทีมกู้ภัย                                         |
| body.priorityLevel | String | ❌       | `"CRITICAL"` / `"HIGH"` / `"NORMAL"` / `"LOW"`        |
| body.status        | String | ❌       | สถานะ Dispatch Order                                  |
| body.dispatchedAt  | String | ❌       | เวลาที่มอบหมาย (ISO 8601)                             |

### Failure Handling

| กรณี                    | การจัดการ                                             |
| ----------------------- | ----------------------------------------------------- |
| Duplicate event         | Idempotency check → skip (ไม่ error)                  |
| RescueRequest ล่ม       | Degraded Mode → `incident_id = ""` (ยังสร้าง mission) |
| Missing required fields | Return error → SNS retry                              |
| Unknown messageType     | Log + skip (ไม่ error)                                |
