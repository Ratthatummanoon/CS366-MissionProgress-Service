# 🔍 Q4: Timeout Analysis

> **"List every outbound call. Does each have an explicit timeout? What is the fallback when it times out?"**

---

## 📋 Outbound Calls ทั้งหมด

### 1. IncidentTracking Service (HTTP GET)

**Source:** `incident_client.go`

```go
httpClient: &http.Client{
    Timeout: 3 * time.Second,  // ✅ explicit timeout
}
```

| เกณฑ์ | ผลลัพธ์ |
|---|---|
| Explicit Timeout? | ✅ **3 วินาที** |
| Fallback? | ✅ **Degraded Mode** — return `nil` → `data_source: "partial"` |

---

### 2. EventBridge PutEvents

**Source:** `publisher.go`

```go
_, err = p.ebClient.PutEvents(ctx, &eventbridge.PutEventsInput{...})
```

| เกณฑ์ | ผลลัพธ์ |
|---|---|
| Explicit Timeout? | ❌ **ไม่มี** — ใช้ `ctx` จาก Lambda (implicit) |
| Fallback? | ✅ **Outbox Pattern** — `saveToOutbox()` |

---

### 3. DynamoDB — MissionRepo.GetMissionByIncidentID

**Source:** `mission_repo.go`

```go
output, err := r.client.Query(ctx, &dynamodb.QueryInput{...})
```

| เกณฑ์ | ผลลัพธ์ |
|---|---|
| Explicit Timeout? | ❌ **ไม่มี** — ใช้ `ctx` จาก Lambda (implicit) |
| Fallback? | ❌ **ไม่มี** — return error → 500 |

---

### 4. DynamoDB — MissionRepo.UpdateMissionStatus

**Source:** `mission_repo.go`

```go
_, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{...})
```

| เกณฑ์ | ผลลัพธ์ |
|---|---|
| Explicit Timeout? | ❌ **ไม่มี** |
| Fallback? | ❌ **ไม่มี** — return error → 500 |

---

### 5. DynamoDB — TimelineRepo.AddTimelineEntry

**Source:** `timeline_repo.go`

```go
_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{...})
```

| เกณฑ์ | ผลลัพธ์ |
|---|---|
| Explicit Timeout? | ❌ **ไม่มี** |
| Fallback? | ⚠️ **บางส่วน** — log error แต่ไม่ return 500 (non-blocking) |

---

### 6. DynamoDB — TimelineRepo.GetTimelineByMissionID

**Source:** `timeline_repo.go`

```go
output, err := r.client.Query(ctx, &dynamodb.QueryInput{...})
```

| เกณฑ์ | ผลลัพธ์ |
|---|---|
| Explicit Timeout? | ❌ **ไม่มี** |
| Fallback? | ⚠️ **บางส่วน** — return empty `[]` แทน |

---

### 7. DynamoDB — OutboxRepo.SaveOutboxEntry

**Source:** `outbox_repo.go`

```go
_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{...})
```

| เกณฑ์ | ผลลัพธ์ |
|---|---|
| Explicit Timeout? | ❌ **ไม่มี** |
| Fallback? | ❌ **ไม่มี** — log error แล้วจบ (event อาจหาย) |

---

## 📊 สรุปตาราง

| # | Outbound Call | Explicit Timeout | Fallback |
|---|---|:---:|:---:|
| 1 | **IncidentTracking** (HTTP GET) | ✅ 3 วินาที | ✅ Degraded Mode |
| 2 | **EventBridge** PutEvents | ❌ | ✅ Outbox Pattern |
| 3 | **DynamoDB** GetMission | ❌ | ❌ → 500 |
| 4 | **DynamoDB** UpdateMission | ❌ | ❌ → 500 |
| 5 | **DynamoDB** AddTimeline | ❌ | ⚠️ Non-blocking |
| 6 | **DynamoDB** GetTimeline | ❌ | ⚠️ Return `[]` |
| 7 | **DynamoDB** SaveOutbox | ❌ | ❌ Event อาจหาย |

---

## ✅ Verdict: **พบปัญหาบางส่วน**

### 🟢 ทำได้ดี

| จุด | เหตุผล |
|---|---|
| IncidentTracking มี explicit timeout 3 วินาที | external service ตัวเดียว → มี timeout + fallback ครบ |
| EventBridge มี Outbox fallback | publish fail → ไม่สูญหาย |

### 🟡 ข้อควรระวัง

| จุด | เหตุผล | ยอมรับได้ไหม |
|---|---|---|
| DynamoDB / EventBridge ไม่มี explicit timeout | ใช้ Lambda execution timeout เป็น implicit boundary (default 15 วินาที) | ⚠️ **ยอมรับได้** — AWS SDK มี default timeout อยู่แล้ว + Lambda มี max execution time |
| Outbox save fail → event หายจริง | เป็น last resort ถ้า DynamoDB ล่มด้วย ก็ทำอะไรไม่ได้ | ⚠️ **ยอมรับได้** — edge case ที่ทั้ง EventBridge + DynamoDB ล่มพร้อมกัน |

### 🔴 ถ้าจะปรับปรุง

```go
// เพิ่ม explicit timeout ให้ทุก outbound call:
func (r *MissionRepo) GetMissionByIncidentID(ctx context.Context, incidentID string) (*models.MissionAssignment, error) {
    timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    output, err := r.client.Query(timeoutCtx, &dynamodb.QueryInput{...})
    // ...
}
```

> **สรุป: External call (IncidentTracking) มี explicit timeout + fallback ครบถ้วน ส่วน AWS SDK calls ไม่มี explicit timeout แต่มี Lambda execution timeout เป็น safety net — ถือว่ายอมรับได้แต่ไม่ perfect**

---
