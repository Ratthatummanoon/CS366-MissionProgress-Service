# **Service Data**

---

## **1) Mission Timeline Data** _(Owned by this service)_

> ข้อมูลประวัติการทำงานของทีมกู้ภัย (Log) แต่ละภารกิจ ใช้สำหรับสร้าง Timeline

| Field Name   | Type     | Required | Description                                                    | Example                                    |
| ------------ | -------- | -------- | -------------------------------------------------------------- | ------------------------------------------ |
| log_id       | UUID     | Yes      | รหัสอ้างอิงของรายการ Log (Primary Key)                         | LOG-556677                                 |
| mission_id   | UUID     | Yes      | รหัสภารกิจ (เชื่อมโยงกับทีมและเหตุการณ์)                       | MIS-998800                                 |
| action_type  | String   | Yes      | ประเภทการกระทำ เช่น `"STATUS_CHANGE"`, `"COMMENT"`, `"UPLOAD"` | STATUS_CHANGE                              |
| description  | String   | Yes      | รายละเอียดของสิ่งที่ทำ                                         | Arrived at location, setting up perimeter. |
| performed_by | String   | Yes      | ID ของทีมกู้ภัยหรือเจ้าหน้าที่ที่ทำรายการ                      | TEAM-01                                    |
| timestamp    | DateTime | Yes      | เวลาที่เกิดเหตุการณ์จริง                                       | 2024-10-15T12:30:00Z                       |

**Notes**:

- Timeline entries จะถูกเรียงตาม `timestamp`

---

## **2) Mission Assignment State** _(Owned by this service)_

> ข้อมูลสถานะปัจจุบันของภารกิจที่ทีมกู้ภัยแต่ละทีมรับผิดชอบ

| Field Name          | Type     | Required | Description                                                 | Example              |
| ------------------- | -------- | -------- | ----------------------------------------------------------- | -------------------- |
| mission_id          | UUID     | Yes      | รหัสภารกิจ (Primary Key)                                    | MIS-998800           |
| dispatch_id         | String   | Yes      | รหัส Dispatch Order จาก Manage Dispatch Service             | DSP-001              |
| request_id          | String   | Yes      | รหัส request จาก RescueRequest Service (Unique, lookup key) | REQ-8812-9901        |
| incident_id         | UUID     | Yes      | รหัสเหตุการณ์ที่เชื่อมโยง (Foreign Key)                     | INC-12345            |
| rescue_team_id      | String   | Yes      | รหัสทีมกู้ภัยที่รับผิดชอบภารกิจนี้                          | TEAM-01              |
| current_status      | String   | Yes      | สถานะปัจจุบันของภารกิจ                                      | ON-SITE              |
| latest_impact_level | Number   | No       | ระดับความรุนแรงล่าสุดที่ประเมินโดยทีมนี้                    | 3                    |
| started_at          | DateTime | Yes      | เวลาที่เริ่มรับภารกิจ                                       | 2024-10-15T12:00:00Z |
| last_updated_at     | DateTime | Yes      | เวลาที่อัปเดตข้อมูลล่าสุด                                   | 2024-10-15T12:35:00Z |

**Notes**:

- `current_status` ควรสอดคล้องกับ State Machine ของระบบ (เช่น `DISPATCHED → EN_ROUTE → ON-SITE → RESOLVED`)
- `latest_impact_level` สามารถ null ได้ หากยังไม่มีการประเมิน
- `request_id` คือ primary lookup key — ใช้ค้นหา mission จาก path parameter `{request_id}`
- 1 `incident_id` มีได้หลาย `request_id` → `incident_id` ไม่ unique สำหรับ lookup
- `description`, `location`, `requestType` ดึงมาจาก RescueRequest Service แบบ on-demand (degraded mode ถ้า service ไม่พร้อม)

---
