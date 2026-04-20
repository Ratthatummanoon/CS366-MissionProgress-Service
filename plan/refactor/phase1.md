## Plan: Refactor Phase 1 — MissionProgress Service

แก้ไขปัญหาที่พบจากการวิเคราะห์ Anti-pattern 3 sessions โดยมี 2 ปัญหาหลัก: **Static Contract Anti-Pattern** (events ไม่มี `schemaVersion`, ไม่มี breaking change policy) และ **Timeout gaps** (DynamoDB calls ไม่มี explicit timeout) พร้อม implement **Outbox Processor** ที่ค้างอยู่ และเพิ่ม **Retry with backoff** ตามคำแนะนำ

---

### สรุปปัญหาที่พบจาก Anti-pattern Analysis

| Session | Anti-pattern | Verdict | สิ่งที่ต้องแก้ |
|---------|-------------|---------|---------------|
| 01 | Distributed Monolith, Shared DB, Chatty Services | ✅ ไม่พบ | ไม่ต้องแก้ |
| 02 | Over-Microservices, God Service | ✅ ไม่พบ | ข้อควรระวังเรื่อง Field Assessment Forwarding |
| 02 | Tight Coupling | ✅ ไม่พบ | แนะนำเพิ่ม Retry with backoff, Outbox Processor |
| 02 | Lack of Observability | ✅ ผ่าน | ไม่ต้องแก้ |
| 03 | Timeout | ⚠️ พบบางส่วน | DynamoDB calls ไม่มี explicit timeout |
| 03 | Static Contract | ❌ พบ | Events ไม่มี `schemaVersion`, ไม่มี breaking change policy |

---

### Steps

#### Phase A: Static Contract Fix (Critical)

**Step 1: เพิ่ม `SchemaVersion` field ใน Event structs**
- ไฟล์: events.go
- เพิ่ม `SchemaVersion string \`json:"schema_version"\`` ให้กับทั้ง 3 structs: `MissionStatusChangedEvent`, `MissionBackupRequestedEvent`, `ImpactLevelUpdatedEvent`
- ค่า initial version: `"1.0"`

**Step 2: อัปเดต Event Publisher ให้ set SchemaVersion** *(depends on Step 1)*
- ไฟล์: publisher.go และ report-progress/main.go
- ทุกจุดที่สร้าง event struct ต้อง set `SchemaVersion: "1.0"` ก่อน publish

**Step 3: อัปเดต Async Contract Documentation** *(parallel with Step 2)*
- ไฟล์: 03-Async-Contract.md
- เพิ่ม `schema_version` field ในตัวอย่าง payload ของทั้ง 3 events

**Step 4: เขียน Breaking Change Policy** *(parallel with Step 2)*
- สร้างไฟล์ใหม่: `docs/policies/breaking-change-policy.md`
- เนื้อหา: นิยาม breaking vs non-breaking change, กระบวนการ notify consumers, deprecation timeline, วิธีใช้ `schema_version`, ตัวอย่าง

---

#### Phase B: Timeout Hardening (Medium)

**Step 5: เพิ่ม Context timeout สำหรับ DynamoDB operations** *(parallel with Phase A)*
- ไฟล์: mission_repo.go, outbox_repo.go, timeline_repo.go
- ใช้ `context.WithTimeout()` ครอบ DynamoDB API call แต่ละตัว, timeout **5 วินาที**
- เปลี่ยน function signature ให้รับ `context.Context` (ถ้ายังไม่มี)
- อัปเดต callers ใน get-mission/main.go และ report-progress/main.go

**Step 6: เพิ่ม Context timeout สำหรับ EventBridge publish** *(parallel with Step 5)*
- ไฟล์: publisher.go
- เพิ่ม `context.WithTimeout()` สำหรับ `PutEvents()`, timeout **5 วินาที**

---

#### Phase C: Outbox Processor (Medium-High)

**Step 7: สร้าง Outbox Processor Lambda** *(depends on Step 1 — ต้องใช้ SchemaVersion)*
- สร้างไฟล์: `src/backend/cmd/outbox-processor/main.go`
- Logic: Query `EventOutbox` GSI `status-index` สำหรับ `PENDING` entries → publish ไป EventBridge → update status เป็น `SENT` (หรือ `FAILED` หลัง 5 retries)

**Step 8: เพิ่ม Repository functions สำหรับ Outbox Processor** *(parallel with Step 7)*
- ไฟล์: outbox_repo.go
- เพิ่ม `GetPendingOutboxEntries()` และ `UpdateOutboxEntryStatus()`

**Step 9: เพิ่ม Terraform + Build script** *(depends on Step 7)*
- lambda.tf — เพิ่ม `aws_lambda_function` สำหรับ `outbox-processor`
- eventbridge.tf — เพิ่ม CloudWatch Events Rule (scheduled every 1 minute) เพื่อ trigger
- build.sh — เพิ่ม build step สำหรับ `outbox-processor`

---

#### Phase D: Retry with Exponential Backoff (Medium)

**Step 10: เพิ่ม Retry สำหรับ IncidentTracking HTTP call** *(parallel with Phase C)*
- ไฟล์: incident_client.go
- Max retries: 2 (total 3 attempts), backoff: 100ms → 200ms → 400ms
- Retry เฉพาะ network errors + 5xx, ปรับ per-request timeout เป็น ~800ms
- ยังคง degrade gracefully (Degraded Mode) ถ้าทุก attempt ล้มเหลว

---

### Relevant Files

| ไฟล์ | การเปลี่ยนแปลง |
|------|---------------|
| events.go | เพิ่ม `SchemaVersion` field ใน 3 structs |
| publisher.go | set SchemaVersion + context timeout |
| report-progress/main.go | set SchemaVersion ตอนสร้าง event, ส่ง context |
| get-mission/main.go | ส่ง context ไป repos |
| mission_repo.go | เพิ่ม context timeout |
| outbox_repo.go | เพิ่ม context timeout + functions ใหม่ |
| timeline_repo.go | เพิ่ม context timeout |
| incident_client.go | retry with backoff |
| `cmd/outbox-processor/main.go` | ไฟล์ใหม่ — Outbox Processor Lambda |
| lambda.tf | เพิ่ม outbox-processor Lambda |
| eventbridge.tf | เพิ่ม scheduled rule |
| build.sh | เพิ่ม build outbox-processor |
| 03-Async-Contract.md | เพิ่ม `schema_version` ใน event examples |
| `docs/policies/breaking-change-policy.md` | document ใหม่ |

---

### Verification

1. **Build**: รัน build.sh → ต้อง compile สำเร็จทั้ง 4 Lambdas
2. **Event schema**: POST `/incidents/{id}/progress` → ตรวจ CloudWatch Logs ว่า event มี `schema_version: "1.0"`
3. **Timeout**: Mock DynamoDB ให้ช้า > 5s → ต้อง return error ไม่ hang
4. **Outbox processor**: สร้าง PENDING entry ใน EventOutbox → trigger processor → ตรวจว่า status เปลี่ยนเป็น SENT
5. **Retry**: Mock IncidentTracking ให้ return 500 → ตรวจ logs ว่ามี retry → สุดท้าย degrade เป็น partial
6. **Terraform plan**: `terraform plan` ต้องแสดง outbox-processor Lambda + scheduled rule
7. **Docs review**: `03-Async-Contract.md` มี `schema_version` ครบ, `breaking-change-policy.md` มีเนื้อหาครบ

---

### Decisions

- **SchemaVersion format**: `"1.0"` (semver-like string) เพื่อรองรับ minor version
- **DynamoDB timeout**: 5 วินาที — สมดุลระหว่าง DynamoDB internal retry กับ Lambda timeout (30s)
- **Outbox processor schedule**: ทุก 1 นาที — trade-off ระหว่าง timeliness กับ cost
- **Retry backoff**: base 100ms, max 2 retries — ให้อยู่ในกรอบ 3s timeout เดิม
- **Max outbox retry**: 5 ครั้ง → mark `FAILED` ให้จัดการ manual
- **Scope exclusion**: Circuit Breaker ไม่ทำใน Phase 1 เพราะ Degraded Mode + Retry ครอบคลุมเพียงพอ
