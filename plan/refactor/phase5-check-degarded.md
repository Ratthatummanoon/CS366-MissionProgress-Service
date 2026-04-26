# Phase 5 — Degraded Mode Audit

> ตรวจสอบว่า MissionProgress Service รับมือได้อย่างไรเมื่อ service ภายนอกล่ม
> วันที่ตรวจ: 24 เมษายน 2569

---

## ภาพรวม External Dependencies

| Dependency                     | ใช้ใน Lambda                          | ประเภทการเรียก  | Degraded Mode ปัจจุบัน                     |
| ------------------------------ | ------------------------------------- | --------------- | ------------------------------------------ |
| **RescueRequest Service**      | `get-mission`                         | Sync HTTP GET   | ✅ return `nil` → `data_source: partial`   |
| **ManageDispatch Service**     | `get-mission`                         | Sync HTTP GET   | ✅ return `nil` → skip dispatch enrichment |
| **RescueTeam Service (GET)**   | `get-mission`                         | Sync HTTP GET   | ✅ return `nil` → skip team enrichment     |
| **RescueTeam Service (PATCH)** | `report-progress`                     | Sync HTTP PATCH | ⚠️ log error แต่ยังบล็อก response อยู่     |
| **EventBridge**                | `report-progress`, `outbox-processor` | Async Publish   | ✅ Outbox fallback                         |
| **DynamoDB**                   | ทุก Lambda                            | CRUD            | ❌ ไม่มี fallback — return 500             |

---

## ข้อบกพร่องที่พบ

---

### 🔴 BUG-01 — Sequential External Calls ใน get-mission (Critical)

**ไฟล์:** `src/backend/cmd/get-mission/main.go`

**ปัญหา:**
3 การเรียก external service ถูก execute แบบ **sequential** (ทีละอัน) ไม่ใช่ parallel:

```go
// Call 1 — RescueRequest Service
requestDetail := rescueRequestClient.GetRequestDetail(requestID)

// Call 2 — ManageDispatch Service
dispatchList := manageDispatchClient.GetDispatchByTeamAndRequest(mission.RescueTeamID)

// Call 3 — RescueTeam Service
teamDetail := rescueTeamClient.GetTeamDetail(mission.RescueTeamID)
```

**ผลกระทบ (worst case เมื่อทุก service ล่ม):**

- แต่ละ client มี timeout 800ms × 3 retries + backoff (100ms + 200ms) = ~2,700ms
- 3 clients รวมกัน = **~8,100ms (8.1 วินาที)**
- Lambda API Gateway มี timeout ปกติ 10 วินาที → handler เกือบ timeout
- user ได้รับข้อมูลแบบ `partial` **ล่าช้า** แทนที่จะได้รับทันที

**สิ่งที่ควรเป็น:**
ใช้ `sync.WaitGroup` หรือ goroutine เรียก 3 services พร้อมกัน → worst case ลดเหลือ ~2,700ms

---

### 🔴 BUG-02 — Idempotency ใน mission-assigned-handler ทำงานผิด (Critical)

**ไฟล์:** `src/backend/cmd/mission-assigned-handler/main.go`

**ปัญหา:**
`mission_id` ถูก generate ใหม่ทุกครั้งที่ handler รัน:

```go
generatedMissionID := "MISS-" + uuid.New().String()[:8]  // ← UUID ใหม่ทุกครั้ง
```

จากนั้นใช้ `CreateMissionIdempotent` ที่ check `attribute_not_exists(mission_id)`:

- เนื่องจาก `mission_id` เป็น UUID ใหม่เสมอ → condition **ผ่านทุกครั้ง**
- ถ้า EventBridge ส่ง event ซ้ำ (at-least-once delivery) → สร้าง mission ซ้ำหลายรายการ
- idempotency check จะ catch ได้เฉพาะกรณี UUID ชนกัน (แทบเป็นไปไม่ได้)

**สิ่งที่ควรเป็น:**

- ต้องตรวจสอบ `dispatch_id` ก่อน (Query ผ่าน `dispatch-index` GSI)
- ถ้ามี mission ที่มี `dispatch_id` เดียวกันอยู่แล้ว → skip
- ขาด method `GetMissionByDispatchID` ใน `mission_repo.go`

---

### 🟠 BUG-03 — UpdateTeamStatus บล็อก Response Path (High)

**ไฟล์:** `src/backend/cmd/report-progress/main.go` บรรทัด ~175

**ปัญหา:**
การเรียก RescueTeam Service เพื่อปลดล็อกทีม (RESOLVED) ถูก comment ว่า "Best-effort" แต่ยังรันแบบ synchronous บน critical response path:

```go
// Best-effort — ถ้าล้มเหลวให้ log แล้วผ่าน — ไม่ fail request
if req.Status == "RESOLVED" && mission.RescueTeamID != "" {
    if err := rescueTeamClient.UpdateTeamStatus(mission.RescueTeamID, "AVAILABLE"); err != nil {
        log.Printf("WARN: ...")
        // ไม่ return error — mission ถูก update สำเร็จแล้ว
    }
}
```

**ผลกระทบ:**

- ถ้า RescueTeam Service ล่ม: `UpdateTeamStatus` จะ retry 3 ครั้ง + backoff = ~2,700ms
- user ต้องรอ response นานขึ้น 2.7 วินาทีสำหรับ RESOLVED update ทั้งที่ mission update สำเร็จแล้ว

**สิ่งที่ควรเป็น:**
ควรเรียกใน goroutine แบบ fire-and-forget:

```go
go func() {
    if err := rescueTeamClient.UpdateTeamStatus(mission.RescueTeamID, "AVAILABLE"); err != nil {
        log.Printf("WARN: ...")
    }
}()
```

---

### 🟠 BUG-04 — `defer resp.Body.Close()` ภายใน Retry Loop (High)

**ไฟล์:**

- `src/backend/internal/client/manage_dispatch_client.go` บรรทัด ~64
- `src/backend/internal/client/rescue_team_client.go` บรรทัด ~77, ~133

**ปัญหา:**
ใช้ `defer` inside for-loop แทนการ close ทันที:

```go
for attempt := 0; attempt <= mdMaxRetries; attempt++ {
    resp, err := c.httpClient.Do(req)
    ...
    defer resp.Body.Close()  // ← defer ไม่รันจนกว่า function จะ return!

    if resp.StatusCode >= 500 {
        // lastErr = ...
        continue  // ← body ยังไม่ถูกปิด แต่ loop ต่อ
    }
}
```

เปรียบเทียบกับ `rescue_request_client.go` ที่ทำถูก:

```go
resp.Body.Close()  // ← ปิดทันที ไม่ใช้ defer
```

**ผลกระทบ:**

- Connection leak: HTTP connections ไม่ถูก return กลับ pool จนกว่า function จะ return
- ในกรณี retry หลายครั้ง = หลาย connection ถูก hold พร้อมกัน
- อาจทำให้ connection pool หมดในกรณีที่มี concurrent Lambda invocations จำนวนมาก

---

### 🟠 BUG-05 — Retry Backoff ไม่ตรวจ Context Cancellation (High)

**ไฟล์:** ทุก client (`rescue_request_client.go`, `manage_dispatch_client.go`, `rescue_team_client.go`)

**ปัญหา:**
ใช้ `time.Sleep(backoff)` ใน retry loop โดยไม่ตรวจ context:

```go
if attempt > 0 {
    backoff := rrBackoffBase * (1 << (attempt - 1))
    time.Sleep(backoff)  // ← ไม่ตรวจ ctx.Done()
}
```

**ผลกระทบ:**

- ถ้า Lambda context ถูก cancel (timeout หมดแล้ว) → code ยังคง sleep ต่อ
- ไม่สามารถหยุด gracefully เมื่อ parent context หมดเวลา
- เสียเวลา billing time ที่ไม่มีประโยชน์

**สิ่งที่ควรเป็น:**

```go
select {
case <-time.After(backoff):
case <-ctx.Done():
    return nil  // abort early
}
```

---

### 🟡 BUG-06 — ไม่มี `GetMissionByDispatchID` ใน MissionRepo (Medium)

**ไฟล์:** `src/backend/internal/repository/mission_repo.go`

**ปัญหา:**
ไม่มี method สำหรับ query mission ด้วย `dispatch_id` ทั้งที่ DynamoDB มี `dispatch_id` เป็น field อยู่ใน model แต่ไม่มี GSI ที่ index บน field นี้

**ผลกระทบ:** ทำให้แก้ BUG-02 ไม่ได้โดยตรง (ต้องสร้าง GSI + method ใหม่ก่อน)

---

### 🟡 BUG-07 — `OldLevel` ใน `ImpactLevelUpdatedEvent` ถูก Hardcode เป็น 0 (Medium)

**ไฟล์:** `src/backend/cmd/report-progress/main.go` บรรทัด ~162

**ปัญหา:**

```go
publisher.PublishImpactLevelUpdated(ctx, models.ImpactLevelUpdatedEvent{
    OldLevel: 0,  // ← hardcoded เสมอ ไม่ได้อ่านค่าเดิม
    NewLevel: *req.NewImpactLevel,
})
```

**ผลกระทบ:**

- Downstream services (IncidentTracking) ที่ subscribe event นี้จะได้รับ `old_level: 0` เสมอ
- ไม่สามารถ detect ว่า impact level เปลี่ยนจริงหรือไม่ได้

**สิ่งที่ควรเป็น:** อ่านค่า `mission.LatestImpactLevel` ก่อน update แล้วใช้เป็น `OldLevel`

---

### 🟡 BUG-08 — `GetPendingOutboxEntries` ไม่มี Limit (Medium)

**ไฟล์:** `src/backend/internal/repository/outbox_repo.go` บรรทัด ~54

**ปัญหา:**

```go
output, err := r.client.Query(ctx, &dynamodb.QueryInput{
    // ไม่มี Limit: aws.Int32(N)
    TableName:  ...
    IndexName:  aws.String("status-index"),
    KeyConditionExpression: aws.String("#s = :status"),
})
```

**ผลกระทบ:**

- ถ้า EventBridge ล่มนานหลายชั่วโมง → Outbox อาจมี PENDING entries หลายพัน entries
- outbox-processor Lambda จะดึงทั้งหมดในครั้งเดียว → อาจ timeout หรือใช้ memory เกิน
- อาจกระทบ Lambda ที่มี 128MB default memory

---

### 🟡 BUG-09 — ManageDispatchClient ไม่มี Auth Header (Medium)

**ไฟล์:** `src/backend/internal/client/manage_dispatch_client.go`

**ปัญหา:**
`RescueRequestClient` และ `RescueTeamClient` ต่างมี `bearerToken` สำหรับ auth:

```go
req.Header.Set("Authorization", "Bearer "+c.bearerToken)
```

แต่ `ManageDispatchClient` ไม่มี auth header เลย แม้จะ read env var ก็ตาม

**ผลกระทบ:**

- ถ้า ManageDispatch Service กำหนดให้ต้องใช้ Auth → ได้รับ 401 ทุกครั้ง
- 401 ถูก handle ใน code path `resp.StatusCode != http.StatusOK` → return `nil` (degraded mode ทันที)
- ทำให้ดู degraded เสมอทั้งที่ service ยังทำงานอยู่

---

### 🟢 BUG-10 — ไม่มี Log เมื่อ Network Error ใน ManageDispatchClient (Low)

**ไฟล์:** `src/backend/internal/client/manage_dispatch_client.go` บรรทัด ~62

**ปัญหา:**

```go
resp, err := c.httpClient.Do(req)
if err != nil {
    lastErr = err
    continue  // ← ไม่มี log!
}
```

เปรียบเทียบ `rescue_request_client.go`:

```go
log.Printf("WARNING: RescueRequestService attempt %d failed (network): %v", attempt+1, err)
```

**ผลกระทบ:** Debug degraded mode ยากเพราะไม่รู้ว่า network error เกิดขึ้นตอนไหน

---

### 🟢 BUG-11 — Timeline Description ว่างเปล่าเมื่อไม่มี Note (Low)

**ไฟล์:** `src/backend/cmd/report-progress/main.go` บรรทัด ~128

**ปัญหา:**

```go
entry := &models.TimelineEntry{
    ActionType:  "STATUS_CHANGE",
    Description: req.Note,  // ← ว่างเปล่าถ้า user ไม่ส่ง note
}
```

**ผลกระทบ:**

- Timeline entry แสดงแค่ `ACTION: STATUS_CHANGE` โดยไม่บอกว่าเปลี่ยนจากอะไรไปอะไร
- ยากต่อการ debug หรือ audit ในภายหลัง

**สิ่งที่ควรเป็น:**

```go
desc := fmt.Sprintf("Status changed: %s → %s", oldStatus, req.Status)
if req.Note != "" {
    desc += ". Note: " + req.Note
}
```

---

## สรุปภาพรวม

| ID     | ระดับ       | ไฟล์ที่ต้องแก้                                                     | ผลกระทบหลัก                         |
| ------ | ----------- | ------------------------------------------------------------------ | ----------------------------------- |
| BUG-01 | 🔴 Critical | `cmd/get-mission/main.go`                                          | Lambda timeout เมื่อทุก service ล่ม |
| BUG-02 | 🔴 Critical | `cmd/mission-assigned-handler/main.go`                             | Duplicate mission records           |
| BUG-03 | 🟠 High     | `cmd/report-progress/main.go`                                      | RESOLVED ช้า 2.7 วิ                 |
| BUG-04 | 🟠 High     | `client/manage_dispatch_client.go`, `client/rescue_team_client.go` | Connection leak                     |
| BUG-05 | 🟠 High     | ทุก client                                                         | ไม่ abort เมื่อ Lambda timeout      |
| BUG-06 | 🟡 Medium   | `repository/mission_repo.go` + DynamoDB GSI                        | Blocker สำหรับ BUG-02               |
| BUG-07 | 🟡 Medium   | `cmd/report-progress/main.go`                                      | Event data ไม่ถูกต้อง               |
| BUG-08 | 🟡 Medium   | `repository/outbox_repo.go`                                        | outbox-processor อาจ timeout        |
| BUG-09 | 🟡 Medium   | `client/manage_dispatch_client.go`                                 | Degraded เสมอถ้า auth ต้องการ       |
| BUG-10 | 🟢 Low      | `client/manage_dispatch_client.go`                                 | Debug ยาก                           |
| BUG-11 | 🟢 Low      | `cmd/report-progress/main.go`                                      | Timeline ไม่มีข้อมูลเพียงพอ         |

---

## ลำดับการแก้ไขที่แนะนำ

```
Round 1 (Critical + บล็อก Deploy):
  BUG-02 → BUG-06 (ต้องทำก่อน BUG-02)
  BUG-01

Round 2 (High — Degraded Mode ทำงานไม่สมบูรณ์):
  BUG-03
  BUG-04
  BUG-05

Round 3 (Medium — Data Correctness):
  BUG-07
  BUG-08
  BUG-09

Round 4 (Low — Observability):
  BUG-10
  BUG-11
```
