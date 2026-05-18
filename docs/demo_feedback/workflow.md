# 📋 MissionProgress Service — Workflow ฉบับ Present (ไม่มี Live Demo)

> **หมายเหตุ:** AWS Account ถูก deactivate — ใช้ไฟล์นี้อธิบาย flow แทน Live Demo  
> เขียนแบบละเอียดเหมือนกำลัง demo จริง — อ่านแล้วเข้าใจได้โดยไม่ต้องเปิด console

---

## ก่อนเริ่ม — ภาพรวมระบบทั้งหมด

ระบบนี้คือระบบจัดการภารกิจกู้ภัยแบบ distributed microservices บน AWS

**5 services ที่ทำงานร่วมกัน:**

| Service             | เจ้าของ       | หน้าที่                                           |
| ------------------- | ------------- | ------------------------------------------------- |
| IncidentTracking    | Krittamet     | สร้างและติดตาม incident ภาพรวม                    |
| RescueRequest       | Phattharaphum | รับแจ้งเหตุจากประชาชน ออก rescue request          |
| Prioritization      | Nattasak      | จัดลำดับความสำคัญ request                         |
| ManageDispatch      | Noppakron     | จัดสรรทีมกู้ภัย ออก dispatch order                |
| **MissionProgress** | **ฉัน**       | **ติดตาม lifecycle ของภารกิจตั้งแต่รับงานจนจบ**  |

**ฉันเข้ามาในระบบตอนไหน:**

```
เกิดเหตุ → IncidentTracking สร้าง Incident
                → RescueRequest รับแจ้งจากประชาชน
                    → Prioritization จัดลำดับ
                        → ManageDispatch จัดสรรทีม
                                         ↓
                              [ฉันเริ่มทำงานตรงนี้]
                              MissionProgress รับ event
                              สร้าง Mission record
                              ติดตามจนภารกิจจบ
```

---

## Infrastructure ของฉัน (ที่ deploy ไปแล้วก่อน deactivate)

**Lambda Functions 7 ตัว** — ทั้งหมดเป็น Go binary, `runtime = "provided.al2023"`, 256MB RAM, timeout 30 วินาที:

| Function                   | หน้าที่                                               |
| -------------------------- | ----------------------------------------------------- |
| `authorizer`               | ตรวจสอบ API Key + ดึง team_id จาก header             |
| `mission-assigned-handler` | รับ SNS event สร้าง Mission record                    |
| `get-mission`              | GET mission พร้อม enrichment จาก 3 services           |
| `list-missions`            | LIST missions ของทีม                                   |
| `report-progress`          | PATCH status + notify downstream services             |
| `presigned-url`            | ออก S3 presigned URL สำหรับอัปโหลดรูปหลักฐาน         |
| `outbox-processor`         | Retry events ที่ publish ล้มเหลว (cron ทุก 1 นาที)   |

**DynamoDB 3 ตาราง:**

| Table               | เก็บอะไร                                | Primary Key                          |
| ------------------- | --------------------------------------- | ------------------------------------ |
| `MissionAssignment` | Mission records ทุกอัน                  | `mission_id` (PK)                    |
| `MissionTimeline`   | ประวัติทุก event ของแต่ละภารกิจ         | `mission_id` (PK) + `timestamp` (SK) |
| `EventOutbox`       | Events ที่ EventBridge publish ล้มเหลว  | `outbox_id` (PK)                     |

**GSI (Global Secondary Index) ใน MissionAssignment:**
- `team-index` — ค้นหา missions ทั้งหมดของทีม
- `request-index` — ค้นหา mission จาก request_id
- `dispatch-index` — ค้นหา mission จาก dispatch_id (ใช้ตรวจ idempotency)

---

## 🌊 Phase 1: เกิดเหตุ (ไม่ใช่ scope ของฉัน — เล่าเพื่อภาพรวม)

**สิ่งที่เกิดขึ้นก่อนที่ฉันจะรับงาน:**

> ลองนึกภาพสถานการณ์จริง: น้ำท่วมที่ถนนสุขุมวิท มีคนติดอยู่ 3 คน

**1. IncidentTracking (Krittamet)** สร้าง Incident ขึ้นมา:
```json
{
  "incident_id": "INC-001",
  "type": "FLOOD",
  "location": "ถนนสุขุมวิท ซอย 5",
  "severity": "HIGH"
}
```

**2. RescueRequest (Phattharaphum)** รับแจ้งจากประชาชน สร้าง Request:
```json
{
  "request_id": "REQ-001",
  "incident_id": "INC-001",
  "description": "น้ำเข้าบ้าน คนติดอยู่ 3 คน",
  "required_capability": "WATER_RESCUE"
}
```

**3. Prioritization (Nattasak)** จัดลำดับ — ความสำคัญ Level 3

**4. ManageDispatch (Noppakron)** จัดสรรทีม Alpha Rescue → สร้าง Dispatch Order:
```json
{
  "dispatch_id": "DISP-xyz",
  "request_id": "REQ-001",
  "team_id": "TEAM-001",
  "priority_level": 3,
  "dispatched_at": "2025-01-01T10:00:00Z"
}
```

---

## 🟢 Phase 2: ฉันรับงาน — Mission ถูกสร้าง (Async)

### Step 1: ManageDispatch publish event ผ่าน SNS

> ⚠️ **จุดสำคัญ:** Diagram เดิมบอก EventBridge แต่ implement จริงใช้ **SNS** เพราะ ManageDispatch เลือก SNS เป็น integration layer — ฉันต้องปรับ consumer ตาม

ManageDispatch publish ไปที่ **SNS Topic** ชื่อ `rescue.mission.dispatch.v1`:
```json
{
  "messageType": "DispatchOrderCreated",
  "dispatchId": "DISP-xyz",
  "requestId": "REQ-001",
  "teamId": "TEAM-001",
  "priorityLevel": 3,
  "dispatchedAt": "2025-01-01T10:00:00Z"
}
```

SNS มี **filter policy** → ส่งเฉพาะ `messageType = "DispatchOrderCreated"` มา trigger Lambda ของฉัน

---

### Step 2: `mission-assigned-handler` Lambda ทำงาน

Lambda ถูก trigger จาก SNS record นี้ทันที ขั้นตอนใน code:

**Step 2a: Idempotency Check** — เคยสร้าง Mission สำหรับ `dispatch_id` นี้แล้วไหม?
```
query DynamoDB → dispatch-index GSI → ค้นหา dispatch_id = "DISP-xyz"
```
- ถ้าเจอ → **skip** (ไม่สร้างซ้ำ)
- ถ้าไม่เจอ → ไปต่อ

> ⚠️ **ทำไมต้องเช็ค?** SNS มี at-least-once delivery guarantee — Lambda อาจถูก invoke ซ้ำได้ ถ้าไม่เช็ค Mission จะซ้ำทันที

**Step 2b: ดึง incident_id จาก RescueRequest (best-effort)**
```
GET /v1/rescue-requests/REQ-001
→ ถ้าได้: incidentId = "INC-001"
→ ถ้า RescueRequest ล่ม: incidentId = "" ← Degraded Mode ยังสร้าง Mission ได้!
```

**Step 2c: สร้าง Mission record**
```
สร้าง mission_id = "MISS-" + uuid[:8]  เช่น "MISS-a1b2c3d4"
เหตุผล: "MISS-" prefix ทำให้ human-readable ระหว่าง demo
```

**Step 2d: บันทึกลง DynamoDB พร้อมกัน 2 ตาราง:**

*ตาราง MissionAssignment:*
```json
{
  "mission_id": "MISS-a1b2c3d4",
  "dispatch_id": "DISP-xyz",
  "request_id": "REQ-001",
  "rescue_team_id": "TEAM-001",
  "incident_id": "INC-001",
  "current_status": "DISPATCHED",
  "priority_level": 3,
  "created_at": "2025-01-01T10:00:05Z"
}
```

*ตาราง MissionTimeline (entry แรก):*
```json
{
  "mission_id": "MISS-a1b2c3d4",
  "timestamp": "2025-01-01T10:00:05Z",
  "action_type": "MISSION_ASSIGNED",
  "description": "Mission created from dispatch DISP-xyz"
}
```

**ผลลัพธ์:** Mission `MISS-a1b2c3d4` อยู่ใน DynamoDB สถานะ `DISPATCHED` พร้อมแล้ว

---

## 🔵 Phase 3: ทีมดูข้อมูล Mission (GET — On-demand Enrichment)

### ทีม Alpha Rescue เปิด Dashboard ดู Mission ของตัวเอง

เรียก **GET `/v1/missions/REQ-001`** พร้อม headers:
```
x-api-key: {api_key}
X-Rescue-Team-ID: TEAM-001
```

**Authorizer Lambda ทำงานก่อน:**
1. อ่าน `x-api-key` จาก header → เทียบกับ `VALID_API_KEY` ใน env var
2. อ่าน `X-Rescue-Team-ID` จาก header
3. ถ้าผ่าน → สร้าง IAM Policy Allow แบบ **wildcard** (`arn:aws:execute-api:*/*/`) ครอบทุก endpoint
4. inject `rescueTeamId` เข้า request context → Lambda ถัดไปใช้ได้เลย

> **ทำไม wildcard?** API Gateway cache Policy ต่อ token — wildcard ทำให้ cache ครอบทุก endpoint ในครั้งเดียว ไม่ต้อง invoke Authorizer ซ้ำ ลด latency และ cost

**`get-mission` Lambda ทำงาน — เรียก 3 services พร้อมกัน (Parallel Goroutines):**

```
                    DynamoDB ──────────────────────────────► Core data
                              │
                    ┌─────────┼─────────────────────────────────────────┐
                    │         │                                           │
             goroutine 1   goroutine 2                            goroutine 3
                    │         │                                           │
              RescueRequest  ManageDispatch                         RescueTeam
              GET /REQ-001   GET /dispatches                       GET /teams/TEAM-001
                    │         │                                           │
              ← incident    ← dispatch details                      ← team details
                info          priority_level                           capabilities
                    │         │                                           │
                    └─────────┴──────────────────── wg.Wait() ───────────┘
                                                         │
                                                   merge ทุก data
                                                   สร้าง response
```

> **ทำไม parallel?** Sequential = 3 × 800ms = 2.4s minimum | Parallel = ~800ms maximum — เร็วกว่า 3x

**Timeout per service:** 800ms per attempt, retry 2 ครั้ง, exponential backoff 100ms → 200ms

**ถ้า service ใดล่ม:**

| Service ล่ม    | ผลลัพธ์                                                          |
| -------------- | ---------------------------------------------------------------- |
| RescueRequest  | `data_source: "partial"`, field `description`, `location` ว่าง  |
| RescueTeam     | `data_source: "partial"`, field `team_name`, `capabilities` ว่าง |
| ManageDispatch | ข้อมูล dispatch ว่าง (ไม่ set partial — supplementary data)     |
| ทั้ง 3 ล่ม     | `data_source: "partial"`, core data (status, timeline) ยังครบ   |

**Response ที่ทีมได้รับ:**
```json
{
  "request_id": "REQ-001",
  "mission_id": "MISS-a1b2c3d4",
  "dispatch_id": "DISP-xyz",
  "rescue_team_id": "TEAM-001",
  "current_status": "DISPATCHED",
  "data_source": "full",
  "team_name": "Alpha Rescue",
  "team_type": "WATER_RESCUE",
  "capabilities": ["boat", "firstaid"],
  "priority_level": 3,
  "description": "น้ำเข้าบ้าน คนติดอยู่ 3 คน",
  "location": "ถนนสุขุมวิท ซอย 5",
  "incident_type": "FLOOD",
  "timeline": [
    {
      "timestamp": "2025-01-01T10:00:05Z",
      "action_type": "MISSION_ASSIGNED",
      "description": "Mission created from dispatch DISP-xyz"
    }
  ],
  "started_at": "2025-01-01T10:00:05Z"
}
```

---

## 🟣 Phase 4: ทีมออกเดินทาง — รายงานสถานะแรก

### ทีมกดปุ่ม "ออกเดินทาง" บน Dashboard

เรียก **PATCH `/v1/missions/REQ-001/status`**:
```json
{
  "new_status": "EN_ROUTE",
  "new_impact_level": 3,
  "note": "เรือออกจากท่าแล้ว ใช้เวลาประมาณ 20 นาที",
  "current_location": "13.7563,100.5018"
}
```

**`report-progress` Lambda ทำงาน — ทีละ Step:**

**Step A: ยืนยัน ownership**
```
query DynamoDB → request-index GSI
KeyCondition:    request_id = "REQ-001"
FilterExpression: rescue_team_id = "TEAM-001"   ← ตรวจว่าเป็นทีมของตัวเอง
```
- ถ้าไม่เจอ → 404 `REQUEST_NOT_FOUND`

**Step B: ตรวจ State Machine**
```
validTransitions["DISPATCHED"] = ["EN_ROUTE"]
→ "EN_ROUTE" อยู่ใน list ✓
```
ถ้าส่ง status ที่ไม่ valid เช่น `DISPATCHED → RESOLVED` → 400 `INVALID_STATE_TRANSITION`

**Step C: Optimistic Lock — Update DynamoDB**
```
UpdateItem:
  Key: mission_id = "MISS-a1b2c3d4"
  SET current_status = "EN_ROUTE"
  ConditionExpression: current_status = "DISPATCHED"  ← ต้องตรงกับที่อ่านมา!
```

> **ทำไม Optimistic Lock?** Lambda scale แบบ horizontal — ถ้า 2 คนกด update พร้อมกัน ConditionExpression ป้องกัน last-write-wins  
> ถ้าแพ้ race: DynamoDB throw `ConditionalCheckFailedException` → ฉัน return 409 `CONCURRENT_UPDATE_CONFLICT` → client retry ได้

**Step D: บันทึก Timeline**
```
PutItem → MissionTimeline:
  action_type: "STATUS_CHANGE"
  old_status: "DISPATCHED"
  new_status: "EN_ROUTE"
  note: "เรือออกจากท่าแล้ว..."
  location: "13.7563,100.5018"
```

**Step E: Publish Event + Notify (Parallel Goroutines, context.Background())**

```
goroutine 1: Publish "MissionStatusChanged" → EventBridge
             ถ้าล้มเหลว → บันทึกลง EventOutbox DynamoDB รอ retry

goroutine 2: POST /rescue-requests/REQ-001/start → RescueRequest
             (best-effort — EN_ROUTE แจ้ง request ว่าทีมออกแล้ว)
             ถ้าล่ม → log WARN ไม่กระทบ 200

wg.Wait()  ← รอทั้ง 2 goroutine เสร็จก่อน return response
```

> **ทำไมใช้ `context.Background()` แทน Lambda context?**  
> Lambda context มี deadline — ถ้าใช้ Lambda context ใน goroutine แล้ว handler return context จะ cancel HTTP call ทันที  
> `context.Background()` ทำให้ goroutine ทำงานต่อได้ แต่ HTTP client ยังมี 800ms timeout อยู่

**Event ที่ publish:**
```json
{
  "source": "MissionProgressService",
  "detail-type": "MissionStatusChanged",
  "detail": {
    "schema_version": "1.0",
    "mission_id": "MISS-a1b2c3d4",
    "request_id": "REQ-001",
    "incident_id": "INC-001",
    "rescue_team_id": "TEAM-001",
    "old_status": "DISPATCHED",
    "new_status": "EN_ROUTE",
    "changed_at": "2025-01-01T10:15:00Z"
  }
}
```

**Event route ไปที่ไหน:**

```
EventBridge Custom Bus: "mission-progress-events"
    │
    ├──→ CloudWatch Logs  (บันทึกเสมอ — observability)
    │
    └──→ IncidentTracking (2 ช่องทาง ขึ้นอยู่กับ variable ที่ set):
         ├── API Destination (HTTP POST ตรงไป endpoint IncidentTracking)
         └── Direct Lambda Invocation
```

**Response ที่ทีมได้รับ:**
```json
{
  "message": "Progress reported successfully",
  "mission_id": "MISS-a1b2c3d4",
  "request_id": "REQ-001",
  "old_status": "DISPATCHED",
  "new_status": "EN_ROUTE",
  "updated_at": "2025-01-01T10:15:00Z"
}
```

---

## 🔵 Phase 4b: ทีมถึงหน้างาน

ทีมกดปุ่ม "ถึงหน้างานแล้ว" บน Dashboard:

```
PATCH /v1/missions/REQ-001/status
{ "new_status": "ON_SITE", "note": "ถึงแล้ว น้ำระดับเอว" }
```

**ขั้นตอนเหมือน Phase 4 ทุกประการ** แต่ transition ต่างกัน:
```
EN_ROUTE → ON_SITE  (valid ✓)
```

Goroutine ที่เรียก downstream สำหรับ ON_SITE:
- Publish `MissionStatusChanged` → EventBridge → IncidentTracking
- ไม่มี sync call เพิ่มเติมสำหรับ transition นี้

ณ จุดนี้ทีมอยู่หน้างาน — เกิดได้ 3 สถานการณ์:

---

## ✅ Phase 5: ภารกิจสำเร็จ — RESOLVED

> สถานการณ์: ทีมช่วยคนออกได้แล้ว น้ำลด ภารกิจเสร็จ

### ทีมกดปุ่ม "ภารกิจสำเร็จ"

```
PATCH /v1/missions/REQ-001/status
{
  "new_status": "RESOLVED",
  "note": "ช่วยคน 3 คนออกมาได้แล้ว ส่งโรงพยาบาล",
  "image_key": "evidence/MISS-a1b2c3d4/final.jpg"
}
```

**Transition:** `ON_SITE → RESOLVED` (หรือ `NEED_BACKUP → RESOLVED`)

**Goroutines สำหรับ RESOLVED — ซับซ้อนที่สุด:**

```
goroutine 1: Publish "MissionStatusChanged" → EventBridge
             ถ้าล้มเหลว → EventOutbox รอ retry

goroutine 2: POST /rescue-requests/REQ-001/resolve → RescueRequest
             (best-effort — ปิด rescue request)
             ถ้า RescueRequest ล่ม → log WARN ไม่กระทบ 200

goroutine 3: PATCH /teams/TEAM-001/status  body: { "status": "AVAILABLE" }
             → RescueTeam (best-effort — คืนทีมกลับเป็น available)
             ถ้า RescueTeam ล่ม → log WARN ไม่กระทบ 200
             ⚠️ trade-off: ทีมยังถูก mark เป็น BUSY ใน RescueTeam
                           แต่ mission ของฉันปิดแล้ว

goroutine 4: PATCH /dispatches/DISP-xyz/status  body: { "status": "RESOLVED" }
             → ManageDispatch (best-effort — ปิด dispatch order)
             ถ้า ManageDispatch ล่ม → log WARN ไม่กระทบ 200
             ⚠️ trade-off: dispatch ยัง OPEN อยู่ใน ManageDispatch
                           แต่ mission ของฉันปิดแล้ว

wg.Wait()  ← รอทุก goroutine เสร็จก่อน return
```

> ⚠️ **Diagram เดิมบอก:** ManageDispatch รับผ่าน SQS  
> **Code จริง:** เป็น sync PATCH โดยตรง — ManageDispatch มี REST endpoint รองรับ ไม่จำเป็นต้องผ่าน SQS (Terraform มี comment ยืนยัน)

**Event route สำหรับ MissionStatusChanged (RESOLVED):**
```
EventBridge → CloudWatch Logs (เสมอ)
           → IncidentTracking (API Destination หรือ Direct Lambda)
           → RescueRequest EventBridge Bus (cross-account)
              ⚠️ Diagram เดิมบอก SQS แต่ RescueRequest เปลี่ยนมาใช้ EventBridge Bus
```

**จบภารกิจ — สิ่งที่เกิดขึ้นสรุป:**
- Mission status = `RESOLVED` ใน DynamoDB ✅
- Timeline บันทึก entry สุดท้าย ✅
- IncidentTracking ได้รับแจ้งอัปเดต incident ✅
- ManageDispatch ปิด dispatch order (ถ้า service ยังอยู่) ✅
- RescueTeam คืน team status เป็น AVAILABLE (ถ้า service ยังอยู่) ✅
- RescueRequest ปิด rescue request (ถ้า service ยังอยู่) ✅

---

## 🆘 Phase 6: ขอกำลังเสริม — NEED_BACKUP (วนกลับ!)

> สถานการณ์: ทีม Alpha ถึงหน้างานแล้ว แต่สถานการณ์หนักกว่าที่คิด น้ำสูง ต้องการทีมเพิ่ม

### ทีมกดปุ่ม "ขอกำลังเสริม"

```
PATCH /v1/missions/REQ-001/status
{
  "new_status": "NEED_BACKUP",
  "note": "น้ำขึ้นเร็ว ต้องการเรือเพิ่ม อีก 1 ทีม",
  "current_location": "13.7563,100.5018"
}
```

**Transition:** `ON_SITE → NEED_BACKUP` (valid ✓)

**สิ่งที่ฉัน publish เพิ่มพิเศษ — `MissionBackupRequested`:**

```json
{
  "source": "MissionProgressService",
  "detail-type": "MissionBackupRequested",
  "detail": {
    "schema_version": "1.0",
    "mission_id": "MISS-a1b2c3d4",
    "incident_id": "INC-001",
    "rescue_team_id": "TEAM-001",
    "requested_at": "2025-01-01T11:00:00Z",
    "location": "13.7563,100.5018"
  }
}
```

**Event route:**
```
EventBridge → CloudWatch Logs (เสมอ)
           → Prioritization SQS (ถ้า var.prioritization_sqs_arn ถูก set)
```

**สิ่งที่เกิดต่อ (ไม่ใช่ scope ของฉันแต่เล่าให้เห็นภาพ):**

```
Prioritization (Nattasak) ได้รับ MissionBackupRequested
    │
    ▼
จัดลำดับใหม่ — เลือกทีม Beta Rescue สำหรับกำลังเสริม
    │
    ▼
ManageDispatch (Noppakron) ได้รับ order ใหม่
→ สร้าง Dispatch Order ใหม่: DISP-abc
→ publish "DispatchOrderCreated" ผ่าน SNS อีกครั้ง
    │
    ▼
[ฉันได้รับ SNS event ใหม่!]
mission-assigned-handler ทำงาน
→ สร้าง Mission ใหม่: "MISS-e5f6g7h8"
→ request_id เดิม = "REQ-001"
→ team_id ใหม่ = "TEAM-002"
→ status = DISPATCHED
```

> **จุดสำคัญ:** ใช้ request_id เดิม แต่ mission_id ใหม่ — 1 incident มีได้หลาย mission!

### เมื่อทีมเสริมถึง — NEED_BACKUP → ON_SITE

```
PATCH /v1/missions/REQ-001/status
{ "new_status": "ON_SITE" }
```

Transition: `NEED_BACKUP → ON_SITE` (valid ✓)

ภารกิจดำเนินต่อ — กลับไป Phase 5 หรือ 6 ได้อีก

---

## ⚠️ Phase 7: ความรุนแรงเปลี่ยน — Impact Level Updated

> สถานการณ์: ทีมอยู่หน้างาน พบว่าสถานการณ์หนักขึ้น น้ำเพิ่มระดับอย่างรวดเร็ว

### ทีมส่ง impact level ใหม่พร้อมกับ status update

```
PATCH /v1/missions/REQ-001/status
{
  "new_status": "ON_SITE",
  "new_impact_level": 5,
  "note": "น้ำสูงขึ้นเร็วมาก มีคนบาดเจ็บ"
}
```

เมื่อ `new_impact_level` แตกต่างจาก `latest_impact_level` เดิม — ฉัน publish event เพิ่ม:

```json
{
  "source": "MissionProgressService",
  "detail-type": "ImpactLevelUpdated",
  "detail": {
    "schema_version": "1.0",
    "mission_id": "MISS-a1b2c3d4",
    "incident_id": "INC-001",
    "old_impact_level": 3,
    "new_impact_level": 5,
    "updated_at": "2025-01-01T10:45:00Z"
  }
}
```

**Event route:**
```
EventBridge → CloudWatch Logs (เสมอ)
           → IncidentTracking (API Destination หรือ Direct Lambda)
              → Krittamet อัปเดต severity ของ incident
           → Prioritization SQS
              → Nattasak จัดลำดับใหม่ตาม impact ที่เพิ่ม
```

---

## 🔄 Phase 8: EventBridge ล้มเหลว — Outbox Pattern

> สถานการณ์: EventBridge มีปัญหา — event publish ล้มเหลว

**ขั้นตอนใน publisher.go:**

```
1. ลอง PutEvents ไปที่ EventBridge (timeout 5 วินาที)
   │
   ├── สำเร็จ → เสร็จ ✅
   │
   └── ล้มเหลว → บันทึกลง EventOutbox DynamoDB:
```

```json
{
  "outbox_id": "OUT-001",
  "created_at": "2025-01-01T10:15:01Z",
  "event_type": "MissionStatusChanged",
  "payload": "{ ...event detail... }",
  "status": "PENDING",
  "retry_count": 0
}
```

**outbox-processor Lambda ทำงานทุก 1 นาที (cron):**

```
scan DynamoDB EventOutbox → status = "PENDING"
    │
    ▼
สำหรับแต่ละ event:
    │
    ├── retry PutEvents → สำเร็จ → update status = "SENT" ✅
    │
    └── ล้มเหลว → retry_count++
         ├── retry_count < 5 → เก็บไว้ retry รอบหน้า
         └── retry_count = 5 → update status = "FAILED" ⚠️
                                log error ไว้ตรวจสอบ
```

> **Event ที่ fail สูงสุด 5 ครั้ง** จะ mark FAILED — แต่ mission data ใน DynamoDB ยังครบถ้วน

---

## 📦 Phase 9: อัปโหลดรูปหลักฐาน (Presigned URL Flow)

> ทีมถ่ายรูปหน้างานเพื่อเป็นหลักฐาน

**Step 1: ขอ upload URL**

```
POST /v1/missions/REQ-001/upload-url
{ "content_type": "image/jpeg" }
```

`presigned-url` Lambda สร้าง presigned PUT URL ของ S3:
```json
{
  "upload_url": "https://s3.amazonaws.com/mission-progress-evidence-xxx/evidence/MISS-a1b2c3d4/photo1.jpg?X-Amz-...",
  "key": "evidence/MISS-a1b2c3d4/photo1.jpg",
  "expires_in": 900
}
```

> Content-Type ที่รองรับ: `image/jpeg`, `image/png`, `image/webp` เท่านั้น

**Step 2: Frontend PUT รูปตรงไป S3** (ไม่ผ่าน Lambda!)

```
PUT {upload_url}
Body: [binary image data]
Content-Type: image/jpeg
```

> เหตุผล: ไม่ต้องให้ image data ผ่าน API Gateway/Lambda — ลด bandwidth cost และ latency

**Step 3: ส่ง image_key ใน status update**

```
PATCH /v1/missions/REQ-001/status
{
  "new_status": "ON_SITE",
  "image_key": "evidence/MISS-a1b2c3d4/photo1.jpg"
}
```

**Step 4: ดูรูปทีหลัง**

```
GET /v1/missions/REQ-001/upload-url?image_key=evidence/MISS-a1b2c3d4/photo1.jpg
→ คืน presigned GET URL (expires 300 วินาที)
```

---

## 🛡️ Phase 10: Degraded Mode — ทดสอบ Resilience

### Scenario A: RescueRequest ล่มตอนสร้าง Mission

```
ManageDispatch → SNS → mission-assigned-handler
                              │
                              ▼
                    GET /rescue-requests/REQ-001 → ❌ timeout (800ms × 3 attempts = 2.4s)
                              │
                              ▼
                    incidentId = ""   ← ยังสร้าง Mission ได้!
                              │
                              ▼
                    DynamoDB: { mission_id: "MISS-...", incident_id: "" }
```

**ผลกระทบ:** `incident_id` ว่าง — IncidentTracking อาจไม่ได้รับ event บางตัว  
**ยังทำงานได้:** Mission สร้างได้ ทีม report status ได้ events ส่งออกได้

---

### Scenario B: RescueTeam ล่มตอน GET Mission

```
GET /v1/missions/REQ-001
    │
    ├── DynamoDB → core data ✅
    ├── RescueRequest → incident info ✅
    ├── ManageDispatch → dispatch info ✅
    └── RescueTeam → ❌ timeout ← retry 2 ครั้ง แล้ว return nil
```

Response:
```json
{
  "data_source": "partial",
  "current_status": "ON_SITE",
  "team_name": "",
  "capabilities": []
}
```

**ทีมยังเห็น mission status ได้** แค่ team detail ว่าง — ระบบกู้ภัยต้องรู้ status เสมอ

---

### Scenario C: ManageDispatch ล่มตอน RESOLVED

```
PATCH /status { new_status: "RESOLVED" }
    │
    ▼
DynamoDB update สำเร็จ ✅
    │
    ▼
goroutine: PATCH /dispatches/DISP-xyz/status → ❌ (800ms × 3 retries, ~2.4s)
    │
    ▼
log WARN "failed to update ManageDispatch"
wg.Wait() รอครบ
    │
    ▼
return 200 ✅
```

**ผลกระทบ:** dispatch ใน ManageDispatch ยัง OPEN อยู่  
**Mission ปิดแล้ว:** ฉัน return 200 สำเร็จ — แค่ ManageDispatch ไม่รู้

---

### Scenario D: EventBridge ล่มตอน Publish

```
PATCH /status → DynamoDB update ✅
    │
    ▼
goroutine: PutEvents → ❌
    │
    ▼
saveToOutbox → DynamoDB EventOutbox ← บันทึกรอ retry
    │
    ▼
return 200 ✅

(ภายใน 1 นาที)
    ▼
outbox-processor cron → retry PutEvents → ✅
```

---

## 🗺️ สรุป Flow ทั้งหมดในภาพเดียว

```
เกิดเหตุ
    │
    ▼
IncidentTracking → RescueRequest → Prioritization → ManageDispatch
                                                          │
                                                    SNS: DispatchOrderCreated
                                                          │
                                                          ▼
                                              ┌───────────────────────┐
                                              │   MissionProgress     │
                                              │  mission-assigned     │
                                              │  idempotency check    │
                                              │  GET incidentId       │
                                              │  สร้าง Mission        │
                                              │  status: DISPATCHED   │
                                              └───────────────────────┘
                                                          │
                                              ทีมกด Dashboard
                                              PATCH /v1/missions/{id}/status
                                                          │
                          ┌───────────────────────────────┼───────────────────────────┐
                          │                               │                           │
                     RESOLVED                        NEED_BACKUP               IMPACT_LEVEL
                          │                               │                      UPDATED
                          │                               ▼                           │
              ┌───────────┴──────────┐             Prioritization          Prioritization
              │           │          │             ManageDispatch           IncidentTracking
         EventBridge  ManageDispatch  RescueTeam   → Mission ใหม่ 🔄
         (IncidentTracking)  sync PATCH  sync PATCH
         cross-account EB    RESOLVED    AVAILABLE
         (RescueRequest)
              │
              ▼
         CloudWatch Logs ← observability เสมอ
```

---

## 📌 State Machine — Transition ที่ valid

| จาก           | ไป            | Trigger                          |
| ------------- | ------------- | -------------------------------- |
| DISPATCHED    | EN_ROUTE      | ทีมออกเดินทาง                    |
| EN_ROUTE      | ON_SITE       | ทีมถึงหน้างาน                    |
| ON_SITE       | NEED_BACKUP   | ทีมขอกำลังเสริม                  |
| ON_SITE       | RESOLVED      | ภารกิจสำเร็จ                     |
| NEED_BACKUP   | ON_SITE       | ทีมเสริมมาถึง                    |
| NEED_BACKUP   | RESOLVED      | ยุติภารกิจ                       |

**ห้ามข้าม:** เช่น `DISPATCHED → RESOLVED` → 400 `INVALID_STATE_TRANSITION`  
**ย้อนกลับไม่ได้:** `RESOLVED` ไม่มีทางออก — ภารกิจจบแล้ว

---

## 🔍 สิ่งที่จะเห็นใน AWS Console (ถ้า account ยังอยู่)

**DynamoDB → MissionAssignment:**
```
mission_id  | current_status | rescue_team_id | request_id | dispatch_id | priority_level
MISS-a1b2c  | RESOLVED        | TEAM-001       | REQ-001    | DISP-xyz    | 3
MISS-e5f6g  | ON_SITE         | TEAM-002       | REQ-001    | DISP-abc    | 3
```

**DynamoDB → MissionTimeline:**
```
mission_id  | timestamp              | action_type      | old_status  | new_status
MISS-a1b2c  | 2025-01-01T10:00:05Z   | MISSION_ASSIGNED | -           | -
MISS-a1b2c  | 2025-01-01T10:15:00Z   | STATUS_CHANGE    | DISPATCHED  | EN_ROUTE
MISS-a1b2c  | 2025-01-01T10:30:00Z   | STATUS_CHANGE    | EN_ROUTE    | ON_SITE
MISS-a1b2c  | 2025-01-01T11:00:00Z   | STATUS_CHANGE    | ON_SITE     | NEED_BACKUP
MISS-a1b2c  | 2025-01-01T11:30:00Z   | STATUS_CHANGE    | NEED_BACKUP | RESOLVED
```

**CloudWatch Logs → `/aws/events/mission-status-changed`:**
```json
{ "source": "MissionProgressService", "detail-type": "MissionStatusChanged", "detail": { ... } }
```

**DynamoDB → EventOutbox (ถ้า EventBridge เคยล้มเหลว):**
```
outbox_id | event_type           | status  | retry_count
OUT-001   | MissionStatusChanged  | SENT    | 1
OUT-002   | MissionBackupRequested | FAILED | 5
```

---

## 💬 ประโยคสำหรับ Present (พูดได้เลย)

> **เปิดด้วย:**  
> "Service ของผมคือ MissionProgress — ทำหน้าที่เหมือน mission control center  
> ผมไม่ได้รับแจ้งเหตุครั้งแรก และไม่ได้จัดสรรทีม  
> แต่เมื่อทีมถูก dispatch มาแล้ว ผมรับงานต่อ ติดตามทุก step จนภารกิจจบ"

> **อธิบาย inbound:**  
> "ผมรับข้อมูลผ่าน SNS — ManageDispatch publish event 'DispatchOrderCreated' มา  
> Lambda ของผมถูก trigger ทันที ก่อนสร้าง Mission จะเช็คก่อนว่า dispatch_id นี้เคยสร้างแล้วไหม  
> เพราะ SNS มี at-least-once delivery — ต้องป้องกัน Mission ซ้ำ"

> **อธิบาย GET Mission:**  
> "ตอนทีมดู Mission ผมเรียก 3 services พร้อมกัน — RescueRequest, ManageDispatch, RescueTeam  
> ถ้า sequential จะช้า 2.4 วินาที แต่ parallel เหลือแค่ 800ms  
> ถ้า service ใดล่ม ผม return partial data — ทีมยังเห็น status ได้เสมอ"

> **อธิบาย PATCH status:**  
> "ตอนทีมกด update ผมใช้ Optimistic Locking ใน DynamoDB  
> ConditionExpression บังคับว่าค่าใน DB ต้องตรงกับที่อ่านมา  
> ถ้า 2 คนกดพร้อมกัน คนที่แพ้ race ได้ 409 แล้ว retry ได้ — ป้องกัน data corruption"

> **อธิบาย resilience:**  
> "ทุก downstream call เป็น best-effort — ถ้าล่มก็ log WARN แล้วตอบ 200 ต่อ  
> mission data อยู่ใน DynamoDB ของผม 100% — ไม่พึ่ง service อื่นสำหรับ write  
> EventBridge ล้มเหลว ผมไม่ทิ้ง event — บันทึกลง Outbox แล้วให้ cron retry ทุก 1 นาที"

> **สรุปท้าย:**  
> "สิ่งที่ผมภูมิใจคือ resilience ครบทุก layer  
> ไม่ว่า service เพื่อนจะล่ม ทีมกู้ภัยยังรายงานสถานะได้เสมอ  
> เพราะ mission data อยู่กับผม และ ทุก side-effect เป็น best-effort"
