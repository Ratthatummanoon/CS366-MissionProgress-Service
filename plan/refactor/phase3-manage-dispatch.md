# Phase 3 Refactor — Integration กับ Manage Dispatch Service

---

## ภาพรวมของการเปลี่ยนแปลง

Phase นี้คือการเชื่อม MissionProgress Service เข้ากับ **Manage Dispatch Service** อย่างถูกต้อง  
ปัจจุบัน `mission-assigned-handler` รับ payload ที่ไม่ตรงกับ event จริงของ Manage Dispatch และ model ยังขาด field สำคัญหลายตัว

| หัวข้อ                             | เดิม                                          | ใหม่                                                          |
| ---------------------------------- | --------------------------------------------- | ------------------------------------------------------------- |
| Event payload ของ handler          | `mission_id`, `rescue_unit_id`, `incident_id` | **`dispatchId`, `teamId`, `requestId`** (ตาม Manage Dispatch) |
| Event `detail-type` ใน EventBridge | `"MissionAssignedEvent"`                      | **`"DispatchOrderCreated"`**                                  |
| Event `source` ใน EventBridge      | `"dispatch-management-service"`               | **`"ManageDispatchService"`**                                 |
| `mission_id`                       | ส่งมาจาก payload                              | **generate UUID ใน handler**                                  |
| `dispatch_id` ใน model             | ไม่มี                                         | **เพิ่ม field ใหม่**                                          |
| `priority_level` ใน model          | ไม่มี                                         | **เพิ่ม field ใหม่**                                          |
| `incident_id` ใน handler           | required จาก payload                          | **optional — ดึงจาก RescueRequest Service**                   |
| Client สำหรับ Manage Dispatch      | ไม่มี                                         | **สร้าง `manage_dispatch_client.go` ใหม่**                    |
| `get-mission` response             | ไม่มีข้อมูล dispatch                          | **เพิ่ม `dispatch_id`, `dispatch_status`, `priority_level`**  |
| Terraform env var (get-mission)    | ไม่มี `MANAGE_DISPATCH_SERVICE_URL`           | **เพิ่ม env var**                                             |

---

## Gap Analysis — สิ่งที่ต้องแก้ไขทั้งหมด

| #   | ปัญหา                                                                                                       | ไฟล์ที่เกี่ยวข้อง                      |
| --- | ----------------------------------------------------------------------------------------------------------- | -------------------------------------- |
| 1   | `MissionAssignedPayload` ไม่ตรงกับ `DispatchOrderCreated` ของ Manage Dispatch                               | `cmd/mission-assigned-handler/main.go` |
| 2   | EventBridge rule ใช้ `detail-type: "MissionAssignedEvent"` แต่ Manage Dispatch ส่ง `"DispatchOrderCreated"` | `terraform/eventbridge.tf`             |
| 3   | ไม่มี `dispatch_id`, `priority_level` ใน `MissionAssignment` model                                          | `internal/models/mission.go`           |
| 4   | `get-mission` ไม่ return ข้อมูล dispatch ให้ caller                                                         | `cmd/get-mission/main.go`              |
| 5   | ไม่มี `ManageDispatchClient` สำหรับ fetch ข้อมูล dispatch                                                   | `internal/client/` (ไฟล์ใหม่)          |
| 6   | `GetMissionResponse` ขาด `dispatch_id`, `dispatch_status`, `priority_level`                                 | `internal/models/requests.go`          |
| 7   | Lambda `get-mission` ไม่มี env var `MANAGE_DISPATCH_SERVICE_URL`                                            | `terraform/lambda.tf`                  |
| 8   | `variables.tf` ไม่มี `manage_dispatch_service_url`                                                          | `terraform/variables.tf`               |
| 9   | Seed data ไม่มี `dispatch_id`                                                                               | `script/seed-data.sh`                  |

---

## สิ่งที่ทำเสร็จแล้ว (ก่อน Phase 3)

| ส่วน                                                        | สถานะ   | หมายเหตุ                                     |
| ----------------------------------------------------------- | ------- | -------------------------------------------- |
| `models/mission.go` — มี `request_id`                       | ✅ Done | Phase 2                                      |
| `client/rescue_request_client.go`                           | ✅ Done | Phase 2                                      |
| `terraform/variables.tf` — มี `dispatch_sqs_arn`            | ✅ Done | เตรียมไว้แล้ว                                |
| EventBridge rule `mission_resolved_dispatch` → Dispatch SQS | ✅ Done | ส่ง RESOLVED event ไปหา Manage Dispatch แล้ว |

---

## ขั้นตอนที่ต้องทำทั้งหมด

---

### ขั้นที่ A — อัปเดต `src/backend/internal/models/mission.go`

เพิ่ม `DispatchID` และ `PriorityLevel` เพื่อเก็บข้อมูลที่ได้จาก Manage Dispatch Service

```go
// MissionAssignment represents a mission assignment record in DynamoDB.
type MissionAssignment struct {
    MissionID         string `json:"mission_id"          dynamodbav:"mission_id"`
    DispatchID        string `json:"dispatch_id"         dynamodbav:"dispatch_id"`        // เพิ่มใหม่
    RequestID         string `json:"request_id"          dynamodbav:"request_id"`
    IncidentID        string `json:"incident_id"         dynamodbav:"incident_id"`
    RescueTeamID      string `json:"rescue_team_id"      dynamodbav:"rescue_team_id"`
    PriorityLevel     int    `json:"priority_level"      dynamodbav:"priority_level"`     // เพิ่มใหม่
    CurrentStatus     string `json:"current_status"      dynamodbav:"current_status"`
    LatestImpactLevel int    `json:"latest_impact_level" dynamodbav:"latest_impact_level"`
    StartedAt         string `json:"started_at"          dynamodbav:"started_at"`
    LastUpdatedAt     string `json:"last_updated_at"     dynamodbav:"last_updated_at"`
}
```

> **หมายเหตุ:** `DispatchID` เป็น reference ไปยัง Manage Dispatch Service  
> `PriorityLevel` มาจาก event payload (ถ้ามี) หรือ default เป็น `0`

---

### ขั้นที่ B — อัปเดต `src/backend/cmd/mission-assigned-handler/main.go`

**ปัญหา:** `MissionAssignedPayload` ปัจจุบันคาดหวัง `mission_id`, `rescue_unit_id`, `incident_id` แต่ Manage Dispatch ส่ง:

```json
{
  "requestId": "REQ-10293",
  "teamId": "TEAM-005",
  "dispatchId": "DSP-88472",
  "status": "PENDING",
  "dispatchedAt": "2026-03-03T14:30:00Z"
}
```

**สิ่งที่ต้องเปลี่ยน:**

#### B.1 — เปลี่ยน `MissionAssignedPayload` struct

```go
// MissionAssignedPayload คือ payload จาก DispatchOrderCreated event ของ Manage Dispatch Service
// field names ใช้ camelCase ตาม Manage Dispatch contract
type MissionAssignedPayload struct {
    DispatchID   string `json:"dispatchId"`
    RequestID    string `json:"requestId"`
    TeamID       string `json:"teamId"`
    PriorityLevel int   `json:"priorityLevel"`
    Status       string `json:"status"`
    DispatchedAt string `json:"dispatchedAt"`
}
```

#### B.2 — เปลี่ยน validation ใน handler

```go
// เดิม (ตรวจสอบ mission_id, rescue_unit_id, incident_id)
if payload.MissionID == "" || payload.RequestID == "" || payload.IncidentID == "" || payload.RescueUnitID == "" {

// ใหม่ (ตรวจสอบ dispatchId, requestId, teamId เท่านั้น — incidentId เป็น optional)
if payload.DispatchID == "" || payload.RequestID == "" || payload.TeamID == "" {
```

#### B.3 — สร้าง `mission_id` ใน handler แทนการรับจาก payload

```go
// Generate mission_id ใหม่เพราะ Manage Dispatch ไม่ส่ง mission_id มาให้
generatedMissionID := "MISS-" + uuid.New().String()[:8]
```

#### B.4 — map ฟิลด์ให้ถูกต้อง

```go
mission := &models.MissionAssignment{
    MissionID:         generatedMissionID,       // generate ใหม่
    DispatchID:        payload.DispatchID,        // DSP-88472
    RequestID:         payload.RequestID,         // REQ-10293
    IncidentID:        "",                        // ไม่ได้รับจาก Manage Dispatch — ปล่อยว่าง
    RescueTeamID:      payload.TeamID,            // TEAM-005
    PriorityLevel:     payload.PriorityLevel,     // 1
    CurrentStatus:     "DISPATCHED",
    LatestImpactLevel: 0,
    StartedAt:         payload.DispatchedAt,
    LastUpdatedAt:     payload.DispatchedAt,
}
```

#### B.5 — อัปเดต timeline entry description

```go
// เดิม
Description: fmt.Sprintf("Mission assigned to %s", payload.RescueUnitID),

// ใหม่
Description: fmt.Sprintf("Dispatch %s assigned to team %s", payload.DispatchID, payload.TeamID),
```

> **เหตุผลที่ไม่ fetch `incident_id` ใน handler:**  
> Handler เป็น async worker — ไม่ควรทำ synchronous HTTP call เพิ่มเพื่อหลีกเลี่ยง latency และ tight coupling  
> `incident_id` จะถูก populate lazy ผ่าน `get-mission` ซึ่งมี `RescueRequestClient` อยู่แล้ว

---

### ขั้นที่ C — อัปเดต EventBridge Rule ใน `terraform/eventbridge.tf`

เปลี่ยน event pattern ให้ตรงกับ event ที่ Manage Dispatch Service ส่งออกมาจริง

```hcl
# เดิม
resource "aws_cloudwatch_event_rule" "mission_assigned" {
  name        = "mission-assigned-rule"
  description = "Capture MissionAssignedEvent from Dispatch service"

  event_pattern = jsonencode({
    source      = ["dispatch-management-service"]
    detail-type = ["MissionAssignedEvent"]
  })
}

# ใหม่
resource "aws_cloudwatch_event_rule" "mission_assigned" {
  name        = "mission-assigned-rule"
  description = "Capture DispatchOrderCreated from Manage Dispatch Service"

  event_pattern = jsonencode({
    source      = ["ManageDispatchService"]
    detail-type = ["DispatchOrderCreated"]
  })
}
```

> **หมายเหตุ:** ต้องประสาน `source` และ `detail-type` กับ Manage Dispatch Service team  
> ให้ตรงกับที่ฝั่งนั้น publish ไปยัง EventBridge

---

### ขั้นที่ D — สร้าง `src/backend/internal/client/manage_dispatch_client.go` (ไฟล์ใหม่)

สร้าง client สำหรับเรียก Manage Dispatch Service  
อ้างอิง: `GET /v1/dispatches?teamId={teamId}` (Sync Contract #2)

```go
package client

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/models"
)

const (
    mdMaxRetries        = 2
    mdBackoffBase       = 100 * time.Millisecond
    mdPerRequestTimeout = 800 * time.Millisecond
)

// ManageDispatchClient calls the Manage Dispatch Service.
type ManageDispatchClient struct {
    baseURL    string
    httpClient *http.Client
}

// NewManageDispatchClient creates a new client.
// Reads MANAGE_DISPATCH_SERVICE_URL from env.
func NewManageDispatchClient() *ManageDispatchClient {
    baseURL := os.Getenv("MANAGE_DISPATCH_SERVICE_URL")
    if baseURL == "" {
        baseURL = "http://localhost:9997"
    }
    return &ManageDispatchClient{
        baseURL: baseURL,
        httpClient: &http.Client{
            Timeout: mdPerRequestTimeout,
        },
    }
}

// GetDispatchByTeamAndRequest ดึงข้อมูล dispatch record สำหรับทีมที่ระบุ
// กรอง status=PENDING และ ACCEPT โดย caller
// Endpoint: GET /v1/dispatches?teamId={teamId}
// Returns nil on failure (degraded mode).
func (c *ManageDispatchClient) GetDispatchByTeamAndRequest(teamID string) *models.DispatchDetail {
    url := fmt.Sprintf("%s/v1/dispatches?teamId=%s", c.baseURL, teamID)

    var lastErr error
    for attempt := 0; attempt <= mdMaxRetries; attempt++ {
        if attempt > 0 {
            backoff := mdBackoffBase * (1 << (attempt - 1))
            log.Printf("INFO: Retry %d/%d for ManageDispatchService after %v", attempt, mdMaxRetries, backoff)
            time.Sleep(backoff)
        }

        req, err := http.NewRequest(http.MethodGet, url, nil)
        if err != nil {
            lastErr = err
            continue
        }
        req.Header.Set("Accept", "application/json")

        resp, err := c.httpClient.Do(req)
        if err != nil {
            lastErr = err
            continue
        }
        defer resp.Body.Close()

        if resp.StatusCode == http.StatusNotFound {
            return nil
        }
        if resp.StatusCode >= 500 {
            lastErr = fmt.Errorf("ManageDispatchService returned %d", resp.StatusCode)
            continue
        }
        if resp.StatusCode != http.StatusOK {
            log.Printf("WARN: ManageDispatchService returned unexpected status %d", resp.StatusCode)
            return nil
        }

        var result models.DispatchListResponse
        if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
            log.Printf("WARN: decode ManageDispatchService response: %v", err)
            return nil
        }

        // หา dispatch record ที่ตรงกับ requestId ที่ต้องการ
        return &result
    }

    log.Printf("WARN: ManageDispatchService unavailable after %d retries: %v", mdMaxRetries, lastErr)
    return nil
}
```

---

### ขั้นที่ E — เพิ่ม model ใหม่ใน `src/backend/internal/models/requests.go`

#### E.1 — เพิ่ม `DispatchDetail` struct (ใช้กับ ManageDispatchClient)

```go
// DispatchItem คือข้อมูลคำสั่งการ 1 รายการจาก Manage Dispatch Service
type DispatchItem struct {
    DispatchID    string `json:"dispatchId"`
    RequestID     string `json:"requestId"`
    Status        string `json:"status"`         // PENDING / ACCEPT / DECLINE
    PriorityLevel int    `json:"priorityLevel"`
    DispatchedAt  string `json:"dispatchedAt"`
}

// DispatchListResponse คือ response จาก GET /v1/dispatches?teamId=...
type DispatchListResponse struct {
    TeamID string         `json:"teamId"`
    Items  []DispatchItem `json:"items"`
}

// DispatchDetail คือข้อมูล dispatch ที่ embed ใน GetMissionResponse
type DispatchDetail struct {
    DispatchID     string `json:"dispatch_id"`
    DispatchStatus string `json:"dispatch_status"`
    PriorityLevel  int    `json:"priority_level"`
    DispatchedAt   string `json:"dispatched_at"`
}
```

#### E.2 — เพิ่ม field ใน `GetMissionResponse`

```go
// GetMissionResponse is the response for GET /missions/{request_id}.
type GetMissionResponse struct {
    RequestID         string          `json:"request_id"`
    IncidentID        string          `json:"incident_id"`
    MissionID         string          `json:"mission_id"`
    DispatchID        string          `json:"dispatch_id,omitempty"`        // เพิ่มใหม่
    RescueTeamID      string          `json:"rescue_team_id"`
    PriorityLevel     int             `json:"priority_level,omitempty"`     // เพิ่มใหม่
    DispatchStatus    string          `json:"dispatch_status,omitempty"`    // เพิ่มใหม่
    CurrentStatus     string          `json:"current_status"`
    LatestImpactLevel int             `json:"latest_impact_level"`
    StartedAt         string          `json:"started_at"`
    LastUpdatedAt     string          `json:"last_updated_at"`
    Description       string          `json:"description,omitempty"`
    Location          string          `json:"location,omitempty"`
    IncidentType      string          `json:"incident_type,omitempty"`
    Timeline          []TimelineEntry `json:"timeline"`
    DataSource        string          `json:"data_source"`
}
```

---

### ขั้นที่ F — อัปเดต `src/backend/cmd/get-mission/main.go`

เพิ่ม `manageDispatchClient` และ enrich response ด้วยข้อมูล dispatch

#### F.1 — เพิ่ม client ใน `var` และ `init()`

```go
var (
    missionRepo          *repository.MissionRepo
    timelineRepo         *repository.TimelineRepo
    rescueRequestClient  *client.RescueRequestClient
    manageDispatchClient *client.ManageDispatchClient  // เพิ่มใหม่
)

func init() {
    // ... (เหมือนเดิม)
    rescueRequestClient  = client.NewRescueRequestClient()
    manageDispatchClient = client.NewManageDispatchClient()  // เพิ่มใหม่
}
```

#### F.2 — เพิ่ม dispatch enrichment หลัง step 3 (ใน handler)

```go
// 3b. Call Manage Dispatch Service เพื่อดึงสถานะล่าสุดของ dispatch (degraded mode on failure)
var dispatchStatus, dispatchedAt string
var priorityLevel int

if mission.DispatchID != "" {
    dispatchList := manageDispatchClient.GetDispatchByTeamAndRequest(mission.RescueTeamID)
    if dispatchList != nil {
        for _, item := range dispatchList.Items {
            if item.DispatchID == mission.DispatchID {
                dispatchStatus = item.Status
                priorityLevel  = item.PriorityLevel
                dispatchedAt   = item.DispatchedAt
                break
            }
        }
    } else {
        log.Printf("INFO: ManageDispatchService unavailable - skipping dispatch enrichment for missionID=%s", mission.MissionID)
    }
}
```

#### F.3 — อัปเดต response struct

```go
return response.JSON(200, models.GetMissionResponse{
    RequestID:         mission.RequestID,
    IncidentID:        mission.IncidentID,
    MissionID:         mission.MissionID,
    DispatchID:        mission.DispatchID,          // เพิ่มใหม่
    RescueTeamID:      mission.RescueTeamID,
    PriorityLevel:     priorityLevel,               // เพิ่มใหม่
    DispatchStatus:    dispatchStatus,               // เพิ่มใหม่
    CurrentStatus:     mission.CurrentStatus,
    LatestImpactLevel: mission.LatestImpactLevel,
    StartedAt:         mission.StartedAt,
    LastUpdatedAt:     mission.LastUpdatedAt,
    Description:       description,
    Location:          location,
    IncidentType:      incidentType,
    Timeline:          timeline,
    DataSource:        dataSource,
}), nil
```

---

### ขั้นที่ G — อัปเดต `terraform/variables.tf`

เพิ่ม variable สำหรับ Manage Dispatch Service

```hcl
variable "manage_dispatch_service_url" {
  description = "URL of the Manage Dispatch Service"
  type        = string
  default     = "http://localhost:9997"
}
```

---

### ขั้นที่ H — อัปเดต `terraform/lambda.tf`

เพิ่ม env var `MANAGE_DISPATCH_SERVICE_URL` ใน Lambda `get_mission`

```hcl
# เดิม
environment {
  variables = {
    TABLE_MISSION                = aws_dynamodb_table.mission_assignment.name
    TABLE_TIMELINE               = aws_dynamodb_table.mission_timeline.name
    RESCUE_REQUEST_SERVICE_URL   = var.rescue_request_service_url
    RESCUE_REQUEST_SERVICE_TOKEN = var.rescue_request_service_token
  }
}

# ใหม่
environment {
  variables = {
    TABLE_MISSION                = aws_dynamodb_table.mission_assignment.name
    TABLE_TIMELINE               = aws_dynamodb_table.mission_timeline.name
    RESCUE_REQUEST_SERVICE_URL   = var.rescue_request_service_url
    RESCUE_REQUEST_SERVICE_TOKEN = var.rescue_request_service_token
    MANAGE_DISPATCH_SERVICE_URL  = var.manage_dispatch_service_url   # เพิ่มใหม่
  }
}
```

---

### ขั้นที่ I — อัปเดต `script/seed-data.sh`

เพิ่ม `dispatch_id` และ `priority_level` ใน seed data items

```bash
# เพิ่ม attribute ใน put-item แต่ละรายการ
'"dispatch_id":    {"S": "DSP-00001"},' \
'"priority_level": {"N": "1"},' \
```

---

### ขั้นที่ J — Build & Verify

```bash
cd src/backend
go build ./...
```

ตรวจสอบว่าไม่มี compile error ก่อน deploy

---

## สรุปลำดับการทำงาน

```
A → B → C → D → E → F → G → H → I → J
```

| ขั้น | ไฟล์ที่แก้ / สร้าง                          | ประเภท    |
| ---- | ------------------------------------------- | --------- |
| A    | `internal/models/mission.go`                | แก้       |
| B    | `cmd/mission-assigned-handler/main.go`      | แก้       |
| C    | `terraform/eventbridge.tf`                  | แก้       |
| D    | `internal/client/manage_dispatch_client.go` | สร้างใหม่ |
| E    | `internal/models/requests.go`               | แก้       |
| F    | `cmd/get-mission/main.go`                   | แก้       |
| G    | `terraform/variables.tf`                    | แก้       |
| H    | `terraform/lambda.tf`                       | แก้       |
| I    | `script/seed-data.sh`                       | แก้       |
| J    | `go build ./...`                            | verify    |

---

## Dependency / ข้อควรระวัง

| ประเด็น                                              | รายละเอียด                                                                                                                               |
| ---------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| **EventBridge source**                               | ต้องประสานกับ Manage Dispatch Service team ว่า publish event ด้วย `source` และ `detail-type` อะไรกันแน่                                  |
| **`incident_id` ว่างเปล่า**                          | หลัง Phase 3 missions ใหม่จะมี `incident_id = ""` — `get-mission` ยังคืน `incident_id` ว่างในกรณีนี้ (acceptable)                        |
| **Dispatch enrichment เป็น degraded mode**           | ถ้า Manage Dispatch ไม่ตอบสนอง `get-mission` ยัง return ได้แต่ `dispatch_status` จะว่าง                                                  |
| **Idempotency ของ handler**                          | `CreateMissionIdempotent` ทำงานด้วย `dispatch_id` เป็น idempotency key ได้ — ต้องตรวจสอบว่า condition expression ใน repo ครอบคลุมกรณีนี้ |
| **`DispatchOrderCreated` vs `DispatchTeamRejected`** | handler ควร ignore `REJECTED` events หรือ log แล้วส่งต่อเป็น no-op                                                                       |
