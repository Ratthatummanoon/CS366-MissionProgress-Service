# **Asynchronous Function Contract**

---

## **Message Contract #1: MissionStatusChanged**

### ข้อมูลทั่วไป

| Field             | Value                                                 |
| ----------------- | ----------------------------------------------------- |
| Message Name      | MissionStatusChangedEvent                             |
| Interaction Style | Asynchronous (Publish/Subscribe)                      |
| Producer          | MissionProgress Service (report-progress Lambda — Go) |
| Consumers         | IncidentTracking, Dispatch Management                 |
| Channel           | EventBridge (mission-progress-events)                 |
| Demo 1            | ✅ CloudWatch Logs                                    |
| Demo 2+           | 🔜 Route ไป Service จริง                              |

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
  "source": "mission-progress-service",
  "detail-type": "MissionStatusChanged",
  "detail": {
    "mission_id": "MSN-001",
    "incident_id": "INC-001",
    "rescue_team_id": "TEAM-ALPHA",
    "old_status": "EN_ROUTE",
    "new_status": "ON_SITE",
    "note": "ถึงจุดเกิดเหตุแล้ว น้ำสูง 1.2m",
    "updated_at": "2025-06-14T09:32:15Z",
    "performed_by": "TEAM-ALPHA"
  }
}
```

---

### Field Definition

| Field                 | Type            | Required | Description              |
| --------------------- | --------------- | -------- | ------------------------ |
| source                | String          | ✅       | mission-progress-service |
| detail-type           | String          | ✅       | MissionStatusChanged     |
| detail.mission_id     | String          | ✅       | รหัสภารกิจ               |
| detail.incident_id    | String          | ✅       | รหัสเหตุการณ์            |
| detail.rescue_team_id | String          | ✅       | ทีมกู้ภัย                |
| detail.old_status     | String          | ✅       | สถานะเดิม                |
| detail.new_status     | String          | ✅       | สถานะใหม่                |
| detail.note           | String          | ❌       | หมายเหตุ                 |
| detail.updated_at     | ISO 8601 String | ✅       | เวลา                     |
| detail.performed_by   | String          | ✅       | ผู้กระทำ                 |

---

### Validation Rules

- `new_status` ต้องถูกต้องตาม State Machine
- `updated_at` ต้องเป็น ISO 8601
- `rescue_team_id` ห้ามว่าง

---

### Consumer Routing

| Consumer         | Rule Filter                        | Demo 1 | Demo 2+    |
| ---------------- | ---------------------------------- | ------ | ---------- |
| IncidentTracking | detail-type = MissionStatusChanged | Logs   | 🔜 Service |
| Dispatch         | + new_status = RESOLVED            | Logs   | 🔜 Service |

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
  "source": "mission-progress-service",
  "detail-type": "MissionBackupRequested",
  "detail": {
    "mission_id": "MSN-001",
    "incident_id": "INC-001",
    "rescue_team_id": "TEAM-ALPHA",
    "old_status": "ON_SITE",
    "new_status": "NEED_BACKUP",
    "note": "ต้องการเรือเพิ่ม",
    "updated_at": "2025-06-14T10:15:00Z",
    "performed_by": "TEAM-ALPHA"
  }
}
```

---

### Field Definition

| Field             | Required | Description          |
| ----------------- | -------- | -------------------- |
| detail.new_status | ✅       | ต้องเป็น NEED_BACKUP |
| detail.old_status | ✅       | ต้องเป็น ON_SITE     |

---

### Consumer Routing

| Consumer       | Rule Filter            | Demo 1 | Demo 2+ |
| -------------- | ---------------------- | ------ | ------- |
| Prioritization | MissionBackupRequested | Logs   | 🔜      |

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
  "source": "mission-progress-service",
  "detail-type": "ImpactLevelUpdated",
  "detail": {
    "mission_id": "MSN-001",
    "incident_id": "INC-001",
    "rescue_team_id": "TEAM-ALPHA",
    "new_impact_level": "HIGH",
    "note": "สถานการณ์รุนแรงขึ้น",
    "updated_at": "2025-06-14T09:35:00Z",
    "performed_by": "TEAM-ALPHA"
  }
}
```

---

### Field Definition

| Field            | Required | Description     |
| ---------------- | -------- | --------------- |
| new_impact_level | ✅       | ระดับความรุนแรง |
| updated_at       | ✅       | ISO 8601        |

---

### Consumer Routing

| Consumer         | Rule Filter        | Demo 1 | Demo 2+ |
| ---------------- | ------------------ | ------ | ------- |
| IncidentTracking | ImpactLevelUpdated | Logs   | 🔜      |
| Prioritization   | ImpactLevelUpdated | Logs   | 🔜      |

---

### Failure Handling

- Outbox Pattern
- EventBridge retry
- Non-blocking - ทีมกู้ภัยยังทำงานได้แม้ Event ส่งไม่ถึง
