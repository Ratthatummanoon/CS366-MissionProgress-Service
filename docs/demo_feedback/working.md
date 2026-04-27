# การทำงานของ MissionProgress Service

## ตำนานสี

| สี         | ความหมาย                                                                |
| ---------- | ----------------------------------------------------------------------- |
| 🔵 น้ำเงิน | **HTTP request** — ใครบางคนยิง request มาหา MP หรือ MP ยิงออกไป         |
| 🟢 เขียว   | **HTTP response** — ตอบกลับ synchronous                                 |
| 🟣 ม่วง    | **Publish event** — ยิง event ออกไปที่ EventBridge แล้วลืมเลย (ไม่รอผล) |
| 🔴 แดง     | **Consume event** — MP หรือ service อื่นรับข้อความจาก Queue มาประมวลผล  |

---

## ภาพรวมสั้น ๆ

```
ใครส่งทีม? → MD publish event → MP รับ → สร้าง Mission
ใครอัปเดต? → ทีมกดปุ่มบน FE → FE ยิง POST มาหา MP → MP บันทึก + ยิง event ต่อ
ใครรับข่าว? → IT, MD, PR ต่างฝ่ายต่างมี SQS ของตัวเองรอรับ event จาก MP
```

---

## 🔴 จุดเริ่มต้น: MP รับงานจาก ManageDispatch

### MD → DEFAULT_EB → MP

> **ManageDispatch (Noppakron) จัดทีมให้ request เสร็จแล้ว → แจ้ง MP**

```
🟣 MD publish "DispatchOrderCreated" เข้า Default EventBridge (bus สาธารณะของทุก service)
        ↓
🔴 MP consume event นี้ → ได้รับ:
        dispatchId    — รหัสคำสั่งการ
        requestId     — request ที่ต้องช่วย
        teamId        — ทีมที่ถูกส่ง
        priorityLevel — ระดับความสำคัญ
        status        — สถานะ dispatch
        dispatchedAt  — เวลาที่ส่ง
        ↓
🔵 MP เรียก RescueRequest Service ทันที (GET /v1/rescue-requests/{requestId})
   → เพื่อดึง incidentId มาเก็บไว้ใน Mission
        ↓
✅ MP สร้าง Mission ใหม่ status: DISPATCHED บันทึกใน DynamoDB
```

---

## 🔵 MP ดึงข้อมูลเพิ่มเติมจาก Services อื่น

### MP → RescueRequest (ตอนสร้าง Mission)

```
🔵 GET /v1/rescue-requests/{requestId}   auth: Bearer token
🟢 ตอบกลับ: master: {
        requestId, incidentId, requestType,
        description, peopleCount,
        latitude, longitude, locationDetails
   }
⚠️  ถ้า RescueRequest ล่ม → MP ยังสร้าง Mission ได้ แต่ incidentId = "" (degraded mode)
```

### MP → ManageDispatch (ตอนมีคนดู Mission)

```
🔵 GET /v1/dispatches?teamId={teamId}   auth: Bearer token
🟢 ตอบกลับ: {
        teamId,
        items[]: { dispatchId, requestId, status, priorityLevel, dispatchedAt }
   }
⚠️  ถ้า ManageDispatch ล่ม → MP ยัง GET Mission ได้ แต่ไม่มีข้อมูล dispatch (degraded mode)
```

### MP → RescueTeam (ตอนมีคนดู Mission)

```
🔵 GET /v1/teams/{teamId}   auth: Bearer token
🟢 ตอบกลับ: {
        team_id, team_name, team_type, status,
        capabilities[], equipment[],
        location: { lat, lng }
   }
⚠️  ถ้า RescueTeam ล่ม → MP ยัง GET Mission ได้ แต่ไม่มีข้อมูลทีม (degraded mode)
```

---

## 🖥️ ทีมกู้ภัยกดปุ่มบน Dashboard

### FE → MP (Sync Inbound — หัวใจหลักของ MP)

```
🔵 POST /missions/{request_id}/progress
   headers: x-api-key, X-Rescue-Team-ID
   body: {
        new_status        — สถานะใหม่ที่ต้องการเปลี่ยน
        new_impact_level  — (optional) ระดับความรุนแรง
        image_key         — (optional) รูปจากหน้างาน
   }
        ↓
   MP ตรวจ state machine ว่า transition นี้ valid ไหม
        ↓
🟢 ตอบกลับ: { mission_id, request_id, old_status, new_status, updated_at }
        ↓
🟣 MP publish events ออกไปต่อ (ดูส่วนถัดไป)
```

**Transitions ที่ทำได้:**

```
DISPATCHED  →  EN_ROUTE
EN_ROUTE    →  ON_SITE
ON_SITE     →  RESOLVED
ON_SITE     →  NEED_BACKUP
NEED_BACKUP →  ON_SITE
NEED_BACKUP →  RESOLVED
```

---

## 🟣 MP ยิง Events ออกไป (Custom EventBridge)

> MP มี Custom EventBridge Bus ชื่อ `mission-progress-events` เป็นของตัวเอง  
> ทุก event ออกจาก MP จะไปที่ bus นี้ก่อน แล้วค่อย route ต่อไปยัง SQS ของแต่ละ service

---

### Event 1: MissionStatusChanged

> **เกิดทุกครั้งที่ status เปลี่ยน**

```
🟣 publish: MissionStatusChanged
   fields: {
        schema_version, mission_id, requestId,
        incident_id, rescue_team_id,
        old_status, new_status,
        changed_at, changed_by
   }
```

route ไปที่:

- **IT_SQS** → Krittamet รับ → อัปเดตสถานะ Incident
- **MD_SQS** → Noppakron รับ → **เฉพาะ `new_status = RESOLVED` เท่านั้น** → ปิด Dispatch Order

---

### Event 2: MissionBackupRequested

> **เกิดเฉพาะ ON_SITE → NEED_BACKUP**

```
🟣 publish: MissionBackupRequested
   fields: {
        schema_version, mission_id,
        incident_id, rescue_team_id,
        requested_at, requested_by, location
   }
```

route ไปที่:

- **PR_SQS** → Nattasak รับ → จัดลำดับใหม่ → MD สร้าง Dispatch Order ใหม่ → **วนกลับมาที่ MP อีกรอบ** 🔄

---

### Event 3: ImpactLevelUpdated

> **เกิดเมื่อ `new_impact_level` มีค่าใน request body**

```
🟣 publish: ImpactLevelUpdated
   fields: {
        schema_version, mission_id,
        incident_id, rescue_team_id,
        old_level, new_level,
        updated_at, updated_by
   }
```

route ไปที่:

- **IT_SQS** → Krittamet รับ → อัปเดตระดับความรุนแรงของ Incident
- **PR_SQS** → Nattasak รับ → พิจารณาจัดทีมใหม่ตามความรุนแรง

---

## 🟣 EventBridge Routing: EB → SQS (3 เส้น)

> EventBridge ไม่ส่ง event ตรงไปยัง service — ส่งเข้า SQS ของแต่ละ service ก่อน  
> แต่ละ SQS มี routing rule กรอง event ที่ตัวเองสนใจเท่านั้น

### เส้นที่ 15: EB → IT_SQS

```
🟣 route: MissionStatusChanged + ImpactLevelUpdated
   เหตุผล: IT ต้องการรู้ทุกครั้งที่ status หรือ impact level เปลี่ยน
           เพื่ออัปเดตภาพรวม Incident
```

### เส้นที่ 16: EB → MD_SQS

```
🟣 route: MissionStatusChanged (เฉพาะ new_status = RESOLVED เท่านั้น)
   เหตุผล: MD สนใจแค่ตอนจบ เพื่อปิด Dispatch Order
           ไม่จำเป็นต้องรู้ทุก transition
```

### เส้นที่ 17: EB → PR_SQS

```
🟣 route: MissionBackupRequested + ImpactLevelUpdated
   เหตุผล: PR ต้องรู้เมื่อทีมขอกำลังเสริม หรือความรุนแรงเปลี่ยน
           เพื่อจัดลำดับและ dispatch ทีมใหม่
```

---

## 🔴 SQS → Consumer Services (3 เส้น)

> แต่ละ service มา poll SQS ของตัวเองเอง เมื่อพร้อมจึงค่อยดึง

### เส้นที่ 18: IT_SQS → IT (Krittamet)

```
🔴 consume: MissionStatusChanged + ImpactLevelUpdated
   รับ fields: mission_id, incident_id, new_status, old/new_level
   ทำ: อัปเดตสถานะ Incident ตาม mission ที่คืบหน้า
       เช่น EN_ROUTE = ทีมลงพื้นที่แล้ว, RESOLVED = จบภารกิจ
```

### เส้นที่ 19: MD_SQS → MD (Noppakron)

```
🔴 consume: MissionStatusChanged (new_status: RESOLVED)
   รับ fields: mission_id, incident_id, new_status: RESOLVED
   ทำ: ปิด Dispatch Order ของภารกิจนั้น
```

### เส้นที่ 20: PR_SQS → PR (Nattasak)

```
🔴 consume: MissionBackupRequested + ImpactLevelUpdated
   รับ fields: mission_id, incident_id, rescue_team_id, location, old/new_level
   ทำ:
       → จัดลำดับ request ใหม่ตามความรุนแรง
       → แจ้ง MD ให้ dispatch ทีมเสริม
       → MD สร้าง DispatchOrderCreated ใหม่ → 🔄 วนกลับมาที่ MP
```

---

## ✅ จบภารกิจ (RESOLVED)

```
🔵 ทีมกดปุ่ม RESOLVED บน Dashboard
        ↓
🟣 MP publish MissionStatusChanged (new_status: RESOLVED)
        ↓
   ├── IT_SQS → Krittamet รับ → อัปเดต Incident ว่าจบแล้ว
   └── MD_SQS → Noppakron รับ → ปิด Dispatch Order
        ↓
🔵 MP ยิง PATCH /v1/teams/{teamId}/status  body: { status: "AVAILABLE" }
   → คืนทีมกลับเป็น available (fire-and-forget ถ้าล้มเหลวแค่ log)
```

---

## SQS คืออะไร?

> SQS = คิวข้อความ  
> แทนที่ EventBridge จะส่ง event ตรงไปยัง service ปลายทาง → EventBridge ส่งเข้าคิวก่อน  
> service ปลายทางมาดึงเองเมื่อพร้อม

**ทำไมต้องมี SQS?**

- ถ้า service ปลายทางล่มอยู่ → ข้อความถูกเก็บไว้ในคิว ไม่หาย
- service ค่อยมาดึงเองเมื่อกลับมา online (at-least-once delivery)

| SQS    | เจ้าของ   | รับ event                                      |
| ------ | --------- | ---------------------------------------------- |
| IT_SQS | Krittamet | `MissionStatusChanged`, `ImpactLevelUpdated`   |
| MD_SQS | Noppakron | `MissionStatusChanged` (RESOLVED เท่านั้น)     |
| PR_SQS | Nattasak  | `MissionBackupRequested`, `ImpactLevelUpdated` |

---

## Nodes (ตัวละคร)

| Node                              | คือใคร           | หน้าที่                                                          |
| --------------------------------- | ---------------- | ---------------------------------------------------------------- |
| **MP** — MissionProgress Service  | ฉัน (รัฐธรรมนูญ) | ศูนย์กลางติดตามสถานะภารกิจ รับ event → บันทึก → กระจาย event ต่อ |
| **FE** — Rescue Team Dashboard    | Frontend ของฉัน  | UI ที่ทีมกู้ภัยใช้กดเปลี่ยนสถานะภารกิจ                           |
| **RR** — RescueRequest Service    | Phattharaphum    | เก็บข้อมูลคำร้องขอความช่วยเหลือจากประชาชน                        |
| **MD** — ManageDispatch Service   | Noppakron        | สร้างและจัดการคำสั่งการส่งทีม (Dispatch Order)                   |
| **RT** — RescueTeam Service       | กมลพันธ์         | วิเคราะห์และแนะนำทีมที่เหมาะสมกับแต่ละ request                   |
| **IT** — IncidentTracking Service | Krittamet        | ติดตามสถานะเหตุการณ์ (Incident) ภาพรวม                           |
| **PR** — Prioritization Service   | Nattasak         | จัดลำดับความสำคัญและ trigger การส่งทีมเสริม                      |

---

## Infrastructure (คนกลาง)

### Default EventBridge (AWS)

- Bus สาธารณะที่ทุก service ใช้ร่วมกัน
- MD publish `DispatchOrderCreated` เข้ามาที่นี่
- MP subscribe อยู่ → รับ event แล้วสร้าง Mission ใหม่

### Custom EventBridge (`mission-progress-events`)

- Bus ที่ MP สร้างเองและเป็นเจ้าของ
- MP publish event ทั้งหมดออกมาที่นี่
- มี routing rules ส่ง event ไปยัง SQS ของแต่ละ service

### SQS คืออะไร?

SQS (Simple Queue Service) คือ **คิวรับ-ส่งข้อความ** ของ AWS
แต่ละ service ที่ต้องการรับ event จาก MP จะสร้าง SQS ของตัวเองไว้รอรับ

- **ทำไมต้องมี SQS แทนที่จะส่งตรง?**  
  ถ้า EventBridge ส่งตรงและ service ปลายทางล่ม → ข้อความหาย  
  SQS จะเก็บข้อความค้างไว้ → service ปลายทางค่อยมาดึงเองเมื่อพร้อม (at-least-once delivery)

| SQS        | เจ้าของ   | รับ event อะไร                                        | service ปลายทางทำอะไร                     |
| ---------- | --------- | ----------------------------------------------------- | ----------------------------------------- |
| **IT_SQS** | Krittamet | `MissionStatusChanged` + `ImpactLevelUpdated`         | อัปเดตสถานะ Incident ตามความคืบหน้าภารกิจ |
| **MD_SQS** | Noppakron | `MissionStatusChanged` (เฉพาะ `new_status: RESOLVED`) | ปิด Dispatch Order เมื่อภารกิจจบ          |
| **PR_SQS** | Nattasak  | `MissionBackupRequested` + `ImpactLevelUpdated`       | จัดลำดับใหม่และ trigger ส่งทีมเสริม       |

---

## เส้นทั้งหมด (ลูกศรใน Diagram)

---

### 1. FE → MP : ทีมกดปุ่มเปลี่ยนสถานะ (Sync Inbound)

```
POST /missions/{request_id}/progress
headers: x-api-key, X-Rescue-Team-ID
body: { new_status, new_impact_level?, image_key? }
```

- เกิดขึ้นเมื่อ: ทีมกู้ภัยกดปุ่มบน Dashboard เช่น "ออกเดินทางแล้ว", "ถึงหน้างาน", "ภารกิจสำเร็จ"
- MP ตอบกลับ: `{ mission_id, request_id, old_status, new_status, updated_at }`

---

### 2. MP → RR : ดึงข้อมูลคำร้อง (Sync Outbound)

```
GET /v1/rescue-requests/{requestId}
auth: Bearer token
```

- เกิดขึ้นเมื่อ: MP ได้รับ `DispatchOrderCreated` → กำลังสร้าง Mission ใหม่
- ต้องการ: `incidentId` เพื่อเก็บไว้ใน Mission record
- ถ้า RR ล่ม: MP ยังสร้าง Mission ได้ แต่ `incidentId` จะเป็น `""` (degraded mode)

---

### 3. MP → MD : ดึงข้อมูล Dispatch (Sync Outbound)

```
GET /v1/dispatches?teamId={teamId}
auth: Bearer token
```

- เกิดขึ้นเมื่อ: มีคนเรียก GET Mission (on-read enrichment)
- ต้องการ: `dispatch status`, `priorityLevel` เพื่อแสดงใน response
- ถ้า MD ล่ม: MP ยังตอบ GET Mission ได้ แต่ไม่มีข้อมูล dispatch (degraded mode)

---

### 4. MP → RT (GET) : ดึงข้อมูลทีม (Sync Outbound)

```
GET /v1/teams/{teamId}
auth: Bearer token
```

- เกิดขึ้นเมื่อ: มีคนเรียก GET Mission (on-read enrichment)
- ต้องการ: `team_name`, `team_type`, `capabilities`, `equipment`, `location`
- ถ้า RT ล่ม: MP ยังตอบ GET Mission ได้ แต่ไม่มีข้อมูลทีม (degraded mode)

---

### 5. MP → RT (PATCH) : คืนทีมกลับเป็น AVAILABLE (Sync Outbound)

```
PATCH /v1/teams/{teamId}/status
body: { status: "AVAILABLE" }
```

- เกิดขึ้นเมื่อ: Mission เปลี่ยนเป็น `RESOLVED`
- เป็น fire-and-forget → ถ้าล้มเหลว MP แค่ log ไม่ block response

---

### 6. MD → DEFAULT_EB : แจ้งว่าจัดทีมแล้ว (Async Publish)

```
event: DispatchOrderCreated
fields: { dispatchId, requestId, teamId, priorityLevel, status, dispatchedAt }
```

- MD เป็นคน publish หลังจาก assign ทีมให้ request เรียบร้อย
- ส่งเข้า Default EventBridge (bus สาธารณะ)

---

### 7. DEFAULT_EB → MP : MP รับงาน (Async Consume)

```
consume: DispatchOrderCreated
→ สร้าง Mission ใหม่ status: DISPATCHED
```

- MP subscribe อยู่ที่ Default EventBridge
- รับ event → ดึงข้อมูลจาก RR → สร้าง Mission record ใน DynamoDB

---

### 8. MP → EB : ประกาศสถานะเปลี่ยน (Async Publish)

```
event: MissionStatusChanged
fields: { schema_version, mission_id, requestId, incident_id, rescue_team_id,
          old_status, new_status, changed_at, changed_by }
transitions: DISPATCHED→EN_ROUTE, EN_ROUTE→ON_SITE,
             ON_SITE→RESOLVED, NEED_BACKUP→ON_SITE, NEED_BACKUP→RESOLVED
```

- ทุกครั้งที่ status เปลี่ยน MP publish event นี้
- ไปที่ Custom EventBridge ของ MP เอง

---

### 9. MP → EB : ขอกำลังเสริม (Async Publish)

```
event: MissionBackupRequested
fields: { schema_version, mission_id, incident_id, rescue_team_id,
          requested_at, requested_by, location }
triggered: ON_SITE → NEED_BACKUP
```

- เกิดเฉพาะเมื่อทีมแจ้งว่าไม่พอและต้องการกำลังเสริม

---

### 10. MP → EB : ความรุนแรงเปลี่ยน (Async Publish)

```
event: ImpactLevelUpdated
fields: { schema_version, mission_id, incident_id, rescue_team_id,
          old_level, new_level, updated_at, updated_by }
triggered: new_impact_level มีค่าใน request body
```

- เกิดเมื่อทีมรายงานว่าระดับความรุนแรงเปลี่ยน เช่น น้ำท่วมหนักขึ้น

---

### 11. EB → IT_SQS → IT : อัปเดต Incident

- routing rule: `MissionStatusChanged` + `ImpactLevelUpdated`
- IT รับแล้วอัปเดตสถานะ Incident ตาม mission ที่คืบหน้า

---

### 12. EB → MD_SQS → MD : ปิด Dispatch Order

- routing rule: `MissionStatusChanged` เฉพาะ `new_status = RESOLVED`
- MD รับแล้วปิด Dispatch Order ของทีมนั้น

---

### 13. EB → PR_SQS → PR : จัดทีมใหม่ (วนกลับ)

- routing rule: `MissionBackupRequested` + `ImpactLevelUpdated`
- PR รับแล้วจัดลำดับใหม่ → MD สร้าง Dispatch Order ใหม่ → MP รับ `DispatchOrderCreated` อีกรอบ (loop กลับ Step 7)
