# 📋 ลำดับการทำงาน Flow ฉบับ Present

---

## 🌊 Phase 1: เกิดเหตุ → ก่อนถึงฉัน (Background)

```
🔥 เกิดเหตุการณ์ เช่น น้ำท่วม
│
▼
🔍 IncidentTracking (Krittamet) สร้าง Incident
│
▼
📝 RescueRequest (Phattharaphum) รับแจ้งจากประชาชน
   1 Incident มีได้หลาย Requests
│
▼
⚡ Prioritization (Nattasak) จัดลำดับความสำคัญ
│
▼
🚀 ManageDispatch (Noppakron) จัดสรรทีมกู้ภัย สร้าง Dispatch Order
```

> ⚠️ Phase นี้ไม่ใช่ scope ของฉัน แต่ต้องเล่าให้เห็นภาพรวม

---

## 🟢 Phase 2: ฉันรับงาน (Async Inbound)

```
Step 1: ManageDispatch (Noppakron) publish event "DispatchOrderCreated"
        → ส่งเข้า Default EventBridge

Step 2: MissionProgress (ฉัน) consume event จาก Default EventBridge
        ได้รับ: dispatchId, requestId, teamId, priorityLevel
```

---

## 🔵 Phase 3: ฉันดึงข้อมูลเพิ่ม (Sync Outbound)

```
Step 3: ฉัน GET → RescueRequest (Phattharaphum)          ← เกิดตอนสร้าง Mission
        endpoint: /v1/rescue-requests/{requestId}
        ได้กลับ: incident_id (เก็บใน Mission record)
        หาก service ล่ม → degraded: incidentId = ""

Step 4: ฉัน GET → ManageDispatch (Noppakron)             ← เกิดตอน GET Mission (on-read)
        ได้กลับ: dispatch details, dispatch status

Step 5: ฉัน GET → RescueTeam (กมลพันธ์)                  ← เกิดตอน GET Mission (on-read)
        ได้กลับ: team name, capabilities, team location

Step 6: ฉันสร้าง Mission สำเร็จ
        สถานะเริ่มต้น: DISPATCHED
        (Steps 4-5 ถูกเรียกเมื่อมีคนดึงข้อมูล Mission ไม่ใช่ตอนสร้าง)
```

---

## � Phase 3.5: RescueTeam รายงานสถานะจากหน้างาน (Sync Inbound)

```
RescueTeam (กมลพันธ์) เรียก POST /progress เพื่ออัปเดตสถานะภารกิจ
fields: new_status, new_impact_level, image_key

→ ฉัน respond: 200 updated mission status
```

> ⚠️ ทุก status transition และ impact level change ใน Phase 4-7 ถูก trigger โดย RescueTeam เรียก endpoint นี้

---

## �🟣 Phase 4: ติดตามภารกิจ → ส่ง Event ออก (Async Outbound)

### Step 7: ทีมออกเดินทาง → ถึงหน้างาน

```
Mission: DISPATCHED → EN_ROUTE → ON_SITE

แต่ละ transition → ฉัน publish "MissionStatusChanged" → Custom EventBridge
   │
   ├──→ CloudWatch Logs (บันทึก log เสมอ)
   │
   └──→ IncidentTracking SQS → Krittamet รับไปอัปเดต incident
```

### Step 8: ทีม ON_SITE แล้ว → แยกเป็น 3 ทาง

```
ทีมถึงหน้างานแล้ว (ON_SITE) → เกิดอะไรขึ้น?

   ทางที่ 1: ✅ ภารกิจสำเร็จ         → ไป Phase 5
   ทางที่ 2: 🆘 ต้องการกำลังเสริม    → ไป Phase 6
   ทางที่ 3: ⚠️ ความรุนแรงเปลี่ยน   → ไป Phase 7
```

---

## ✅ Phase 5: ภารกิจสำเร็จ (RESOLVED)

```
Step 9: Mission: ON_SITE → RESOLVED  (หรือ NEED_BACKUP → RESOLVED)

ฉัน publish "MissionStatusChanged" → Custom EventBridge
   │
   ├──→ CloudWatch Logs (บันทึก log เสมอ)
   │
   ├──→ IncidentTracking SQS
   │    → Krittamet รับไปอัปเดต incident
   │
   └──→ Dispatch SQS (เฉพาะ RESOLVED เท่านั้น!)
        → Noppakron รับไปปิด Dispatch Order

+ ฉัน notify → RescueTeam (Sync, fire-and-forget)
        → set team กลับเป็น AVAILABLE

🏁 จบภารกิจ
```

---

## 🆘 Phase 6: ขอกำลังเสริม (วนกลับ!)

```
Step 10: ทีม ON_SITE แจ้งว่าไม่พอ → ON_SITE → NEED_BACKUP

ฉัน publish "MissionBackupRequested" → Custom EventBridge
   │
   ├──→ CloudWatch Logs (บันทึก log เสมอ)
   │
   └──→ Prioritization SQS
        → Nattasak รับไปจัดลำดับใหม่
           │
           ▼
        ⚡ Prioritization จัดลำดับใหม่
           │
           ▼
        🚀 ManageDispatch จัดสรรทีมเสริม → สร้าง Dispatch Order ใหม่
           │
           ▼
        🟣 publish "DispatchOrderCreated" → Default EventBridge
           │
           ▼
        📋 กลับมาหาฉัน! (วน Phase 2 ใหม่)
           → ฉันสร้าง Mission ใหม่อีกตัวสำหรับทีมเสริม
           → ผูกกับ Incident เดียวกัน
```

> **🔄 นี่คือจุดที่ flow วนกลับ!**

```
Step 10b: ทีมเสริมถึงหน้างาน → Mission เดิม: NEED_BACKUP → ON_SITE

ฉัน publish "MissionStatusChanged" → Custom EventBridge
   │
   ├──→ CloudWatch Logs (บันทึก log เสมอ)
   │
   └──→ IncidentTracking SQS → Krittamet รับไปอัปเดต incident
```

---

## ⚠️ Phase 7: ความรุนแรงเปลี่ยน

```
Step 11: RescueTeam เรียก POST /progress ด้วย new_impact_level
         สถานการณ์เปลี่ยน เช่น น้ำท่วมหนักขึ้น
         impact: LOW → HIGH

ฉัน publish "ImpactLevelUpdated" → Custom EventBridge
   │
   ├──→ CloudWatch Logs (บันทึก log เสมอ)
   │
   ├──→ IncidentTracking SQS
   │    → Krittamet รับไปอัปเดตความรุนแรงของ incident
   │
   └──→ Prioritization SQS
        → Nattasak รับไปจัดลำดับความสำคัญใหม่
```

---

## 🗺️ สรุป Flow ทั้งหมดในภาพเดียว

```
เกิดเหตุ → Request → Prioritize → Dispatch
                                      │
                                      ▼
                              ════════════════
                              ║ ฉัน (MP)     ║
                              ║ สร้าง Mission ║
                              ════════════════
                                      │
                    ┌─────────────────┼─────────────────┐
                    │                 │                 │
                    ▼                 ▼                 ▼
              ✅ RESOLVED       🆘 BACKUP         ⚠️ IMPACT
                    │            REQUESTED          UPDATED
                    │                 │                 │
                    ▼                 │                 ▼
             → Krittamet              │          → Krittamet
             → Noppakron              │          → Nattasak
               (ปิด dispatch)        │
             → RescueTeam
               (set AVAILABLE)       │
                    │                 ▼
                    ▼           → Nattasak
                  🏁 จบ          (จัดลำดับใหม่)
                                      │
                                      ▼
                                → Noppakron
                                  (dispatch ทีมใหม่)
                                      │
                                      ▼
                                กลับมาหาฉัน!
                                Mission ใหม่ 🔄
```

---

> **"เมื่อเกิดเหตุ ระบบจะผ่านขั้นตอน Request → Prioritize → Dispatch แล้วส่ง event มาถึง MissionProgress**
>
> **ผมรับ event แล้วไปดึงข้อมูลจาก 3 services มาสร้าง Mission**
>
> **จากนั้นติดตามภารกิจ ซึ่งมี 3 ทางที่เกิดได้:**
>
> 1. **สำเร็จ** → ส่ง event ให้ IncidentTracking อัปเดต และ Dispatch ปิด order
> 2. **ขอกำลังเสริม** → ส่ง event ให้ Prioritization จัดลำดับใหม่ → Dispatch จัดทีมใหม่ → **วนกลับมาหาผมอีกรอบเป็น Mission ใหม่**
> 3. **ความรุนแรงเปลี่ยน** → ส่ง event ให้ IncidentTracking และ Prioritization รับไปอัปเดต
>
> **ทุก event ที่ผมส่งออกจะถูกบันทึกลง CloudWatch Logs เสมอเพื่อ observability"**

---
