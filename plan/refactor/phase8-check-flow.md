# Phase 8 — ตรวจสอบ Flow ว่าตรงกับ Demo Feedback หรือไม่

อ้างอิง:

- `docs/demo_feedback/diagram-flow.md` — Mermaid diagram แสดง service interaction
- `docs/demo_feedback/workflow.md` — ลำดับขั้นตอนการทำงานแบบ step-by-step

---

## สรุปผลการตรวจสอบ

| จุด                                            | diagram-flow                                       | workflow.md                      | โค้ดจริง                                                          | ผล  |
| ---------------------------------------------- | -------------------------------------------------- | -------------------------------- | ----------------------------------------------------------------- | --- |
| Async Inbound: DispatchOrderCreated → MP       | Default EventBridge                                | Step 1-2                         | `mission-assigned-handler` listen on default bus                  | ✅  |
| Sync: GET RescueRequest                        | `/v1/rescue-requests/{requestId}`                  | Step 3                           | `rescue_request_client.go` GET เดียวกัน + Bearer token            | ✅  |
| Sync: GET ManageDispatch                       | GET dispatch order                                 | Step 4                           | `manage_dispatch_client.go` มีอยู่ แต่ไม่ถูกเรียกตอนสร้าง mission | ⚠️  |
| Sync: GET RescueTeam                           | GET team info                                      | Step 5                           | `rescue_team_client.go` มีอยู่ แต่ไม่ถูกเรียกตอนสร้าง mission     | ⚠️  |
| Publish MissionStatusChanged                   | Custom EventBridge                                 | Step 7, 9                        | `publisher.go` + Terraform rule                                   | ✅  |
| Publish MissionBackupRequested                 | Custom EventBridge                                 | Step 10                          | เมื่อ status = `NEED_BACKUP`                                      | ✅  |
| Publish ImpactLevelUpdated                     | Custom EventBridge                                 | Step 11                          | เมื่อ `new_impact_level` ถูกส่งมาใน request body                  | ✅  |
| EventBridge → CloudWatch Logs (เสมอ)           | ทั้ง 3 events                                      | —                                | Terraform: rule ทั้ง 3 มี CW target                               | ✅  |
| EventBridge → IT SQS (conditional)             | MissionStatusChanged + ImpactLevelUpdated          | —                                | Terraform: `count = var.incident_tracking_sqs_arn != ""`          | ✅  |
| EventBridge → Dispatch SQS (RESOLVED only)     | MissionStatusChanged RESOLVED                      | —                                | Terraform: filter `new_status = ["RESOLVED"]`                     | ✅  |
| EventBridge → Prioritization SQS (conditional) | BackupRequested + ImpactLevelUpdated               | —                                | Terraform: 2 targets แยกกัน                                       | ✅  |
| Initial mission status (diagram)               | `DISPATCHED` (ระบุชัดใน arrow label)               | "PENDING" _(abstract)_           | `"DISPATCHED"` — ตรงกับ diagram                                   | ✅  |
| Status flow (diagram)                          | DISPATCHED→EN_ROUTE→ON_SITE→RESOLVED + NEED_BACKUP | PENDING→IN_PROGRESS _(abstract)_ | `statemachine.go` ตรงกับ diagram ทุก transition                   | ✅  |
| BackupRequested trigger (diagram)              | `triggered: ON_SITE→NEED_BACKUP`                   | Phase 6: NEED_BACKUP             | `if req.Status == "NEED_BACKUP"` — ตรงกัน                         | ✅  |

---

## รายละเอียด: จุดที่ตรงกัน ✅

### 1. Async Inbound — DispatchOrderCreated

- `mission-assigned-handler/main.go` รับ event จาก Default EventBridge ถูกต้อง
- Parse fields: `dispatchId`, `requestId`, `teamId`, `priorityLevel` ตรงกับ diagram
- มี idempotency check ด้วย `dispatch_id` เพื่อรับมือ at-least-once delivery

### 2. Sync: GET RescueRequest (ตอนสร้าง mission)

- เรียก `GET /v1/rescue-requests/{requestId}` พร้อม Bearer token
- ดึง `incidentID` กลับมาเก็บใน mission record
- หาก service ล่ม → degraded mode (incidentID = "")

### 3. Publisher — ทั้ง 3 events

- `publisher.go` publish ไปที่ custom bus `mission-progress-events`
- ทั้ง `MissionStatusChanged`, `MissionBackupRequested`, `ImpactLevelUpdated` มี struct ครบ
- มี Outbox fallback เมื่อ EventBridge ล่ม

### 4. EventBridge routing (Terraform)

- CloudWatch Logs: ทุก event เสมอ ✅
- IncidentTracking SQS: `MissionStatusChanged` + `ImpactLevelUpdated` (conditional ARN) ✅
- Dispatch SQS: เฉพาะ `new_status = RESOLVED` เท่านั้น ✅
- Prioritization SQS: `MissionBackupRequested` + `ImpactLevelUpdated` (conditional ARN) ✅

### 5. Sync calls ใน `get-mission`

- `get-mission/main.go` เรียก RescueRequest, ManageDispatch, RescueTeam แบบ parallel (goroutine)
- ข้อมูลทั้ง 3 service ถูก embed ใน response ของ GET endpoint

---

## รายละเอียด: จุดที่ไม่ตรง / เบี่ยงเบน ⚠️ ❌

### ⚠️ Status Names ใน workflow.md ยังไม่อัปเดตตาม diagram ใหม่

**diagram-flow.md (เวอร์ชันใหม่) บอก:**

- Initial status: `DISPATCHED` (ระบุชัดใน arrow label ของ `DEFAULT_EB → MP`)
- Transitions: `DISPATCHED→EN_ROUTE→ON_SITE→RESOLVED`, `ON_SITE→NEED_BACKUP`, `NEED_BACKUP→ON_SITE|RESOLVED`

**workflow.md ยังคงใช้:**

```
สถานะเริ่มต้น: PENDING
Mission: PENDING → IN_PROGRESS (Step 7)
Mission: IN_PROGRESS → RESOLVED (Phase 5)
```

**โค้ดจริง (`statemachine.go`):** ตรงกับ diagram ใหม่ทุก transition ✅

**ผลกระทบ**: diagram และโค้ดตรงกันแล้ว แต่ workflow.md ยังใช้ชื่อ abstract (PENDING/IN_PROGRESS) ซึ่งไม่ตรงกับ status จริง — ควรอัปเดต workflow.md

### ⚠️ Sync calls (Steps 3-5) เกิดขึ้นที่ GET ไม่ใช่ตอนสร้าง Mission

**workflow.md บอก (Steps 3-6):**

> ฉัน GET → RescueRequest → GET → ManageDispatch → GET → RescueTeam → รวมข้อมูล → สร้าง Mission

**โค้ดจริง:**

- `mission-assigned-handler` เรียกแค่ RescueRequest (เพื่อเอา `incidentID`)
- ManageDispatch และ RescueTeam **ไม่ถูกเรียกตอนสร้าง mission**
- ทั้ง 3 services ถูกเรียกที่ `get-mission` (GET endpoint) แทน — enrichment on read

**ผลกระทบ**: ข้อมูลทีม (ชื่อ, capabilities) และ dispatch details ไม่ถูกเก็บใน DynamoDB  
จะหายไปถ้า RescueTeam/ManageDispatch service ล่มในตอน GET

### ⚠️ ImpactLevelUpdated — trigger condition

**workflow.md บอก (Phase 7):**

> เป็น phase แยกต่างหาก เมื่อ impact level เปลี่ยน เช่น LOW → HIGH

**โค้ดจริง:**

- ถูก publish เมื่อ `req.NewImpactLevel != nil` ใน `report-progress`
- ไม่มี validation ว่า new level ต้องต่างจาก old level
- สามารถ publish พร้อมกับ `MissionStatusChanged` ในการเรียก API ครั้งเดียวได้
- ไม่มี dedicated endpoint สำหรับ update impact เพียงอย่างเดียว

### ⚠️ RescueTeam status update (RESOLVED) — ไม่มีใน diagram/workflow

**โค้ดจริง:**

```go
// report-progress/main.go
if req.Status == "RESOLVED" && mission.RescueTeamID != "" {
    go rescueTeamClient.UpdateTeamStatus(ctx, teamID, "AVAILABLE")
}
```

- เมื่อ RESOLVED → เรียก RescueTeam Service เพื่อ set ทีมเป็น AVAILABLE (fire-and-forget)
- Diagram และ workflow ไม่ได้กล่าวถึง sync outbound นี้เลย
- ถือเป็น extra behavior ที่ไม่ได้ document

---

## การเปลี่ยนแปลงใน diagram-flow.md (เวอร์ชันล่าสุด)

| จุดที่เปลี่ยน                  | ก่อน                   | หลัง                                                   | ผลต่อการ match               |
| ------------------------------ | ---------------------- | ------------------------------------------------------ | ---------------------------- |
| `DEFAULT_EB → MP` arrow        | ไม่ระบุ initial status | เพิ่ม `create Mission status: DISPATCHED`              | ❌→✅ ตรงกับโค้ด             |
| `MissionStatusChanged` arrow   | ไม่ระบุ transitions    | เพิ่ม transitions ทั้งหมด                              | ❌→✅ ตรงกับ statemachine.go |
| `MissionBackupRequested` arrow | ไม่ระบุ trigger        | เพิ่ม `triggered: ON_SITE→NEED_BACKUP`                 | ✅ ยืนยันชัดขึ้น             |
| CW_LOG3 label                  | `ImpactLevelUpdated`   | `ImpactUpdated`                                        | cosmetic เท่านั้น            |
| SQS consume labels             | สั้น                   | เพิ่ม context (เช่น EN_ROUTE=ลงพื้นที่แล้ว, วนกลับ 🔄) | cosmetic เท่านั้น            |

---

## สรุปสิ่งที่ต้องพิจารณา

| #   | ประเด็น                                                                                           | ความเร่งด่วน                                   |
| --- | ------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| 1   | อัปเดต workflow.md ให้ใช้ชื่อ status จริง (DISPATCHED/EN_ROUTE/ON_SITE) แทน PENDING/IN_PROGRESS   | สูง — workflow.md ยังไม่ตรงกับ diagram และโค้ด |
| 2   | Document ว่า enrichment data จาก ManageDispatch/RescueTeam เกิดที่ GET (on-read) ไม่ใช่ที่ CREATE | กลาง — อธิบายการออกแบบ                         |
| 3   | เพิ่ม RescueTeam AVAILABLE notification ใน diagram/workflow                                       | ต่ำ — extra behavior ที่ไม่ได้ document        |
| 4   | พิจารณา validate `new_impact_level != old_level` ก่อน publish ImpactLevelUpdated                  | ต่ำ — quality                                  |
