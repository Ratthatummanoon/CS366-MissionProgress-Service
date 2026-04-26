# Phase 4 Refactor — Integration กับ RescueTeam Service

---

## ภาพรวมของการเปลี่ยนแปลง

Phase นี้คือการเชื่อม MissionProgress Service เข้ากับ **RescueTeam Service** ซึ่งเป็น Source of Truth ของข้อมูลทีมกู้ภัย  
MissionProgress มี `rescue_team_id` อยู่ใน DynamoDB แล้ว แต่ยังไม่เคย query ข้อมูลจาก RescueTeam Service เลย

| หัวข้อ                           | เดิม                              | ใหม่                                                                             |
| -------------------------------- | --------------------------------- | -------------------------------------------------------------------------------- |
| ข้อมูลทีมใน `get-mission`        | มีแค่ `rescue_team_id` (ID เปล่า) | **เพิ่ม `team_name`, `team_type`, `capabilities`, `equipment`, `team_location`** |
| แจ้ง RescueTeam เมื่อ RESOLVED   | ไม่มี (team ยังค้างสถานะ BUSY)    | **เรียก `PATCH /v1/teams/{team_id}/status` → `AVAILABLE`**                       |
| Client สำหรับ RescueTeam         | ไม่มี                             | **สร้าง `rescue_team_client.go` ใหม่**                                           |
| Model ข้อมูลทีม                  | ไม่มี                             | **สร้าง `RescueTeamDetail` struct**                                              |
| Lambda `get-mission` env var     | ไม่มี `RESCUE_TEAM_SERVICE_URL`   | **เพิ่ม env var**                                                                |
| Lambda `report-progress` env var | ไม่มี `RESCUE_TEAM_SERVICE_URL`   | **เพิ่ม env var**                                                                |

---

## Gap Analysis — สิ่งที่ต้องแก้ไขทั้งหมด

| #   | ปัญหา                                                                                           | ไฟล์ที่เกี่ยวข้อง             |
| --- | ----------------------------------------------------------------------------------------------- | ----------------------------- |
| 1   | ไม่มี `RescueTeamClient`                                                                        | `internal/client/` (ไฟล์ใหม่) |
| 2   | `GetMissionResponse` ขาด `team_name`, `team_type`, `capabilities`, `equipment`, `team_location` | `internal/models/requests.go` |
| 3   | `get-mission` ไม่ enrich response ด้วยข้อมูลทีม                                                 | `cmd/get-mission/main.go`     |
| 4   | `report-progress` ไม่แจ้ง RescueTeam เมื่อ RESOLVED                                             | `cmd/report-progress/main.go` |
| 5   | Lambda `get-mission` ไม่มี `RESCUE_TEAM_SERVICE_URL` / `RESCUE_TEAM_SERVICE_TOKEN`              | `terraform/lambda.tf`         |
| 6   | Lambda `report-progress` ไม่มี `RESCUE_TEAM_SERVICE_URL` / `RESCUE_TEAM_SERVICE_TOKEN`          | `terraform/lambda.tf`         |
| 7   | `variables.tf` ไม่มี `rescue_team_service_url` / `rescue_team_service_token`                    | `terraform/variables.tf`      |

---

## สิ่งที่มีอยู่แล้ว (ก่อน Phase 4)

| ส่วน                                                        | สถานะ               | หมายเหตุ                               |
| ----------------------------------------------------------- | ------------------- | -------------------------------------- |
| `mission.RescueTeamID`                                      | ✅ มีแล้ว           | เก็บใน DynamoDB ตั้งแต่ต้น             |
| `MissionStatusChangedEvent` — publish `new_status=RESOLVED` | ✅ มีแล้ว           | แต่ไม่มีใครรับแล้ว notify RescueTeam   |
| EventBridge rule `mission_resolved_dispatch` → Dispatch SQS | ✅ มีแล้ว           | ส่ง RESOLVED ไปหา Manage Dispatch แล้ว |
| `rescue_request_client.go` pattern                          | ✅ ใช้เป็น template | retry + degraded mode เหมือนกัน        |

---

## ขั้นตอนที่ต้องทำทั้งหมด

---

### ขั้นที่ A — เพิ่ม model ใน `src/backend/internal/models/rescue_team.go` (ไฟล์ใหม่)

สร้าง model แยกสำหรับ RescueTeam Service เพื่อรับ response จาก `GET /v1/teams/{team_id}`  
(อ้างอิง: Sync Contract #2 — Get Rescue Team Detail)

```go
package models

// RescueTeamLocation คือ sub-struct ของ location จาก RescueTeam Service.
type RescueTeamLocation struct {
    Lat       float64 `json:"lat"`
    Lng       float64 `json:"lng"`
    UpdatedAt string  `json:"updated_at,omitempty"`
    Source    string  `json:"source,omitempty"`
}

// RescueTeamDetail คือ response body จาก GET /v1/teams/{team_id}.
// อ้างอิง: Sync Contract #2
type RescueTeamDetail struct {
    TeamID       string             `json:"team_id"`
    TeamName     string             `json:"team_name"`
    TeamType     string             `json:"team_type"`
    Status       string             `json:"status"`
    Location     RescueTeamLocation `json:"location"`
    Capabilities []string           `json:"capabilities"`
    Equipment    []string           `json:"equipment,omitempty"`
    UpdatedAt    string             `json:"updated_at,omitempty"`
}
```

---

### ขั้นที่ B — สร้าง `src/backend/internal/client/rescue_team_client.go` (ไฟล์ใหม่)

สร้าง client 2 method:

- `GetTeamDetail(teamID)` — GET /v1/teams/{team_id} สำหรับ enrich get-mission
- `UpdateTeamStatus(teamID, status)` — PATCH /v1/teams/{team_id}/status สำหรับ free team เมื่อ RESOLVED

Auth: `Authorization: Bearer <token>` อ้างอิงจาก Global Definitions ของ Sync Contract

```go
package client

import (
    "bytes"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/Ratthatummanoon/CS366-MissionProgress-Service/internal/models"
)

const (
    rtMaxRetries        = 2
    rtBackoffBase       = 100 * time.Millisecond
    rtPerRequestTimeout = 800 * time.Millisecond
)

// RescueTeamClient calls the RescueTeam Service.
type RescueTeamClient struct {
    baseURL     string
    bearerToken string
    httpClient  *http.Client
}

// NewRescueTeamClient creates a new client.
// Reads RESCUE_TEAM_SERVICE_URL and RESCUE_TEAM_SERVICE_TOKEN from env.
func NewRescueTeamClient() *RescueTeamClient {
    baseURL := os.Getenv("RESCUE_TEAM_SERVICE_URL")
    if baseURL == "" {
        baseURL = "http://localhost:9996"
    }
    token := os.Getenv("RESCUE_TEAM_SERVICE_TOKEN")
    return &RescueTeamClient{
        baseURL:     baseURL,
        bearerToken: token,
        httpClient: &http.Client{
            Timeout: rtPerRequestTimeout,
        },
    }
}

// GetTeamDetail fetches rescue team details from the RescueTeam Service.
// Endpoint: GET /v1/teams/{team_id}
// Auth: Authorization: Bearer <token>
// Retries up to 2 times on network errors and 5xx.
// Returns nil on failure (degraded mode — caller ยังคืน response ได้โดยไม่มีข้อมูลทีม).
func (c *RescueTeamClient) GetTeamDetail(teamID string) *models.RescueTeamDetail {
    url := fmt.Sprintf("%s/v1/teams/%s", c.baseURL, teamID)

    var lastErr error
    for attempt := 0; attempt <= rtMaxRetries; attempt++ {
        if attempt > 0 {
            backoff := rtBackoffBase * (1 << (attempt - 1)) // 100ms, 200ms
            log.Printf("INFO: Retry %d/%d for RescueTeamService.GetTeamDetail after %v", attempt, rtMaxRetries, backoff)
            time.Sleep(backoff)
        }

        req, err := http.NewRequest(http.MethodGet, url, nil)
        if err != nil {
            lastErr = err
            continue
        }
        req.Header.Set("Accept", "application/json")
        if c.bearerToken != "" {
            req.Header.Set("Authorization", "Bearer "+c.bearerToken)
        }

        resp, err := c.httpClient.Do(req)
        if err != nil {
            lastErr = err
            continue
        }
        defer resp.Body.Close()

        if resp.StatusCode == http.StatusNotFound {
            log.Printf("WARN: RescueTeamService team_id=%s not found (404)", teamID)
            return nil
        }
        if resp.StatusCode >= 500 {
            lastErr = fmt.Errorf("RescueTeamService returned %d", resp.StatusCode)
            continue
        }
        if resp.StatusCode != http.StatusOK {
            log.Printf("WARN: RescueTeamService returned unexpected status %d", resp.StatusCode)
            return nil
        }

        var detail models.RescueTeamDetail
        if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
            log.Printf("WARN: decode RescueTeamService response: %v", err)
            return nil
        }
        return &detail
    }

    log.Printf("WARN: RescueTeamService unavailable after %d retries: %v", rtMaxRetries, lastErr)
    return nil
}

// UpdateTeamStatus calls PATCH /v1/teams/{team_id}/status to update team availability.
// Endpoint: PATCH /v1/teams/{team_id}/status
// Auth: Authorization: Bearer <token>
// เรียกใช้ตอน mission RESOLVED เพื่อ free team กลับไปเป็น AVAILABLE.
// Best-effort — ถ้าล้มเหลว log แล้วผ่าน (ไม่ block response กลับ caller).
func (c *RescueTeamClient) UpdateTeamStatus(teamID, status string) error {
    url := fmt.Sprintf("%s/v1/teams/%s/status", c.baseURL, teamID)

    body, _ := json.Marshal(map[string]string{"status": status})

    var lastErr error
    for attempt := 0; attempt <= rtMaxRetries; attempt++ {
        if attempt > 0 {
            backoff := rtBackoffBase * (1 << (attempt - 1))
            log.Printf("INFO: Retry %d/%d for RescueTeamService.UpdateTeamStatus after %v", attempt, rtMaxRetries, backoff)
            time.Sleep(backoff)
        }

        req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
        if err != nil {
            lastErr = err
            continue
        }
        req.Header.Set("Content-Type", "application/json")
        if c.bearerToken != "" {
            req.Header.Set("Authorization", "Bearer "+c.bearerToken)
        }

        resp, err := c.httpClient.Do(req)
        if err != nil {
            lastErr = err
            continue
        }
        resp.Body.Close()

        if resp.StatusCode == http.StatusOK {
            log.Printf("INFO: RescueTeamService team_id=%s status updated to %s", teamID, status)
            return nil
        }
        if resp.StatusCode >= 500 {
            lastErr = fmt.Errorf("RescueTeamService returned %d on status update", resp.StatusCode)
            continue
        }
        // 4xx — validation error หรือ not found → ไม่ retry
        log.Printf("WARN: RescueTeamService status update failed: HTTP %d for team_id=%s", resp.StatusCode, teamID)
        return fmt.Errorf("RescueTeamService rejected status update: HTTP %d", resp.StatusCode)
    }
    return fmt.Errorf("RescueTeamService unavailable after retries: %w", lastErr)
}
```

---

### ขั้นที่ C — อัปเดต `src/backend/internal/models/requests.go`

#### C.1 — เพิ่ม field ใน `GetMissionResponse`

```go
// GetMissionResponse is the response for GET /missions/{request_id}.
type GetMissionResponse struct {
    RequestID         string             `json:"request_id"`
    IncidentID        string             `json:"incident_id"`
    MissionID         string             `json:"mission_id"`
    DispatchID        string             `json:"dispatch_id,omitempty"`
    RescueTeamID      string             `json:"rescue_team_id"`
    // --- Team detail fields (เพิ่มใหม่ Phase 4) ---
    TeamName          string             `json:"team_name,omitempty"`
    TeamType          string             `json:"team_type,omitempty"`
    Capabilities      []string           `json:"capabilities,omitempty"`
    Equipment         []string           `json:"equipment,omitempty"`
    TeamLocation      *TeamLocationSnap  `json:"team_location,omitempty"`
    // -----------------------------------------------
    PriorityLevel     int                `json:"priority_level,omitempty"`
    DispatchStatus    string             `json:"dispatch_status,omitempty"`
    CurrentStatus     string             `json:"current_status"`
    LatestImpactLevel int                `json:"latest_impact_level"`
    StartedAt         string             `json:"started_at"`
    LastUpdatedAt     string             `json:"last_updated_at"`
    Description       string             `json:"description,omitempty"`
    Location          string             `json:"location,omitempty"`
    IncidentType      string             `json:"incident_type,omitempty"`
    Timeline          []TimelineEntry    `json:"timeline"`
    DataSource        string             `json:"data_source"`
}

// TeamLocationSnap คือ snapshot ตำแหน่งทีมที่ embed ใน GetMissionResponse
type TeamLocationSnap struct {
    Lat float64 `json:"lat"`
    Lng float64 `json:"lng"`
}
```

> **หมายเหตุ:** ใช้ `omitempty` เพื่อให้ response ไม่แตกหาก RescueTeam Service ไม่ตอบสนอง

---

### ขั้นที่ D — อัปเดต `src/backend/cmd/get-mission/main.go`

เพิ่ม `rescueTeamClient` และ enrich response ด้วยข้อมูลทีม

#### D.1 — เพิ่มใน `var` และ `init()`

```go
var (
    missionRepo          *repository.MissionRepo
    timelineRepo         *repository.TimelineRepo
    rescueRequestClient  *client.RescueRequestClient
    manageDispatchClient *client.ManageDispatchClient
    rescueTeamClient     *client.RescueTeamClient     // เพิ่มใหม่
)

func init() {
    // ... (เหมือนเดิม)
    rescueRequestClient  = client.NewRescueRequestClient()
    manageDispatchClient = client.NewManageDispatchClient()
    rescueTeamClient     = client.NewRescueTeamClient()  // เพิ่มใหม่
}
```

#### D.2 — เพิ่ม team enrichment หลัง step 3b (ใน handler)

```go
// 3c. Call RescueTeam Service เพื่อดึงข้อมูลทีม (degraded mode on failure)
var teamName, teamType string
var capabilities, equipment []string
var teamLocation *models.TeamLocationSnap

teamDetail := rescueTeamClient.GetTeamDetail(mission.RescueTeamID)
if teamDetail != nil {
    teamName     = teamDetail.TeamName
    teamType     = teamDetail.TeamType
    capabilities = teamDetail.Capabilities
    equipment    = teamDetail.Equipment
    teamLocation = &models.TeamLocationSnap{
        Lat: teamDetail.Location.Lat,
        Lng: teamDetail.Location.Lng,
    }
} else {
    log.Printf("INFO: RescueTeamService unavailable - returning partial team data for teamID=%s", mission.RescueTeamID)
    if dataSource == "full" {
        dataSource = "partial"
    }
}
```

#### D.3 — อัปเดต response struct

```go
return response.JSON(200, models.GetMissionResponse{
    RequestID:         mission.RequestID,
    IncidentID:        mission.IncidentID,
    MissionID:         mission.MissionID,
    DispatchID:        mission.DispatchID,
    RescueTeamID:      mission.RescueTeamID,
    TeamName:          teamName,          // เพิ่มใหม่
    TeamType:          teamType,          // เพิ่มใหม่
    Capabilities:      capabilities,      // เพิ่มใหม่
    Equipment:         equipment,         // เพิ่มใหม่
    TeamLocation:      teamLocation,      // เพิ่มใหม่
    PriorityLevel:     priorityLevel,
    DispatchStatus:    dispatchStatus,
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

### ขั้นที่ E — อัปเดต `src/backend/cmd/report-progress/main.go`

เพิ่มการแจ้ง RescueTeam Service เมื่อ mission เปลี่ยนสถานะเป็น `RESOLVED`

#### E.1 — เพิ่มใน `var` และ `init()`

```go
var (
    missionRepo      *repository.MissionRepo
    timelineRepo     *repository.TimelineRepo
    publisher        *evtpub.Publisher
    rescueTeamClient *client.RescueTeamClient  // เพิ่มใหม่
)

func init() {
    // ... (เหมือนเดิม)
    rescueTeamClient = client.NewRescueTeamClient()  // เพิ่มใหม่
}
```

#### E.2 — เพิ่ม call UpdateTeamStatus ใน handler (หลัง step 8 publish events)

```go
// 8b. Notify RescueTeam Service ให้ free team กลับเป็น AVAILABLE เมื่อ RESOLVED
// Best-effort: ถ้าล้มเหลวให้ log แล้วผ่าน — ไม่ fail request
if req.Status == "RESOLVED" && mission.RescueTeamID != "" {
    if err := rescueTeamClient.UpdateTeamStatus(mission.RescueTeamID, "AVAILABLE"); err != nil {
        log.Printf("WARN: failed to update RescueTeam status to AVAILABLE for team=%s: %v",
            mission.RescueTeamID, err)
        // ไม่ return error — mission ถูก update สำเร็จแล้ว
    }
}
```

> **เหตุผลที่ใช้ best-effort (ไม่ fail request):**
>
> - Mission ถูก update ใน DynamoDB สำเร็จแล้ว → ความสำเร็จหลักเกิดขึ้นแล้ว
> - `MissionStatusChangedEvent` ถูก publish ผ่าน EventBridge → Manage Dispatch ได้รับแล้ว
> - RescueTeam Service มี idempotent PATCH — ถ้า retry สำเร็จทีหลังก็ไม่เสียหาย
> - ไม่ควรให้ NetworkError จาก RescueTeam ทำให้ RESOLVED ล้มเหลวในมุมมองของ Rescue Team

---

### ขั้นที่ F — อัปเดต `terraform/variables.tf`

เพิ่ม variable สำหรับ RescueTeam Service

```hcl
variable "rescue_team_service_url" {
  description = "URL of the RescueTeam Service"
  type        = string
  default     = "http://localhost:9996"
}

variable "rescue_team_service_token" {
  description = "Bearer token for authenticating with RescueTeam Service"
  type        = string
  sensitive   = true
  default     = "mock-dispatcher-token-123"
}
```

> **หมายเหตุ:** RescueTeam Service ใช้ `mock-dispatcher-token-123` เป็น mock token ตาม Sync Contract

---

### ขั้นที่ G — อัปเดต `terraform/lambda.tf`

เพิ่ม env var ใน 2 Lambda ที่ต้องใช้ RescueTeam Service

#### G.1 — Lambda `get_mission`

```hcl
environment {
  variables = {
    TABLE_MISSION                = aws_dynamodb_table.mission_assignment.name
    TABLE_TIMELINE               = aws_dynamodb_table.mission_timeline.name
    RESCUE_REQUEST_SERVICE_URL   = var.rescue_request_service_url
    RESCUE_REQUEST_SERVICE_TOKEN = var.rescue_request_service_token
    MANAGE_DISPATCH_SERVICE_URL  = var.manage_dispatch_service_url      # Phase 3
    RESCUE_TEAM_SERVICE_URL      = var.rescue_team_service_url          # เพิ่มใหม่
    RESCUE_TEAM_SERVICE_TOKEN    = var.rescue_team_service_token        # เพิ่มใหม่
  }
}
```

#### G.2 — Lambda `report_progress`

```hcl
environment {
  variables = {
    TABLE_MISSION     = aws_dynamodb_table.mission_assignment.name
    TABLE_TIMELINE    = aws_dynamodb_table.mission_timeline.name
    TABLE_OUTBOX      = aws_dynamodb_table.event_outbox.name
    EVENT_BUS_NAME    = aws_cloudwatch_event_bus.mission_events.name
    RESCUE_TEAM_SERVICE_URL   = var.rescue_team_service_url    # เพิ่มใหม่
    RESCUE_TEAM_SERVICE_TOKEN = var.rescue_team_service_token  # เพิ่มใหม่
  }
}
```

---

### ขั้นที่ H — Build & Verify

```bash
cd src/backend
go build ./...
```

ตรวจสอบว่าไม่มี compile error ก่อน deploy

---

## สรุปลำดับการทำงาน

```
A → B → C → D → E → F → G → H
```

| ขั้น | ไฟล์ที่แก้ / สร้าง                      | ประเภท    |
| ---- | --------------------------------------- | --------- |
| A    | `internal/models/rescue_team.go`        | สร้างใหม่ |
| B    | `internal/client/rescue_team_client.go` | สร้างใหม่ |
| C    | `internal/models/requests.go`           | แก้       |
| D    | `cmd/get-mission/main.go`               | แก้       |
| E    | `cmd/report-progress/main.go`           | แก้       |
| F    | `terraform/variables.tf`                | แก้       |
| G    | `terraform/lambda.tf`                   | แก้       |
| H    | `go build ./...`                        | verify    |

---

## Dependency / ข้อควรระวัง

| ประเด็น                                 | รายละเอียด                                                                                                                                                    |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Authentication token**                | RescueTeam ใช้ `Authorization: Bearer <dispatcher_token>` — ปัจจุบัน mock เป็น `mock-dispatcher-token-123` ต้องเปลี่ยนเป็น token จริงก่อน production          |
| **Team enrichment เป็น degraded mode**  | ถ้า RescueTeam ไม่ตอบสนอง `get-mission` ยังคืน response ได้ แต่ `team_name`, `capabilities`, `team_location` จะว่าง และ `data_source` เปลี่ยนเป็น `"partial"` |
| **`UpdateTeamStatus` เป็น best-effort** | ถ้าล้มเหลวจะ log แต่ไม่ fail request — ควร monitor log เพื่อตรวจสอบว่าไม่มี team ค้างสถานะ BUSY นานเกินไป                                                     |
| **Double-free ของ team**                | ถ้า RESOLVED ถูกเรียกหลายครั้ง (retry) `UpdateTeamStatus` จะถูกเรียกหลายครั้งด้วย — RescueTeam Service ระบุว่า PATCH idempotent จึงปลอดภัย                    |
| **`RescueTeamID` อาจว่างเปล่า**         | ใน missions เก่าที่ยังไม่ได้ผ่าน Phase 3 อาจมี `RescueTeamID` ว่าง — ต้อง guard ก่อนเรียก client (ใน handler ตรวจ `mission.RescueTeamID != ""` อยู่แล้ว)      |
| **Phase ordering**                      | Phase 4 ขึ้นอยู่กับ Phase 3 (model `MissionAssignment` ที่มี `DispatchID`) — ต้องทำ Phase 3 ก่อน หรือทำพร้อมกันในไฟล์เดียว                                    |
