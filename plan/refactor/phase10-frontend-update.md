# Phase 10 — Frontend Update: ดูทุกอย่างผ่าน UI แทน curl

## เป้าหมาย

ทำให้ดูสถานะการต่อ **ทุก service** ที่ interact ด้วย และข้อมูล enrichment ทั้งหมดได้ผ่าน UI โดยไม่ต้อง curl

---

## Services ที่ interact ด้วย

| Service            | Endpoint                              | ข้อมูลที่ได้                                  | Field ใน response                                                      |
| ------------------ | ------------------------------------- | --------------------------------------------- | ---------------------------------------------------------------------- |
| **RescueTeam**     | `GET /v1/teams/{teamId}`              | ชื่อทีม, ประเภท, ความสามารถ, อุปกรณ์, ตำแหน่ง | `team_name`, `team_type`, `capabilities`, `equipment`, `team_location` |
| **RescueRequest**  | `GET /v1/rescue-requests/{requestId}` | ประเภทเหตุ, รายละเอียด, สถานที่               | `incident_type`, `description`, `location`                             |
| **ManageDispatch** | `GET /v1/dispatches?teamId=...`       | สถานะคำสั่งการ, ระดับความสำคัญ                | `dispatch_status`, `priority_level`                                    |

Backend ส่งทุก field มาใน `GET /missions/{request_id}` แล้ว — frontend แค่ยังไม่รับ

---

## สิ่งที่ต้องเพิ่ม

### 1. `lib/types.ts` — เพิ่ม fields ที่ขาดหายใน `MissionDetailResponse`

```ts
// fields ที่ backend ส่งมาแล้วแต่ frontend ยังไม่มี
team_name?: string
team_type?: string
capabilities?: string[]
equipment?: string[]
team_location?: { lat: number; lng: number }
dispatch_status?: string
priority_level?: number
```

---

### 2. `app/mission/page.tsx` — เพิ่ม Service Status Bar

แสดงก่อน header card เพื่อให้รู้ทันทีว่า service ไหนออนไลน์อยู่

```
┌──────────────────────────────────────────────────────────┐
│ ● RescueTeam    ● RescueRequest    ⚠ ManageDispatch      │
└──────────────────────────────────────────────────────────┘
```

logic:

- `team_name` มีค่า → RescueTeam ● เขียว
- `description` หรือ `incident_type` มีค่า → RescueRequest ● เขียว
- `dispatch_status` มีค่า → ManageDispatch ● เขียว
- ว่าง → service นั้น ⚠ เหลือง (degraded)

---

### 3. `app/mission/page.tsx` — เพิ่ม card "ข้อมูลทีม" (RescueTeam)

แสดงหลัง header card

```
┌─────────────────────────────────────────────────────┐
│ 🚒 ข้อมูลทีม                        ● เชื่อมต่อแล้ว │
│                                                     │
│ ชื่อทีม    TEAM-ALPHA                               │
│ ประเภท     Rescue                                   │
│ ความสามารถ  Fire Rescue, Swift Water Rescue         │
│ อุปกรณ์    Stretcher, AED, Rope                     │
└─────────────────────────────────────────────────────┘
```

- ถ้า `team_name` ว่าง → แสดง "⚠ RescueTeam Service ไม่พร้อมใช้งาน"

---

### 4. `app/mission/page.tsx` — เพิ่มข้อมูล Dispatch ใน header card (ManageDispatch)

เพิ่มใน grid ที่มีอยู่แล้ว (บริเวณ `data_source === "full"`)

```
ระดับความสำคัญ   🔴 3 — สูง
สถานะ Dispatch   DISPATCHED
```

- ถ้า `dispatch_status` ว่าง → ไม่แสดง row นี้ (ซ่อนไปเงียบ ๆ)

---

### 5. `app/mission/page.tsx` — แก้ partial warning ให้ระบุ service จริง ๆ

ปัจจุบัน hard-code ว่า "RescueRequest Service ไม่พร้อมใช้งาน" เสมอ

แก้ให้แสดง service ที่ขาดจริงตาม field:

- `description` ว่าง → "RescueRequest Service"
- `team_name` ว่าง → "RescueTeam Service"
- แสดงรวมกันถ้าหายหลายตัว เช่น "RescueRequest, RescueTeam Service ไม่พร้อมใช้งาน"

---

## ไฟล์ที่แก้

| ไฟล์                   | การเปลี่ยนแปลง                                                             |
| ---------------------- | -------------------------------------------------------------------------- |
| `lib/types.ts`         | เพิ่ม 7 fields ใน `MissionDetailResponse`                                  |
| `app/mission/page.tsx` | เพิ่ม Service Status Bar + Team card + Dispatch info + แก้ partial warning |

---

## วิธีทดสอบหลัง deploy

1. seed data → เปิดหน้า mission detail
2. ดู Service Status Bar ทันทีว่า service ไหนออนไลน์
   - ● เขียวทั้ง 3 → ทุก service พร้อม
   - ⚠ เหลือง → service นั้น degraded ไม่ต้อง curl ตรวจสอบ
