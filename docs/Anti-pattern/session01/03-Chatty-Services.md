## 3. Chatty Services

### 3.1 วิเคราะห์

**GET /incidents/{id}** — คืนข้อมูลครบในครั้งเดียว:
> mission info + timeline entries + incident detail (ถ้ามี) + data_source

Caller ไม่ต้องเรียกซ้ำ ไม่มี endpoint ที่คืนแค่ ID แล้วต้องไป query ต่อ

**ทดสอบ journey: "ทีมกู้ภัยดูภารกิจแล้วรายงานความคืบหน้า"**
1. `GET /incidents/INC-001` → ได้ข้อมูลภารกิจ + timeline ครบ **(1 hop)**
2. `POST /incidents/INC-001/progress` → ได้ผลลัพธ์ old/new status กลับมาทันที **(1 hop)**

**รวม 2 hops** — สมเหตุสมผลเพราะเป็นคนละ action (อ่าน vs เขียน)

### 3.2 Detected?
**No** — ไม่พบ Chatty Services

### 3.3 Fixed?
**N/A** — ไม่มีปัญหาต้องแก้