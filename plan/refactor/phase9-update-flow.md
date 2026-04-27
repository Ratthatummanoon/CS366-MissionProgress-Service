# Phase 9: ตรวจสอบ Codebase vs Diagram Flow

> ตรวจสอบเมื่อ: 27 เมษายน 2569  
> เทียบระหว่าง `docs/demo_feedback/diagram-flow.md` กับ codebase จริง

---

## ✅ สิ่งที่ตรงกัน (Implemented & Matches)

### Async Inbound — `mission-assigned-handler`

| รายการ                                                 | Diagram | Code | ไฟล์                                   |
| ------------------------------------------------------ | ------- | ---- | -------------------------------------- |
| Consume `DispatchOrderCreated` จาก Default EventBridge | ✅      | ✅   | `cmd/mission-assigned-handler/main.go` |
| Fields: `dispatchId, requestId, teamId, priorityLevel` | ✅      | ✅   | `MissionAssignedPayload` struct        |
| สร้าง Mission status: `DISPATCHED`                     | ✅      | ✅   | `mission.CurrentStatus = "DISPATCHED"` |
| Idempotency check by `dispatchId`                      | ✅      | ✅   | `GetMissionByDispatchID()`             |
| Degraded: `incidentId = ""` ถ้า RescueRequest ล่ม      | ✅      | ✅   | fallback in handler                    |

### Sync Outbound at CREATE — ดึง incidentId

| รายการ                                | Diagram | Code | ไฟล์                                           |
| ------------------------------------- | ------- | ---- | ---------------------------------------------- |
| `GET /v1/rescue-requests/{requestId}` | ✅      | ✅   | `rescue_request_client.go`                     |
| Auth: Bearer token                    | ✅      | ✅   | `Authorization` header                         |
| ดึง `incident_id` มาเก็บใน Mission    | ✅      | ✅   | `incidentID = requestDetail.Master.IncidentID` |

### Sync Outbound at GET (on-read) — get-mission

| รายการ                                              | Diagram | Code | ไฟล์                              |
| --------------------------------------------------- | ------- | ---- | --------------------------------- |
| เรียก 3 services พร้อมกัน (parallel)                | ✅      | ✅   | `sync.WaitGroup` + 3 goroutines   |
| RescueRequest → description, location, incidentType | ✅      | ✅   | `GetRequestDetail()`              |
| ManageDispatch → dispatch details, status, priority | ✅      | ✅   | `GetDispatchByTeamAndRequest()`   |
| RescueTeam → team name, capabilities, location      | ✅      | ✅   | `GetTeamDetail()`                 |
| Degraded mode ถ้า service ล่ม                       | ✅      | ✅   | `dataSource = "partial"` fallback |

### Sync Inbound — RescueTeam เรียก POST /progress

| รายการ                                                    | Diagram | Code | ไฟล์                                 |
| --------------------------------------------------------- | ------- | ---- | ------------------------------------ |
| Endpoint: `POST /missions/{request_id}/progress`          | ✅      | ✅   | `cmd/report-progress/main.go`        |
| Request fields: `new_status, new_impact_level, image_key` | ✅      | ✅   | `ReportProgressRequest` struct       |
| Response 200: updated mission status                      | ✅      | ✅   | `ReportProgressResponse`             |
| Auth via `X-Rescue-Team-ID`                               | ✅      | ✅   | authorizer context / header fallback |

### State Machine Transitions

| Transition             | Diagram | Code |
| ---------------------- | ------- | ---- |
| DISPATCHED → EN_ROUTE  | ✅      | ✅   |
| EN_ROUTE → ON_SITE     | ✅      | ✅   |
| ON_SITE → NEED_BACKUP  | ✅      | ✅   |
| ON_SITE → RESOLVED     | ✅      | ✅   |
| NEED_BACKUP → ON_SITE  | ✅      | ✅   |
| NEED_BACKUP → RESOLVED | ✅      | ✅   |

> ไฟล์: `internal/statemachine/statemachine.go`

### Async Outbound — Events Published

| Event                    | Trigger                              | Diagram | Code                                         |
| ------------------------ | ------------------------------------ | ------- | -------------------------------------------- |
| `MissionStatusChanged`   | ทุก transition                       | ✅      | ✅ (`publisher.PublishMissionStatusChanged`) |
| `MissionBackupRequested` | `new_status == NEED_BACKUP`          | ✅      | ✅ (`publisher.PublishBackupRequested`)      |
| `ImpactLevelUpdated`     | `new_impact_level != oldImpactLevel` | ✅      | ✅ (`publisher.PublishImpactLevelUpdated`)   |

### EventBridge Routing (Terraform)

| Rule                                                | Target | Diagram | Terraform                                 |
| --------------------------------------------------- | ------ | ------- | ----------------------------------------- |
| MissionStatusChanged → CloudWatch Logs              | ✅     | ✅      | `eventbridge.tf`                          |
| MissionBackupRequested → CloudWatch Logs            | ✅     | ✅      | `eventbridge.tf`                          |
| ImpactLevelUpdated → CloudWatch Logs                | ✅     | ✅      | `eventbridge.tf`                          |
| MissionStatusChanged → IncidentTracking SQS         | ✅     | ✅      | `mission_status_changed_incident_sqs`     |
| MissionStatusChanged (RESOLVED only) → Dispatch SQS | ✅     | ✅      | `mission_resolved_dispatch` rule          |
| MissionBackupRequested → Prioritization SQS         | ✅     | ✅      | `backup_requested_prioritization_sqs`     |
| ImpactLevelUpdated → IncidentTracking SQS           | ✅     | ✅      | `impact_level_updated_incident_sqs`       |
| ImpactLevelUpdated → Prioritization SQS             | ✅     | ✅      | `impact_level_updated_prioritization_sqs` |

### Phase 5: RESOLVED → Notify RescueTeam

| รายการ                              | Diagram | Code           |
| ----------------------------------- | ------- | -------------- |
| Notify RescueTeam ให้ set AVAILABLE | ✅      | ✅             |
| Fire-and-forget (non-blocking)      | ✅      | ✅ (goroutine) |

---

## ❌ สิ่งที่ไม่ตรง / ขาดหาย

### ~~Gap 1: HTTP Method ของการ notify RescueTeam~~ ✅ แก้แล้ว

- แก้ diagram: `PUT` → `PATCH /v1/teams/{teamId}/status` (ตามที่ code ใช้จริง)

---

### ~~Gap 2: ManageDispatch GET endpoint ไม่ตรง contract~~ ✅ แก้แล้ว

- แก้ diagram: `GET dispatch order, fields: dispatch_id` → `GET /v1/dispatches?teamId={teamId}` (ตามที่ code ใช้จริง)

> ⚠️ ยังควรตรวจสอบกับ Noppakron ว่ามี `GET /v1/dispatches/{dispatch_id}` ไหม ถ้ามีควรเปลี่ยน code ให้ query ตรงแทน (ลด data mismatch risk)

---

### Gap 3: `peopleCount` ไม่ถูก map เข้า Response

|                       | Diagram       | Code                                             |
| --------------------- | ------------- | ------------------------------------------------ |
| RescueRequest ตอบ     | `peopleCount` | มีใน model (`PeopleCount int`)                   |
| `GetMissionResponse`  | ควรมี         | ❌ ไม่มี field `people_count`                    |
| `get-mission` handler | ควร map       | ❌ ไม่ได้ map `requestDetail.Master.PeopleCount` |

**Action**: เพิ่ม field `PeopleCount int json:"people_count,omitempty"` ใน `GetMissionResponse` และ map ค่าใน `get-mission/main.go`

---

### ~~Gap 4: Fields ใน `MissionStatusChangedEvent` มากกว่า diagram~~ ✅ แก้แล้ว

- เพิ่ม `changed_by` เข้า diagram แล้ว

---

### Gap 5: Lambdas เพิ่มเติมไม่อยู่ใน diagram

| Lambda             | Endpoint                                    | มีใน Diagram |
| ------------------ | ------------------------------------------- | ------------ |
| `list-missions`    | `GET /missions` (list by team)              | ❌           |
| `presigned-url`    | `POST /missions/{request_id}/presigned-url` | ❌           |
| `outbox-processor` | Scheduled (ทุก 1 นาที)                      | ❌           |
| `authorizer`       | Lambda Authorizer                           | ❌           |

**Action**: ไม่จำเป็นต้องเพิ่มทุก Lambda ใน diagram หลัก แต่ควร note ไว้ว่ามีอยู่

---

### Gap 6: Outbox Pattern ไม่อยู่ใน diagram

- Code มี DynamoDB Outbox table + `outbox-processor` Lambda (retry ทุก 1 นาที)
- ถ้า EventBridge `PutEvents` ล้มเหลว → save to Outbox → retry ภายหลัง
- ไม่ได้แสดงใน diagram และไม่ได้กล่าวถึงใน workflow.md

**Action**: อาจเพิ่ม note เล็ก ๆ ใน diagram หรือ workflow ว่า "EventBridge publish มี Outbox fallback"

---

## 📋 สรุป Action Items

| #   | รายการ                                                   | Priority  | ไฟล์ที่ต้องแก้                                  | สถานะ                 |
| --- | -------------------------------------------------------- | --------- | ----------------------------------------------- | --------------------- |
| 1   | ~~ตรวจสอบ PUT vs PATCH กับ RescueTeam contract~~         | 🔴 High   | `diagram-flow.md`                               | ✅ แก้แล้ว            |
| 2   | ตรวจสอบ ManageDispatch endpoint (by dispatch_id vs list) | 🟡 Medium | `manage_dispatch_client.go`                     | ⏳ รอตรวจสอบ contract |
| 3   | เพิ่ม `people_count` ใน `GetMissionResponse` + map ค่า   | 🟡 Medium | `models/requests.go`, `cmd/get-mission/main.go` | ❌ ยังไม่แก้          |
| 4   | ~~อัปเดต diagram: เพิ่ม `changed_by`~~                   | 🟢 Low    | `diagram-flow.md`                               | ✅ แก้แล้ว            |
| 5   | อัปเดต workflow: เพิ่ม note เรื่อง Outbox fallback       | 🟢 Low    | `docs/demo_feedback/workflow.md`                | ❌ ยังไม่แก้          |
| 6   | ~~ลบ CloudWatch Logs ออกจาก diagram (internal)~~         | 🟢 Low    | `diagram-flow.md`                               | ✅ แก้แล้ว            |
