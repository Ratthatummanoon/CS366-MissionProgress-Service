## 1. Distributed Monolith

### 1.1 วิเคราะห์
**Service สามารถ start ได้แม้ทุกอย่างจะ offline** เพราะ:
- `init()` ของทุก Lambda แค่สร้าง AWS SDK client objects (DynamoDB, EventBridge, HTTP client) ซึ่ง**ไม่ได้เรียก network call จริงตอนสร้าง** — จะเรียกจริงเมื่อมี request เข้ามาเท่านั้น
- ไม่มี shared config file — แต่ละ Lambda ใช้ environment variables ของตัวเอง (`TABLE_MISSION`, `TABLE_TIMELINE`, `INCIDENT_SERVICE_URL` ฯลฯ) กำหนดผ่าน Terraform
- เมื่อ IncidentTracking Service ล่ม → เข้า **Degraded Mode** คืนข้อมูลเท่าที่มี (`data_source: "partial"`)
- เมื่อ EventBridge ล่ม → fallback ไป **Outbox Table** ไม่ fail request หลัก

### 1.2 Detected?
**No** — ไม่พบ Distributed Monolith

### 1.3 Fixed?
**N/A** — ไม่มีปัญหาต้องแก้

