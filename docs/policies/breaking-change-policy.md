# Breaking Change Policy — MissionProgress Service

## วัตถุประสงค์

นโยบายนี้กำหนดแนวทางการจัดการ breaking changes สำหรับ API contracts (ทั้ง Sync และ Async) ของ MissionProgress Service เพื่อปกป้อง consumers จากการเปลี่ยนแปลงที่ไม่คาดคิด

---

## นิยาม

### Non-Breaking Change (ไม่ต้องแจ้ง consumer ล่วงหน้า)

- **เพิ่ม field ใหม่** — เช่น เพิ่ม `gps_accuracy` ใน event payload
- **เพิ่ม optional query parameter** — เช่น เพิ่ม `?sort=desc` ใน GET endpoint
- **เพิ่ม enum value ใหม่** — เช่น เพิ่มสถานะ `CANCELLED` (แต่ต้อง document)
- **เปลี่ยนแปลง internal implementation** — ที่ไม่กระทบ contract

### Breaking Change (ต้องแจ้ง consumer ล่วงหน้า)

- **ลบ field** — เช่น ลบ `old_status` จาก event
- **เปลี่ยน field type** — เช่น `impact_level` จาก `int` เป็น `string`
- **เปลี่ยนชื่อ field** — เช่น `mission_id` → `msn_id`
- **เปลี่ยน field จาก optional เป็น required** หรือกลับกัน
- **เปลี่ยน URL path** — เช่น `/incidents/{id}/progress` → `/missions/{id}/progress`
- **เปลี่ยน HTTP method**
- **เปลี่ยน event name (detail-type)**
- **เปลี่ยนความหมายของ field** — เช่น `timestamp` จาก UTC เป็น local time

---

## การใช้ Schema Version

ทุก async event ต้องมี field `schema_version` ใน payload:

```json
{
  "schema_version": "1.0",
  "mission_id": "MSN-001",
  ...
}
```

### กฎ Version

| การเปลี่ยนแปลง              | Version Action            | ตัวอย่าง                        |
| --------------------------- | ------------------------- | ------------------------------- |
| เพิ่ม field ใหม่ (optional) | Bump minor: `1.0` → `1.1` | เพิ่ม `gps_accuracy`            |
| ลบ/เปลี่ยน field (breaking) | Bump major: `1.x` → `2.0` | เปลี่ยน type ของ `impact_level` |
| Bug fix ใน payload          | Bump minor: `1.0` → `1.1` | แก้ format ของ timestamp        |

---

## กระบวนการแจ้ง Consumer

### สำหรับ Non-Breaking Changes

1. อัปเดต documentation (`03-Async-Contract.md` หรือ `02-Sync-Contract.md`)
2. Bump minor version ของ `schema_version`
3. Deploy ได้เลย

### สำหรับ Breaking Changes

1. **ประกาศ** — แจ้ง consumers ทั้งหมดผ่านช่องทางทีม (Slack/LINE/email) อย่างน้อย **2 สัปดาห์** ก่อน deploy
2. **Document** — อัปเดต contract docs พร้อมระบุ deprecated fields และ timeline
3. **Dual Support** — support ทั้ง schema version เก่าและใหม่เป็นเวลาอย่างน้อย **2 sprints** (4 สัปดาห์)
4. **Migrate** — ช่วย consumers migrate โดยให้ตัวอย่าง code
5. **Sunset** — ลบ version เก่าหลังจาก consumers ทุกตัว migrate เสร็จ

---

## Consumers ปัจจุบัน

| Consumer                      | Events ที่ Subscribe                       | Owner     |
| ----------------------------- | ------------------------------------------ | --------- |
| IncidentTracking Service      | MissionStatusChanged, ImpactLevelUpdated   | Krittamet |
| Dispatch Management Service   | MissionStatusChanged (RESOLVED only)       | Noppakron |
| Rescue Prioritization Service | MissionBackupRequested, ImpactLevelUpdated | Nattasak  |

---

## ตัวอย่าง

### ตัวอย่าง Non-Breaking: เพิ่ม field ใหม่

**Before (v1.0):**

```json
{
  "schema_version": "1.0",
  "mission_id": "MSN-001",
  "new_status": "ON_SITE"
}
```

**After (v1.1):**

```json
{
  "schema_version": "1.1",
  "mission_id": "MSN-001",
  "new_status": "ON_SITE",
  "gps_accuracy": 5.2
}
```

Consumer ที่ไม่รู้จัก `gps_accuracy` จะ ignore field นี้โดยอัตโนมัติ — ไม่ต้องแก้ code

### ตัวอย่าง Breaking: เปลี่ยน field type

**Before (v1.x):**

```json
{
  "schema_version": "1.2",
  "old_level": 3,
  "new_level": 5
}
```

**After (v2.0):**

```json
{
  "schema_version": "2.0",
  "old_level": "HIGH",
  "new_level": "CRITICAL"
}
```

ต้องแจ้ง consumers ล่วงหน้า 2 สัปดาห์ และ support ทั้ง v1.x และ v2.0 เป็นเวลา 4 สัปดาห์
