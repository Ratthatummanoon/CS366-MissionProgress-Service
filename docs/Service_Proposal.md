# Service Overview

ภาพรวมของบริการ (Service Overview)  
MissionProgress Service

1. Service Owner

- นายรัฐธรรมนูญ โคสาแสง (Ratthatummanoon Kosasang)
- รหัสนักศึกษา 6609612178

2. Service Purpose

MissionProgress Service คือบริการสำหรับทีมกู้ภัย (Rescue Team) เพื่อใช้ในการรายงานความคืบหน้าของภารกิจที่ได้รับมอบหมาย อัปเดตสถานะของเหตุการณ์ (Incident Status) และบันทึกรายละเอียดการปฏิบัติงานหน้างาน (Action Logs) เพื่อให้ศูนย์สั่งการได้รับข้อมูลที่ถูกต้องและเป็นปัจจุบันที่สุด

3. Pain Point ที่แก้ไข

ปัจจุบันศูนย์สั่งการ (Command Center) ขาดการมองเห็นภาพรวมและการติดตามสถานะการทำงานของทีมกู้ภัยแบบเรียลไทม์ (Lack of real-time visibility) รวมถึงขาดข้อมูลประเมินความรุนแรง (Impact Assessment) ที่ถูกต้องแม่นยำจากหน้างานจริง ทำให้การตัดสินใจสั่งการหรือสนับสนุนทรัพยากรเกิดความล่าช้าและผิดพลาด

4\. Target Users

- Rescue Team (ผู้ใช้งานหลัก): ใช้สำหรับอัปเดตสถานะ แจ้งพิกัดเมื่อถึงหน้างาน และประเมินความรุนแรง
- Dispatcher: ใช้ดูข้อมูล Timeline การทำงานเพื่อติดตามผล (ผ่านการดึงข้อมูลไปแสดงผล)

5\. Service Boundary

- In-scope Responsibilities (สิ่งที่บริการนี้รับผิดชอบ)
  - รับผิดชอบการเปลี่ยนสถานะของภารกิจกู้ภัย (State Transitions) ตาม Workflow: DISPATCHED → EN_ROUTE → ON-SITE → NEED_BACKUP → RESOLVED
  - บันทึก Log การปฏิบัติงาน (Timeline) เช่น เวลาที่ถึงจุดเกิดเหตุ, การกระทำที่ทำลงไป (Evacuation start, First aid applied)
  - รับข้อมูลประเมินความรุนแรง (Impact Level) และความเร่งด่วน (Priority) ใหม่จากหน้างานเพื่อปรับปรุงข้อมูลเหตุการณ์
  - แก้ไขรายละเอียดตำแหน่ง (Exact Location Description) หากหมุดพิกัดเดิมคลาดเคลื่อน
- Out-of-scope / Not Responsible For (ไม่รับผิดชอบ)
  - ไม่รับผิดชอบการ "สั่งการ" หรือ "มอบหมายงาน" (Assignment) ให้ทีมกู้ภัย (เป็นหน้าที่ของ _Manage Dispatch Service_)
  - ไม่รับผิดชอบการค้นหาเส้นทาง (เป็นหน้าที่ของ _SafeRoute Service_)
  - ไม่รับผิดชอบการจัดการทรัพยากรโรงพยาบาลหรือการส่งตัวผู้ป่วย (เป็นหน้าที่ของ _HospitalResourceStatus Service_)

6\. Autonomy / Decision Logic  
บริการมีความเป็นอิสระในการตัดสินใจเกี่ยวกับ:

- การตรวจสอบสถานะ (Status Validation): ตรวจสอบว่าการเปลี่ยนสถานะสมเหตุสมผลหรือไม่ เช่น ต้องเป็น ON-SITE ก่อนจึงจะสามารถกด RESOLVED ได้ หรือการกด NEED_BACKUP จะต้อง Trigger การแจ้งเตือน
- การอัปเดตข้อมูลวิกฤต (Critical Data Update): สามารถตัดสินใจปรับ Impact Level หรือ Priority ของ Incident ได้ทันทีตามข้อมูลจากหน้างานโดยไม่ต้องรอการอนุมัติ เพื่อให้ระบบอื่น ๆ (เช่น Rescue Prioritization) นำข้อมูลไปคำนวณใหม่ได้ทันที

การตัดสินใจอิงจาก:

- Input จากทีมกู้ภัยหน้างาน (User Action)
- สถานะปัจจุบันของ Incident (Current State)

บริการสามารถตัดสินใจได้เองภายใต้ business rules ที่กำหนด โดยไม่ต้องรอ/ต้องรอการอนุมัติจากมนุษย์ในกรณีปกติ  
7\. Owned Data  
ข้อมูลที่บริการนี้เป็นเจ้าของและดูแลโดยตรง (Source of Truth)

- Mission Timeline / Action Logs: ข้อมูลประวัติการทำงานทั้งหมดที่เป็น Array of objects (เก็บ เวลา, เหตุการณ์, รายละเอียด, ผู้กระทำ)
- Operational Context Updates: ข้อมูลบริบทหน้างานล่าสุดที่ทีมกู้ภัยส่งมา เช่น last_update_at, updated_by (Rescue Unit ID)

8\. Linked Data (Reference Only)  
ข้อมูลที่บริการนี้ต้องอ้างอิงจากบริการอื่น แต่ไม่ได้เป็นเจ้าของหลัก

- Incident Master Data: อ้างอิง incident*id, incident_type, incident_description จาก \_IncidentTracking Service* (หรือ Core Incident Schema) เพื่อแสดงผลให้ทีมกู้ภัยดู
- Rescue Team Info: อ้างอิง ID ของทีมกู้ภัยจากระบบจัดการทีม (เพื่อระบุว่าใครเป็นคนส่ง Log)

9\. Non-Functional Requirements

- High Availability (ความพร้อมใช้งานสูง): บริการต้องพร้อมใช้งานตลอด 24/7 (SLA 99.9%) เนื่องจากเป็นระบบกู้ภัยที่หยุดชะงักไม่ได้ หากระบบล่มจะส่งผลต่อความปลอดภัยของผู้ประสบภัย
- Low Latency (ความหน่วงต่ำ): การส่งข้อมูลสถานะ (Update Status) และการดึงข้อมูล (Read Timeline) ต้องใช้เวลาตอบสนองไม่เกิน 500ms เพื่อให้ศูนย์สั่งการเห็นข้อมูลเป็น Real-time
- Concurrent Handling (การรองรับผู้ใช้พร้อมกัน): รองรับการยิง Request พร้อมกันจากทีมกู้ภัยหลายร้อยทีมในช่วงวิกฤต (Spike Traffic) ได้โดยไม่ล่ม
- Data Integrity (ความถูกต้องของข้อมูล): ข้อมูล Timeline ต้องเรียงลำดับเวลาถูกต้องเสมอ (Sequential Consistency) ห้ามข้อมูลสลับกัน
- Resilience (ความทนทาน): หากการเชื่อมต่ออินเทอร์เน็ตของทีมกู้ภัยหลุดชั่วคราว แอปพลิเคชันควรสามารถเก็บข้อมูลไว้ในเครื่อง (Local Cache) และส่งใหม่ (Retry) เมื่อต่อเน็ตได้ (Client-side consideration แต่กระทบการออกแบบ API ให้รองรับ Idempotency)

# Sync Contract

Synchronous Function Contract  
Base URL: https://api.disaster-management.net/mission-progress/v1

# API Contract \#1: Report Progress & Update Status

(รวม Action A และ Action C: ใช้บันทึก Timeline และเปลี่ยนสถานะในครั้งเดียว)

### ข้อมูลทั่วไป

- Name: reportMissionProgress
- Method: POST
- Path: /incidents/{incident_id}/progress
- Type: Synchronous

### คำอธิบาย

### ใช้สำหรับทีมกู้ภัยเพื่อ "บันทึกการปฏิบัติงาน" (Create new record) ลงใน Timeline และ "อัปเดตสถานะ" (Update status) ของเหตุการณ์ เช่น แจ้งว่าถึงที่เกิดเหตุแล้ว หรือแจ้งปิดงาน

### Request

Path/Query Params

- incident_id (Path, UUID): รหัสเหตุการณ์ที่กำลังปฏิบัติภารกิจ

  Headers

- Content-Type: application/json

- X-Rescue-Team-ID: \<รหัสทีมกู้ภัยที่ส่งข้อมูล\> (ใช้ระบุตัวตนผู้รายงาน)

  Body:

  {

  "status": "ON-SITE", // สถานะใหม่ (Enum: DISPATCHED, EN_ROUTE, ON-SITE, NEED_BACKUP, RESOLVED)

  "action_description": "ถึงจุดเกิดเหตุแล้ว กำลังเริ่มปฐมพยาบาลเบื้องต้น", // รายละเอียดการทำงาน

  "current_location": {

      "latitude": 13.7563,

      "longitude": 100.5018

  },

  "new_impact_level": 3 // (Optional) ส่งมาเฉพาะเมื่อต้องการปรับระดับความรุนแรงใหม่

  }

### Response

Success:

{

"incident_id": "INC-12345",

"current_status": "ON-SITE",

"timestamp": "2024-10-15T12:35:00Z",

"message": "Progress reported successfully"

}

Error:

Error (400 Bad Request): ข้อมูลไม่ถูกต้อง หรือ Status flow ผิดลำดับ

{

"error": {

    "code": "INVALID\_STATE\_TRANSITION",

    "message": "Cannot transition from 'DISPATCHED' to 'RESOLVED' directly. Must be 'ON-SITE' first.",

    "timestamp": "2024-10-15T12:35:05Z"

}

}

Error (404 Not Found): ไม่พบ incident_id ในระบบ

{

"error": {

    "code": "INCIDENT\_NOT\_FOUND",

    "message": "Incident ID INC-12345 does not exist.",

    "timestamp": "2024-10-15T12:35:05Z"

}

}

### Dependency / Reliability

- Internal Data: บันทึกข้อมูลลงฐานข้อมูลของ MissionProgress Service (Timeline Table)
- External Service (Notification): หาก status เป็น NEED_BACKUP หรือ RESOLVED บริการนี้อาจต้องเรียกไปที่ Rescue Prioritization Service หรือ IncidentTracking Service เพื่อแจ้งเตือนทันที (แต่ถ้าออกแบบเป็น Asynchronous จะไประบุใน Async Contract แทน)
- Validation: ตรวจสอบว่า status ที่ส่งมาถูกต้องตาม Flow หรือไม่ (เช่น จาก DISPATCHED ข้ามไป RESOLVED เลยไม่ได้)

# API Contract \#2: View Incident Mission Details

(Action B: ดูข้อมูลเหตุการณ์)

### ข้อมูลทั่วไป

- Name: getMissionDetails
- Method: GET
- Path: /incidents/{incident_id}
- Type: Synchronous

### คำอธิบาย

### ใช้ดึงข้อมูลรายละเอียดของเหตุการณ์ รวมถึง "ประวัติการทำงานทั้งหมด (Timeline)" เพื่อให้ทีมกู้ภัยดูย้อนหลังหรือตรวจสอบข้อมูลก่อนเริ่มงาน

### Request

Path/Query Param

- incident_id (Path, UUID): รหัสเหตุการณ์ที่ต้องการดู

  Headers

- X-Rescue-Team-ID: \<รหัสทีมกู้ภัย\>

  Body

  ไม่มี

  Validation

- ตรวจสอบว่า incident_id มีอยู่จริงหรือไม่

### Response

Success:

{

"incident_id": "INC-12345",

"description": "น้ำท่วมสูง 2 เมตร ติดค้างในบ้าน 3 คน", // ดึงจาก IncidentTracking หรือ Cache

"current_status": "ON-SITE",

"exact_location": "13.7563, 100.5018",

"timeline": \[

    {

      "time": "2024-10-15T12:00:00Z",

      "action": "Mission Assigned",

      "by": "Dispatcher"

    },

    {

      "time": "2024-10-15T12:10:00Z",

      "action": "En Route",

      "detail": "ออกจากฐาน กำลังเดินทาง",

      "by": "Rescue Team A"

    }

\]

}

Error:

Error (404 Not Found): ไม่พบข้อมูลเหตุการณ์

{

"error": {

    "code": "INCIDENT\_NOT\_FOUND",

    "message": "Incident ID INC-12345 not found.",

    "timestamp": "2024-10-15T12:10:05Z"

}

}

### Dependency / Reliability

- Dependency (Critical): ต้องดึงข้อมูล Master Data (Description, Location) จาก IncidentTracking Service
  - _Failure Handling:_ หาก IncidentTracking Service ล่ม ให้แสดงข้อมูลล่าสุดที่ Cache ไว้ใน MissionProgress Service แทน หรือแสดงเฉพาะ Timeline ที่ตนเองถืออยู่เพื่อให้งานเดินต่อได้

# Async Contract

Asynchronous Function Contract

## Message Contract \#1: Mission Status Changed

ใช้แจ้งระบบอื่น ๆ เมื่อสถานะของภารกิจเปลี่ยนแปลง (เช่น ทีมกู้ภัยถึงหน้างาน หรือ ปิดงานแล้ว)

### ข้อมูลทั่วไป

- Message Name: MissionStatusChangedEvent
- Interaction Style: Asynchronous (Publish/Subscribe)
- Producer: MissionProgress Service
- Consumer: IncidentTracking Service, Rescue Prioritization Service, Dispatch Management Service
- Channel/Queue: mission.status.updates.v1
- Version: v1

### คำอธิบาย

### Event นี้จะถูกส่งออกมาเมื่อทีมกู้ภัยทำการเปลี่ยนสถานะ (Update Status) สำเร็จ เพื่อให้ระบบ Incident Tracking อัปเดตสถานะรวมของเหตุการณ์ และระบบ Dispatch ทราบสถานะล่าสุดของทีม

### Request

### Message Headers

- Event-ID: evt-889900
- Source: MissionProgressService
- Content-Type: application/json
- Timestamp: 2024-10-15T12:30:05Z

  Message Body

```
  {

  "incident_id": "INC-12345",

  "rescue_team_id": "TEAM-01",

  "previous_status": "EN_ROUTE",

  "current_status": "ON-SITE",

  "changed_at": "2024-10-15T12:30:00Z",

  "location_snapshot": {

      "lat": 13.7563,

      "lng": 100.5018

  }

  }
```

Field Definition:

| Field             | Type     | Required | Description                                                            |
| ----------------- | -------- | -------- | ---------------------------------------------------------------------- |
| incident_id       | UUID     | Yes      | รหัสของเหตุการณ์ที่ถูกอัปเดต                                           |
| rescue_team_id    | String   | Yes      | รหัสทีมกู้ภัยที่ปฏิบัติงาน                                             |
| current_status    | String   | Yes      | สถานะใหม่ (Enum: DISPATCHED, EN_ROUTE, ON-SITE, NEED_BACKUP, RESOLVED) |
| changed_at        | DateTime | Yes      | เวลาที่เกิดการเปลี่ยนแปลงสถานะ                                         |
| location_snapshot | Object   | No       | พิกัดล่าสุดขณะเปลี่ยนสถานะ                                             |

Validation Rules

- incident_id (UUID, Required): รหัสเหตุการณ์
- current_status (String, Required): สถานะใหม่ (EN_ROUTE, ON-SITE, RESOLVED)

Response

ในกรณีที่เป็น Event ส่วนใหญ่จะไม่มีการตอบกลับ แต่ถ้าใน Template มีช่องให้ใส่ จะหมายถึง Acknowledge หรือ Confirmation จากระบบรับ

    Message Headers

- Correlation-ID: evt-889900 (อ้างอิง ID เดิม)
- Status: PROCESSED

Success Message Body

```
{

    "status": "SUCCESS",

    "message": "Event received and processed.",

    "processed\_at": "2024-10-15T12:30:06Z"

}
```

Reject/Error Message Body

```
{

    "status": "FAILED",

    "error\_code": "INVALID\_SCHEMA",

    "message": "Missing required field: rescue\_team\_id"

}
```

Field Definition:

| Field      | Type   | Required | Description                             |
| ---------- | ------ | -------- | --------------------------------------- |
| status     | String | Yes      | ผลลัพธ์การรับข้อความ (SUCCESS / FAILED) |
| message    | String | Yes      | รายละเอียดผลลัพธ์                       |
| error_code | String | No       | รหัสข้อผิดพลาด (กรณี Failed)            |

    Validation Rules

- incident_id: ต้องเป็นรูปแบบ UUID ที่ถูกต้องและมีอยู่ในระบบจริง
- current_status: ต้องเป็นค่าที่กำหนดไว้ใน State Machine เท่านั้น (DISPATCHED, EN_ROUTE, ON-SITE, NEED_BACKUP, RESOLVED)
- rescue_team_id: ห้ามเป็นค่าว่าง (Null/Empty) ต้องระบุตัวตนทีมกู้ภัยเสมอ
- changed_at: ต้องเป็นรูปแบบ Timestamp (ISO 8601\) และห้ามเป็นเวลาในอนาคต

# Service Data

# 1\) Mission Timeline Data (Owned by this service)

ข้อมูลประวัติการทำงานอย่างละเอียดของทีมกู้ภัย (Log) ที่เกิดขึ้นในแต่ละภารกิจ ใช้สำหรับสร้าง Timeline

| Field Name   | Type     | Required | Description                                              | Example                                    |
| ------------ | -------- | -------- | -------------------------------------------------------- | ------------------------------------------ |
| log_id       | UUID     | Yes      | รหัสอ้างอิงของรายการ Log (Primary Key)                   | LOG-556677                                 |
| mission_id   | UUID     | Yes      | รหัสภารกิจ (เชื่อมโยงกับทีมและเหตุการณ์)                 | MIS-998800                                 |
| action_type  | String   | Yes      | ประเภทการกระทำ เช่น "STATUS_CHANGE", "COMMENT", "UPLOAD" | STATUS_CHANGE                              |
| description  | String   | Yes      | รายละเอียดของสิ่งที่ทำ                                   | Arrived at location, setting up perimeter. |
| performed_by | String   | Yes      | ID ของทีมกู้ภัยหรือเจ้าหน้าที่ที่ทำรายการ                | TEAM-01                                    |
| timestamp    | DateTime | Yes      | เวลาที่เกิดเหตุการณ์จริง                                 | 2024-10-15T12:30:00Z                       |
| gps_location | String   | No       | พิกัด GPS ณ จุดที่บันทึกข้อมูล (Lat, Long)               | 13.7563, 100.5018                          |

#

#

# 2\) Mission Assignment State (Owned by this service)

ข้อมูลสถานะปัจจุบันของภารกิจที่ทีมกู้ภัยแต่ละทีมรับผิดชอบ (Current State)

| Field Name          | Type     | Required | Description                                    | Example              |
| ------------------- | -------- | -------- | ---------------------------------------------- | -------------------- |
| mission_id          | UUID     | Yes      | รหัสภารกิจ (Primary Key)                       | MIS-998800           |
| incident_id         | UUID     | Yes      | รหัสเหตุการณ์ที่เชื่อมโยง (Foreign Key)        | INC-12345            |
| rescue_team_id      | String   | Yes      | รหัสทีมกู้ภัยที่รับผิดชอบภารกิจนี้             | TEAM-01              |
| current_status      | String   | Yes      | สถานะปัจจุบันของภารกิจ                         | ON-SITE              |
| latest_impact_level | Integer  | No       | ระดับความรุนแรงล่าสุดที่ประเมินโดยทีมนี้ (1-4) | 3                    |
| started_at          | DateTime | Yes      | เวลาที่เริ่มรับภารกิจ                          | 2024-10-15T12:00:00Z |
| last_updated_at     | DateTime | Yes      | เวลาที่อัปเดตข้อมูลล่าสุด                      | 2024-10-15T12:35:00Z |

# Service Architecture

**Service Architecture**  
**![][image1]**

Mermaid  
flowchart TD  
 %% Actors  
 User\[Rescue Team\\n(Web Browser on Mobile)\]

    %% Frontend Hosting
    subgraph Frontend\_Hosting \[AWS Amplify / Vercel\]
        WebApp\[MissionProgress Web Client\\n(React/Vue.js)\]
    end

    %% AWS Backend
    subgraph Backend\_Services \[AWS Backend (Serverless)\]
        style Backend\_Services fill:\#f9f9f9,stroke:\#333,stroke-width:2px

        %% Entry Point
        APIGateway\[Amazon API Gateway\\n(REST API)\]

        %% Auth
        Cognito\[Amazon Cognito\\n(User Login)\]

        %% Logic
        Lambda\[AWS Lambda\\n(Node.js API Handler)\]

        %% Storage
        DynamoDB\[(Amazon DynamoDB\\nMission Data)\]
        S3Bucket\[Amazon S3\\n(Image Uploads)\]

        %% Messaging
        EventBridge{Amazon EventBridge\\n(Async Events)}
    end

    %% External
    subgraph Consumers \[External Systems\]
        IncidentService\[IncidentTracking\]
        DispatchService\[DispatchManagement\]
    end

    %% Flows
    User \-- HTTPS \--\> WebApp
    WebApp \-- 1\. Login \--\> Cognito
    WebApp \-- 2\. API Call (Bearer Token) \--\> APIGateway

    APIGateway \--\> Lambda

    Lambda \-- 3\. Read/Write Data \--\> DynamoDB
    Lambda \-- 4\. Generate Presigned URL \--\> S3Bucket
    WebApp \-.-\>|5. Direct Upload Image| S3Bucket

    Lambda \-- 6\. Publish Event \--\> EventBridge
    EventBridge \--\> Consumers

**Components**  
1\. **MissionProgress Web Client (Frontend):**

- พัฒนาด้วย Web Framework ยอดนิยม (เช่น React, Vue, Next.js) ที่ทีมถนัด
- Ease of Impl: ไม่ต้องทำ Mobile App Native แค่ทำ Responsive Web ให้แสดงผลดีบนมือถือ
- Hosting: ฝากไฟล์เว็บไว้บน AWS Amplify หรือ S3 \+ CloudFront

2\. **Amazon Cognito (Authentication):**

- ใช้สำหรับหน้า Login ของทีมกู้ภัย
- Ease of Impl: ไม่ต้องเขียนระบบ Login เอง ใช้ Cognito มีหน้า Login สำเร็จรูปให้ หรือใช้ Library `aws-amplify` ในโค้ด Frontend ได้เลย

3\. **Amazon API Gateway \+ AWS Lambda (Backend API):**

- ทำหน้าที่เป็น Backend รับ Request จากหน้าเว็บ
- **Ease of Impl:** เขียน Function แค่ไฟล์เดียว (เช่น `handler.js` ใน Node.js) ก็รับส่งข้อมูลได้แล้ว ไม่ต้องตั้ง Server (EC2) ให้ยุ่งยาก

4\. **Amazon S3 (Image Storage):**

- ใช้เก็บรูปภาพหน้างาน
- **Ease of Impl:** ให้ Frontend อัปโหลดรูปตรงไป S3 (ใช้ Presigned URL) จะง่ายกว่าการส่งรูปผ่าน API Server ซึ่งซับซ้อนและช้า

5\. **Amazon DynamoDB (Database):**

- เก็บข้อมูล JSON ตรงๆ (NoSQL) เหมาะกับ Web App ที่ Schema อาจมีการเปลี่ยนบ่อยๆ ตอนทำโปรเจกต์

**Explanation**  
1\. **User Access & Authentication (การเข้าถึงและยืนยันตัวตน):**

- ทีมกู้ภัยเข้าใช้งานผ่าน **Web Browser** บนมือถือ (ไม่ต้องลงแอป)
- ระบบใช้ **Amazon Cognito** ในการจัดการ User Management เมื่อกู้ภัย Login สำเร็จ จะได้รับ **JWT Token** เพื่อใช้แนบไปกับทุก Request ยืนยันว่า "ฉันคือทีมกู้ภัย A จริงๆ"

2\. **Evidence Upload Flow (การอัปโหลดหลักฐานหน้างาน):**

- เมื่อทีมกู้ภัยต้องการส่งรูปภาพหน้างาน (Evidence) เพื่อลดภาระของ Server ระบบใช้เทคนิค **Direct Upload**:
  - Frontend ขอ "Presigned URL" จาก API
  - Frontend อัปโหลดไฟล์รูปภาพตรงไปยัง **Amazon S3**
  - เมื่ออัปโหลดเสร็จ จะได้ "Image Key" (ที่อยู่ไฟล์) มาเพื่อเตรียมส่งพร้อมข้อมูลข้อความ

3\. **Core Transaction Processing (การประมวลผลคำสั่งหลัก):**

- User ส่งข้อมูลสถานะ (Status) และรายละเอียด (Logs) ผ่าน **Amazon API Gateway** (REST API)
- **AWS Lambda (Business Logic)** ถูกปลุกขึ้นมาทำงาน เพื่อตรวจสอบกฎทางธุรกิจ (Business Rules) เช่น _“สถานะปัจจุบันคือ En-route จะข้ามไป Resolved เลยไม่ได้ ต้องเป็น On-site ก่อน”_
- หากตรวจสอบผ่าน Lambda จะบันทึกข้อมูล State ล่าสุดและ Timeline ลงใน **Amazon DynamoDB** ซึ่งออกแบบมาให้รับแรงกระแทกจากการเขียนข้อมูลจำนวนมาก (Write-heavy) ได้ดี

4\. **Asynchronous Event Propagation (การกระจายข่าวแบบอะซิงโครนัส):**

- ทันทีที่บันทึกข้อมูลสำเร็จ Lambda จะทำหน้าที่เป็น Producer ส่ง Event ชื่อ `MissionStatusChangedEvent` ไปยัง **Amazon EventBridge**
- **EventBridge** จะทำหน้าที่เป็น Router กระจายข่าวสารไปยัง Service ปลายทางที่ Subscribe ไว้:
  - ส่งไป **IncidentTracking Service** เพื่ออัปเดตสถานะภาพรวมของเหตุการณ์
  - ส่งไป **Dispatch Management Service** (กรณีสถานะเป็น Resolved) เพื่อปลดล็อกทีมกู้ภัยให้ว่างพร้อมรับงานใหม่

5\. **Reliability & Failure Handling (ความทนทาน):**

- หากระบบปลายทาง (เช่น IncidentTracking) ล่ม หรือส่ง Event ไม่ผ่าน ระบบจะส่ง Message นั้นเข้าสู่ **Dead Letter Queue (DLQ)** โดยอัตโนมัติ เพื่อให้ Admin หรือระบบ Retry มาตามเก็บตกข้อมูลทีหลัง ทำให้ข้อมูลสถานะภารกิจไม่มีวันสูญหาย

# Service Interaction

**Service Interaction**  
**Mermaid**  
graph TD  
 %% \--- Nodes Definition \---  
 subgraph Upstream \[Upstream Services\]  
 RescueApp(\[Rescue Team Web App\])  
 CommandDash(\[Command Center Dashboard\])  
 end

    subgraph Core\_Service \[MissionProgress Service\]
        MissionAPI\[MissionProgress API\]
        EventBus{EventBridge}
    end

    subgraph Downstream \[Downstream Services\]
        IncidentService\[IncidentTracking Service\]
        DispatchService\[Dispatch Management Service\]
        PriorityService\[Rescue Prioritization Service\]
    end

    %% \--- Upstream Interactions (Inbound) \---
    RescueApp \-- "1. POST /progress (Update Status & Log)" \--\> MissionAPI
    RescueApp \-- "2. GET /missions (View Timeline)" \--\> MissionAPI
    CommandDash \-- "3. GET /missions (Monitor Real-time)" \--\> MissionAPI

    %% \--- Internal & Downstream Sync Interactions \---
    MissionAPI \-- "4. GET /incidents (Verify & Get Info)" \--\> IncidentService

    %% \--- Downstream Async Interactions (Outbound Events) \---
    MissionAPI \-.-\>|"5. Publish Events"| EventBus

    EventBus \-.-\>|"Event: MissionStatusChanged"| IncidentService
    EventBus \-.-\>|"Event: MissionStatusChanged (Resolved Only)"| DispatchService
    EventBus \-.-\>|"Event: MissionBackupRequested / ImpactUpdated"| PriorityService

    %% \--- Styling \---
    linkStyle 0,1,2,3 stroke:\#333,stroke-width:2px; %% Sync Calls (Solid Line)
    linkStyle 4,5,6,7 stroke:\#d4a017,stroke-width:2px,stroke-dasharray: 5 5; %% Async Events (Dotted Line)

    style MissionAPI fill:\#f9f9f9,stroke:\#333,stroke-width:2px
    style EventBus fill:\#FF9900,stroke:\#333,stroke-width:1px

**คำอธิบาย Flow:**

- **เส้นทึบ (Solid Lines):** คือการเรียกแบบ **Synchronous** (รอผลตอบกลับทันที) เช่น การรายงานความคืบหน้า หรือการดึงข้อมูลไปแสดง
- **เส้นประ (Dotted Lines):** คือการส่ง **Asynchronous Events** (ไม่รอผล) ผ่าน EventBridge เพื่อกระจายข่าวไปยัง Service อื่นๆ (Incident, Dispatch, Prioritization)

**Upstream Services (บริการต้นทางที่เรียกใช้งาน MissionProgress Service)**

- **Rescue Team Web Application (Frontend)**
  - เรียกใช้งาน `POST /missions/{incident_id}/progress` แบบ synchronous เพื่อรายงานสถานะล่าสุด (เช่น En-route, On-site) อัปโหลดรูปภาพหลักฐาน และบันทึก Log การปฏิบัติงาน
  - เรียกใช้งาน `GET /missions/{incident_id}` แบบ synchronous เพื่อดึงประวัติการทำงาน (Timeline) ทั้งหมดของภารกิจนั้นมาแสดงผลบนหน้าจอมือถือของทีมกู้ภัย
- **Command Center Dashboard Service**
  - เรียกใช้งาน `GET /missions/{incident_id}` แบบ synchronous เพื่อดึงข้อมูล Timeline และสถานะปัจจุบันของภารกิจไปแสดงบนหน้าจอของผู้สั่งการ (Dispatcher) ทำให้เห็นความเคลื่อนไหวหน้างานแบบเรียลไทม์

**Downstream Services (บริการปลายทางที่ MissionProgress Service เรียกหรือส่งข้อมูลไป)**

- **IncidentTracking Service**
  - ถูกเรียกใช้งานแบบ synchronous ผ่าน `GET /incidents/{incident_id}` เพื่อดึงข้อมูลรายละเอียดเหตุการณ์ (เช่น พิกัด, คำอธิบาย) มาแสดงให้ทีมกู้ภัยตรวจสอบความถูกต้องก่อนเริ่มงาน
  - รับ event `MissionStatusChangedEvent` ที่ส่งออกจาก MissionProgress Service ผ่าน EventBridge แบบ asynchronous เพื่ออัปเดตสถานะรวมของเหตุการณ์ในระบบหลัก (เช่น เปลี่ยนสถานะ Incident เป็น "In Progress")
- **Dispatch Management Service**
  - รับ event `MissionStatusChangedEvent` (เฉพาะเมื่อสถานะเป็น **RESOLVED**) ผ่าน EventBridge แบบ asynchronous เพื่อรับทราบว่าภารกิจเสร็จสิ้น และทำการปลดล็อกสถานะของทีมกู้ภัยให้กลับมาเป็น "Available" พร้อมรับงานใหม่
- **Rescue Prioritization Service**
  - รับ event `MissionBackupRequestedEvent` หรือข้อมูลการอัปเดต `Impact Level` ผ่าน EventBridge แบบ asynchronous เพื่อนำค่าความรุนแรงล่าสุดจากหน้างาน ไปคำนวณลำดับความสำคัญ (Priority Score) ของเคสใหม่อีกครั้ง

# Dependency Mapping

### **Dependency Mapping – MissionProgress Service**

#### **1\. IncidentTracking Service**

- Service Owner: Krittamet Damthongkam
- Type: Service
- Interaction Style: Hybrid (Synchronous & Asynchronous)
- Purpose:
  - (Sync \- GET): ดึงข้อมูลรายละเอียดของเหตุการณ์ (`description`, `location`) เพื่อนำมาแสดงผลให้ทีมกู้ภัยดูบนหน้า Web App ก่อนเริ่มงาน
  - (Async \- Event): รับ Event `MissionStatusChanged` เพื่ออัปเดตสถานะรวมของเหตุการณ์ในระบบหลัก
- Criticality: Critical
- Failure Handling:
  - กรณีดึงข้อมูลไม่สำเร็จ (Read Fail): ระบบจะแสดงข้อความแจ้งเตือน "ไม่สามารถโหลดรายละเอียดเหตุการณ์ได้" แต่ยังอนุญาตให้ทีมกู้ภัยกดปุ่มอัปเดตสถานะได้ (Degraded Mode) โดยอิงจาก `incident_id` ที่มีอยู่
  - กรณีส่ง Event ไม่สำเร็จ (Write Fail): ระบบจะใช้กลไก Retry ของ EventBridge และหากยังไม่สำเร็จจะส่งเข้า Dead Letter Queue (DLQ) เพื่อรอการ Re-process ภายหลัง

#### **2\. Dispatch Management Service**

- Service Owner: Noppakron Songkroh
- Type: Service
- Interaction Style: Asynchronous (Event-Driven)
- Purpose: รับ Event `MissionStatusChanged` (เฉพาะสถานะ `RESOLVED`) เพื่อทำการปลดล็อกสถานะของทีมกู้ภัยในระบบจัดตารางงาน (Dispatch) ให้เปลี่ยนจาก `BUSY` เป็น `AVAILABLE` พร้อมรับงานใหม่
- Criticality: High
- Failure Handling:
  - ใช้ Exponential Backoff Retry ในระดับ Infrastructure (EventBridge) หากระบบ Dispatch ล่มชั่วคราว Event จะถูกส่งซ้ำเรื่อยๆ จนกว่าจะสำเร็จ เพื่อป้องกันปัญหาสถานะทีมกู้ภัยค้าง

#### **3\. Rescue Prioritization Service**

- Service Owner: Nattasak Chonmanat
- Type: Service
- Interaction Style: Asynchronous (Event-Driven)
- Purpose: รับข้อมูลระดับความรุนแรง (`Impact Level`) ที่ทีมกู้ภัยประเมินใหม่จากหน้างาน เพื่อนำไปคำนวณคะแนนความเร่งด่วน (Priority Score) ใหม่
- Criticality: Medium (Non-blocking for rescue operation)
- Failure Handling:
  - Fire-and-forget: หากส่งข้อมูลไม่สำเร็จ ระบบกู้ภัยหน้างานยังคงทำงานต่อได้ตามปกติโดยไม่ต้องรอผลลัพธ์ และไม่มีการ Retry ที่ซับซ้อน เพื่อลดภาระระบบ

#### **4\. Amazon S3 (Infrastructure)**

- Type: Object Storage
- Interaction Style: Synchronous (Direct Client Upload)
- Purpose: เก็บไฟล์รูปภาพและวิดีโอที่เป็นหลักฐานการปฏิบัติงาน (Evidence) จากหน้างาน เพื่อลดภาระการประมวลผลของ Application Server
- Criticality: Medium
- Failure Handling:
  - หากการอัปโหลดรูปภาพล่ม (Upload Fail) Frontend จะแจ้งเตือนผู้ใช้งาน และอนุญาตให้ User กด "ข้าม" (Skip) การส่งรูปภาพ เพื่อให้สามารถส่งเฉพาะสถานะ (Text Status) ที่สำคัญกว่าไปก่อนได้

#### **5\. Amazon Cognito (Infrastructure)**

- Type: Identity Provider (Auth Service)
- Interaction Style: Synchronous
- Purpose: ยืนยันตัวตน (Authentication) และตรวจสอบสิทธิ์ (Authorization) ของทีมกู้ภัยก่อนที่จะอนุญาตให้ส่งข้อมูลเข้าสู่ระบบ
- Criticality: Critical
- Failure Handling:
  - หากระบบ Auth ล่ม User จะไม่สามารถ Login เข้าใช้งานได้ (Hard Failure) ต้องรอจนกว่าระบบจะกลับมา

#### **6\. Amazon EventBridge (Infrastructure)**

- Type: Event Bus
- Interaction Style: Asynchronous
- Purpose: เป็นตัวกลางในการรับและกระจาย Event (Message Broker) จาก MissionProgress Service ไปยัง Service ปลายทางอื่นๆ
- Criticality: Critical
- Failure Handling:
  - หาก Event Bus มีปัญหา ระบบ Backend (Lambda) จะบันทึก Event ที่ส่งไม่ผ่านลงใน Local Database (DynamoDB \- Outbox Table) ชั่วคราว และจะมี Cron Job มาดึงไปส่งใหม่เมื่อระบบกลับมาปกติ (Outbox Pattern)
