# 🔍 Q3: Tight Coupling Analysis

> **"What happens to your service if every synchronous dependency goes down? Is a fallback defined for each?"**

---

## 📋 ระบุ Synchronous Dependencies ทั้งหมด

จาก Dependency Mapping + Source Code พบ Sync Dependencies ดังนี้:

| # | Sync Dependency | ใช้ใน Endpoint ไหน | Source Code |
|---|---|---|---|
| 1 | **IncidentTracking Service** (HTTP GET) | `GET /incidents/{id}` | `incident_client.go` → `get-mission/main.go` |
| 2 | **DynamoDB** (AWS Managed) | ทุก endpoint | `mission_repo.go`, `timeline_repo.go` |
| 3 | **EventBridge** (Publish) | `POST /progress` | `publisher.go` |
| 4 | **API Gateway + Authorizer** | ทุก endpoint | `authorizer/main.go` |
| 5 | **Amazon S3** (Presigned URL) | `POST /presigned-url` (Demo 2+) | ยังไม่ implement |

---

## 🧪 สถานการณ์: ถ้า Sync Dependency ทุกตัวล่มพร้อมกัน

### Dependency 1: IncidentTracking Service ล่ม

```go
// incident_client.go
func (c *IncidentClient) GetIncidentDetail(incidentID string) *models.IncidentDetail {
    resp, err := c.httpClient.Get(url)
    if err != nil {
        log.Printf("WARNING: IncidentTracking Service unavailable: %v", err)
        return nil  // ← return nil = degraded mode
    }
    ...
}
```

```go
// get-mission/main.go
incidentDetail := incidentClient.GetIncidentDetail(incidentID)
if incidentDetail != nil {
    // full mode
} else {
    dataSource = "partial"  // ← Degraded gracefully
}
```

| เกณฑ์ | ผลลัพธ์ |
|-------|---------|
| Service ยังทำงานได้? | ✅ **ได้** — return `data_source: "partial"` |
| Fallback defined? | ✅ **Degraded Mode** — ตัดข้อมูล description, location, incident_type ออก |
| Timeout กำหนด? | ✅ **3 วินาที** (`Timeout: 3 * time.Second`) |
| User experience? | ⚠️ ทีมกู้ภัยเห็นข้อมูลไม่ครบ แต่ยังทำงานได้ |

### ✅ Verdict: **ผ่าน — มี Fallback ชัดเจน**

---

### Dependency 2: EventBridge ล่ม

```
// จาก 07-Dependency-Mapping.md
EventBridge Publish ล้มเหลว
        │
        ▼
บันทึกลง EventOutbox Table (DynamoDB)
{
  "outbox_id": "OBX-uuid",
  "event_type": "MissionStatusChanged",
  "event_payload": "{...}",
  "status": "PENDING",
  "retry_count": 0
}
```

จาก `report-progress/main.go`:
```go
// 8. Publish events (non-blocking with outbox fallback)
publisher.PublishMissionStatusChanged(ctx, ...)
```

| เกณฑ์ | ผลลัพธ์ |
|-------|---------|
| Service ยังทำงานได้? | ✅ **ได้** — POST request ไม่ fail |
| Fallback defined? | ✅ **Outbox Pattern** — เก็บ event ไว้ retry ทีหลัง |
| Data integrity? | ✅ สถานะถูกบันทึกใน DynamoDB แล้วก่อน publish |

### ✅ Verdict: **ผ่าน — มี Outbox Pattern**

---

### Dependency 3: DynamoDB ล่ม

```go
// get-mission/main.go
mission, err := missionRepo.GetMissionByIncidentID(ctx, incidentID)
if err != nil {
    log.Printf("ERROR: query mission: %v", err)
    return response.Error(500, "INTERNAL_ERROR", "Failed to query mission"), nil
}
```

```go
// report-progress/main.go
if err := missionRepo.UpdateMissionStatus(ctx, mission); err != nil {
    log.Printf("ERROR: update mission: %v", err)
    return response.Error(500, "INTERNAL_ERROR", "Failed to update mission status"), nil
}
```

| เกณฑ์ | ผลลัพธ์ |
|-------|---------|
| Service ยังทำงานได้? | ❌ **ไม่ได้** — return 500 |
| Fallback defined? | ❌ **ไม่มี** — ไม่มี cache layer, ไม่มี retry logic |
| สมเหตุสมผลไหม? | ✅ **สมเหตุสมผล** — DynamoDB เป็น core data store, ถ้าล่มก็ไม่ควรทำงานต่อ |

### ⚠️ Verdict: **ยอมรับได้ — เป็น core infrastructure ไม่ใช่ external service**

> DynamoDB เป็น **internal data store** ที่ AWS guarantee 99.999% availability — การไม่มี fallback สำหรับ primary database ถือว่า **ไม่ใช่ anti-pattern** เพราะถ้า DB ล่ม service ไม่ควรตอบข้อมูลผิดๆ

---

### Dependency 4: API Gateway + Authorizer ล่ม

| เกณฑ์ | ผลลัพธ์ |
|-------|---------|
| Service ยังทำงานได้? | ❌ **ไม่ได้** — request เข้าไม่ถึง Lambda |
| Fallback defined? | ❌ **ไม่มี** (เขียนว่า Demo 2+ จะใช้ local cache) |
| สมเหตุสมผลไหม? | ✅ **สมเหตุสมผล** — เป็น ingress layer, ถ้าล่มก็ไม่มีทางรับ request |

### ⚠️ Verdict: **ยอมรับได้ — เป็น AWS Managed + ingress layer**

---

### Dependency 5: Amazon S3 ล่ม (Demo 2+)

| เกณฑ์ | ผลลัพธ์ |
|-------|---------|
| Fallback defined? | ✅ **"ข้ามได้"** — ส่งเฉพาะ Text Status |
| Impact? | ⚠️ อัปโหลดรูปไม่ได้ แต่ core workflow ยังทำงาน |

### ✅ Verdict: **ผ่าน — มี graceful skip**

---

## 📊 สรุป Tight Coupling Assessment

| # | Sync Dependency | ล่มแล้วเกิดอะไร | Fallback | Verdict |
|---|---|---|---|---|
| 1 | **IncidentTracking** | Degraded Mode (`partial`) | ✅ Return ข้อมูลที่มี | ✅ **ไม่ coupled** |
| 2 | **EventBridge** | เก็บ Outbox → retry later | ✅ Outbox Pattern | ✅ **ไม่ coupled** |
| 3 | **DynamoDB** | 500 Error | ❌ ไม่มี | ⚠️ ยอมรับได้ (core DB) |
| 4 | **API GW + Auth** | Request เข้าไม่ถึง | ❌ ไม่มี | ⚠️ ยอมรับได้ (ingress) |
| 5 | **S3** (Demo 2+) | ข้ามอัปโหลดรูป | ✅ Skip → Text only | ✅ **ไม่ coupled** |

---

## ✅ Overall Q3 Verdict: **ผ่าน — ไม่พบ Tight Coupling Anti-Pattern**

### 🟢 จุดที่ทำได้ดีมาก

| สิ่งที่ทำ | ทำไมดี |
|-----------|--------|
| **Degraded Mode** สำหรับ IncidentTracking | sync dependency ตัวเดียวที่เป็น external service → มี fallback ชัดเจน |
| **Outbox Pattern** สำหรับ EventBridge | event publish ล้มเหลวไม่ทำให้ request fail |
| **Timeout 3 วินาที** ใน incident_client.go | ป้องกัน cascading failure ไม่ให้ค้างนาน |
| **`data_source` field** ใน response | บอก caller ว่าข้อมูลครบหรือไม่ → transparency |

### 🟡 ข้อควรปรับปรุงเล็กน้อย

| จุด | คำแนะนำ |
|-----|---------|
| **ไม่มี Circuit Breaker** ใน `incident_client.go` | ถ้า IncidentTracking ล่มนาน ทุก request จะ wait 3 วินาทีก่อน timeout → ควรเพิ่ม circuit breaker เพื่อ fail fast |
| **ไม่มี Retry with backoff** ใน `incident_client.go` | ปัจจุบัน call 1 ครั้ง → fail → degraded ทันที → อาจ miss กรณี transient failure |
| **Outbox Processor ยังไม่ implement** | Demo 1 มีแค่ save → ยังไม่มี retry mechanism จริง |

```
// แนะนำเพิ่ม Circuit Breaker pattern:
type IncidentClient struct {
    baseURL       string
    httpClient    *http.Client
    failCount     int
    lastFailTime  time.Time
    circuitOpen   bool
}

func (c *IncidentClient) GetIncidentDetail(id string) *models.IncidentDetail {
    if c.circuitOpen && time.Since(c.lastFailTime) < 30*time.Second {
        log.Printf("Circuit OPEN — skip IncidentTracking call")
        return nil  // fail fast
    }
    // ... normal call ...
}
```

---
