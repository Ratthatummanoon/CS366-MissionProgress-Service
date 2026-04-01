## 4. Overall Reflection

### 4.1 Anti-pattern ไหนประเมินยากที่สุด?

**Chatty Services** — เพราะต้องมองจากมุมของ **caller** ไม่ใช่แค่มุม implementation ของตัวเอง ต้องคิดว่า "คนที่เรียก API ของเราต้องเรียกกี่ครั้งถึงจะได้ข้อมูลครบ?" ซึ่งต่างจากอีก 2 anti-pattern ที่ดูจาก code/infra ของตัวเองได้เลย นอกจากนี้ `report-progress` มี internal calls หลายอัน (DynamoDB + EventBridge) ทำให้ต้องแยกให้ชัดว่า **"หลาย call ภายใน" ≠ "chatty ต่อ caller"**

### 4.2 สิ่งที่จะทำก่อน Session 2

**Document API contract กับ IncidentTracking Service ให้ชัดเจนขึ้น** — โดยเฉพาะ:
- กำหนด error response ที่คาดหวัง (4xx, 5xx) ไม่ใช่แค่ happy path 200
- ระบุ retry policy / circuit breaker ที่ชัดเจน (ตอนนี้มีแค่ timeout 3 วินาทีแล้ว fallback)
- เตรียม implement **outbox-processor Lambda** เพื่อ retry events ที่ publish ไม่สำเร็จ ตามที่ระบุไว้ใน "สิ่งที่ยังไม่ได้ทำใน Demo 1"
