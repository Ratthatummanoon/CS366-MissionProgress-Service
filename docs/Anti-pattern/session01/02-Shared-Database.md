## 2. Shared Database

### 2.1 วิเคราะห์
**Database เป็นของ service นี้คนเดียว:**
- เป็นเจ้าของ DynamoDB 3 ตาราง: `MissionAssignment`, `MissionTimeline`, `EventOutbox`
- ไม่ได้ connect ไป database ของ service อื่น
- การดึงข้อมูลจาก IncidentTracking ใช้ **HTTP API call** (`incident_client.go`) ไม่ได้เข้าถึง database ของเขาตรง
- Contract ใน README ก็ระบุชัดว่าเรียกผ่าน `GET {INCIDENT_SERVICE_URL}/incidents/{id}` เท่านั้น

### 2.2 Detected?
**No** — ไม่พบ Shared Database

### 2.3 Fixed?
**N/A** — ไม่มีปัญหาต้องแก้
