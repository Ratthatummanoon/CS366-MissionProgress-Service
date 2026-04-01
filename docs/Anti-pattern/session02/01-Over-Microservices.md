# 🔍 Anti-Pattern Analysis — Q1 & Q2

---

## Q1: Over-Microservices

> **"Can you describe your service in one sentence without naming another service?"**

### ดูจาก Section 2: Service Purpose

> *"MissionProgress Service คือบริการสำหรับทีมกู้ภัย (Rescue Team) เพื่อใช้ในการ**รายงานความคืบหน้าของภารกิจ**ที่ได้รับมอบหมาย **อัปเดตสถานะของเหตุการณ์ (Incident Status)** และ**บันทึกรายละเอียดการปฏิบัติงานหน้างาน (Action Logs)** เพื่อให้ศูนย์สั่งการได้รับข้อมูลที่ถูกต้องและเป็นปัจจุบันที่สุด"*

### ✅ Verdict: **ผ่าน — ไม่พบ Over-Microservices**

| เกณฑ์ | ผลลัพธ์ |
|--------|---------|
| อธิบายได้ในประโยคเดียว? | ✅ ได้ — *"บริการรายงานความคืบหน้าภารกิจ อัปเดตสถานะ และบันทึก Action Logs ของทีมกู้ภัย"* |
| ต้องอ้างชื่อ service อื่นในการอธิบายตัวเอง? | ✅ **ไม่ต้อง** — ประโยค Service Purpose ไม่มีชื่อ service อื่นเลย |
| มี Owned Data เป็นของตัวเอง? | ✅ มี — `Mission Timeline / Action Logs`, `Operational Context Updates` |
| มี Decision Logic อิสระ? | ✅ มี — State Validation, Field Assessment Acceptance, Idempotency Decision |

> Service นี้มี **identity ชัดเจน** อธิบายตัวเองได้โดยลำพัง ไม่ใช่ thin proxy ที่แค่ forward ข้อมูลอย่างเดียว

---

## Q2: God Service

> **"Does your out-of-scope section exist and is it specific? Can you describe the service without using 'and' more than once?"**

### 2.1 Out-of-Scope Section

| เกณฑ์ | ผลลัพธ์ |
|--------|---------|
| มี Out-of-scope section? | ✅ มี — Section 5 ❌ Out-of-scope |
| เป็นรายการที่เจาะจง? | ✅ เจาะจงมาก — ระบุทั้ง **สิ่งที่ไม่ทำ** และ **ใครรับผิดชอบแทน** |

Out-of-scope ที่ระบุไว้:
| สิ่งที่ปฏิเสธชัดเจน | ให้ service ไหนแทน |
|---|---|
| การสั่งการ/มอบหมายงาน | Manage Dispatch Service |
| การค้นหาเส้นทาง | SafeRoute Service |
| การจัดการทรัพยากรโรงพยาบาล | HospitalResourceStatus Service |
| เป็น Source of Truth ของ Impact/Priority | IncidentTracking Service |

> 💡 ดีมากที่ระบุว่า *"MissionProgress แค่ Forward ข้อมูล"* ไม่ได้อ้างตัวเป็นเจ้าของ

### 2.2 "AND" Test

ลองอธิบาย Service ดู:

> *"MissionProgress Service รับผิดชอบจัดการ state transition ของภารกิจกู้ภัย **และ** บันทึก timeline การปฏิบัติงานหน้างาน"*

— ใช้ "and" **1 ครั้ง** ✅

### ⚠️ แต่มีข้อสังเกต — In-Scope มี 5 responsibilities:

| # | Responsibility | จำเป็นต่อ core purpose? |
|---|---|---|
| 1 | State Transition Management | ✅ Core |
| 2 | Timeline / Action Log Recording | ✅ Core |
| 3 | Field Assessment Forwarding | ⚠️ เป็น **forwarding** — ทำไมไม่ให้ client ส่งตรงไป IncidentTracking? |
| 4 | Evidence Image Management | ⚠️ จัดการรูปภาพ S3 → อาจเป็น concern แยกได้ |
| 5 | Event Publishing | ✅ Infrastructure concern ปกติ |

### ✅ Verdict: **ผ่าน — แต่มีข้อควรระวัง**

| เกณฑ์ | ผลลัพธ์ |
|--------|---------|
| Out-of-scope มีอยู่จริง? | ✅ มี และเจาะจง |
| AND Test (≤1 ครั้ง)? | ✅ ผ่าน |
| God Service? | ✅ **ไม่ใช่** — responsibility ยังอยู่ใน domain เดียวกัน |

### ⚠️ คำแนะนำ

| จุดที่ควรระวัง | เหตุผล |
|---|---|
| **Field Assessment Forwarding** | ถ้า logic เป็นแค่ *"รับมาแล้ว publish ต่อ"* โดยไม่มี validation/transformation → อาจเป็น unnecessary middleman ได้ในอนาคต |
| **Evidence Image Management** | ถ้าเริ่มมี logic ซับซ้อน (resize, OCR, tagging) → ควรแยกออกเป็น service ใหม่ |

---

## 📊 สรุป Q1

| Anti-Pattern | Status | หมายเหตุ |
|:---|:---:|:---|
| **Over-Microservices** | ✅ ไม่พบ | มี owned data, decision logic, อธิบายตัวเองได้โดยไม่ต้องอ้าง service อื่น |

---