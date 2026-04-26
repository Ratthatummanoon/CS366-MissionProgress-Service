# Phase 7 — Service Endpoint Implementation Audit

> ตรวจสอบว่า endpoint / interaction ทั้งหมดทั้ง sync และ async ถูก implement ครบแล้วหรือไม่
> ตรวจสอบทั้ง Lambda code (`src/backend/cmd/`), client code (`src/backend/internal/client/`), และ Terraform (`terraform/`)

---

## ผลสรุป

**ครบทุก interaction — ไม่มีอะไรค้างอยู่**

---

## 1. Inbound Synchronous (API Endpoints ที่ Service เปิดรับ)

| #   | Endpoint                                        | Lambda                  | Terraform Route                             | สถานะ |
| :-- | :---------------------------------------------- | :---------------------- | :------------------------------------------ | :---- |
| ①   | `POST /missions/{request_id}/progress`          | `cmd/report-progress/`  | `POST /missions/{request_id}/progress`      | ✅    |
| ②   | `GET /missions/{request_id}`                    | `cmd/get-mission/`      | `GET /missions/{request_id}`                | ✅    |
| ③   | `GET /missions/{request_id}` (from Dispatch UI) | ใช้ endpoint เดียวกับ ② | —                                           | ✅    |
| ⑥   | `POST /missions/{request_id}/presigned-url`     | `cmd/presigned-url/`    | `POST /missions/{request_id}/presigned-url` | ✅    |
| ⑦   | `GET /missions?team_id=`                        | `cmd/list-missions/`    | `GET /missions`                             | ✅    |

---

## 2. Outbound Synchronous (Service เรียก External)

| #   | Endpoint                                                         | Client File                                           | ใช้ใน Lambda                                      | สถานะ |
| :-- | :--------------------------------------------------------------- | :---------------------------------------------------- | :------------------------------------------------ | :---- |
| ④   | `GET /v1/rescue-requests/{requestId}` (RescueRequest)            | `client/rescue_request_client.go`                     | `get-mission`, `mission-assigned-handler`         | ✅    |
| ④b  | `GET /v1/dispatches?teamId=` (ManageDispatch)                    | `client/manage_dispatch_client.go`                    | `get-mission`                                     | ✅    |
| ④c  | `GET /v1/teams/{teamId}` (RescueTeam)                            | `client/rescue_team_client.go` → `GetTeamDetail()`    | `get-mission`                                     | ✅    |
| ④d  | `PATCH /v1/teams/{teamId}/status` (RescueTeam — ปล่อย AVAILABLE) | `client/rescue_team_client.go` → `UpdateTeamStatus()` | `report-progress` (เมื่อ `new_status = RESOLVED`) | ✅    |

> **หมายเหตุ ④d:** interaction นี้ implement แล้วในโค้ด แต่ยังไม่บันทึกใน `docs/proposals/06-Service-Interaction.md`
> ต้องเพิ่มเป็น row ④d ในตาราง RescueTeam Service section

---

## 3. Outbound Asynchronous (Events ที่ Publish ออกไป)

| Event                             | Trigger                       | Lambda             | EventBridge Rule                    | SQS Target (conditional)                               | สถานะ |
| :-------------------------------- | :---------------------------- | :----------------- | :---------------------------------- | :----------------------------------------------------- | :---- |
| `MissionStatusChanged`            | ทุก status change             | `report-progress`  | `mission-status-changed-rule` → CWL | `incident_tracking_sqs_arn` (IncidentTracking)         | ✅    |
| `MissionStatusChanged` (RESOLVED) | `new_status = RESOLVED`       | `report-progress`  | `mission-resolved-dispatch-rule`    | `dispatch_sqs_arn` (Dispatch)                          | ✅    |
| `MissionBackupRequested`          | `new_status = NEED_BACKUP`    | `report-progress`  | `backup-requested-rule` → CWL       | `prioritization_sqs_arn` (Prioritization)              | ✅    |
| `ImpactLevelUpdated`              | มี `new_impact_level` ใน body | `report-progress`  | `impact-level-updated-rule` → CWL   | `incident_tracking_sqs_arn` + `prioritization_sqs_arn` | ✅    |
| Outbox retry                      | EventBridge publish ล้มเหลว   | `outbox-processor` | scheduled `rate(1 minute)`          | —                                                      | ✅    |

> SQS targets ทั้งหมดใช้ `count = var.xxx_sqs_arn != "" ? 1 : 0` — เปิดใช้งานอัตโนมัติเมื่อ SQS ARN ถูกส่งเข้ามาทาง Terraform variable

---

## 4. Inbound Asynchronous (Events ที่รับเข้ามา)

| Event                  | Source                  | Lambda Handler                  | EventBridge Rule                      | สถานะ |
| :--------------------- | :---------------------- | :------------------------------ | :------------------------------------ | :---- |
| `DispatchOrderCreated` | `ManageDispatchService` | `cmd/mission-assigned-handler/` | `mission-assigned-rule` (default bus) | ✅    |

Handler ทำ:

1. Idempotency check ด้วย `dispatch_id`
2. เรียก RescueRequest Service เพื่อดึง `incident_id` (degraded mode ถ้าล้มเหลว)
3. สร้าง MissionAssignment record (`status = DISPATCHED`)
4. สร้าง Timeline entry แรก

---

## 5. Build Script

ไฟล์ `script/build.sh` build Lambda ครบทั้ง 7 functions:

```
report-progress, get-mission, authorizer, outbox-processor,
presigned-url, list-missions, mission-assigned-handler
```

---

## 6. งานที่ต้องทำต่อ

### 6.1 อัปเดต `docs/proposals/06-Service-Interaction.md`

เพิ่ม interaction ④d ที่ implement แล้วแต่ยังไม่บันทึกใน docs:

**Section 6: RescueTeam Service — ตารางการสื่อสาร** เพิ่ม row:

| #   | ช่องทาง                                         | ทิศทาง                       | รายละเอียด                                                                                | สถานะ          |
| :-- | :---------------------------------------------- | :--------------------------- | :---------------------------------------------------------------------------------------- | :------------- |
| ④d  | Sync `PATCH /v1/teams/{teamId}/status` (Bearer) | MissionProgress → RescueTeam | เมื่อ `new_status = RESOLVED` → ปล่อยทีมกลับเป็น AVAILABLE (best-effort, fire-and-forget) | ✅ Implemented |

**Mermaid Diagram** เพิ่ม arrow:

```
MS -. "④d PATCH /v1/teams/{teamId}/status (RESOLVED → AVAILABLE)" .-> RescueTeamAPI
```
