# **Dependency Mapping**

---

## **Dependency 1: IncidentTracking Service**

### Overview

| Field             | Value                               |
| ----------------- | ----------------------------------- |
| Service Owner     | Krittamet Damthongkam               |
| Type              | Service                             |
| Interaction Style | Hybrid (Synchronous + Asynchronous) |
| Criticality       | High (รองรับ Degraded Mode)         |

---

### Purpose

- ใช้ดึงข้อมูล Incident (description, location, type)
- เป็น **Source of Truth** ของ Incident

---

### Interaction

| ทิศทาง         | ช่องทาง                             | รายละเอียด                                                   | Demo 1                          | Demo 2+               |
| -------------- | ----------------------------------- | ------------------------------------------------------------ | ------------------------------- | --------------------- |
| ~~ขาออก Sync~~ | ~~HTTP GET /missions/{request_id}~~ | ~~ดึงข้อมูล Incident~~ → ย้ายไปใช้ RescueRequest Service แทน | ~~⚠️ Mock (timeout → partial)~~ | ~~✅ URL จริง [TBD]~~ |
| ขาออก Async    | EventBridge MissionStatusChanged    | อัปเดตสถานะ Incident                                         | ✅ → CloudWatch Logs            | 🔜 → Service จริง     |
| ขาออก Async    | EventBridge ImpactLevelUpdated      | อัปเดต Impact Level                                          | ✅ → CloudWatch Logs            | 🔜 → Service จริง     |

> **หมายเหตุ:** Synchronous call ออก IncidentTracking ถูกยกเลิกแล้ว — ข้อมูล description/location/type ดึงมาจาก RescueRequest Service แทน (เพราะ RescueRequest Service เป็นเจ้าของ request context)

---

### Failure Handling

| กรณี                  | การจัดการ                                                        |
| --------------------- | ---------------------------------------------------------------- |
| Sync GET ล้มเหลว      | **Degraded Mode** → ส่งเฉพาะข้อมูลที่มี (`data_source: partial`) |
| Event Publish ล้มเหลว | **Outbox Pattern** → retry ภายหลัง                               |

---

### TBD

- API path & response format
- Service URL
- การ subscribe EventBridge

---

## **Dependency 2: Dispatch Management Service**

### Overview

| Field             | Value                        |
| ----------------- | ---------------------------- |
| Service Owner     | Noppakron Songkroh           |
| Type              | Service                      |
| Interaction Style | Sync + Async (Bidirectional) |
| Criticality       | Critical                     |

---

### Interaction

| ทิศทาง       | ช่องทาง                          | รายละเอียด                                            | Demo 1             | Demo 2+ |
| ------------ | -------------------------------- | ----------------------------------------------------- | ------------------ | ------- |
| ขาเข้า Async | EventBridge MissionAssignedEvent | สร้าง Mission Record                                  | ❌ Seed Data       | 🔜      |
| ขาออก Async  | MissionStatusChanged (RESOLVED)  | ปลดล็อกทีม (BUSY → AVAILABLE)                         | ✅ CloudWatch Logs | 🔜      |
| ขาออก Sync   | HTTP GET /v1/dispatch/{id}       | ดึงข้อมูล Dispatch Order (เสริม get-mission response) | Degraded Mode      | [TBD]   |
| ขาเข้า Sync  | GET /missions/{request_id}       | ดู Timeline + รูปภาพ                                  | [TBD]              | [TBD]   |

---

### Failure Handling

| กรณี            | การจัดการ                                       |
| --------------- | ----------------------------------------------- |
| ไม่ได้รับ Event | Demo 1: ไม่กระทบ / Demo 2+: Mission ไม่ถูกสร้าง |
| Publish ล้มเหลว | Outbox Pattern                                  |

---

## **Dependency 3: Rescue Prioritization**

### Overview

| Field             | Value                 |
| ----------------- | --------------------- |
| Service Owner     | Nattasak Chonmanat    |
| Type              | Service               |
| Interaction Style | Async (Event-Driven)  |
| Criticality       | Medium (Non-blocking) |

---

### Interaction

| ทิศทาง      | Event                  | รายละเอียด           | Demo 1             | Demo 2+ |
| ----------- | ---------------------- | -------------------- | ------------------ | ------- |
| ขาออก Async | MissionBackupRequested | คำนวณ Priority ใหม่  | ✅ CloudWatch Logs | 🔜      |
| ขาออก Async | ImpactLevelUpdated     | อัปเดตลำดับความสำคัญ | ✅ CloudWatch Logs | 🔜      |

---

### Failure Handling

- **Non-blocking** → ระบบหลักยังทำงานได้
- EventBridge มี retry 24 ชม.
- ใช้ **Outbox Pattern** เป็น safety net

---

## **Dependency 4: Amazon S3 (Evidence Storage)**

### Overview

| Field             | Value                           |
| ----------------- | ------------------------------- |
| Type              | Infrastructure (Object Storage) |
| Interaction Style | Synchronous                     |
| Criticality       | Medium                          |
| Demo 2+           | Presigned URL + Direct Upload   |

---

### Purpose

- เก็บรูปภาพหลักฐาน
- Upload ผ่าน **Presigned URL**

---

### Failure Handling

| กรณี                 | การจัดการ               |
| -------------------- | ----------------------- |
| Generate URL ล้มเหลว | Retry                   |
| Upload ล้มเหลว       | ข้ามได้ → ส่ง Text ก่อน |
| S3 ล่ม               | Degrade → ใช้ Text only |

---

## **Dependency 5: RescueRequest Service**

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

---

## **Dependency 6: API Gateway + Authorizer**

### Overview

| Field             | Value                 |
| ----------------- | --------------------- |
| Type              | Infrastructure (Auth) |
| Interaction Style | Sync                  |
| Criticality       | Critical              |

---

### Purpose

- รับ HTTP Request ทั้งหมด
- ตรวจสอบ `x-api-key`, `X-Rescue-Team-ID`

---

### Failure Handling

| กรณี            | การจัดการ                 |
| --------------- | ------------------------- |
| API Key ผิด     | 403                       |
| ไม่มี Header    | 403                       |
| Authorizer ล่ม  | 500                       |
| API Gateway ล่ม | ใช้ local cache (Demo 2+) |

---

## **Dependency 7: Amazon EventBridge**

### Overview

| Field       | Value                   |
| ----------- | ----------------------- |
| Type        | Event Bus               |
| Bus Name    | mission-progress-events |
| Interaction | Async                   |
| Criticality | Critical                |

---

### Events

| Event                  | Trigger             | Demo 1          | Demo 2+                   |
| ---------------------- | ------------------- | --------------- | ------------------------- |
| MissionStatusChanged   | สถานะเปลี่ยน        | CloudWatch Logs | Incident + Dispatch       |
| MissionBackupRequested | NEED_BACKUP         | CloudWatch Logs | Prioritization            |
| ImpactLevelUpdated     | มี new_impact_level | CloudWatch Logs | Incident + Prioritization |

---

### Failure Handling

**ระดับ 1: Publish ล้มเหลว**

- ใช้ **Outbox Pattern**
- Retry ผ่าน processor (Demo 2+)

**ระดับ 2: Delivery ล้มเหลว**

- EventBridge retry อัตโนมัติ (24 ชม.)
- รองรับ DLQ

---

## **Dependency 8: RescueTeam Service**

### Overview

| Field             | Value                         |
| ----------------- | ----------------------------- |
| Service Owner     | กมลพันธ์ กันธายอด             |
| Type              | Service                       |
| Interaction Style | Synchronous (REST)            |
| Criticality       | Medium (รองรับ Degraded Mode) |

---

### Purpose

- ดึงข้อมูลทีมกู้ภัย (ชื่อ, ความสามารถ) เพื่อเสริมข้อมูลใน `get-mission` response

### Interaction

| ทิศทาง     | ช่องทาง         | รายละเอียด                             | Demo 1        | Demo 2+ |
| ---------- | --------------- | -------------------------------------- | ------------- | ------- |
| ขาออก Sync | HTTP GET /teams | ดึงข้อมูลทีม (ล้มเหลว → Degraded Mode) | Degraded Mode | [TBD]   |

### Failure Handling

| กรณี             | การจัดการ                                                        |
| ---------------- | ---------------------------------------------------------------- |
| Sync GET ล้มเหลว | **Degraded Mode** → ส่งเฉพาะข้อมูลที่มี (`data_source: partial`) |

### TBD

- Service URL
- Bearer token สำหรับ service-to-service auth

---

## **📊 สรุปภาพรวม**

| #   | Dependency         | Type           | Interaction   | Criticality | Demo 1      | Demo 2+      |
| --- | ------------------ | -------------- | ------------- | ----------- | ----------- | ------------ |
| 1   | IncidentTracking   | Service        | Async only    | Medium      | Logs        | 🔜           |
| 2   | Dispatch           | Service        | Bidirectional | Critical    | Seed + Logs | 🔜           |
| 3   | Prioritization     | Service        | Async         | Medium      | Logs        | 🔜           |
| 4   | S3                 | Infrastructure | Upload        | Medium      | ❌          | 🔜           |
| 5   | RescueRequest      | Service        | Sync          | High        | ❌          | ✅ Real      |
| 6   | API Gateway + Auth | Infrastructure | Sync          | Critical    | ✅          | ✅           |
| 7   | EventBridge        | Infrastructure | Async         | Critical    | Logs        | Real Targets |
| 8   | RescueTeam         | Service        | Sync          | Medium      | Degraded    | [TBD]        |

---
