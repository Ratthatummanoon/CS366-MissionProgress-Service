# Phase 2 Refactor — Migration จาก IncidentTracking Service → RescueRequest Service

---

## ภาพรวมของการเปลี่ยนแปลง

การ refactor นี้ไม่ใช่แค่เปลี่ยนชื่อตัวแปร — แต่คือการเปลี่ยน **external dependency** ที่ service เรียกออกไป

| หัวข้อ                   | เดิม                                            | ใหม่                                                                                                       |
| ------------------------ | ----------------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| External Service         | IncidentTracking Service                        | **RescueRequest Service**                                                                                  |
| Lookup key ใน path param | `incident_id`                                   | **`request_id`**                                                                                           |
| Endpoint ที่เรียก        | `GET /incidents/{incident_id}`                  | **`GET /v1/rescue-requests/{requestId}`**                                                                  |
| Service Owner            | Krittamet Damthongkam                           | **Phattharaphum Kingchai**                                                                                 |
| Auth header              | ไม่มี                                           | **`Authorization: Bearer <token>`**                                                                        |
| Response schema          | flat: `{ description, location, incidentType }` | nested: `{ request: { description, location: { province, ... }, requestType }, current: { status, ... } }` |
| Client file              | `internal/client/incident_client.go`            | **`internal/client/rescue_request_client.go`**                                                             |
| Model file               | `internal/models/incident.go`                   | **`internal/models/rescue_request.go`**                                                                    |
| Terraform env var        | `INCIDENT_SERVICE_URL`                          | **`RESCUE_REQUEST_SERVICE_URL`**                                                                           |

---

## สถานะปัจจุบัน — สิ่งที่ทำเสร็จแล้ว ✅

ส่วน routing และ `request_id` ทำเสร็จในโค้ดแล้วทั้งหมด:

| ส่วน                                                               | สถานะ   | หมายเหตุ                             |
| ------------------------------------------------------------------ | ------- | ------------------------------------ |
| `models/mission.go` — เพิ่ม `RequestID`                            | ✅ Done |                                      |
| `models/requests.go` — เพิ่ม `request_id` ใน response structs      | ✅ Done |                                      |
| `repository/mission_repo.go` — `GetMissionByRequestID`             | ✅ Done |                                      |
| `mission-assigned-handler/main.go` — รับ `request_id` จาก payload  | ✅ Done |                                      |
| `get-mission/main.go` — path param เปลี่ยนเป็น `request_id`        | ✅ Done | แต่ยัง call `incidentClient` อยู่ ⚠️ |
| `report-progress/main.go` — path param เปลี่ยนเป็น `request_id`    | ✅ Done |                                      |
| `presigned-url/main.go` — path param เปลี่ยนเป็น `request_id`      | ✅ Done |                                      |
| `terraform/api_gateway.tf` — routes `/missions/{request_id}`       | ✅ Done |                                      |
| `terraform/dynamodb.tf` — GSI `request-index`                      | ✅ Done |                                      |
| `frontend/lib/types.ts` — เพิ่ม `request_id`                       | ✅ Done |                                      |
| `frontend/lib/api.ts` — URL paths เปลี่ยนเป็น `/missions/`         | ✅ Done |                                      |
| `frontend/app/dashboard/page.tsx` — ใช้ `request_id` navigate      | ✅ Done |                                      |
| `frontend/app/mission/page.tsx` — รับ `requestId` จาก searchParams | ✅ Done |                                      |
| `script/seed-data.sh` — เพิ่ม `request_id` ใน items                | ✅ Done |                                      |

---

## สิ่งที่ยังไม่ได้ทำ — THE CRITICAL GAP ⚠️

`get-mission/main.go` บรรทัด 63 ยังเรียกบริการเดิมอยู่:

```go
// ยังอยู่ใน get-mission/main.go — ยังไม่ได้เปลี่ยน
incidentDetail := incidentClient.GetIncidentDetail(mission.IncidentID)
```

และ `incident_client.go`:

```go
// ยัง call IncidentTracking Service อยู่ — ต้องเปลี่ยนทั้งหมด
url := fmt.Sprintf("%s/incidents/%s", c.baseURL, incidentID)
// env: INCIDENT_SERVICE_URL
```

---

## ขั้นตอนที่ยังต้องทำทั้งหมด

---

### ขั้นที่ A — สร้าง `src/backend/internal/models/rescue_request.go` (ไฟล์ใหม่)

สร้าง model ที่ตรงกับ response schema ของ RescueRequest Service
(อ้างอิง: `docs/RescueRequest Service/RescueRequest Service sync contract.txt` — `#2 GetRescueRequest`)

```go
package models

// RescueRequestLocation คือ sub-struct ของ location จาก RescueRequest Service.
type RescueRequestLocation struct {
    Latitude        float64                `json:"latitude"`
    Longitude       float64                `json:"longitude"`
    LocationDetails map[string]string      `json:"locationDetails,omitempty"`
    Province        string                 `json:"province,omitempty"`
    District        string                 `json:"district,omitempty"`
    Subdistrict     string                 `json:"subdistrict,omitempty"`
    AddressLine     string                 `json:"addressLine,omitempty"`
}

// RescueRequestMaster คือข้อมูลหลักของคำร้องจาก RescueRequest Service.
type RescueRequestMaster struct {
    RequestID    string                `json:"requestId"`
    IncidentID   string                `json:"incidentId"`
    RequestType  string                `json:"requestType"`
    Description  string                `json:"description,omitempty"`
    PeopleCount  int                   `json:"peopleCount"`
    Location     RescueRequestLocation `json:"location"`
    SubmittedAt  string                `json:"submittedAt,omitempty"`
}

// RescueRequestCurrent คือ current state ของคำร้องจาก RescueRequest Service.
type RescueRequestCurrent struct {
    Status      string `json:"status,omitempty"`
    AssignedAt  string `json:"assignedAt,omitempty"`
    LastUpdatedAt string `json:"lastUpdatedAt,omitempty"`
}

// RescueRequestDetail คือ response body จาก GET /v1/rescue-requests/{requestId}.
type RescueRequestDetail struct {
    Request RescueRequestMaster  `json:"request"`
    Current RescueRequestCurrent `json:"current"`
}
```

> **ทำอะไรกับ `models/incident.go`:** เก็บไว้ก่อน ไม่ลบ — ยังมี `IncidentDetail` ซึ่ง embedded ใน `GetMissionResponse` ผ่าน field `description`, `location`, `incident_type` อยู่ (จะลบทีหลังเมื่อ migration เสร็จ 100%)

---

### ขั้นที่ B — สร้าง `src/backend/internal/client/rescue_request_client.go` (ไฟล์ใหม่)

สร้าง client ใหม่สำหรับเรียก RescueRequest Service
(อ้างอิง: endpoint `GET /v1/rescue-requests/{requestId}`, auth: `Authorization: Bearer <token>`)

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
    rrMaxRetries        = 2
    rrBackoffBase       = 100 * time.Millisecond
    rrPerRequestTimeout = 800 * time.Millisecond
)

// RescueRequestClient calls the RescueRequest Service.
type RescueRequestClient struct {
    baseURL    string
    bearerToken string
    httpClient *http.Client
}

// NewRescueRequestClient creates a new client with retry support.
// Reads RESCUE_REQUEST_SERVICE_URL and RESCUE_REQUEST_SERVICE_TOKEN from env.
func NewRescueRequestClient() *RescueRequestClient {
    baseURL := os.Getenv("RESCUE_REQUEST_SERVICE_URL")
    if baseURL == "" {
        baseURL = "http://localhost:9998"
    }
    token := os.Getenv("RESCUE_REQUEST_SERVICE_TOKEN")
    return &RescueRequestClient{
        baseURL:     baseURL,
        bearerToken: token,
        httpClient: &http.Client{
            Timeout: rrPerRequestTimeout,
        },
    }
}

// GetRequestDetail fetches rescue request details from the RescueRequest Service.
// Endpoint: GET /v1/rescue-requests/{requestId}
// Auth: Authorization: Bearer <token>
// Retries up to 2 times on network errors and 5xx.
// Returns nil on failure (degraded mode).
func (c *RescueRequestClient) GetRequestDetail(requestID string) *models.RescueRequestDetail {
    url := fmt.Sprintf("%s/v1/rescue-requests/%s", c.baseURL, requestID)

    var lastErr error
    for attempt := 0; attempt <= rrMaxRetries; attempt++ {
        if attempt > 0 {
            backoff := rrBackoffBase * (1 << (attempt - 1)) // 100ms, 200ms
            log.Printf("INFO: Retry %d/%d for RescueRequestService after %v", attempt, rrMaxRetries, backoff)
            time.Sleep(backoff)
        }

        req, err := http.NewRequest(http.MethodGet, url, nil)
        if err != nil {
            log.Printf("WARNING: RescueRequestService build request failed: %v", err)
            return nil
        }
        if c.bearerToken != "" {
            req.Header.Set("Authorization", "Bearer "+c.bearerToken)
        }
        req.Header.Set("Accept", "application/json")

        resp, err := c.httpClient.Do(req)
        if err != nil {
            lastErr = err
            log.Printf("WARNING: RescueRequestService attempt %d failed (network): %v", attempt+1, err)
            continue
        }

        if resp.StatusCode >= 500 {
            resp.Body.Close()
            lastErr = fmt.Errorf("status %d", resp.StatusCode)
            log.Printf("WARNING: RescueRequestService attempt %d failed (5xx): status %d", attempt+1, resp.StatusCode)
            continue
        }

        if resp.StatusCode == http.StatusNotFound {
            resp.Body.Close()
            log.Printf("WARNING: RescueRequestService returned 404 for requestId=%s", requestID)
            return nil
        }

        if resp.StatusCode != http.StatusOK {
            resp.Body.Close()
            log.Printf("WARNING: RescueRequestService returned status %d (non-retryable)", resp.StatusCode)
            return nil
        }

        var detail models.RescueRequestDetail
        if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
            resp.Body.Close()
            log.Printf("WARNING: Failed to decode RescueRequestService response: %v", err)
            return nil
        }
        resp.Body.Close()
        return &detail
    }

    log.Printf("WARNING: RescueRequestService all retries exhausted for requestId=%s: %v", requestID, lastErr)
    return nil
}

// FormatLocation formats the structured location into a human-readable string.
// e.g., "123 ม.2 ถ.ห้วยแก้ว, สุเทพ, เมืองเชียงใหม่, เชียงใหม่"
func FormatLocation(loc models.RescueRequestLocation) string {
    parts := []string{}
    if loc.AddressLine != "" {
        parts = append(parts, loc.AddressLine)
    }
    if loc.Subdistrict != "" {
        parts = append(parts, loc.Subdistrict)
    }
    if loc.District != "" {
        parts = append(parts, loc.District)
    }
    if loc.Province != "" {
        parts = append(parts, loc.Province)
    }
    result := ""
    for i, p := range parts {
        if i > 0 {
            result += ", "
        }
        result += p
    }
    return result
}
```

---

### ขั้นที่ C — อัปเดต `src/backend/cmd/get-mission/main.go`

เปลี่ยน dependency จาก `IncidentClient` → `RescueRequestClient`

**C.1** เปลี่ยน import และ var:

```go
// เดิม
var (
    missionRepo    *repository.MissionRepo
    timelineRepo   *repository.TimelineRepo
    incidentClient *client.IncidentClient
)

// ใหม่
var (
    missionRepo          *repository.MissionRepo
    timelineRepo         *repository.TimelineRepo
    rescueRequestClient  *client.RescueRequestClient
)
```

**C.2** เปลี่ยน init():

```go
// เดิม
incidentClient = client.NewIncidentClient()

// ใหม่
rescueRequestClient = client.NewRescueRequestClient()
```

**C.3** เปลี่ยน call ใน handler (step 3):

```go
// เดิม — ใช้ mission.IncidentID เรียก IncidentTracking
incidentDetail := incidentClient.GetIncidentDetail(mission.IncidentID)
if incidentDetail != nil {
    description = incidentDetail.Description
    location    = incidentDetail.Location
    incidentType = incidentDetail.IncidentType
} else {
    dataSource = "partial"
    log.Printf("INFO: IncidentTracking unavailable - returning partial data for %s", mission.IncidentID)
}

// ใหม่ — ใช้ requestID เรียก RescueRequest Service
requestDetail := rescueRequestClient.GetRequestDetail(requestID)
if requestDetail != nil {
    description  = requestDetail.Request.Description
    location     = client.FormatLocation(requestDetail.Request.Location)
    incidentType = requestDetail.Request.RequestType
} else {
    dataSource = "partial"
    log.Printf("INFO: RescueRequestService unavailable - returning partial data for requestID=%s", requestID)
}
```

> **หมายเหตุ:** `requestID` มาจาก path param ที่ parse ไว้แล้วตั้งแต่ step 1 ของ handler — ใช้ได้เลยโดยไม่ต้องส่ง `mission.IncidentID` ไปอีก

---

### ขั้นที่ D — อัปเดต `terraform/variables.tf`

**D.1** เพิ่ม variable ใหม่ (ต่อท้ายไฟล์):

```hcl
variable "rescue_request_service_url" {
  description = "URL of the RescueRequest Service"
  type        = string
  default     = "http://localhost:9998"
}

variable "rescue_request_service_token" {
  description = "Bearer token for authenticating with RescueRequest Service (staff access)"
  type        = string
  sensitive   = true
  default     = ""
}
```

**D.2** `incident_service_url` — เก็บไว้ก่อน (ยังมี SQS/EventBridge dependency อื่นที่อาจอ้างอิง) หรือ comment out ก็ได้ถ้ามั่นใจว่าไม่ใช้แล้ว

---

### ขั้นที่ E — อัปเดต `terraform/lambda.tf`

**E.1** `aws_lambda_function.get_mission` — เปลี่ยน env vars:

```hcl
# เดิม
environment {
  variables = {
    TABLE_MISSION        = aws_dynamodb_table.mission_assignment.name
    TABLE_TIMELINE       = aws_dynamodb_table.mission_timeline.name
    INCIDENT_SERVICE_URL = var.incident_service_url
  }
}

# ใหม่
environment {
  variables = {
    TABLE_MISSION                = aws_dynamodb_table.mission_assignment.name
    TABLE_TIMELINE               = aws_dynamodb_table.mission_timeline.name
    RESCUE_REQUEST_SERVICE_URL   = var.rescue_request_service_url
    RESCUE_REQUEST_SERVICE_TOKEN = var.rescue_request_service_token
  }
}
```

**E.2** `aws_lambda_function.report_progress` — ลบ `INCIDENT_SERVICE_URL` ออก (Lambda นี้ไม่ได้ใช้ incidentClient เลย แต่ถูกใส่ไว้ผิด):

```hcl
# เดิม
environment {
  variables = {
    TABLE_MISSION        = aws_dynamodb_table.mission_assignment.name
    TABLE_TIMELINE       = aws_dynamodb_table.mission_timeline.name
    TABLE_OUTBOX         = aws_dynamodb_table.event_outbox.name
    EVENT_BUS_NAME       = aws_cloudwatch_event_bus.mission_events.name
    INCIDENT_SERVICE_URL = var.incident_service_url   # ← ลบออก
  }
}

# ใหม่
environment {
  variables = {
    TABLE_MISSION  = aws_dynamodb_table.mission_assignment.name
    TABLE_TIMELINE = aws_dynamodb_table.mission_timeline.name
    TABLE_OUTBOX   = aws_dynamodb_table.event_outbox.name
    EVENT_BUS_NAME = aws_cloudwatch_event_bus.mission_events.name
  }
}
```

---

### ขั้นที่ F — จัดการ `src/backend/internal/client/incident_client.go`

ไฟล์นี้ไม่มี Lambda ใดเรียกอีกแล้วหลังขั้นที่ C เสร็จ

**ตัวเลือก (เลือกอย่างใดอย่างหนึ่ง):**

| ตัวเลือก                                                | เหมาะกับ                        |
| ------------------------------------------------------- | ------------------------------- |
| ลบไฟล์ทิ้ง                                              | clean codebase, ไม่มี confusion |
| Rename เป็น `incident_client.go.bak` หรือ comment out   | ยังต้องการ reference            |
| เพิ่ม `// Deprecated: ใช้ rescue_request_client.go แทน` | เก็บไว้ แต่ชัดเจน               |

> แนะนำ: **ลบทิ้ง** เพราะ `incident.go` model ก็ไม่จำเป็นอีกต่อไปเมื่อใช้ `rescue_request.go` แทน

---

### ขั้นที่ G — ลบ `src/backend/internal/models/incident.go`

หลังจาก `get-mission` เปลี่ยนมาใช้ `RescueRequestDetail` แล้ว struct `IncidentDetail` ไม่มี consumer อีกแล้ว

```go
// incident.go — ลบได้ทั้งไฟล์หลังขั้น C เสร็จ
type IncidentDetail struct {
    IncidentID   string `json:"incident_id"`
    Description  string `json:"description"`
    Location     string `json:"location"`
    IncidentType string `json:"incident_type"`
}
```

> ตรวจสอบก่อนลบ: `go build ./...` ต้องผ่านหลังลบ

---

## Frontend — สิ่งที่ยังต้องแก้ใน Source Code

Backend เปลี่ยน field `location` จาก string → formatted string จาก RescueRequest จะไม่กระทบ frontend (ยังเป็น `string` เหมือนเดิม)

แต่มี 2 จุดที่ยังแสดง `incident_id` ต่อ user แทนที่จะเป็น `request_id`:

**จุดที่ 1: `src/frontend/app/mission/page.tsx` บรรทัด ~180**

```tsx
// เดิม — แสดง incident_id เป็น heading
<h1 className="text-2xl font-bold text-gray-900">{mission.incident_id}</h1>
<p className="text-sm text-gray-500 mt-1">Mission ID: {mission.mission_id}</p>

// ใหม่ — ใช้ request_id เป็น heading
<h1 className="text-2xl font-bold text-gray-900">{mission.request_id}</h1>
<p className="text-sm text-gray-500 mt-1">
  Incident: {mission.incident_id} | Mission: {mission.mission_id}
</p>
```

**จุดที่ 2: `src/frontend/app/dashboard/page.tsx` บรรทัด ~175** — ✅ ทำเสร็จแล้ว (แสดง `request_id` เป็น primary, `incident_id | mission_id` เป็น sub-text)

---

## เอกสารที่ต้องอัปเดต

---

### Doc 1 — `docs/proposals/07-Dependency-Mapping.md` (สำคัญที่สุด)

**เพิ่ม Dependency ใหม่: RescueRequest Service** (ต่อจาก Dependency 4):

```markdown
## Dependency 5: RescueRequest Service

### Overview

| Field             | Value                       |
| ----------------- | --------------------------- |
| Service Owner     | Phattharaphum Kingchai      |
| Type              | Service                     |
| Interaction Style | Synchronous (REST)          |
| Criticality       | High (รองรับ Degraded Mode) |

### Purpose

- ดึงข้อมูล Request (description, location, requestType, peopleCount)
- เป็น **Source of Truth** ของ Rescue Request ที่ผูกกับภารกิจนี้

### Endpoint ที่เรียก

`GET /v1/rescue-requests/{requestId}`

- Auth: `Authorization: Bearer <token>`
- รับ `requestId` จาก path parameter ของ `GET /missions/{request_id}`

### Interaction

| ทิศทาง     | ช่องทาง                        | รายละเอียด                                  | Demo 2            |
| ---------- | ------------------------------ | ------------------------------------------- | ----------------- |
| ขาออก Sync | HTTP GET /v1/rescue-requests/… | ดึงข้อมูล Request (ล้มเหลว → Degraded Mode) | ✅ URL จริง [TBD] |

### Failure Handling

| กรณี             | การจัดการ                                                        |
| ---------------- | ---------------------------------------------------------------- |
| Sync GET ล้มเหลว | **Degraded Mode** → ส่งเฉพาะข้อมูลที่มี (`data_source: partial`) |

### TBD

- Service URL (prod)
- Bearer token สำหรับ service-to-service auth
```

**อัปเดต Dependency 1 (IncidentTracking Service):**

```markdown
### Interaction (อัปเดต)

| ทิศทาง         | ช่องทาง                          | รายละเอียด                                                   |
| -------------- | -------------------------------- | ------------------------------------------------------------ |
| ~~ขาออก Sync~~ | ~~HTTP GET /incidents/{id}~~     | ~~ดึงข้อมูล Incident~~ → ย้ายไปใช้ RescueRequest Service แทน |
| ขาออก Async    | EventBridge MissionStatusChanged | อัปเดตสถานะ Incident                                         |
| ขาออก Async    | EventBridge ImpactLevelUpdated   | อัปเดต Impact Level                                          |

> **หมายเหตุ:** Synchronous call ออก IncidentTracking ถูกยกเลิกแล้ว — ข้อมูล description/location/type ดึงมาจาก RescueRequest Service แทน (เพราะ RescueRequest Service เป็นเจ้าของ request context)
```

---

### Doc 2 — `docs/proposals/02-Sync-Contract.md`

เปลี่ยน path ทุกจุด:

| เดิม                                     | ใหม่                                   |
| ---------------------------------------- | -------------------------------------- |
| `/incidents/{incident_id}`               | `/missions/{request_id}`               |
| `/incidents/{incident_id}/progress`      | `/missions/{request_id}/progress`      |
| `/incidents/{incident_id}/presigned-url` | `/missions/{request_id}/presigned-url` |
| `GET /incidents`                         | `GET /missions`                        |

Path Parameter table ทุก endpoint:

| เดิม                           | ใหม่                                                                          |
| ------------------------------ | ----------------------------------------------------------------------------- |
| `incident_id` \| รหัสเหตุการณ์ | `request_id` \| รหัส request จาก RescueRequest Service (เช่น `REQ-8812-9901`) |

Error codes:

```json
// เดิม
{ "error": "INCIDENT_NOT_FOUND", "message": "No mission found for incident: INC-99999" }

// ใหม่
{ "error": "REQUEST_NOT_FOUND", "message": "No mission found for request: REQ-99999" }
```

เพิ่ม `request_id` ใน success response JSON ทุก endpoint

---

### Doc 3 — `docs/proposals/04-Service-Data.md`

ตาราง **Mission Assignment** — เพิ่ม row `request_id`:

| Field Name   | Type   | Required | Description                                                 | Example       |
| ------------ | ------ | -------- | ----------------------------------------------------------- | ------------- |
| `request_id` | string | Yes      | รหัส request จาก RescueRequest Service (Unique, lookup key) | REQ-8812-9901 |

เพิ่มหมายเหตุ:

> - `request_id` คือ primary lookup key — ใช้ค้นหา mission จาก path parameter `{request_id}`
> - 1 `incident_id` มีได้หลาย `request_id` → `incident_id` ไม่ unique สำหรับ lookup
> - `description`, `location`, `requestType` ดึงมาจาก RescueRequest Service แบบ on-demand (degraded mode ถ้า service ไม่พร้อม)

---

### Doc 4 — `docs/proposals/05-Service-Architecture.md`

ตาราง Routes (API Gateway):

| เดิม                                 | ใหม่                                        |
| ------------------------------------ | ------------------------------------------- |
| `GET /incidents/{id}`                | `GET /missions/{request_id}`                |
| `POST /incidents/{id}/progress`      | `POST /missions/{request_id}/progress`      |
| `POST /incidents/{id}/presigned-url` | `POST /missions/{request_id}/presigned-url` |

ตาราง DynamoDB GSI — เพิ่ม `request-index`:

| Table             | PK         | GSI                                                                                         |
| ----------------- | ---------- | ------------------------------------------------------------------------------------------- |
| MissionAssignment | mission_id | `request-index` (request_id), `team-index` (rescue_team_id), `incident-index` (incident_id) |

External Dependencies section — เพิ่ม:

> - **RescueRequest Service** (sync, Degraded Mode): `GET /v1/rescue-requests/{requestId}` — ดึง description/location/requestType
> - ~~IncidentTracking Service (sync)~~ — ยกเลิก synchronous call แล้ว (ยังรับ async events อยู่)

---

### Doc 5 — `docs/proposals/06-Service-Interaction.md`

Mermaid diagram — เปลี่ยน edge labels:

| เดิม                                 | ใหม่                                        |
| ------------------------------------ | ------------------------------------------- |
| `GET /incidents/{id}`                | `GET /missions/{request_id}`                |
| `POST /incidents/{id}/progress`      | `POST /missions/{request_id}/progress`      |
| `POST /incidents/{id}/presigned-url` | `POST /missions/{request_id}/presigned-url` |

เพิ่ม node และ arrow แสดง call ออกไป RescueRequest Service:

```
MissionProgress --> RescueRequestService: GET /v1/rescue-requests/{request_id}
RescueRequestService --> MissionProgress: { request: { description, location, requestType } }
```

ตาราง Interaction ทุก endpoint — เปลี่ยน path + เพิ่ม `request_id` ใน response JSON

---

### Doc 6 — `docs/proposals/07-Dependency-Mapping.md`

ดูรายละเอียดใน Doc 1 ข้างต้น

---

### Doc 7 — `docs/contract/contract_demo1.md`

```
// เดิม
1. GET /incidents/{incident_id}
2. POST /incidents/{incident_id}/progress

// ใหม่
1. GET /missions/{request_id}
2. POST /missions/{request_id}/progress
```

Path param: `request_id` | รหัส request จาก RescueRequest Service (เช่น `REQ-003`)

ตัวอย่าง curl:

```bash
# เดิม
curl -X GET ".../v1/incidents/INC-003" ...

# ใหม่
curl -X GET ".../v1/missions/REQ-003" ...
```

Error 404: เปลี่ยน `INCIDENT_NOT_FOUND` → `REQUEST_NOT_FOUND`

---

### Doc 8 — `docs/contract/contract_demo2.md`

```
// เดิม
1. GET /incidents/{incident_id}
2. POST /incidents/{incident_id}/progress
3. POST /incidents/{incident_id}/presigned-url
4. GET /incidents

// ใหม่
1. GET /missions/{request_id}
2. POST /missions/{request_id}/progress
3. POST /missions/{request_id}/presigned-url
4. GET /missions
```

เพิ่มแถวใน "สิ่งที่เพิ่มจาก Demo 1":

| ฟีเจอร์                                                                     | Demo 1 | Demo 2 |
| --------------------------------------------------------------------------- | ------ | ------ |
| `request_id` as lookup key (จาก RescueRequest Service แทน IncidentTracking) | ❌     | ✅     |
| ดึง request context จาก RescueRequest Service (description/location)        | ❌     | ✅     |

---

## สิ่งที่ไม่เปลี่ยน

| ส่วน                                   | เหตุผล                                                                                |
| -------------------------------------- | ------------------------------------------------------------------------------------- |
| `incident_client.go` → ลบทิ้ง          | ไม่มี consumer แล้ว                                                                   |
| Events ที่ publish ออกไป (EventBridge) | ยังมี `incident_id` ใน events — ถูกต้อง เพราะ downstream ต้องการ context ของ incident |
| `list-missions` Lambda                 | ยัง query ด้วย `rescue_team_id` เหมือนเดิม                                            |
| `outbox-processor` Lambda              | ไม่เกี่ยวกับ external service call                                                    |
| State machine logic                    | ไม่เปลี่ยน                                                                            |
| `authorizer` Lambda                    | ไม่เกี่ยวกับ path parameter หรือ external service                                     |
| `docs/proposals/03-Async-Contract.md`  | async events ยังใช้ `incident_id` เป็น context — ถูกต้อง                              |

---

## Design Decisions ที่ต้องระวัง

### 1. Field Mapping: `requestType` → `incident_type`

RescueRequest Service ส่งมาเป็น `requestType` (เช่น `flood_rescue`) แต่ `GetMissionResponse` มี field `incident_type`
→ **Map ตรงๆ**: `incidentType = requestDetail.Request.RequestType`
→ Frontend ยังรับ field `incident_type` เหมือนเดิม — ไม่กระทบ

### 2. `location` เป็น string

`GetMissionResponse.Location` เป็น `string` ส่วน RescueRequest ส่งเป็น structured object
→ **Format ด้วย `client.FormatLocation()`** → `"addressLine, subdistrict, district, province"`
→ ไม่ต้องเปลี่ยน frontend types.ts

### 3. Auth Token

RescueRequest Service ต้องการ `Authorization: Bearer <token>` สำหรับ staff access
→ ต้องได้ token จริงจาก Phattharaphum ก่อน deploy
→ Demo: ถ้ายังไม่มี token → client ส่ง request โดยไม่มี auth header → อาจได้ 401 → Degraded Mode (`data_source: partial`)

### 4. Request ID format

RescueRequest Service ใช้ format `^REQ-[0-9]{4}-[0-9]{4}$` เช่น `REQ-8812-9901`
→ Seed data ปัจจุบันใช้ `REQ-001` ซึ่งไม่ตรง pattern
→ ไม่กระทบ backend (ไม่มี validation ฝั่งเรา) แต่ควร align กับ RescueRequest Service จริงก่อน integration

---

## Checklist ก่อน Deploy

### Backend (งานที่เหลือ)

- [ ] **ขั้น A** — สร้าง `models/rescue_request.go` (RescueRequestDetail, RescueRequestLocation structs)
- [ ] **ขั้น B** — สร้าง `client/rescue_request_client.go` (client + FormatLocation)
- [ ] **ขั้น C** — อัปเดต `cmd/get-mission/main.go` (เปลี่ยน incidentClient → rescueRequestClient, เปลี่ยน call + field mapping)
- [ ] **ขั้น D** — อัปเดต `terraform/variables.tf` (เพิ่ม rescue_request_service_url, rescue_request_service_token)
- [ ] **ขั้น E** — อัปเดต `terraform/lambda.tf` (env vars get-mission + ลบ INCIDENT_SERVICE_URL จาก report-progress)
- [ ] **ขั้น F/G** — ลบ `client/incident_client.go` และ `models/incident.go` (หลัง build ผ่าน)
- [ ] `go build ./...` — ต้องผ่านทุก Lambda

### Frontend (งานที่เหลือ)

- [ ] `src/frontend/app/mission/page.tsx` — เปลี่ยน heading จาก `mission.incident_id` → `mission.request_id`

### Infrastructure

- [ ] รับ Bearer token จาก Phattharaphum (RescueRequest Service owner)
- [ ] รับ Service URL จริงของ RescueRequest Service
- [ ] `terraform plan` — ตรวจ diff ก่อน apply
- [ ] `terraform apply`

### เอกสาร

- [ ] **Doc 1** — `docs/proposals/07-Dependency-Mapping.md` (เพิ่ม RescueRequest Service + อัปเดต IncidentTracking)
- [ ] **Doc 2** — `docs/proposals/02-Sync-Contract.md`
- [ ] **Doc 3** — `docs/proposals/04-Service-Data.md`
- [ ] **Doc 4** — `docs/proposals/05-Service-Architecture.md`
- [ ] **Doc 5** — `docs/proposals/06-Service-Interaction.md`
- [ ] **Doc 6** — `docs/contract/contract_demo1.md`
- [ ] **Doc 7** — `docs/contract/contract_demo2.md`
