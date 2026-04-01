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


## 📊 สรุป Q2

| Anti-Pattern | Status | หมายเหตุ |
|:---|:---:|:---|
| **God Service** | ✅ ไม่พบ | Out-of-scope ชัดเจน, AND test ผ่าน, แต่ควรระวัง scope creep จาก Field Assessment Forwarding และ Image Management |

---