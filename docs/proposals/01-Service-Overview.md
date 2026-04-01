# **ภาพรวมของบริการ (Service Overview)**

# MissionProgress Service

---

## 1. Service Owner

| รายละเอียด       | ค่า                                              |
| :--------------- | :----------------------------------------------- |
| **ชื่อ**         | นายรัฐธรรมนูญ โคสาแสง (Ratthatummanoon Kosasang) |
| **รหัสนักศึกษา** | 6609612178                                       |

---

## 2. Service Purpose

MissionProgress Service คือบริการสำหรับทีมกู้ภัย (Rescue Team) เพื่อใช้ในการรายงานความคืบหน้าของภารกิจที่ได้รับมอบหมาย อัปเดตสถานะของเหตุการณ์ (Incident Status) และบันทึกรายละเอียดการปฏิบัติงานหน้างาน (Action Logs) เพื่อให้ศูนย์สั่งการได้รับข้อมูลที่ถูกต้องและเป็นปัจจุบันที่สุด

---

## 3. Pain Point ที่แก้ไข

ปัจจุบันศูนย์สั่งการ (Command Center) ขาดการมองเห็นภาพรวมและการติดตามสถานะการทำงานของทีมกู้ภัยแบบเรียลไทม์ (Lack of real-time visibility) รวมถึงขาดข้อมูลประเมินความรุนแรง (Impact Assessment) ที่ถูกต้องแม่นยำจากหน้างานจริง ทำให้การตัดสินใจสั่งการหรือสนับสนุนทรัพยากรเกิดความล่าช้าและผิดพลาด

---

## 4. Target Users

| ผู้ใช้งาน                       | บทบาท                                                                 |
| :------------------------------ | :-------------------------------------------------------------------- |
| **Rescue Team** (ผู้ใช้งานหลัก) | ใช้สำหรับอัปเดตสถานะ แจ้งพิกัดเมื่อถึงหน้างาน และประเมินความรุนแรง    |
| **Dispatcher**                  | ใช้ดูข้อมูล Timeline การทำงานเพื่อติดตามผล (ผ่านการดึงข้อมูลไปแสดงผล) |

---

## 5. Service Boundary

### ✅ In-scope Responsibilities (สิ่งที่บริการนี้รับผิดชอบ)

| ความรับผิดชอบ                       | รายละเอียด                                                                                                                                                             |
| :---------------------------------- | :--------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **State Transition Management**     | รับผิดชอบการเปลี่ยนสถานะภารกิจตาม Workflow: `DISPATCHED` → `EN_ROUTE` → `ON_SITE` → `NEED_BACKUP` → `RESOLVED` พร้อม Validate ว่าการเปลี่ยนสถานะสมเหตุสมผล             |
| **Timeline / Action Log Recording** | บันทึก Log การปฏิบัติงานทุกรายการ เช่น เวลาที่ถึงจุดเกิดเหตุ, การกระทำ (Evacuation start, First aid applied), ผู้กระทำ                                                 |
| **Field Assessment Forwarding**     | รับข้อมูลประเมิน Impact Level / Priority จากทีมกู้ภัยหน้างาน บันทึกเป็น Action Log แล้ว Publish Event ไปยัง IncidentTracking (ผู้เป็นเจ้าของข้อมูล) เพื่ออัปเดตค่าจริง |
| **Evidence Image Management**       | รับและจัดเก็บหลักฐานภาพถ่ายจากหน้างาน (Evidence Images) ผ่าน S3 Presigned URL พร้อมเชื่อม Image Key กับ Timeline                                                       |
| **Event Publishing**                | Publish Domain Events (`MissionStatusChangedEvent`, `FieldAssessmentUpdatedEvent`) ไปยัง SNS เพื่อแจ้ง Service อื่นๆ                                                   |

### ❌ Out-of-scope / Not Responsible For (ไม่รับผิดชอบ)

| สิ่งที่ไม่รับผิดชอบ                                            | บริการที่รับผิดชอบ                                            |
| :------------------------------------------------------------- | :------------------------------------------------------------ |
| การ "สั่งการ" หรือ "มอบหมายงาน" ให้ทีมกู้ภัย                   | Manage Dispatch Service                                       |
| การค้นหาเส้นทาง                                                | SafeRoute Service                                             |
| การจัดการทรัพยากรโรงพยาบาล / การส่งตัวผู้ป่วย                  | HospitalResourceStatus Service                                |
| การเป็น Source of Truth ของ Impact Level / Priority / Location | IncidentTracking Service (MissionProgress แค่ Forward ข้อมูล) |

---

## 6. Autonomy / Decision Logic

บริการมีความเป็นอิสระในการตัดสินใจเกี่ยวกับ:

### 1. Status Validation (การตรวจสอบสถานะ)

- ตรวจสอบว่าการเปลี่ยนสถานะสมเหตุสมผลหรือไม่ตามตาราง State Transition ที่กำหนด
- เช่น ต้องเป็น `ON_SITE` ก่อนจึงจะ `RESOLVED` ได้
- `NEED_BACKUP` จะ Trigger การ Publish Event เพื่อแจ้งเตือน

### 2. Field Assessment Acceptance (การรับข้อมูลจากหน้างาน)

- รับข้อมูล Impact Level / Priority ที่ทีมกู้ภัยปรับจากหน้างาน ได้ทันที โดยไม่ต้องรอการอนุมัติ
- บันทึกลง Timeline เป็น Action Log พร้อม Forward ผ่าน Event ให้ IncidentTracking อัปเดต

### 3. Idempotency Decision (การตัดสินใจเรื่อง Duplicate)

- ตรวจสอบ Idempotency Key ว่า Request นี้เคยถูกประมวลผลแล้วหรือยัง
- ถ้าเคย → Return cached response ทันทีโดยไม่ประมวลผลซ้ำ

### การตัดสินใจอิงจาก

| แหล่งข้อมูล               | รายละเอียด                |
| :------------------------ | :------------------------ |
| Input จากทีมกู้ภัยหน้างาน | User Action               |
| สถานะปัจจุบันของภารกิจ    | Current State ใน DynamoDB |
| Idempotency Records       | ใน DynamoDB               |

> _บริการตัดสินใจได้เองภายใต้ Business Rules ที่กำหนด โดยไม่ต้องรอการอนุมัติจากมนุษย์ในกรณีปกติ_

---

## 7. Owned Data

> ข้อมูลที่บริการนี้เป็นเจ้าของและดูแลโดยตรง (Source of Truth)

| ข้อมูล                             | รายละเอียด                                                                                        |
| :--------------------------------- | :------------------------------------------------------------------------------------------------ |
| **Mission Timeline / Action Logs** | ข้อมูลประวัติการทำงานทั้งหมดที่เป็น Array of objects (เก็บ เวลา, เหตุการณ์, รายละเอียด, ผู้กระทำ) |
| **Operational Context Updates**    | ข้อมูลบริบทหน้างานล่าสุดที่ทีมกู้ภัยส่งมา เช่น last_update_at, updated_by (Rescue Unit ID)        |

---

## 8. Linked Data (Reference Only)

> ข้อมูลที่บริการนี้ต้องอ้างอิงจากบริการอื่น แต่ไม่ได้เป็นเจ้าของหลัก

| ข้อมูล                   | แหล่งอ้างอิง                                           | วัตถุประสงค์                                                                       |
| :----------------------- | :----------------------------------------------------- | :--------------------------------------------------------------------------------- |
| **Incident Master Data** | _IncidentTracking Service_ (หรือ Core Incident Schema) | อ้างอิง incident_id, incident_type, incident_description เพื่อแสดงผลให้ทีมกู้ภัยดู |
| **Rescue Team Info**     | ระบบจัดการทีม                                          | อ้างอิง ID ของทีมกู้ภัย เพื่อระบุว่าใครเป็นคนส่ง Log                               |

---

## 9. Non-Functional Requirements

| NFR                           | รายละเอียด                                                        | สอดคล้องกับ Architecture อย่างไร                                                                                                                                                                                                                                              |
| :---------------------------- | :---------------------------------------------------------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **High Availability (99.9%)** | ระบบต้องพร้อมใช้งาน 24/7 หยุดชะงักไม่ได้                          | Lambda + DynamoDB + S3 + SNS + SQS ทั้งหมดเป็น AWS Managed Services มี built-in HA หลาย AZ                                                                                                                                                                                    |
| **Low Latency (<500ms)**      | Update Status และ Read Timeline ต้องตอบสนอง <500ms                | Go Lambda cold start \~80-150ms, DynamoDB single-digit ms, API GW \~10-20ms → รวม \~100-200ms ปกติ                                                                                                                                                                            |
| **Concurrent Handling**       | รองรับหลายร้อยทีมพร้อมกันในช่วงวิกฤต                              | Lambda auto-scales (concurrent executions), DynamoDB On-Demand auto-scales (no capacity planning)                                                                                                                                                                             |
| **Data Integrity**            | Timeline ต้องเรียงลำดับเวลาถูกต้อง (Sequential Consistency)       | DynamoDB Sort Key ใช้ ISO 8601 timestamp → Query ด้วย ScanIndexForward=true ได้เรียงตามเวลา                                                                                                                                                                                   |
| **Resilience & Idempotency**  | รองรับ Network หลุดชั่วคราว, Client Retry ได้โดยไม่เกิด Duplicate | Client-side: เก็บ Pending Actions ใน localStorage, Retry ด้วย Idempotency Key เดิม — Server-side: Lambda ตรวจสอบ Idempotency Key ใน DynamoDB ด้วย `attribute_not_exists(PK)` Condition → ป้องกัน Duplicate แบบ Atomic — Event-level: SNS Delivery Failure → DLQ → Retry later |
| **Data Durability**           | ข้อมูลต้องไม่สูญหาย                                               | DynamoDB 99.99% durability, S3 99.99% durability, SNS + DLQ ป้องกัน Event loss                                                                                                                                                                                                |
