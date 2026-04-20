# Refactor Phase 1 — Completion Report

**วันที่ดำเนินการ:** 20 เมษายน 2569  
**สถานะ:** ✅ เสร็จสมบูรณ์ (Build ผ่าน)

---

## สรุปการแก้ไข

จากการวิเคราะห์ Anti-pattern ใน 3 sessions พบปัญหา 2 จุดหลัก พร้อมคำแนะนำเพิ่มเติม 2 ข้อ ทั้งหมดได้รับการแก้ไขในรอบนี้:

| #   | ปัญหา / คำแนะนำ                                              | ที่มา          | สถานะ        |
| --- | ------------------------------------------------------------ | -------------- | ------------ |
| 1   | Static Contract Anti-Pattern — Events ไม่มี `schema_version` | Session 03     | ✅ แก้ไขแล้ว |
| 2   | ไม่มี Breaking Change Policy                                 | Session 03     | ✅ สร้างแล้ว |
| 3   | DynamoDB calls ไม่มี explicit timeout                        | Session 03     | ✅ แก้ไขแล้ว |
| 4   | ยังไม่มี Outbox Processor Lambda                             | Session 01, 02 | ✅ สร้างแล้ว |
| 5   | ไม่มี Retry with backoff สำหรับ external HTTP call           | Session 02     | ✅ เพิ่มแล้ว |
| 6   | EventBridge publish ไม่มี explicit timeout                   | Session 03     | ✅ แก้ไขแล้ว |

---

## Phase A: Static Contract Fix ✅

### Step 1: เพิ่ม `SchemaVersion` field ใน Event structs

- **ไฟล์:** `src/backend/internal/models/events.go`
- **การเปลี่ยนแปลง:** เพิ่ม `SchemaVersion string \`json:"schema_version"\`` ให้กับทั้ง 3 structs:
  - `MissionStatusChangedEvent`
  - `MissionBackupRequestedEvent`
  - `ImpactLevelUpdatedEvent`

### Step 2: อัปเดต Report-Progress Handler

- **ไฟล์:** `src/backend/cmd/report-progress/main.go`
- **การเปลี่ยนแปลง:** ทุกจุดที่สร้าง event struct ตอน publish ได้ set `SchemaVersion: "1.0"` แล้ว (3 จุด)

### Step 3: อัปเดต Async Contract Documentation

- **ไฟล์:** `docs/proposals/03-Async-Contract.md`
- **การเปลี่ยนแปลง:**
  - เพิ่ม `"schema_version": "1.0"` ใน Message Format ของทั้ง 3 events
  - เพิ่ม `detail.schema_version` ใน Field Definition table ของ MissionStatusChanged
  - เพิ่ม `detail.schema_version` ใน Field Definition table ของ MissionBackupRequested

### Step 4: สร้าง Breaking Change Policy

- **ไฟล์ใหม่:** `docs/policies/breaking-change-policy.md`
- **เนื้อหาครอบคลุม:**
  - นิยาม Breaking vs Non-Breaking Change พร้อมตัวอย่าง
  - กฎ Schema Version (major/minor bumping)
  - กระบวนการแจ้ง Consumer (2 สัปดาห์ล่วงหน้า, dual support 4 สัปดาห์)
  - รายชื่อ Consumers ปัจจุบัน (IncidentTracking, Dispatch, Prioritization)
  - ตัวอย่าง Non-Breaking (เพิ่ม field) และ Breaking (เปลี่ยน field type)

---

## Phase B: Timeout Hardening ✅

### Step 5: เพิ่ม Context Timeout สำหรับ DynamoDB Operations

- **ไฟล์ที่แก้ไข:**
  - `src/backend/internal/repository/mission_repo.go` — 3 functions: `GetMissionByIncidentID()`, `UpdateMissionStatus()`, `CreateMission()`
  - `src/backend/internal/repository/outbox_repo.go` — 1 function: `SaveOutboxEntry()`
  - `src/backend/internal/repository/timeline_repo.go` — 2 functions: `AddTimelineEntry()`, `GetTimelineByMissionID()`
- **Timeout:** 5 วินาที ต่อ operation (ใช้ `context.WithTimeout()`)
- **รวม:** 6 DynamoDB operations ทั้งหมดมี explicit timeout

### Step 6: เพิ่ม Context Timeout สำหรับ EventBridge Publish

- **ไฟล์:** `src/backend/internal/events/publisher.go`
- **การเปลี่ยนแปลง:** เพิ่ม `context.WithTimeout(ctx, 5*time.Second)` ครอบ `PutEvents()` call
- **Timeout:** 5 วินาที

---

## Phase C: Outbox Processor ✅

### Step 7: สร้าง Outbox Processor Lambda

- **ไฟล์ใหม่:** `src/backend/cmd/outbox-processor/main.go`
- **Logic:**
  1. Query `EventOutbox` GSI `status-index` สำหรับ entries ที่ status = `PENDING`
  2. สำหรับแต่ละ entry: validate payload → publish ไป EventBridge (5s timeout)
  3. สำเร็จ → update status เป็น `SENT`
  4. ล้มเหลว + retry < 5 → update `retry_count`, คง status `PENDING`
  5. ล้มเหลว + retry ≥ 5 → update status เป็น `FAILED`
- **Max retries:** 5 ครั้ง

### Step 8: เพิ่ม Repository Functions

- **ไฟล์:** `src/backend/internal/repository/outbox_repo.go`
- **Functions ใหม่:**
  - `GetPendingOutboxEntries(ctx)` — query GSI `status-index` where status = `PENDING`
  - `UpdateOutboxEntryStatus(ctx, outboxID, status, retryCount, lastError)` — update status + retry_count + last_error

### Step 9: Terraform + Build Script

- **`terraform/lambda.tf`** — เพิ่ม `aws_lambda_function.outbox_processor` (timeout 60s) + `aws_lambda_permission` สำหรับ EventBridge
- **`terraform/eventbridge.tf`** — เพิ่ม `aws_cloudwatch_event_rule.outbox_processor_schedule` (rate 1 minute) + target
- **`script/build.sh`** — เพิ่ม `"outbox-processor"` ใน `FUNCTIONS` array

---

## Phase D: Retry with Exponential Backoff ✅

### Step 10: Retry สำหรับ IncidentTracking HTTP Call

- **ไฟล์:** `src/backend/internal/client/incident_client.go`
- **การเปลี่ยนแปลง:**
  - Per-request timeout: 800ms (ลดจาก 3s เดิม)
  - Max retries: 2 (total 3 attempts)
  - Backoff: 100ms → 200ms (exponential, base 100ms)
  - Retry เฉพาะ: network errors + HTTP 5xx
  - ไม่ retry: HTTP 4xx (return nil ทันที)
  - Total worst-case time: 800ms + 100ms + 800ms + 200ms + 800ms = 2.7s (ยังอยู่ใน budget)
  - Degraded Mode ยังทำงานเหมือนเดิม — return nil ถ้าทุก attempt ล้มเหลว

---

## Build Verification

```
$ cd src/backend && go build ./...
# No errors — all 4 Lambda functions compile successfully
```

**Lambdas ที่ build ได้:**

1. `cmd/report-progress` ✅
2. `cmd/get-mission` ✅
3. `cmd/authorizer` ✅
4. `cmd/outbox-processor` ✅ (ใหม่)

---

## สรุปไฟล์ที่เปลี่ยนแปลง

### ไฟล์ที่แก้ไข (8 ไฟล์)

| ไฟล์                                               | การเปลี่ยนแปลง                                                                          |
| -------------------------------------------------- | --------------------------------------------------------------------------------------- |
| `src/backend/internal/models/events.go`            | เพิ่ม `SchemaVersion` field ใน 3 event structs                                          |
| `src/backend/internal/events/publisher.go`         | เพิ่ม context timeout 5s สำหรับ EventBridge `PutEvents()`                               |
| `src/backend/cmd/report-progress/main.go`          | set `SchemaVersion: "1.0"` ตอนสร้าง event (3 จุด)                                       |
| `src/backend/internal/repository/mission_repo.go`  | เพิ่ม `context.WithTimeout(5s)` ใน 3 functions                                          |
| `src/backend/internal/repository/outbox_repo.go`   | เพิ่ม timeout + 2 functions ใหม่ (`GetPendingOutboxEntries`, `UpdateOutboxEntryStatus`) |
| `src/backend/internal/repository/timeline_repo.go` | เพิ่ม `context.WithTimeout(5s)` ใน 2 functions                                          |
| `src/backend/internal/client/incident_client.go`   | เพิ่ม retry with exponential backoff (3 attempts, 100ms base)                           |
| `terraform/lambda.tf`                              | เพิ่ม outbox-processor Lambda function + permission                                     |
| `terraform/eventbridge.tf`                         | เพิ่ม scheduled rule (rate 1 min) + target                                              |
| `script/build.sh`                                  | เพิ่ม `outbox-processor` ใน build list                                                  |
| `docs/proposals/03-Async-Contract.md`              | เพิ่ม `schema_version` ใน event payload examples + field tables                         |

### ไฟล์ใหม่ (2 ไฟล์)

| ไฟล์                                       | คำอธิบาย                                       |
| ------------------------------------------ | ---------------------------------------------- |
| `src/backend/cmd/outbox-processor/main.go` | Outbox Processor Lambda — retry pending events |
| `docs/policies/breaking-change-policy.md`  | Breaking Change Policy document                |

---

## สิ่งที่ไม่ได้ทำใน Phase 1 (พิจารณาใน Phase ถัดไป)

| รายการ                                     | เหตุผลที่เลื่อน                               |
| ------------------------------------------ | --------------------------------------------- |
| Circuit Breaker                            | Degraded Mode + Retry ครอบคลุมเพียงพอในตอนนี้ |
| แยก Evidence Image Management เป็น service | ยังไม่ซับซ้อนพอ (ข้อควรระวังจาก Session 02)   |
| Field Assessment Forwarding review         | ต้องดูร่วมกับ service อื่นที่เป็น consumer    |
