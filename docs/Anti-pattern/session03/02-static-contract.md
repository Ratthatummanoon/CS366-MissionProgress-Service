

# 🔍 Q5: Static Contract Analysis

> **"Do all endpoints use /v1/? Do async events have schemaVersion? Is there a written policy for breaking changes?"**

---

## เกณฑ์ 1: ทุก Endpoint ใช้ `/v1/` ไหม?

### ตรวจจาก Doc (`02-Sync-Contract.md`):

```
Base URL: https://api.disaster-management.net/mission-progress/v1
```
✅ มี `/v1`

### ตรวจจาก Terraform (`api_gateway.tf`):

```hcl
resource "aws_api_gateway_stage" "v1" {
  stage_name = "v1"   // ✅ stage name = v1
}
```

URL จริงจะเป็น:
```
https://{api-id}.execute-api.{region}.amazonaws.com/v1/incidents/{id}
https://{api-id}.execute-api.{region}.amazonaws.com/v1/incidents/{id}/progress
```

### ✅ Verdict: **ผ่าน — ทุก endpoint อยู่ภายใต้ `/v1/`**

---

## เกณฑ์ 2: Async Events มี `schemaVersion` ไหม?

### ตรวจจาก Doc (`03-Async-Contract.md`):

```json
{
  "source": "mission-progress-service",
  "detail-type": "MissionStatusChanged",
  "detail": {
    "mission_id": "MSN-001",
    "incident_id": "INC-001",
    ...
  }
  // ❌ ไม่มี schemaVersion
}
```

### ตรวจจาก Code (`events.go`):

```go
type MissionStatusChangedEvent struct {
    MissionID    string `json:"mission_id"`
    IncidentID   string `json:"incident_id"`
    RescueTeamID string `json:"rescue_team_id"`
    OldStatus    string `json:"old_status"`
    NewStatus    string `json:"new_status"`
    ChangedAt    string `json:"changed_at"`
    ChangedBy    string `json:"changed_by"`
    // ❌ ไม่มี SchemaVersion field
}

type MissionBackupRequestedEvent struct {
    // ❌ ไม่มี SchemaVersion
}

type ImpactLevelUpdatedEvent struct {
    // ❌ ไม่มี SchemaVersion
}
```

### ตรวจจาก `publisher.go`:

```go
Entries: []ebtypes.PutEventsRequestEntry{
    {
        Source:       aws.String(source),
        DetailType:  aws.String(detailType),
        Detail:      aws.String(string(data)),
        // ❌ ไม่มี schemaVersion ใน payload
    },
}
```

### ❌ Verdict: **ไม่ผ่าน — ไม่มี `schemaVersion` ใน event ใดเลย ทั้ง 3 events**

---

## เกณฑ์ 3: มี Written Policy สำหรับ Breaking Changes ไหม?

### ตรวจจาก Doc ทั้งหมดที่มี:

| ไฟล์ | มี Breaking Change Policy? |
|---|:---:|
| `01-Service-Overview.md` | ❌ |
| `02-Sync-Contract.md` | ❌ |
| `03-Async-Contract.md` | ❌ |
| `04-Service-Data.md` | ไม่ได้ดู |
| `05-Service-Architecture.md` | ไม่ได้ดู |
| `README.md` | ไม่ได้ดู |

### ❌ Verdict: **ไม่ผ่าน — ไม่มี written policy สำหรับ breaking changes**

---

## 📊 สรุป Q5

| เกณฑ์ | Status | หลักฐาน |
|---|:---:|---|
| ทุก endpoint ใช้ `/v1/`? | ✅ **ผ่าน** | Terraform `stage_name = "v1"` + Doc Base URL |
| Async events มี `schemaVersion`? | ❌ **ไม่ผ่าน** | ไม่มีใน event struct ทั้ง 3 ตัว |
| มี breaking change policy? | ❌ **ไม่ผ่าน** | ไม่พบในเอกสารใดเลย |

## ❌ Overall Q5 Verdict: **พบ Static Contract Anti-Pattern**

---

## 🛠️ แนวทางแก้ไข

### Fix 1: เพิ่ม `schemaVersion` ใน Event Struct

**`events.go` — แก้ไข:**

```go
type MissionStatusChangedEvent struct {
    SchemaVersion string `json:"schemaVersion"`
    MissionID     string `json:"mission_id"`
    IncidentID    string `json:"incident_id"`
    RescueTeamID  string `json:"rescue_team_id"`
    OldStatus     string `json:"old_status"`
    NewStatus     string `json:"new_status"`
    ChangedAt     string `json:"changed_at"`
    ChangedBy     string `json:"changed_by"`
}

type MissionBackupRequestedEvent struct {
    SchemaVersion string `json:"schemaVersion"`
    MissionID     string `json:"mission_id"`
    IncidentID    string `json:"incident_id"`
    RescueTeamID  string `json:"rescue_team_id"`
    RequestedAt   string `json:"requested_at"`
    RequestedBy   string `json:"requested_by"`
    Location      string `json:"location,omitempty"`
}

type ImpactLevelUpdatedEvent struct {
    SchemaVersion string `json:"schemaVersion"`
    MissionID     string `json:"mission_id"`
    IncidentID    string `json:"incident_id"`
    RescueTeamID  string `json:"rescue_team_id"`
    OldLevel      int    `json:"old_level"`
    NewLevel      int    `json:"new_level"`
    UpdatedAt     string `json:"updated_at"`
    UpdatedBy     string `json:"updated_by"`
}
```

### Fix 2: ใส่ `schemaVersion` ตอน publish (`report-progress/main.go`)

```go
publisher.PublishMissionStatusChanged(ctx, models.MissionStatusChangedEvent{
    SchemaVersion: "1.0",
    MissionID:     mission.MissionID,
    // ... เหมือนเดิม
})

publisher.PublishBackupRequested(ctx, models.MissionBackupRequestedEvent{
    SchemaVersion: "1.0",
    // ...
})

publisher.PublishImpactLevelUpdated(ctx, models.ImpactLevelUpdatedEvent{
    SchemaVersion: "1.0",
    // ...
})
```

### Fix 3: อัปเดต `03-Async-Contract.md`

เพิ่ม `schemaVersion` ในทุก Message Format:

```json
{
  "source": "mission-progress-service",
  "detail-type": "MissionStatusChanged",
  "detail": {
    "schemaVersion": "1.0",
    "mission_id": "MSN-001",
    "incident_id": "INC-001",
    ...
  }
}
```

และเพิ่มใน Field Definition table:

```markdown
| Field                | Type   | Required | Description           |
| -------------------- | ------ | -------- | --------------------- |
| detail.schemaVersion | String | ✅       | Schema version ("1.0")|
```

### Fix 4: เพิ่ม Breaking Change Policy

เพิ่มใน `03-Async-Contract.md` หรือสร้างไฟล์ใหม่:

```markdown
## Breaking Change Policy

### Sync API (REST)
- Breaking changes ต้องเพิ่ม version ใหม่ (เช่น `/v2/`)
- `/v1/` ต้อง support อย่างน้อย 3 เดือนหลัง `/v2/` เปิดใช้
- Non-breaking changes (เพิ่ม field ใหม่ optional) ทำได้โดยไม่ต้องขึ้น version

### Async Events (EventBridge)
- เพิ่ม field ใหม่ = non-breaking (consumer ต้อง ignore unknown fields)
- ลบ field / เปลี่ยน type = breaking → ต้องขึ้น schemaVersion
- Consumer ต้องเช็ค schemaVersion ก่อน process

### อะไรคือ Breaking Change?
| Breaking ❌                    | Non-Breaking ✅          |
| ----------------------------- | ----------------------- |
| ลบ field ที่มีอยู่              | เพิ่ม field ใหม่ (optional) |
| เปลี่ยน type ของ field         | เพิ่ม enum value ใหม่     |
| เปลี่ยน URL path              | เพิ่ม query parameter ใหม่ |
| เปลี่ยนความหมายของ field       | เพิ่ม header ใหม่         |
```

---
