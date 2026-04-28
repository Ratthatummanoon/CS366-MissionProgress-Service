# Phase 11 — Endpoint Coverage vs Frontend UI

> วันที่: 29 เมษายน 2569  
> เป้าหมาย: ตรวจสอบว่า endpoint ทุกตัวที่ curl ได้นั้นถูก implement ลงใน frontend UI แล้วหรือยัง เพื่อลดการพึ่งพา curl และให้ทุกฟีเจอร์เข้าถึงได้ผ่าน UI แทน

---

## Endpoints ทั้งหมดที่ curl ได้ (จาก terraform/api_gateway.tf + Lambda handlers)

| #   | Method | Path                                   | Lambda            | Headers Required                                                  |
| --- | ------ | -------------------------------------- | ----------------- | ----------------------------------------------------------------- |
| 1   | `GET`  | `/missions`                            | `list-missions`   | `x-api-key`, `X-Rescue-Team-ID`                                   |
| 2   | `GET`  | `/missions/{request_id}`               | `get-mission`     | `x-api-key`, `X-Rescue-Team-ID`                                   |
| 3   | `POST` | `/missions/{request_id}/progress`      | `report-progress` | `x-api-key`, `X-Rescue-Team-ID`, `Content-Type: application/json` |
| 4   | `POST` | `/missions/{request_id}/presigned-url` | `presigned-url`   | `x-api-key`, `X-Rescue-Team-ID`, `Content-Type: application/json` |

> หมายเหตุ: Endpoint `OPTIONS` บนทุก path เป็น CORS preflight ไม่นับเป็น business endpoint  
> Lambda `mission-assigned-handler` และ `outbox-processor` เป็น internal/async handler ไม่ใช่ HTTP API endpoint

---

## การเปรียบเทียบ Endpoint vs Frontend

### ✅ 1. `GET /missions` — List Missions

- **curl ตัวอย่าง:**
  ```bash
  curl -X GET "$API_URL/missions?status=EN_ROUTE" \
    -H "x-api-key: $API_KEY" \
    -H "X-Rescue-Team-ID: TEAM-ALPHA"
  ```
- **Frontend:** ✅ **Implemented** — หน้า `/dashboard`
  - ไฟล์: `src/frontend/app/dashboard/page.tsx`
  - เรียกผ่าน `client.listMissions(status?)` ใน `lib/api.ts`
  - แสดง mission card ทุกตัว พร้อม filter ตาม status (DISPATCHED / EN_ROUTE / ON_SITE / NEED_BACKUP / RESOLVED / ALL)
  - มีปุ่ม "รีเฟรช" และ summary card นับจำนวนแต่ละ status

---

### ✅ 2. `GET /missions/{request_id}` — Get Mission Detail

- **curl ตัวอย่าง:**
  ```bash
  curl -X GET "$API_URL/missions/REQ-001" \
    -H "x-api-key: $API_KEY" \
    -H "X-Rescue-Team-ID: TEAM-ALPHA"
  ```
- **Frontend:** ✅ **Implemented** — หน้า `/mission?id={request_id}`
  - ไฟล์: `src/frontend/app/mission/page.tsx`
  - เรียกผ่าน `client.getMission(requestId)` ใน `lib/api.ts`
  - แสดงข้อมูลครบ: mission header, ข้อมูลทีม (RescueTeam), dispatch info (ManageDispatch), incident details (RescueRequest)
  - แสดง service status bar (สีเขียว/เหลือง) บอกว่า external service ใดตอบสนองได้
  - แสดง State Machine Diagram (ผ่าน `StateMachineDiagram` component)
  - แสดง Timeline log ทุกรายการ

---

### ✅ 3. `POST /missions/{request_id}/progress` — Report Progress

- **curl ตัวอย่าง:**
  ```bash
  curl -X POST "$API_URL/missions/REQ-001/progress" \
    -H "x-api-key: $API_KEY" \
    -H "X-Rescue-Team-ID: TEAM-ALPHA" \
    -H "Content-Type: application/json" \
    -d '{
      "new_status": "ON_SITE",
      "note": "ถึงจุดเกิดเหตุแล้ว",
      "current_location": "13.7563,100.5018",
      "new_impact_level": 3,
      "image_key": "missions/REQ-001/photo.jpg"
    }'
  ```
- **Frontend:** ✅ **Implemented** — หน้า `/mission?id={request_id}` (section "อัปเดตสถานะภารกิจ")
  - ไฟล์: `src/frontend/app/mission/page.tsx` — function `handleSubmit`
  - เรียกผ่าน `client.reportProgress(requestId, body)` ใน `lib/api.ts`
  - แสดง valid transitions จาก State Machine เท่านั้น (ป้องกัน invalid transition)
  - ฟอร์มมีช่อง: สถานะใหม่, หมายเหตุ, ตำแหน่งปัจจุบัน, ระดับความรุนแรง (1–4), แนบรูปภาพ
  - หลัง submit สำเร็จ ระบบ reload mission อัตโนมัติ

---

### ⚠️ 4. `POST /missions/{request_id}/presigned-url` — Get Presigned Upload URL

- **curl ตัวอย่าง:**
  ```bash
  curl -X POST "$API_URL/missions/REQ-001/presigned-url" \
    -H "x-api-key: $API_KEY" \
    -H "X-Rescue-Team-ID: TEAM-ALPHA" \
    -H "Content-Type: application/json" \
    -d '{
      "file_name": "photo.jpg",
      "content_type": "image/jpeg"
    }'
  ```
- **Frontend:** ⚠️ **Implemented แบบ Implicit** — ไม่มี UI standalone แต่ถูกเรียกโดยอัตโนมัติ
  - ไฟล์: `src/frontend/app/mission/page.tsx` — ใน `handleSubmit` เมื่อ `imageFile !== null`
  - เรียกผ่าน `client.getPresignedUrl(requestId, file.name, file.type)` แล้วตามด้วย `client.uploadFile(uploadUrl, file)`
  - **ผู้ใช้เห็นเพียง:** ช่อง "แนบรูปภาพ" ใน progress form — ไม่รู้ว่า presigned-url ถูกเรียกเบื้องหลัง
  - **ไม่มี UI สำหรับ:** ดู `upload_url`, `image_key`, `expires_in` โดยตรง
  - **ไม่มี UI สำหรับ:** upload รูปภาพโดยไม่ต้อง submit progress พร้อมกัน

---

## สรุป Coverage Matrix

| #   | Endpoint                                    | Frontend Page     | สถานะ                | หมายเหตุ                                    |
| --- | ------------------------------------------- | ----------------- | -------------------- | ------------------------------------------- |
| 1   | `GET /missions`                             | `/dashboard`      | ✅ Fully Implemented | filter + summary card + refresh             |
| 2   | `GET /missions/{request_id}`                | `/mission?id=...` | ✅ Fully Implemented | enriched data, service health bar, timeline |
| 3   | `POST /missions/{request_id}/progress`      | `/mission?id=...` | ✅ Fully Implemented | form พร้อม state machine validation         |
| 4   | `POST /missions/{request_id}/presigned-url` | `/mission?id=...` | ⚠️ Implicit Only     | เรียกอัตโนมัติตอนแนบรูป ไม่มี standalone UI |

---

## สิ่งที่ยังขาด / ควรปรับปรุง

### ❌ ไม่มี standalone UI สำหรับ presigned-url

- ปัจจุบัน: ผู้ใช้ไม่สามารถขอ presigned URL แยกต่างหากได้ ต้องผ่าน progress form เท่านั้น
- **แนวทาง:** อาจไม่จำเป็น เพราะ use case หลักคืออัปโหลดรูปพร้อม progress update ซึ่ง UI รองรับแล้ว

### ❌ ไม่มีหน้าแสดงรูปภาพหลักฐาน (Evidence Viewer)

- Timeline แสดง `image_key` แต่ไม่แสดง thumbnail หรือ link ดูรูปจริง
- ผู้ใช้เห็นเพียง "📷 มีรูปภาพแนบ" แต่ไม่สามารถดูรูปภาพนั้นได้

### ❌ ไม่มี Query Parameter UI สำหรับ `GET /missions`

- API รองรับ `?status=` filter แต่ frontend ใช้ client-side filter จาก data ที่โหลดมาทั้งหมด (**ไม่ได้ส่ง query param จริงไปยัง API**)
- ตรวจสอบใน `dashboard/page.tsx`: เรียก `client.listMissions(status)` ซึ่งส่ง `?status=` ไปจริง → ✅ ถูกต้องแล้ว

### ❌ ไม่มีหน้า Search / Filter ตาม request_id หรือ incident_id

- ปัจจุบันต้องรู้ request_id ก่อนจึงจะเข้า `/mission?id=...` ได้
- Dashboard แสดงรายการแล้ว แต่ไม่มีช่อง search

### ❌ ไม่มี Pagination UI

- API ปัจจุบันคืน missions ทั้งหมดในครั้งเดียว (ไม่มี pagination ใน backend) แต่ถ้าข้อมูลมาก frontend ไม่มี virtual scroll หรือ load more

---

## สรุปภาพรวม

**ทุก endpoint หลักถูก implement ลง frontend ครบแล้ว** สามารถใช้งานผ่าน UI ได้ทั้งหมดโดยไม่ต้อง curl  
จุดที่ยังเป็น implicit คือ `presigned-url` ซึ่ง flow การทำงานถูกต้องแล้ว แต่ผู้ใช้ไม่เห็น URL หรือ image_key โดยตรง  
สิ่งที่ควรปรับปรุงเพิ่มเติมคือ Evidence Viewer (แสดงรูปภาพจาก image_key ใน timeline) และ Search bar
