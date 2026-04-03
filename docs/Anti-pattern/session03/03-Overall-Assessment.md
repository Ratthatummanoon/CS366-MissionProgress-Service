# 📋 Overall Assessment

---

## 1. Most Significant Gap (ช่องว่างที่สำคัญที่สุด)

### **Async Events ไม่มี `schemaVersion` และไม่มี Breaking Change Policy**

**หลักฐานจาก Code (`events.go`):**

```go
// ทั้ง 3 event structs ไม่มี version field ใดเลย
type MissionStatusChangedEvent struct {
    MissionID    string `json:"mission_id"`
    IncidentID   string `json:"incident_id"`
    // ... ไม่มี SchemaVersion
}
```

**หลักฐานจาก Doc (`03-Async-Contract.md`):**

```json
{
  "source": "mission-progress-service",
  "detail-type": "MissionStatusChanged",
  "detail": {
    "mission_id": "MSN-001",
    "incident_id": "INC-001"
    // ไม่มี "schemaVersion": "1.0"
  }
}
```

**ผลกระทบจริง:**

> Service นี้มี **downstream consumer 3 ตัว** (IncidentTracking, Dispatch, Prioritization) ที่จะ subscribe events เหล่านี้ใน Demo 2+ — ถ้าวันหนึ่ง event schema เปลี่ยน (เช่น เปลี่ยน `new_impact_level` จาก String เป็น Integer) **consumer ทุกตัวจะพังโดยไม่มีทางรู้ล่วงหน้า** เพราะไม่มี version ให้เช็ค และไม่มี policy บอกว่าอะไรคือ breaking change

---

## 2. One Thing This Service Does Well (สิ่งที่ทำได้ดีมาก)

### **Degraded Mode + Outbox Pattern — Resilience Design ที่ครบถ้วน**

**หลักฐานจาก Code (`incident_client.go`):**

```go
// Explicit timeout 3 วินาที
httpClient: &http.Client{
    Timeout: 3 * time.Second,
}

// Return nil on failure → graceful degradation
func (c *IncidentClient) GetIncidentDetail(incidentID string) *models.IncidentDetail {
    resp, err := c.httpClient.Get(url)
    if err != nil {
        log.Printf("WARNING: IncidentTracking Service unavailable: %v", err)
        return nil  // ← ไม่ panic, ไม่ fail
    }
}
```

**หลักฐานจาก Code (`get-mission/main.go`):**

```go
// Degraded mode — ยังตอบ client ได้แม้ dependency ล่ม
incidentDetail := incidentClient.GetIncidentDetail(incidentID)
if incidentDetail != nil {
    description = incidentDetail.Description
} else {
    dataSource = "partial"  // ← บอก client ตรงๆ ว่าข้อมูลไม่ครบ
}
```

**หลักฐานจาก Code (`publisher.go`):**

```go
// EventBridge fail → ไม่ทำให้ request fail
if err != nil {
    log.Printf("ERROR: EventBridge publish failed for %s: %v - saving to outbox", detailType, err)
    p.saveToOutbox(ctx, detailType, string(data))  // ← fallback ทันที
    return
}
```

**ทำไมดี:**

> ในระบบกู้ภัย ทีมกู้ภัยหน้างาน **ต้องรายงานสถานะได้เสมอ** ไม่ว่า service อื่นจะล่มหรือไม่ — การออกแบบ Degraded Mode + Outbox Pattern ทำให้ **core workflow ไม่เคยหยุด** เพราะ dependency ภายนอก ซึ่งตรงกับ pain point ที่ระบุไว้ใน Service Overview

---

## 3. Is It Safe to Depend On? (ถ้าเราเป็น Consumer ปลอดภัยไหม?)

### คำตอบ: **ปลอดภัยในระดับหนึ่ง แต่ยังมีความเสี่ยง**

### ✅ สิ่งที่ทำให้มั่นใจได้

| จุด | หลักฐาน |
|---|---|
| API มี versioning `/v1/` | Terraform: `stage_name = "v1"` |
| Error format สม่ำเสมอ | `response.go` ใช้ `Error()` function เดียวทุก endpoint |
| มี `X-Trace-Id` ทุก response | `response.go` → `buildHeaders(traceID)` (หลังแก้แล้ว) |
| Sync API contract ละเอียดมาก | `02-Sync-Contract.md` ระบุทุก error case + response format |
| Events มี Outbox fallback | ไม่ต้องกังวลว่า event จะหายเงียบๆ |

### ❌ สิ่งที่ยังเป็นความเสี่ยง

| ความเสี่ยง | ผลกระทบต่อ consumer |
|---|---|
| Events ไม่มี `schemaVersion` | ถ้า schema เปลี่ยน consumer พังโดยไม่รู้ตัว |
| ไม่มี breaking change policy | ไม่มีสัญญาว่า field ไหนจะไม่ถูกลบ/เปลี่ยน |
| Doc กับ Code อาจไม่ sync | เช่น Doc เดิมไม่มี `traceId` แต่ code มีแล้ว |

### 📝 สิ่งที่ต้องขอจาก Owner ก่อน Integrate

```
1. เพิ่ม schemaVersion ใน event payload ทุกตัว
   → เพื่อให้ consumer เช็คได้ว่า schema ที่รับมาเป็น version ไหน

2. เขียน Breaking Change Policy
   → บอกให้ชัดว่าอะไร breaking / non-breaking
   → แจ้งล่วงหน้ากี่วันก่อนเปลี่ยน

3. ยืนยันว่า Event field names จะ stable
   → เช่น "mission_id" จะไม่เปลี่ยนเป็น "missionId" 
   → ใน Doc ระบุว่า field ไหน guaranteed vs optional

4. Confirm error format ที่แก้ใหม่ (traceId) deploy แล้ว
   → Doc กับ Code ต้อง sync กัน
```

---
