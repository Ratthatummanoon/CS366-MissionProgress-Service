# Phase 6 — Full-Project Scan: Orphaned Code & Configuration

> สแกนทั้ง project หาสิ่งที่ไม่เกี่ยวข้อง, dead code, และ configuration ที่ผิดพลาด
> วันที่ตรวจ: 24 เมษายน 2569

---

## สรุปสิ่งที่พบ

| #   | หมวดหมู่            | ตำแหน่ง                                | ความรุนแรง | สถานะ        |
| --- | ------------------- | -------------------------------------- | ---------- | ------------ |
| 1   | Config Bug          | `terraform/lambda.tf` (get-mission)    | 🔴 High    | ✅ แก้ไขแล้ว |
| 2   | Config Bug          | `terraform/variables.tf`               | 🔴 High    | ✅ แก้ไขแล้ว |
| 3   | Data Bug            | `cmd/mission-assigned-handler/main.go` | 🔴 High    | ✅ แก้ไขแล้ว |
| 4   | Data Bug            | `cmd/presigned-url/main.go`            | 🔴 High    | ✅ แก้ไขแล้ว |
| 5   | Dead Code           | `internal/repository/mission_repo.go`  | 🟡 Medium  | ✅ แก้ไขแล้ว |
| 6   | Dead Code           | `internal/repository/mission_repo.go`  | 🟡 Medium  | ✅ แก้ไขแล้ว |
| 7   | Orphaned Config     | `terraform/variables.tf`               | 🟡 Medium  | ✅ แก้ไขแล้ว |
| 8   | Dead Infrastructure | `terraform/dynamodb.tf`                | 🟡 Medium  | ✅ แก้ไขแล้ว |
| 9   | Security            | `terraform/variables.tf`               | 🟡 Medium  | ✅ แก้ไขแล้ว |
| 10  | Empty Directory     | `internal/statemachine/response/`      | 🟢 Low     | ✅ แก้ไขแล้ว |

---

## รายละเอียดปัญหาแต่ละข้อ

---

### 🔴 ISSUE-01 — `MANAGE_DISPATCH_SERVICE_TOKEN` ขาดหายใน Lambda env ของ `get-mission`

**ไฟล์ที่เกี่ยวข้อง:**

- `src/backend/internal/client/manage_dispatch_client.go`
- `terraform/lambda.tf`

**ปัญหา:**
`NewManageDispatchClient()` อ่าน `MANAGE_DISPATCH_SERVICE_TOKEN` จาก environment variable เพื่อใส่ใน Authorization header แต่ Lambda `get-mission` ไม่ได้รับ env var นี้:

```go
// manage_dispatch_client.go
token := os.Getenv("MANAGE_DISPATCH_SERVICE_TOKEN")  // อ่านจาก env
...
req.Header.Set("Authorization", "Bearer "+c.bearerToken)  // ใช้ใน header
```

```hcl
# lambda.tf — get-mission Lambda (ปัจจุบัน)
environment {
  variables = {
    TABLE_MISSION                = aws_dynamodb_table.mission_assignment.name
    TABLE_TIMELINE               = aws_dynamodb_table.mission_timeline.name
    RESCUE_REQUEST_SERVICE_URL   = var.rescue_request_service_url
    RESCUE_REQUEST_SERVICE_TOKEN = var.rescue_request_service_token
    MANAGE_DISPATCH_SERVICE_URL  = var.manage_dispatch_service_url
    # ❌ MANAGE_DISPATCH_SERVICE_TOKEN หายไป
    RESCUE_TEAM_SERVICE_URL      = var.rescue_team_service_url
    RESCUE_TEAM_SERVICE_TOKEN    = var.rescue_team_service_token
  }
}
```

**ผลกระทบ:** ทุก request ที่ส่งไปยัง Manage Dispatch Service จะมี `Authorization: Bearer ` (ค่าว่าง) → อาจถูก reject ด้วย 401/403 → `dispatchStatus` และ `priorityLevel` ใน `GetMissionResponse` จะว่างเสมอ

**การแก้ไข:**

1. เพิ่ม `variable "manage_dispatch_service_token"` ใน `terraform/variables.tf`
2. เพิ่ม `MANAGE_DISPATCH_SERVICE_TOKEN = var.manage_dispatch_service_token` ใน `get-mission` Lambda env

---

### 🔴 ISSUE-02 — Terraform variable `manage_dispatch_service_token` ไม่มีอยู่ใน `variables.tf`

**ไฟล์ที่เกี่ยวข้อง:**

- `terraform/variables.tf`

**ปัญหา:**
`terraform/lambda.tf` ต้องการ `var.manage_dispatch_service_token` (ตามที่ควรเพิ่มใน ISSUE-01) แต่ `variables.tf` ยังไม่มีการประกาศตัวแปรนี้เลย

เปรียบเทียบกับ service token อื่นที่มีอยู่แล้ว:

```hcl
# มีอยู่แล้ว (ถูกต้อง)
variable "rescue_request_service_token" { ... }
variable "rescue_team_service_token" { ... }

# ❌ ขาดหายไป
variable "manage_dispatch_service_token" { ... }
```

**การแก้ไข:** เพิ่ม block ต่อไปนี้ใน `variables.tf`:

```hcl
variable "manage_dispatch_service_token" {
  description = "Bearer token for authenticating with Manage Dispatch Service"
  type        = string
  sensitive   = true
  default     = ""
}
```

---

### 🔴 ISSUE-03 — `MissionAssignment.IncidentID` ถูกบันทึกเป็น empty string เสมอ

**ไฟล์ที่เกี่ยวข้อง:**

- `src/backend/cmd/mission-assigned-handler/main.go`

**ปัญหา:**
`DispatchOrderCreated` event จาก Manage Dispatch Service ไม่รวม `incidentId` มาด้วย ทำให้ `MissionAssignedPayload` struct ไม่มี field นี้:

```go
// mission-assigned-handler/main.go
type MissionAssignedPayload struct {
    DispatchID    string `json:"dispatchId"`
    RequestID     string `json:"requestId"`
    TeamID        string `json:"teamId"`
    PriorityLevel int    `json:"priorityLevel"`
    Status        string `json:"status"`
    DispatchedAt  string `json:"dispatchedAt"`
    // ❌ ไม่มี IncidentID
}

// สร้าง mission โดย IncidentID เป็น "" ตลอดไป
mission := &models.MissionAssignment{
    ...
    IncidentID: "",  // ← ว่างเสมอ
    ...
}
```

**ผลกระทบ (Ripple Effect):**

1. `MissionStatusChangedEvent.IncidentID = ""` → IncidentTracking Service รับ event แต่ไม่รู้ว่า incident ใด
2. `ImpactLevelUpdatedEvent.IncidentID = ""` → เช่นเดียวกัน
3. `MissionBackupRequestedEvent.IncidentID = ""` → เช่นเดียวกัน
4. `GET /missions/{request_id}` response มี `"incident_id": ""` → Frontend แสดงข้อมูลไม่ครบ

**แนวทางแก้ไข:**

- **Option A (Recommended):** เพิ่ม HTTP call ไปยัง RescueRequest Service ใน `mission-assigned-handler` ด้วย `payload.RequestID` เพื่อดึง `incidentId` ก่อนสร้าง mission — handler นี้ยังไม่มี external client ใดๆ เลย
- **Option B:** หากทำ Option A ไม่ได้ ให้ดึง `IncidentID` แบบ lazy ใน `get-mission` handler (เมื่อ mission.IncidentID == "" ให้เรียก RescueRequest Service เพื่อ fill)

---

### 🔴 ISSUE-04 — S3 key path มี double-slash เพราะ `IncidentID` ว่าง

**ไฟล์ที่เกี่ยวข้อง:**

- `src/backend/cmd/presigned-url/main.go`

**ปัญหา:**
S3 key ถูกสร้างโดยใช้ `mission.IncidentID` ซึ่งเป็น `""` เสมอ (จาก ISSUE-03):

```go
// presigned-url/main.go
imageKey := fmt.Sprintf("evidence/%s/%s/%d-%s",
    mission.IncidentID,  // ← ว่างเสมอ
    rescueTeamID,
    timestamp,
    req.FileName,
)
// ผลลัพธ์: "evidence//TEAM-ALPHA/1745000000-photo.jpg"
//                     ^^ double-slash ← ผิด
```

**ผลกระทบ:** S3 object ถูกบันทึกด้วย path ที่ผิดรูปแบบ (`evidence//...`) ทำให้ไม่สามารถค้นหาหรือ organize ไฟล์ได้ถูกต้อง

**การแก้ไข:** เปลี่ยนไปใช้ `mission.MissionID` แทน เพราะ `MissionID` ถูก generate เสมอ:

```go
// แก้ไขเป็น
imageKey := fmt.Sprintf("evidence/%s/%s/%d-%s",
    mission.MissionID,   // ← ใช้ mission_id ที่มีค่าเสมอ
    rescueTeamID,
    timestamp,
    req.FileName,
)
// ผลลัพธ์: "evidence/MISS-a1b2c3d4/TEAM-ALPHA/1745000000-photo.jpg"
```

---

### 🟡 ISSUE-05 — `MissionRepo.GetMissionByIncidentID` — Dead Code

**ไฟล์ที่เกี่ยวข้อง:**

- `src/backend/internal/repository/mission_repo.go` (line 27–50)

**ปัญหา:**
`GetMissionByIncidentID` ถูก define แต่ไม่มี Lambda handler ใดเรียกใช้เลย:

```go
// mission_repo.go — defined แต่ไม่มีใครเรียก
func (r *MissionRepo) GetMissionByIncidentID(ctx context.Context, incidentID string) (*models.MissionAssignment, error) {
    // queries incident-index GSI
}
```

**หลักฐาน:** `grep` ทั้ง project (`cmd/**/*.go`) ไม่พบการเรียกใช้ method นี้

**การแก้ไข:** ลบ method นี้ออก (และดู ISSUE-08 เกี่ยวกับ GSI ที่ตามมา)

---

### 🟡 ISSUE-06 — `MissionRepo.CreateMissionIdempotent` — Dead Code

**ไฟล์ที่เกี่ยวข้อง:**

- `src/backend/internal/repository/mission_repo.go` (line 97–115)

**ปัญหา:**
`CreateMissionIdempotent` ถูก define แต่ไม่ถูกใช้ `mission-assigned-handler` ทำ idempotency ผ่าน `GetMissionByDispatchID` + `CreateMission` แทน:

```go
// mission_repo.go — defined แต่ไม่มีใครเรียก
func (r *MissionRepo) CreateMissionIdempotent(ctx context.Context, mission *models.MissionAssignment) error {
    // PutItem with ConditionExpression: attribute_not_exists(mission_id)
}

// mission-assigned-handler/main.go — ใช้ 2 steps แทน
existing, err := missionRepo.GetMissionByDispatchID(ctx, payload.DispatchID)  // step 1: check
...
missionRepo.CreateMission(ctx, mission)  // step 2: create (non-idempotent)
```

**การแก้ไข:** ลบ `CreateMissionIdempotent` ออก (หรือเปลี่ยน handler ให้ใช้ method นี้แทนเพื่อ atomic idempotency ที่แข็งแกร่งกว่า)

---

### 🟡 ISSUE-07 — `variable "incident_service_url"` — Orphaned Terraform Variable

**ไฟล์ที่เกี่ยวข้อง:**

- `terraform/variables.tf` (line 26–30)

**ปัญหา:**
`incident_service_url` ถูก declare ใน `variables.tf` แต่ไม่มี Lambda ใดใช้งานในปัจจุบัน:

```hcl
# variables.tf — ยังอยู่
variable "incident_service_url" {
  description = "URL of the IncidentTracking Service"
  type        = string
  default     = "http://localhost:9999"
}
```

**ประวัติ:** ตาม `plan/refactor/phase2-rescue-request.md` — IncidentTracking client ถูกแทนที่ด้วย RescueRequest client ใน Phase 2 และ `INCIDENT_SERVICE_URL` env var ถูกลบออกจากทุก Lambda แล้ว แต่ Terraform variable ยังไม่ถูกลบ

**ยืนยันจาก tfstate.backup:** Lambda `get-mission` เคยมี `INCIDENT_SERVICE_URL` แต่ปัจจุบัน `lambda.tf` ไม่มีแล้ว

**การแก้ไข:** ลบ `variable "incident_service_url"` ออกจาก `variables.tf`

---

### 🟡 ISSUE-08 — `incident-index` GSI ใน DynamoDB อาจ Unused

**ไฟล์ที่เกี่ยวข้อง:**

- `terraform/dynamodb.tf`

**ปัญหา:**
`incident-index` GSI ถูกสร้างบน `MissionAssignment` table สำหรับรองรับ `GetMissionByIncidentID` ซึ่งเป็น dead code (ISSUE-05):

```hcl
# dynamodb.tf
global_secondary_index {
  name            = "incident-index"    # ← รองรับ GetMissionByIncidentID ที่ไม่ใครเรียก
  hash_key        = "incident_id"
  projection_type = "ALL"
}
```

นอกจากนี้ เนื่องจาก `incident_id` เป็น empty string เสมอ (ISSUE-03) → GSI นี้ก็ไม่ useful อยู่ดีแม้จะมีการเรียก

**การแก้ไข:** ลบ GSI นี้ออก (พร้อมกับ attribute `incident_id` ใน table definition ถ้าไม่มีการใช้งานอื่น)

> **หมายเหตุ:** หาก ISSUE-03 ได้รับการแก้ไข (IncidentID ถูก populate จริง) และมีแผนใช้งาน `GetMissionByIncidentID` ในอนาคต ให้คง GSI นี้ไว้

---

### 🟡 ISSUE-09 — Hardcoded Default Tokens ใน `variables.tf`

**ไฟล์ที่เกี่ยวข้อง:**

- `terraform/variables.tf`

**ปัญหา:**
มีค่า `default` ที่เป็น hardcoded credential สองค่า:

```hcl
variable "api_key_value" {
  ...
  default     = "mission-progress-api-key-2024"  # ← hardcoded API key
}

variable "rescue_team_service_token" {
  ...
  default     = "mock-dispatcher-token-123"  # ← hardcoded token
}
```

**ความเสี่ยง:** หาก deploy โดยไม่ตั้งค่า override ใน `terraform.tfvars` → credential ที่ทุกคนรู้ถูกใช้ใน production

**การแก้ไข:** เปลี่ยน default เป็น `""` และบังคับให้ระบุค่าจริงก่อน deploy:

```hcl
variable "api_key_value" {
  description = "API Key value for authentication"
  type        = string
  sensitive   = true
  # ไม่มี default — บังคับให้ระบุ
}
```

---

### 🟢 ISSUE-10 — Empty Directory `internal/statemachine/response/`

**ไฟล์ที่เกี่ยวข้อง:**

- `src/backend/internal/statemachine/response/`

**ปัญหา:**
Directory ว่างเปล่า ไม่มีไฟล์ใด อาจเป็น leftover จาก design เดิมที่ไม่ได้ implement:

```
src/backend/internal/statemachine/
├── statemachine.go
└── response/           ← ว่าง
    └── (ไม่มีไฟล์เลย)
```

**การแก้ไข:** ลบ directory ว่างนี้ออก:

```bash
rmdir src/backend/internal/statemachine/response
```

---

## แผนการแก้ไข (Prioritized)

### รอบที่ 1 — Critical Fixes (แก้ทันที)

| ลำดับ | Action                                               | ไฟล์ที่แก้ไข                |
| ----- | ---------------------------------------------------- | --------------------------- |
| 1     | เพิ่ม `variable "manage_dispatch_service_token"`     | `terraform/variables.tf`    |
| 2     | เพิ่ม `MANAGE_DISPATCH_SERVICE_TOKEN` ใน get-mission | `terraform/lambda.tf`       |
| 3     | แก้ S3 key path ให้ใช้ `mission.MissionID`           | `cmd/presigned-url/main.go` |

### รอบที่ 2 — Data Fix (ต้องพิจารณา contract กับ Manage Dispatch ก่อน)

| ลำดับ | Action                                                   | ไฟล์ที่แก้ไข                                                  |
| ----- | -------------------------------------------------------- | ------------------------------------------------------------- |
| 4     | เพิ่ม RescueRequestClient ใน mission-assigned-handler    | `cmd/mission-assigned-handler/main.go`, `terraform/lambda.tf` |
| 4     | หรือ: lazy-fill IncidentID ใน get-mission หาก field ว่าง | `cmd/get-mission/main.go`                                     |

### รอบที่ 3 — Cleanup (ทำเพื่อ code hygiene)

| ลำดับ | Action                                                | ไฟล์ที่แก้ไข                                  |
| ----- | ----------------------------------------------------- | --------------------------------------------- |
| 5     | ลบ `GetMissionByIncidentID` method                    | `internal/repository/mission_repo.go`         |
| 6     | ลบ `CreateMissionIdempotent` method                   | `internal/repository/mission_repo.go`         |
| 7     | ลบ `variable "incident_service_url"`                  | `terraform/variables.tf`                      |
| 8     | ลบ `incident-index` GSI (พร้อมกับ attribute ถ้าลบได้) | `terraform/dynamodb.tf`                       |
| 9     | ลบ hardcoded default tokens                           | `terraform/variables.tf`                      |
| 10    | ลบ empty directory                                    | `src/backend/internal/statemachine/response/` |

---

## Dependency Map ของปัญหา

```
ISSUE-03 (IncidentID ว่าง)
    └─→ ISSUE-04 (S3 key double-slash) ← แก้ได้อิสระ
    └─→ ISSUE-08 (incident-index GSI unused)
    └─→ ผล: events ที่ publish มี incident_id = ""

ISSUE-01 (MANAGE_DISPATCH_SERVICE_TOKEN ขาด)
    └─→ ISSUE-02 (variable ขาดใน variables.tf) ← ต้องแก้คู่กัน

ISSUE-05 (GetMissionByIncidentID dead)
    └─→ ISSUE-08 (incident-index GSI unused) ← ลบได้หลังลบ method
```
