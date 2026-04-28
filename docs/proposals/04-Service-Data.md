# **Service Data**

---

## **1) Mission Timeline Data** _(Owned by this service)_

> ข้อมูลประวัติการทำงานของทีมกู้ภัย (Log) แต่ละภารกิจ ใช้สำหรับสร้าง Timeline

**DynamoDB Table:** `MissionTimeline` | PK: `mission_id` | SK: `timestamp` | GSI: `log-id-index` (hash: `log_id`)

| Field Name   | Type     | Required | Description                                                      | Example                                    |
| ------------ | -------- | -------- | ---------------------------------------------------------------- | ------------------------------------------ |
| mission_id   | String   | Yes      | รหัสภารกิจ (Partition Key)                                       | MISS-a1b2c3d4                              |
| timestamp    | DateTime | Yes      | เวลาที่เกิดเหตุการณ์จริง (Sort Key)                              | 2024-10-15T12:30:00Z                       |
| log_id       | String   | Yes      | UUID ของรายการ Log (GSI key)                                     | 550e8400-e29b-41d4-a716-446655440000       |
| action_type  | String   | Yes      | ประเภทการกระทำ: `STATUS_CHANGE`, `MISSION_ASSIGNED`, `UPLOAD`    | STATUS_CHANGE                              |
| description  | String   | Yes      | รายละเอียดของสิ่งที่ทำ (auto-generated: "Status changed: X → Y") | Status changed: EN_ROUTE → ON_SITE         |
| performed_by | String   | Yes      | ID ของทีมกู้ภัยหรือ SYSTEM                                       | TEAM-01                                    |
| old_status   | String   | No       | สถานะเดิม (เฉพาะ STATUS_CHANGE)                                  | EN_ROUTE                                   |
| new_status   | String   | No       | สถานะใหม่ (เฉพาะ STATUS_CHANGE)                                  | ON_SITE                                    |
| note         | String   | No       | หมายเหตุเพิ่มเติมจากทีมกู้ภัย                                    | ถึงจุดเกิดเหตุแล้ว น้ำสูง 1.2m             |
| gps_location | String   | No       | พิกัดจากทีม (เนื้อหาจาก `current_location` field)                | 13.7563,100.5018                           |
| image_key    | String   | No       | S3 Key ของรูปภาพหลักฐาน (จาก presigned-url API)                  | evidence/MISS-abc/TEAM-01/1718352735-x.jpg |

**Notes**:

- Timeline entries จะถูกเรียงตาม `timestamp` (Sort Key)
- `action_type = MISSION_ASSIGNED` เมื่อ `mission-assigned-handler` สร้าง Mission ใหม่
- `image_key` เชื่อม Evidence รูปภาพเข้ากับ Timeline entry สำหรับดึง view URL ภายหลัง
- `gps_location` จะถูกเก็บใน DynamoDB ด้วย attribute name `gps_location`

---

## **2) Mission Assignment State** _(Owned by this service)_

> ข้อมูลสถานะปัจจุบันของภารกิจที่ทีมกู้ภัยแต่ละทีมรับผิดชอบ

**DynamoDB Table:** `MissionAssignment` | PK: `mission_id` | GSI: `request-index` (request_id), `team-index` (rescue_team_id), `dispatch-index` (dispatch_id)

| Field Name          | Type     | Required | Description                                               | Example              |
| ------------------- | -------- | -------- | --------------------------------------------------------- | -------------------- |
| mission_id          | String   | Yes      | รหัสภารกิจ (Primary Key, รูปแบบ: `MISS-{uuid8}`)          | MISS-a1b2c3d4        |
| dispatch_id         | String   | Yes      | รหัส Dispatch Order จาก Manage Dispatch Service (GSI key) | DSP-001              |
| request_id          | String   | Yes      | รหัส request จาก RescueRequest Service (lookup key, GSI)  | REQ-8812-9901        |
| incident_id         | String   | Yes      | รหัสเหตุการณ์ (ดึงจาก RescueRequest Service ตอนเกิด)      | INC-12345            |
| rescue_team_id      | String   | Yes      | รหัสทีมกู้ภัยที่รับผิดชอบภารกิจนี้ (GSI key)              | TEAM-01              |
| priority_level      | Integer  | No       | ลำดับความสำคัญจาก Manage Dispatch Service                 | 2                    |
| current_status      | String   | Yes      | สถานะปัจจุบันของภารกิจ                                    | ON_SITE              |
| latest_impact_level | Integer  | No       | ระดับความรุนแรงล่าสุดที่ประเมินโดยทีมนี้                  | 3                    |
| started_at          | DateTime | Yes      | เวลาที่เริ่มรับภารกิจ (จาก `dispatchedAt`)                | 2024-10-15T12:00:00Z |
| last_updated_at     | DateTime | Yes      | เวลาที่อัปเดตข้อมูลล่าสุด                                 | 2024-10-15T12:35:00Z |

**Notes**:

- `current_status` ค่าที่ถูกต้อง: `DISPATCHED`, `EN_ROUTE`, `ON_SITE`, `NEED_BACKUP`, `RESOLVED`
- `latest_impact_level` = 0 ถ้ายังไม่มีการประเมิน (Integer, ไม่ใช่ null)
- `request_id` คือ primary lookup key — ใช้ค้นหา mission จาก path parameter `{request_id}`
- `dispatch_id` ใช้เป็น idempotency key ใน `mission-assigned-handler`
- 1 `incident_id` มีได้หลาย `request_id` → `incident_id` ไม่ unique สำหรับ lookup
- description, location, requestType ดึงจาก RescueRequest Service แบบ on-demand (degraded mode ถ้า service ไม่พร้อม)

---

## **3) Event Outbox** _(Owned by this service)_

> ใช้สำหรับ Outbox Pattern — บันทึก EventBridge events ที่ยังไม่ได้ส่ง หรือส่งไม่สำเร็จ ให้ `outbox-processor` Lambda retry

**DynamoDB Table:** `EventOutbox` | PK: `outbox_id` | SK: `created_at` | GSI: `status-index` (status) | TTL: enabled

| Field Name    | Type    | Required | Description                                    | Example                        |
| ------------- | ------- | -------- | ---------------------------------------------- | ------------------------------ |
| outbox_id     | String  | Yes      | UUID ของ outbox record (Partition Key)         | 550e8400-e29b-41d4-a716-...    |
| created_at    | String  | Yes      | ISO 8601 เวลาที่สร้าง record (Sort Key)        | 2025-06-14T09:32:15Z           |
| event_type    | String  | Yes      | ชื่อ detail-type ของ event                     | MissionStatusChanged           |
| event_payload | String  | Yes      | JSON string ของ event detail                   | {"mission_id": "MISS-a1b2..."} |
| status        | String  | Yes      | สถานะการส่ง: `PENDING`, `SENT`, `FAILED`       | PENDING                        |
| retry_count   | Integer | Yes      | จำนวนครั้งที่ retry (max 5)                    | 0                              |
| ttl           | Integer | No       | Unix timestamp สำหรับ DynamoDB TTL auto-delete | 1718438535                     |

**Notes**:

- `outbox-processor` Lambda ดึง records ที่ `status = PENDING/FAILED` และ `retry_count < 5`
- เมื่อส่ง EventBridge สำเร็จ → อัปเดต `status = SENT`
- เมื่อ retry ครบ 5 ครั้ง → `status = FAILED` (dead letter)
- TTL ช่วยลบ SENT records อัตโนมัติ
