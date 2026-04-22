# อธิบายทุกเส้นเชื่อม — แยกตาม 3 Services

---

## 1. IncidentTracking Service (Krittamet) — 4 เส้น

### เส้น 1.1: 🔵 MP → IncidentTracking (Sync Request)

```
MissionProgress ──GET /incidents/{id}──▶ IncidentTracking
fields: incident_id
```

**เกิดขึ้นเมื่อ:** ทีมกู้ภัยเปิดดูรายละเอียดภารกิจ (GET /incidents/INC-001)

**สิ่งที่เกิดขึ้น:**

- MissionProgress มีข้อมูลภารกิจ (mission_id, status, timeline) อยู่แล้วใน DB ของตัวเอง
- แต่ **ไม่มี** คำอธิบายเหตุการณ์ เช่น "น้ำท่วมหนัก" หรือพิกัดเหตุการณ์
- ข้อมูลเหล่านี้อยู่ที่ IncidentTracking Service → ต้องไป **ขอ** มา
- เลยส่ง HTTP GET ไปถาม "INC-001 คือเหตุการณ์อะไร?"

**เปรียบเทียบ:** เหมือนตำรวจหน้างานโทรกลับศูนย์ถามว่า "เคสนี้รายละเอียดเป็นยังไง?"

---

### เส้น 1.2: 🟢 IncidentTracking → MP (Sync Response)

```
IncidentTracking ──response 200──▶ MissionProgress
fields: description, location, incident_type, incident_id
```

**สิ่งที่เกิดขึ้น:**

- IncidentTracking ตอบกลับว่า INC-001 = "น้ำท่วมหนักบริเวณถนนพหลโยธิน" พิกัด 13.7563,100.5018 ประเภท FLOOD
- MissionProgress เอาข้อมูลนี้ไป **รวม** กับข้อมูล mission ของตัวเอง แล้วตอบกลับ Frontend

**ถ้า IncidentTracking ล่ม?**

- MissionProgress ไม่พัง — ใช้ Degraded Mode
- ตอบข้อมูล mission ได้ปกติ แค่ไม่มี description, location, incident_type
- บอก Frontend ว่า `data_source: "partial"` (ข้อมูลไม่ครบ)

**เปรียบเทียบ:** ศูนย์ตอบกลับว่า "เป็นเคสน้ำท่วม ที่ถนนพหลโยธิน" ถ้าศูนย์ไม่รับสาย → ตำรวจก็ทำงานต่อได้ แค่ไม่มีรายละเอียด

---

### เส้น 1.3: 🟣 EventBridge → IncidentTracking SQS (Async Route)

```
EventBridge ──route──▶ IncidentTracking SQS
events: MissionStatusChanged + ImpactLevelUpdated
fields: mission_id, incident_id, new_status, old/new_level
```

**เกิดขึ้นเมื่อ:**

- ทีมกู้ภัยอัปเดตสถานะ (เช่น เปลี่ยนเป็น ON_SITE) → MissionStatusChanged ถูก publish
- ทีมกู้ภัยปรับระดับความรุนแรง (เช่น จาก 2 เป็น 4) → ImpactLevelUpdated ถูก publish

**ทำไม IncidentTracking ต้องรู้?**

- เพราะ IncidentTracking เป็น **ศูนย์รวมข้อมูลเหตุการณ์** ต้องรู้ว่าภารกิจไหนสถานะอะไรแล้ว ระดับรุนแรงเท่าไหร่
- เอาไปอัปเดตในฐานข้อมูลของตัวเอง

**เปรียบเทียบ:** ทีมกู้ภัยรายงานกลับศูนย์ว่า "ถึงหน้างานแล้ว สถานการณ์รุนแรงกว่าที่คิด"

---

### เส้น 1.4: 🔴 IncidentTracking SQS → IncidentTracking (Async Consume)

```
IncidentTracking SQS ──consume──▶ IncidentTracking
fields: mission_id, incident_id, new_status, old/new_level
```

**สิ่งที่เกิดขึ้น:**

- IncidentTracking อ่าน message จาก SQS queue ของตัวเอง
- เอา new_status / new_level ไปอัปเดตข้อมูลเหตุการณ์ในฐานข้อมูล

**ทำไมต้องผ่าน SQS?**

- เพราะเป็น **async** — MissionProgress ไม่ต้องรอ IncidentTracking ประมวลผลเสร็จ
- ถ้า IncidentTracking ล่มชั่วคราว → message ยังอยู่ใน SQS → พอฟื้นก็อ่านได้

---

## 2. Dispatch Service (Noppakron) — 4 เส้น

### เส้น 2.1: 🟣 Dispatch → Dispatch EventBridge (Async Publish)

```
Dispatch ──publish: MissionAssignedEvent──▶ Dispatch EventBridge
fields: mission_id, rescue_unit_id, incident_id, assigned_at
```

**เกิดขึ้นเมื่อ:** ศูนย์สั่งการมอบหมายภารกิจใหม่ให้ทีมกู้ภัย

**สิ่งที่เกิดขึ้น:**

- Dispatch Service ตัดสินใจว่า "INC-001 ให้ TEAM-ALPHA รับผิดชอบ"
- publish event บอกว่า mission_id=MSN-001, ทีม=TEAM-ALPHA, เหตุการณ์=INC-001, เวลาที่มอบหมาย

**เปรียบเทียบ:** ศูนย์สั่งการประกาศทางวิทยุว่า "ทีม Alpha รับเคส INC-001 ด่วน"

---

### เส้น 2.2: 🔴 Dispatch EventBridge → MP (Async Consume)

```
Dispatch EventBridge ──consume──▶ MissionProgress
fields: mission_id, rescue_unit_id, incident_id, assigned_at
```

**สิ่งที่เกิดขึ้น:**

- MissionProgress รับ event แล้ว **สร้าง mission record ใหม่** อัตโนมัติ
  - สถานะเริ่มต้น = `DISPATCHED`
  - สร้าง Timeline entry: "Mission assigned to TEAM-ALPHA"
- ถ้า event มาซ้ำ (Dispatch ส่งอีกรอบ) → **skip** ไม่สร้างซ้ำ (idempotent)

**ทำไมสำคัญ?**

- นี่คือ **จุดเริ่มต้นของทุกภารกิจ** — ถ้าไม่มี event นี้ จะไม่มี mission ให้อัปเดตสถานะ
- Rescue Team จะเรียก POST /progress ไม่ได้ถ้ายังไม่มี mission record

**เปรียบเทียบ:** ทีม Alpha ได้ยินทางวิทยุ → จดลงสมุดว่า "เรารับเคส INC-001 แล้ว เวลา 08:45"

---

### เส้น 2.3: 🟣 EventBridge → Dispatch SQS (Async Route — RESOLVED only)

```
EventBridge ──route: MissionStatusChanged (RESOLVED only)──▶ Dispatch SQS
fields: mission_id, incident_id, new_status, changed_at
```

**เกิดขึ้นเมื่อ:** ทีมกู้ภัยรายงานว่าภารกิจเสร็จสิ้น (new_status = RESOLVED)

**ทำไม filter เฉพาะ RESOLVED?**

- Dispatch Service ไม่สนใจสถานะระหว่างทาง (EN_ROUTE, ON_SITE ฯลฯ)
- สนใจแค่ว่า **ภารกิจจบแล้ว** → เพื่อปลดทีมกู้ภัยกลับมาพร้อมรับงานใหม่

**เปรียบเทียบ:** ศูนย์สั่งการรอฟังแค่ "เคสจบแล้ว" เพื่อจะได้ส่งทีม Alpha ไปเคสต่อไปได้

---

### เส้น 2.4: 🔴 Dispatch SQS → Dispatch (Async Consume)

```
Dispatch SQS ──consume──▶ Dispatch
fields: mission_id, incident_id, new_status, changed_at
```

**สิ่งที่เกิดขึ้น:**

- Dispatch อ่าน message → รู้ว่า MSN-001 จบแล้ว
- อัปเดตสถานะทีม TEAM-ALPHA เป็น "พร้อมรับงานใหม่"

---

## 3. Prioritization Service (Nattasak) — 2 เส้น

### เส้น 3.1: 🟣 EventBridge → Prioritization SQS (Async Route)

```
EventBridge ──route──▶ Prioritization SQS
events: MissionBackupRequested + ImpactLevelUpdated
fields: mission_id, incident_id, rescue_team_id, location, old/new_level
```

**เกิดขึ้นเมื่อ:**

- **MissionBackupRequested:** ทีมกู้ภัยเปลี่ยนสถานะเป็น `NEED_BACKUP` (ต้องการกำลังเสริม)
- **ImpactLevelUpdated:** ทีมกู้ภัยปรับระดับความรุนแรง (เช่น 2 → 4)

**ทำไม Prioritization ต้องรู้?**

- เพราะ Prioritization Service ทำหน้าที่ **จัดลำดับความสำคัญของเหตุการณ์**
- ถ้าทีมขอ backup → เคสนี้อาจต้องขยับขึ้นเป็นลำดับแรก
- ถ้า impact level เพิ่ม → ต้องคำนวณ Priority Score ใหม่

**เปรียบเทียบ:** ทีมหน้างานตะโกนว่า "สถานการณ์แย่กว่าที่คิด ต้องการคนเพิ่ม!" → ฝ่ายจัดลำดับต้องปรับแผนว่าเคสไหนสำคัญกว่า

---

### เส้น 3.2: 🔴 Prioritization SQS → Prioritization (Async Consume)

```
Prioritization SQS ──consume──▶ Prioritization
fields: mission_id, incident_id, rescue_team_id, location, old/new_level
```

**สิ่งที่เกิดขึ้น:**

- Prioritization อ่าน message
- **MissionBackupRequested** → เอา location + incident_id ไปพิจารณาว่าส่งทีมไหนไปช่วย
- **ImpactLevelUpdated** → เอา new_level ไปคำนวณ Priority Score ใหม่ → จัดลำดับเคสใหม่

---

## สรุปภาพรวม — ทำไมทุกเส้นต้องมี

```
เวลาเกิดเหตุ 1 เคส จะเกิดอะไรขึ้น:

1️⃣  Dispatch มอบหมายงาน
    DS ──publish──▶ DS_EB ──consume──▶ MP สร้าง mission (DISPATCHED)

2️⃣  ทีมกู้ภัยออกเดินทาง (POST new_status=EN_ROUTE)
    MP ──publish──▶ EB ──route──▶ IT_SQS ──consume──▶ IT อัปเดตสถานะ

3️⃣  ทีมถึงหน้างาน พบว่ารุนแรงมาก (POST new_status=NEED_BACKUP + impact=4)
    MP ──publish──▶ EB ──route──▶ IT_SQS (StatusChanged + ImpactUpdated)
                       ──route──▶ PR_SQS (BackupRequested + ImpactUpdated)

4️⃣  ทีมแก้ไขเสร็จ (POST new_status=RESOLVED)
    MP ──publish──▶ EB ──route──▶ IT_SQS (StatusChanged)
                       ──route──▶ DS_SQS (StatusChanged, RESOLVED only)
    → Dispatch ปลดทีมพร้อมรับงานใหม่
```

> **หลักง่าย ๆ:** MissionProgress เป็น **ตัวกลางรายงานสถานการณ์** — รับคำสั่งจาก Dispatch, รับรายงานจากทีมหน้างาน, แล้วกระจายข่าวให้ทุก service ที่เกี่ยวข้องรู้
