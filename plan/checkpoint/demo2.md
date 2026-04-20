### Plan Demo 2 — Full Service Integration

ทำทุกอย่างตาม proposals ให้ครบ: เพิ่ม 3 Lambda ใหม่ (presigned-url, list-missions, **mission-assigned-handler**), S3 Evidence Bucket, เปลี่ยน EventBridge targets จาก CloudWatch ไปยัง **SQS queues** ของเพื่อนจริง, ต่อ IncidentTracking จริง (degraded fallback คงอยู่), รองรับ `image_key` ใน progress report

---

### Steps

#### Phase A: S3 + Presigned URL Lambda

**Step 1: สร้าง S3 Evidence Bucket ใน Terraform**
- สร้างไฟล์ `terraform/s3.tf`
- `aws_s3_bucket` ชื่อ `${var.project_name}-evidence-${data.aws_caller_identity.current.account_id}`
- CORS: allow `PUT` method จาก `*` origin (สำหรับ presigned URL upload)
- Block public access เปิด (ใช้ presigned URL เท่านั้น)
- เพิ่ม `data "aws_caller_identity" "current" {}` ใน main.tf

**Step 2: เพิ่ม presigned URL request/response models**
- ไฟล์: requests.go
- เพิ่ม:
  ```
  PresignedURLRequest  { FileName, ContentType }
  PresignedURLResponse { UploadURL, ImageKey, ExpiresIn, Message }
  ```

**Step 3: สร้าง Presigned URL Lambda (Go)**
- สร้าง `src/backend/cmd/presigned-url/main.go`
- Logic:
  1. Parse `incident_id` จาก path + `X-Rescue-Team-ID` จาก authorizer
  2. Parse body: `{ "file_name": "...", "content_type": "image/jpeg" }`
  3. Validate `content_type` ∈ `{image/jpeg, image/png, image/webp}` → 400 `INVALID_CONTENT_TYPE` ถ้าไม่ตรง
  4. **ตรวจว่า mission มีอยู่** — เรียก `missionRepo.GetMissionByIncidentID()` → 404 `INCIDENT_NOT_FOUND` ถ้าไม่พบ
  5. สร้าง S3 key: `evidence/{incident_id}/{team_id}/{unix_timestamp}-{file_name}`
  6. Generate presigned PUT URL ด้วย `s3.PresignClient` (expire 300s)
  7. Return `{ upload_url, image_key, expires_in: 300, message }`
  8. **Degraded**: ถ้า S3 presigned generation ล้มเหลว → return 500 แต่ mission ยังทำงานได้ (text-only fallback)
- ต้อง `go get github.com/aws/aws-sdk-go-v2/service/s3`

**Step 4: เพิ่ม Terraform + API Gateway สำหรับ presigned-url**
- lambda.tf: เพิ่ม `aws_lambda_function.presigned_url` (env: `EVIDENCE_BUCKET`, `TABLE_MISSION`) + permission
- api_gateway.tf:
  - Resource: `/incidents/{incident_id}/presigned-url`
  - Method: POST (CUSTOM auth)
  - Integration: AWS_PROXY → presigned-url Lambda
  - OPTIONS (CORS)
- อัปเดต deployment `triggers` ให้รวม resources ใหม่

---

#### Phase B: List Missions API (parallel กับ Phase A)

**Step 5: เพิ่ม Repository function `GetMissionsByTeamID()`**
- ไฟล์: mission_repo.go
- Query GSI `team-index` (PK = `rescue_team_id`)
- Optional FilterExpression: `current_status = :st` ถ้ามี status filter
- `context.WithTimeout(5s)` เหมือน functions อื่น

**Step 6: เพิ่ม ListMissionsResponse model**
- ไฟล์: requests.go
- เพิ่ม `ListMissionsResponse { TeamID, TotalMissions, Missions []MissionAssignment }`

**Step 7: สร้าง List Missions Lambda (Go)**
- สร้าง `src/backend/cmd/list-missions/main.go`
- **ตัดสินใจ**: ใช้ `X-Rescue-Team-ID` จาก authorizer เป็น `team_id` — ไม่ต้อง query param `team_id` แยก เพราะ:
  - ป้องกัน team อื่นดึงข้อมูลทีมอื่น (security)
  - `X-Rescue-Team-ID` มีอยู่แล้วจาก authorizer
- Parse optional query param `status` เป็น filter
- Validate `status` ถ้ามีค่า → ต้องเป็น valid status ใน state machine
- ถ้าไม่พบ missions → return `200 OK` กับ `missions: []` (ไม่ใช่ 404)

**Step 8: เพิ่ม Terraform + API Gateway สำหรับ list-missions**
- lambda.tf: เพิ่ม `aws_lambda_function.list_missions` (env: `TABLE_MISSION`) + permission
- api_gateway.tf:
  - ใช้ resource `/incidents` ที่มีอยู่แล้ว
  - Method: GET (CUSTOM auth)
  - Integration: AWS_PROXY → list-missions Lambda
  - OPTIONS (CORS) สำหรับ `/incidents`
- อัปเดต deployment `triggers`

---

#### Phase C: MissionAssignedEvent Handler — รับ event จาก Dispatch (parallel กับ Phase A, B)

**Step 9: สร้าง Mission Assigned Handler Lambda (Go)**
- สร้าง `src/backend/cmd/mission-assigned-handler/main.go`
- **ทำอะไร**: รับ `MissionAssignedEvent` จาก Dispatch ผ่าน EventBridge → สร้าง mission record อัตโนมัติ
- Logic:
  1. Parse EventBridge event payload (CloudWatch Events format)
  2. Extract fields: `mission_id`, `rescue_unit_id` (→ rescue_team_id), `incident_id`, `assigned_at`
  3. เรียก `missionRepo.CreateMission()` สร้าง `MissionAssignment` ด้วย `current_status = "DISPATCHED"`
  4. เรียก `timelineRepo.AddTimelineEntry()` สร้าง timeline entry แรก (action_type = `MISSION_ASSIGNED`)
  5. **Idempotency**: ใช้ DynamoDB conditional write `attribute_not_exists(mission_id)` — ถ้ามีอยู่แล้วจะ skip (ไม่ error)
  6. **Degraded mode**: ถ้า DynamoDB write fail → Lambda return error → EventBridge จะ retry อัตโนมัติ (24 ชม.)
- payload จาก Dispatch ที่คาดหวัง (ตาม proposal 06):
  ```json
  {
    "mission_id": "MSN-001",
    "rescue_unit_id": "TEAM-ALPHA",
    "incident_id": "INC-001",
    "assigned_at": "2025-06-14T08:45:00Z"
  }
  ```

**Step 10: เพิ่ม Terraform สำหรับ mission-assigned-handler**
- lambda.tf: เพิ่ม `aws_lambda_function.mission_assigned_handler` (env: `TABLE_MISSION`, `TABLE_TIMELINE`) + EventBridge permission
- eventbridge.tf: เพิ่ม rule สำหรับ `MissionAssignedEvent` — กรอง event จาก Dispatch
  ```hcl
  event_pattern = {
    source      = ["dispatch-management-service"]
    detail-type = ["MissionAssignedEvent"]
  }
  ```
- Target: mission-assigned-handler Lambda
- **ตัดสินใจ**: ถ้ายังไม่ได้ต่อ Dispatch จริง → ยังใช้ seed-data.sh ได้เหมือนเดิม แต่ infrastructure พร้อมรับ event แล้ว

---

#### Phase D: เพิ่ม `image_key` ใน Report Progress Flow

**Step 11: เพิ่ม `image_key` field ใน request/response**
- ไฟล์: requests.go
  - `ReportProgressRequest`: เพิ่ม `ImageKey string \`json:"image_key,omitempty"\``
- ไฟล์: timeline.go
  - `TimelineEntry`: เพิ่ม `ImageKey string \`json:"image_key,omitempty" dynamodbav:"image_key,omitempty"\``

**Step 12: อัปเดต report-progress handler ให้ส่ง image_key ไป timeline**
- ไฟล์: main.go
- ตรง step 7 (สร้าง timeline entry): เพิ่ม `ImageKey: req.ImageKey`
- ไม่ต้อง validate image_key — ถ้ามีค่าก็บันทึก, ถ้าไม่มีก็ข้ามไป (optional)

---

#### Phase E: EventBridge Real Targets → SQS (depends on Phase A-D build เสร็จ)

**ตัดสินใจ: ใช้ SQS เป็น target** — เพราะ:
- Decouple กว่า Lambda direct invoke
- มี built-in dead-letter queue
- เพื่อนจัดการ processing pace เอง
- ไม่ต้อง cross-account Lambda permission ซับซ้อน

**Step 13: เพิ่ม Terraform variables สำหรับ consumer SQS ARNs**
- ไฟล์: variables.tf
  ```
  variable "incident_tracking_sqs_arn" { default = "" }
  variable "dispatch_sqs_arn" { default = "" }
  variable "prioritization_sqs_arn" { default = "" }
  ```
- `default = ""` → deploy ได้แม้ยังไม่มี ARN (ใช้ CloudWatch Logs เดิม)

**Step 14: เพิ่ม EventBridge → SQS targets (conditional)**
- ไฟล์: eventbridge.tf
- **MissionStatusChanged → IncidentTracking** (ถ้า ARN != ""):
  - ใช้ rule เดิม `mission_status_changed`
  - เพิ่ม target ไป SQS ของ IncidentTracking (ใช้ `count`)
- **MissionStatusChanged → Dispatch (เฉพาะ RESOLVED)** — **rule ใหม่**:
  - สร้าง rule ใหม่ `mission-resolved-dispatch-rule` ด้วย event pattern:
    ```json
    {
      "source": ["MissionProgressService"],
      "detail-type": ["MissionStatusChanged"],
      "detail": { "new_status": ["RESOLVED"] }
    }
    ```
  - Target: SQS ของ Dispatch (ใช้ `count`)
- **MissionBackupRequested → Prioritization** (ถ้า ARN != ""):
  - ใช้ rule เดิม `backup_requested`
  - เพิ่ม target ไป SQS ของ Prioritization
- **ImpactLevelUpdated → IncidentTracking + Prioritization** (ถ้า ARN != ""):
  - ใช้ rule เดิม `impact_level_updated`
  - เพิ่ม 2 targets: SQS IncidentTracking + SQS Prioritization
- **CloudWatch Logs targets คงอยู่ทั้งหมด** — สำหรับ monitoring

**Step 15: เพิ่ม SQS resource policy**
- ไฟล์: eventbridge.tf
- EventBridge ต้อง permission `sqs:SendMessage` → ใช้ `aws_sqs_queue_policy` หรือ IAM role
- ถ้าเพื่อนสร้าง SQS เอง → ขอให้เพื่อนเพิ่ม resource policy ด้วย
- ถ้า same account → ใช้ `aws_sqs_queue_policy` ให้ EventBridge send ได้

---

#### Phase F: IncidentTracking Real Connection

**Step 16: เปลี่ยน `incident_service_url` เป็น URL จริง**
- ไฟล์: variables.tf — เปลี่ยน default เมื่อได้ URL จากเพื่อน
- **ไม่ต้องแก้ code** — incident_client.go รองรับอยู่แล้ว (retry + degraded mode)
- ถ้ายังไม่มี URL: deploy ด้วย default เดิม → degraded mode อัตโนมัติ
- **ตรวจ response mapping**: ตรวจว่า response ของ Krittamet ตรงกับ `IncidentDetail` struct (fields: `incident_id`, `description`, `location`, `incident_type`) — ถ้าไม่ตรงแก้ struct หรือเพิ่ม mapping

---

#### Phase G: Build Script + Documentation

**Step 17: อัปเดต build script**
- ไฟล์: build.sh
- `FUNCTIONS` array เพิ่ม: `"presigned-url"`, `"list-missions"`, `"mission-assigned-handler"`
- รวม 7 Lambdas: report-progress, get-mission, authorizer, outbox-processor, presigned-url, list-missions, mission-assigned-handler

**Step 18: เขียน contract_demo2.md**
- ไฟล์: contract_demo2.md
- เนื้อหา:
  - Base URL + Auth headers
  - **4 sync endpoints** (GET mission, POST progress, POST presigned-url, GET list-missions) พร้อม request/response examples
  - **3 outbound async events** + real SQS consumer targets
  - **1 inbound async event** (MissionAssignedEvent from Dispatch)
  - Dependency: IncidentTracking URL, consumer SQS ARNs

**Step 19: เขียน demo script ใน demo2.md**
- **Scenario 1 — Sync GET (full mode)**: GET mission → `data_source: "full"` (IncidentTracking ต่อจริง)
- **Scenario 2 — Sync POST + Async events**: POST progress → event publish → ตรวจ CloudWatch + SQS ของเพื่อน
- **Scenario 3 — Async ถึงเพื่อน**: ให้เพื่อนยืนยันว่าได้รับ event + แสดงผลในระบบ
- **Scenario 4 — Degraded mode**: เปลี่ยน URL เป็น invalid → GET mission → `data_source: "partial"` + log retry
- **Scenario 5 — Outbox processor**: สร้าง PENDING entry → รอ 1 นาที → status → SENT

---

#### Phase H: Build, Deploy & Verify

**Step 20: Build & Test**
- build.sh → ต้อง compile 7 Lambdas สำเร็จ
- ทดสอบ presigned-url: ขอ URL → `curl -X PUT` upload ภาพจริง → ภาพอยู่ใน S3
- ทดสอบ list-missions: GET `/incidents` → missions ของ team
- ทดสอบ mission-assigned-handler: ส่ง test event ผ่าน `aws events put-events` → mission ถูกสร้าง

**Step 21: Deploy & Integration Test**
- deploy.sh → terraform apply
- ทดสอบ 5 demo scenarios ตามจริง

---

### Relevant Files

**ไฟล์ใหม่ (6):**

| ไฟล์ | คำอธิบาย |
|------|---------|
| `src/backend/cmd/presigned-url/main.go` | Presigned URL Lambda |
| `src/backend/cmd/list-missions/main.go` | List Missions Lambda |
| `src/backend/cmd/mission-assigned-handler/main.go` | รับ MissionAssignedEvent จาก Dispatch |
| `terraform/s3.tf` | S3 Evidence Bucket |
| contract_demo2.md | Demo 2 contract |
| demo2.md | Demo 2 script |

**ไฟล์ที่แก้ไข (10):**

| ไฟล์ | การเปลี่ยนแปลง |
|------|---------------|
| requests.go | +PresignedURLRequest/Response, +ListMissionsResponse, +ImageKey in ReportProgressRequest |
| timeline.go | +ImageKey field |
| mission_repo.go | +GetMissionsByTeamID(), +CreateMission conditional write |
| main.go | ส่ง image_key ไป timeline entry |
| go.mod | +S3 SDK dependency |
| main.tf | +aws_caller_identity data source |
| lambda.tf | +3 Lambda functions (presigned-url, list-missions, mission-assigned-handler) |
| api_gateway.tf | +2 routes + CORS (presigned-url, list-missions) + deployment triggers |
| eventbridge.tf | +SQS targets (conditional), +Dispatch RESOLVED rule, +MissionAssigned rule |
| variables.tf | +3 SQS ARN variables |
| build.sh | +3 functions ใน build list |

---

### Verification

1. **Build**: build.sh → 7 Lambdas compile
2. **Terraform plan**: แสดง S3, 3 Lambdas ใหม่, API routes, EventBridge targets
3. **Presigned URL**: POST → ได้ URL → `curl -X PUT -T photo.jpg "{url}"` → ภาพใน S3
4. **List missions**: GET `/incidents` → missions ของ team + test status filter
5. **MissionAssigned**: `aws events put-events` ด้วย MissionAssignedEvent → mission record ถูกสร้าง
6. **image_key flow**: POST presigned-url → upload → POST progress กับ image_key → timeline มี image_key
7. **Full mode**: GET mission → `data_source: "full"` (IncidentTracking ต่อจริง)
8. **Degraded mode**: เปลี่ยน URL invalid → `data_source: "partial"`
9. **Async → SQS**: POST progress → ตรวจ SQS ของเพื่อนมี message
10. **Dispatch filter**: POST progress เป็น RESOLVED → ตรวจว่า Dispatch SQS ได้รับ, POST progress เป็น ON_SITE → Dispatch SQS ไม่ได้รับ

---

### Decisions (ตัดสินใจแล้ว ไม่ต้องถามเพื่อน)

| หัวข้อ | Decision | เหตุผล |
|--------|----------|--------|
| Consumer target type | **SQS queues** | Decouple กว่า, มี DLQ ในตัว, เพื่อนจัดการ pace เอง |
| List missions team_id | ใช้ `X-Rescue-Team-ID` จาก authorizer | ป้องกัน team ดึงข้อมูล team อื่น |
| IncidentTracking URL | ใส่ผ่าน Terraform variable, default = mock | ไม่ต้องแก้ code, degraded mode ทำงานอัตโนมัติ |
| MissionAssigned source | `dispatch-management-service` | ตาม proposal 06 |
| Dispatch filter | Rule แยก ด้วย `detail.new_status: RESOLVED` | ตาม proposal 06 routing spec |
| S3 bucket access | Presigned URL only, block public | Security best practice |
| S3 failure | Return 500, mission ยังทำงานได้ (text-only) | ตาม dependency mapping #4 |
| CloudWatch Logs targets | คงไว้ทั้งหมด | Monitoring ไม่ควรลบ |
| MissionAssigned idempotency | `attribute_not_exists(mission_id)` | ป้องกัน duplicate creation |

---

