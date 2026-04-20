# 📖 MissionProgress Service — คู่มือการติดตั้งและใช้งานฉบับสมบูรณ์

> สำหรับผู้เริ่มต้นที่ต้องการทดสอบระบบตั้งแต่เริ่มต้นจนถึงการล้างทรัพยากร (Cleanup)
> ใช้ร่วมกับ **AWS Academy Learner Lab**

---

## สารบัญ

1. [ภาพรวมของระบบ](#1-ภาพรวมของระบบ)
2. [สิ่งที่ต้องเตรียมก่อนเริ่มต้น (Prerequisites)](#2-สิ่งที่ต้องเตรียมก่อนเริ่มต้น-prerequisites)
3. [ขั้นตอนที่ 1 — เปิดใช้งาน AWS Learner Lab](#3-ขั้นตอนที่-1--เปิดใช้งาน-aws-learner-lab)
4. [ขั้นตอนที่ 2 — ตั้งค่า AWS CLI บนเครื่องของคุณ](#4-ขั้นตอนที่-2--ตั้งค่า-aws-cli-บนเครื่องของคุณ)
5. [ขั้นตอนที่ 3 — Clone โปรเจกต์](#5-ขั้นตอนที่-3--clone-โปรเจกต์)
6. [ขั้นตอนที่ 4 — ติดตั้ง Dependencies](#6-ขั้นตอนที่-4--ติดตั้ง-dependencies)
7. [ขั้นตอนที่ 5 — Build โปรเจกต์](#7-ขั้นตอนที่-5--build-โปรเจกต์)
8. [ขั้นตอนที่ 6 — Deploy ขึ้น AWS](#8-ขั้นตอนที่-6--deploy-ขึ้น-aws)
9. [ขั้นตอนที่ 7 — Seed ข้อมูลตัวอย่าง](#9-ขั้นตอนที่-7--seed-ข้อมูลตัวอย่าง)
10. [ขั้นตอนที่ 8 — ดู Output ที่สำคัญ (API URL, API Key, Frontend URL)](#10-ขั้นตอนที่-8--ดู-output-ที่สำคัญ)
11. [ขั้นตอนที่ 9 — ทดสอบ API ด้วย curl / Postman](#11-ขั้นตอนที่-9--ทดสอบ-api-ด้วย-curl--postman)
12. [ขั้นตอนที่ 10 — ใช้งาน Frontend (Web UI)](#12-ขั้นตอนที่-10--ใช้งาน-frontend-web-ui)
13. [ขั้นตอนที่ 11 — ตรวจสอบทรัพยากรบน AWS Console](#13-ขั้นตอนที่-11--ตรวจสอบทรัพยากรบน-aws-console)
14. [ขั้นตอนที่ 12 — Cleanup (ลบทรัพยากรทั้งหมด)](#14-ขั้นตอนที่-12--cleanup-ลบทรัพยากรทั้งหมด)
15. [Appendix: State Machine & API Reference](#15-appendix-state-machine--api-reference)
16. [Troubleshooting (แก้ปัญหาที่พบบ่อย)](#16-troubleshooting-แก้ปัญหาที่พบบ่อย)

---

## 1. ภาพรวมของระบบ

**MissionProgress Service** คือบริการสำหรับทีมกู้ภัย (Rescue Team) ใช้ในการ:

- ดูรายการภารกิจที่ได้รับมอบหมาย
- รายงานความคืบหน้าของภารกิจ (เปลี่ยนสถานะ)
- บันทึก Timeline การปฏิบัติงาน
- อัปโหลดหลักฐานภาพถ่ายจากหน้างาน
- Publish Events แจ้ง Service อื่น ๆ

### สถาปัตยกรรม

```
Frontend (Next.js → S3 Static Website)
        │
        ▼
  API Gateway (REST)
   ├── Lambda Authorizer (ตรวจ API Key + Team ID)
   │
   ├── GET  /incidents              → list-missions Lambda
   ├── GET  /incidents/{id}         → get-mission Lambda
   ├── POST /incidents/{id}/progress → report-progress Lambda
   └── POST /incidents/{id}/presigned-url → presigned-url Lambda
           │
           ├── DynamoDB (MissionAssignment, MissionTimeline, EventOutbox)
           ├── EventBridge (publish events → CloudWatch Logs)
           └── S3 (Evidence Storage)
```

### Tech Stack

| Layer           | เทคโนโลยี                               |
| --------------- | --------------------------------------- |
| Backend         | Go 1.24 (AWS Lambda, `provided.al2023`) |
| Frontend        | Next.js 16 + React 19 + TailwindCSS 4   |
| API Gateway     | Amazon API Gateway (REST)               |
| Database        | Amazon DynamoDB (PAY_PER_REQUEST)       |
| Async Messaging | Amazon EventBridge                      |
| Storage         | Amazon S3 (Evidence + Frontend Hosting) |
| IaC             | Terraform ~5.0                          |
| Auth            | API Key + Lambda Authorizer             |

### เจ้าของ Service

**Service Owner:** นายรัฐธรรมนูญ โคสาแสง (6609612178)

### Dependencies — เชื่อมต่อกับ Service เพื่อน

| บริการ (Service)               | เจ้าของ                                | วิธีเชื่อมต่อ                                      | ตัวแปร Terraform                         | Degraded Mode                                                         |
| ------------------------------ | -------------------------------------- | -------------------------------------------------- | ---------------------------------------- | --------------------------------------------------------------------- |
| **IncidentTracking Service**   | กฤตเมธ ดำทองคำ (Krittamet Damthongkam) | HTTP GET `/incidents/{id}` (Sync)                  | `incident_service_url`                   | `data_source: "partial"` (ไม่มี description, location, incident_type) |
| **IncidentTracking SQS**       | กฤตเมธ ดำทองคำ                         | EventBridge → SQS (Async)                          | `incident_tracking_sqs_arn`              | Events ไปที่ CloudWatch Logs แทน                                      |
| **Dispatch SQS**               | เจ้าของ Dispatch Service               | EventBridge → SQS (Async, เฉพาะ RESOLVED)          | `dispatch_sqs_arn`                       | Events ไปที่ CloudWatch Logs แทน                                      |
| **Prioritization SQS**         | เจ้าของ Prioritization Service         | EventBridge → SQS (Async)                          | `prioritization_sqs_arn`                 | Events ไปที่ CloudWatch Logs แทน                                      |
| **Dispatch Service (Inbound)** | เจ้าของ Dispatch Service               | EventBridge → Lambda (Async, MissionAssignedEvent) | — (Event bus: `mission-progress-events`) | ใช้ seed-data แทน                                                     |

### EventBridge Events ที่ Publish (Outbound)

| Event                    | เงื่อนไข                      | ส่งไปยัง                                                            |
| ------------------------ | ----------------------------- | ------------------------------------------------------------------- |
| `MissionStatusChanged`   | ทุกครั้งที่สถานะเปลี่ยน       | CloudWatch Logs, IncidentTracking SQS, Dispatch SQS (RESOLVED only) |
| `MissionBackupRequested` | สถานะใหม่ = `NEED_BACKUP`     | CloudWatch Logs, Prioritization SQS                                 |
| `ImpactLevelUpdated`     | มี `new_impact_level` ใน body | CloudWatch Logs, IncidentTracking SQS, Prioritization SQS           |

### EventBridge Events ที่รับ (Inbound)

| Event                  | มาจาก                         | Handler Lambda             | การทำงาน                                    |
| ---------------------- | ----------------------------- | -------------------------- | ------------------------------------------- |
| `MissionAssignedEvent` | `dispatch-management-service` | `mission-assigned-handler` | สร้าง mission (DISPATCHED) + Timeline entry |

### DynamoDB Tables

| ชื่อ Table        | คำอธิบาย                    | Keys                                |
| ----------------- | --------------------------- | ----------------------------------- |
| MissionAssignment | เก็บข้อมูลภารกิจ            | PK: `mission_id`, GSI: `team-index` |
| MissionTimeline   | เก็บ Timeline การปฏิบัติงาน | PK: `mission_id`, SK: `timestamp`   |
| EventOutbox       | เก็บ Event ที่รอ retry      | PK: `event_id`                      |

---

## 2. สิ่งที่ต้องเตรียมก่อนเริ่มต้น (Prerequisites)

ก่อนเริ่มต้น ให้ตรวจสอบว่าเครื่องของคุณมีเครื่องมือเหล่านี้ติดตั้งแล้ว:

### 2.1 ตรวจสอบเครื่องมือที่จำเป็น

เปิด Terminal แล้วรันคำสั่งต่อไปนี้ทีละบรรทัด:

```bash
# ตรวจสอบ Go (ต้องการ 1.24+)
go version
# ✅ ผลลัพธ์ที่คาดหวัง: go version go1.24.x darwin/arm64 (หรือ linux/amd64)

# ตรวจสอบ Node.js (ต้องการ 18+)
node -v
# ✅ ผลลัพธ์ที่คาดหวัง: v18.x.x หรือสูงกว่า

# ตรวจสอบ npm
npm -v
# ✅ ผลลัพธ์ที่คาดหวัง: 9.x.x หรือสูงกว่า

# ตรวจสอบ Terraform (ต้องการ 1.0+)
terraform -v
# ✅ ผลลัพธ์ที่คาดหวัง: Terraform v1.x.x

# ตรวจสอบ AWS CLI (ต้องการ v2)
aws --version
# ✅ ผลลัพธ์ที่คาดหวัง: aws-cli/2.x.x ...

# ตรวจสอบ Git
git --version
# ✅ ผลลัพธ์ที่คาดหวัง: git version 2.x.x

# ตรวจสอบ zip (ใช้สำหรับ build Lambda)
zip --version
# ✅ ผลลัพธ์ที่คาดหวัง: Copyright (c) ...
```

### 2.2 วิธีติดตั้งเครื่องมือ (ถ้ายังไม่มี)

#### macOS (ใช้ Homebrew)

```bash
# ติดตั้ง Homebrew (ถ้ายังไม่มี)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# ติดตั้งเครื่องมือ
brew install go           # Go
brew install node         # Node.js + npm
brew install terraform    # Terraform
brew install awscli       # AWS CLI v2
brew install git          # Git
brew install zip          # zip (macOS มักมีอยู่แล้ว)
```

#### Windows (ใช้ winget หรือดาวน์โหลดจากเว็บ)

```powershell
winget install GoLang.Go
winget install OpenJS.NodeJS.LTS
winget install Hashicorp.Terraform
winget install Amazon.AWSCLI
winget install Git.Git
```

> ⚠️ **Windows**: แนะนำใช้ **WSL2 (Windows Subsystem for Linux)** หรือ **Git Bash** เพื่อรัน shell script (.sh) ได้

#### Linux (Ubuntu/Debian)

```bash
# Go
sudo snap install go --classic

# Node.js (ผ่าน nvm)
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.0/install.sh | bash
nvm install --lts

# Terraform
sudo apt-get update && sudo apt-get install -y gnupg software-properties-common
wget -O- https://apt.releases.hashicorp.com/gpg | gpg --dearmor | sudo tee /usr/share/keyrings/hashicorp-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
sudo apt update && sudo apt install terraform

# AWS CLI v2
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip && sudo ./aws/install

# zip
sudo apt install zip
```

### 2.3 สิ่งที่ต้องมี

- **บัญชี AWS Academy Learner Lab** ที่ active อยู่
- **เว็บเบราว์เซอร์** (Chrome, Firefox, Safari)
- **Postman** (optional — สำหรับทดสอบ API แบบ GUI) — ดาวน์โหลดจาก https://www.postman.com/downloads/

---

## 3. ขั้นตอนที่ 1 — เปิดใช้งาน AWS Learner Lab

### 3.1 เข้าสู่ระบบ AWS Academy

1. เปิดเว็บเบราว์เซอร์ แล้วไปที่ **AWS Academy** (URL ที่อาจารย์ให้มา เช่น `https://awsacademy.instructure.com/`)
2. **ล็อกอิน** ด้วย Email และ Password ที่ลงทะเบียนไว้
3. หลังล็อกอินสำเร็จ จะเห็นหน้า **Dashboard** ของ Canvas LMS

### 3.2 เปิด Learner Lab

1. ที่เมนูด้านซ้าย คลิก **"Courses"** (หลักสูตร)
2. คลิกเข้าไปในคอร์สที่มีชื่อว่า **"AWS Academy Learner Lab"** (หรือชื่อที่อาจารย์ตั้งไว้)
3. ที่เมนูด้านซ้ายของคอร์ส คลิก **"Modules"**
4. คลิก **"Learner Lab"** ในรายการ Module

### 3.3 Start Lab

1. จะเห็นหน้า Learner Lab Console ด้านบนจะมี:
   - 🔴 **วงกลมสีแดง** ข้าง **"AWS"** — หมายถึง Lab ยังไม่ได้เปิด
   - ปุ่ม **"Start Lab"** ▶️
2. คลิกปุ่ม **"Start Lab"** แล้วรอประมาณ 1–3 นาที
3. สังเกตวงกลมจะเปลี่ยนเป็น 🟢 **สีเขียว** — หมายถึง Lab พร้อมใช้งานแล้ว

> ⏱️ **หมายเหตุ**: Lab session มีเวลาจำกัด (ปกติ 4 ชั่วโมง) และมี budget จำกัด ตรวจสอบงบประมาณที่เหลือได้จากตัวเลขด้านบน เช่น `$XX.XX remaining`

### 3.4 ดึง AWS Credentials

1. เมื่อวงกลมเป็น 🟢 **สีเขียว** แล้ว ให้คลิกที่ **"AWS Details"** (อยู่ข้าง ๆ ปุ่ม Start Lab)
2. จะเห็นกล่องข้อมูล:
   - **AWS Account ID**: เลขบัญชี AWS ของคุณ
   - **Cloud Access**: มีลิงก์ **"Show"** ข้าง **AWS CLI**
3. คลิก **"Show"** ข้าง **AWS CLI** จะแสดงข้อมูลดังนี้:

```
[default]
aws_access_key_id=ASIAxxxxxxxxxxxxxxxx
aws_secret_access_key=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
aws_session_token=xxxxxxxxxxxxxxx...ยาวมาก...xxxxxxxx
```

4. **คัดลอก (Copy) ข้อความทั้งหมด** ที่แสดง (ตั้งแต่ `[default]` ถึงบรรทัดสุดท้ายของ `aws_session_token`)

> ⚠️ **สำคัญมาก**: Credentials เหล่านี้จะ **หมดอายุ** เมื่อ Lab session จบ ทุกครั้งที่ Start Lab ใหม่ต้องทำขั้นตอนนี้ซ้ำ

### 3.5 เปิด AWS Console (Optional)

- ถ้าต้องการดูทรัพยากรบน AWS Console ให้คลิกที่คำว่า **"AWS" 🟢** (ตัวหนังสือ "AWS" ที่มีวงกลมสีเขียว) จะเปิดหน้าต่างใหม่เข้าสู่ AWS Management Console โดยอัตโนมัติ
- **ภูมิภาค (Region)** ค่าเริ่มต้นคือ **us-east-1 (N. Virginia)** — ตรวจสอบที่มุมขวาบนของ Console ว่าแสดงเป็น **"N. Virginia"**

---

## 4. ขั้นตอนที่ 2 — ตั้งค่า AWS CLI บนเครื่องของคุณ

### 4.1 วาง Credentials

เปิด Terminal แล้วรันคำสั่ง:

```bash
# สร้างโฟลเดอร์ .aws (ถ้ายังไม่มี)
mkdir -p ~/.aws
```

เปิดไฟล์ `~/.aws/credentials` ด้วย text editor:

```bash
# macOS/Linux
nano ~/.aws/credentials

# หรือใช้ VS Code
code ~/.aws/credentials
```

**วาง (Paste) ข้อมูล Credentials** ที่คัดลอกมาจากขั้นตอน 3.4 ทับเนื้อหาเดิมทั้งหมด:

```ini
[default]
aws_access_key_id=ASIAxxxxxxxxxxxxxxxx
aws_secret_access_key=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
aws_session_token=xxxxxxxxxxxxxxx...ยาวมาก...xxxxxxxx
```

บันทึกไฟล์แล้วปิด (ถ้าใช้ nano กด `Ctrl+O` → `Enter` → `Ctrl+X`)

### 4.2 ตั้ง Region เริ่มต้น

```bash
# สร้างหรือแก้ไขไฟล์ config
nano ~/.aws/config
```

ใส่เนื้อหา:

```ini
[default]
region = us-east-1
output = json
```

บันทึกและปิด

### 4.3 ทดสอบ AWS CLI

```bash
aws sts get-caller-identity
```

✅ **ผลลัพธ์ที่คาดหวัง** (ถ้าสำเร็จ):

```json
{
  "UserId": "AROA...:user...",
  "Account": "123456789012",
  "Arn": "arn:aws:sts::123456789012:assumed-role/voclabs/user..."
}
```

❌ **ถ้าได้ error** เช่น:

```
An error occurred (ExpiredTokenException) when calling the GetCallerIdentity operation: The security token included in the request is expired
```

→ กลับไปขั้นตอน 3.4 เพื่อดึง Credentials ใหม่ (Lab session อาจหมดอายุ)

---

## 5. ขั้นตอนที่ 3 — Clone โปรเจกต์

```bash
# ไปยังโฟลเดอร์ที่ต้องการเก็บโปรเจกต์ (เช่น Desktop)
cd ~/Desktop

# Clone repository
git clone <REPOSITORY_URL> CS366-MissionProgress-Service

# เข้าไปในโฟลเดอร์โปรเจกต์
cd CS366-MissionProgress-Service
```

> 📌 แทนที่ `<REPOSITORY_URL>` ด้วย URL ของ Git repository จริง

### ตรวจสอบโครงสร้างโปรเจกต์

```bash
ls -la
```

✅ ต้องเห็นโฟลเดอร์เหล่านี้:

```
docs/
plan/
script/
src/
terraform/
README.md
```

---

## 6. ขั้นตอนที่ 4 — ติดตั้ง Dependencies

### 6.1 ติดตั้ง Go Dependencies (Backend)

```bash
cd src/backend
go mod download
```

✅ ผลลัพธ์: ไม่มี error แสดง (อาจมีข้อความ download modules)

```bash
# กลับไปที่ root ของโปรเจกต์
cd ../..
```

### 6.2 ติดตั้ง Node.js Dependencies (Frontend)

```bash
cd src/frontend
npm ci
```

✅ ผลลัพธ์: แสดงข้อความ `added XXX packages` โดยไม่มี error

```bash
# กลับไปที่ root ของโปรเจกต์
cd ../..
```

---

## 7. ขั้นตอนที่ 5 — Build โปรเจกต์

### 7.1 ให้สิทธิ์ Script

```bash
chmod +x script/*.sh
```

### 7.2 รัน Build Script

```bash
bash script/build.sh
```

Script นี้จะทำสิ่งต่อไปนี้:

1. สร้างโฟลเดอร์ `terraform/build/`
2. Cross-compile Go code สำหรับ Lambda ทั้ง 7 functions:
   - `report-progress`
   - `get-mission`
   - `authorizer`
   - `outbox-processor`
   - `presigned-url`
   - `list-missions`
   - `mission-assigned-handler`
3. สร้างไฟล์ `.zip` สำหรับแต่ละ function ไว้ใน `terraform/build/`
4. Build Frontend (Next.js) แบบ static export ไว้ใน `src/frontend/out/`

✅ **ผลลัพธ์ที่คาดหวัง** (ตัวอย่าง):

```
=== Building Lambda functions ===
--- Building report-progress ---
--- report-progress built successfully ---
--- Building get-mission ---
--- get-mission built successfully ---
--- Building authorizer ---
--- authorizer built successfully ---
--- Building outbox-processor ---
--- outbox-processor built successfully ---
--- Building presigned-url ---
--- presigned-url built successfully ---
--- Building list-missions ---
--- list-missions built successfully ---
--- Building mission-assigned-handler ---
--- mission-assigned-handler built successfully ---

=== Build complete (Lambda) ===
Zip files in .../terraform/build:
-rw-r--r--  report-progress.zip
-rw-r--r--  get-mission.zip
-rw-r--r--  authorizer.zip
-rw-r--r--  outbox-processor.zip
-rw-r--r--  presigned-url.zip
-rw-r--r--  list-missions.zip
-rw-r--r--  mission-assigned-handler.zip

=== Building Frontend ===
...
=== Frontend build complete ===
```

### 7.3 ตรวจสอบว่า Build สำเร็จ

```bash
# ตรวจสอบไฟล์ zip ทั้งหมด
ls -la terraform/build/*.zip
```

✅ ต้องเห็นไฟล์ `.zip` ทั้ง 7 ไฟล์

```bash
# ตรวจสอบ Frontend build output
ls src/frontend/out/
```

✅ ต้องเห็นไฟล์ `index.html` และโฟลเดอร์อื่น ๆ

---

## 8. ขั้นตอนที่ 6 — Deploy ขึ้น AWS

### 8.1 ตรวจสอบ AWS Credentials อีกครั้ง

```bash
aws sts get-caller-identity
```

✅ ต้องได้ผลลัพธ์เป็น JSON แสดง Account ID (ไม่ใช่ error)

> ⚠️ ถ้าได้ error → กลับไปขั้นตอน 3.4 เพื่อดึง Credentials ใหม่

### 8.2 Terraform Init

```bash
cd terraform
terraform init
```

✅ **ผลลัพธ์ที่คาดหวัง**:

```
Initializing the backend...
Initializing provider plugins...
- Finding hashicorp/aws versions matching "~> 5.0"...
- Installing hashicorp/aws v5.x.x...
- Installed hashicorp/aws v5.x.x (signed by HashiCorp)

Terraform has been successfully initialized!
```

### 8.3 Terraform Plan (ตรวจสอบก่อน Deploy)

```bash
terraform plan
```

✅ **ผลลัพธ์ที่คาดหวัง**: แสดงรายการทรัพยากรที่จะสร้าง เช่น:

```
Plan: XX to add, 0 to change, 0 to destroy.
```

ตรวจสอบว่าไม่มี error สีแดง

### 8.4 Terraform Apply (Deploy จริง)

```bash
terraform apply -auto-approve
```

⏱️ **ใช้เวลาประมาณ 2–5 นาที**

✅ **ผลลัพธ์ที่คาดหวัง**:

```
Apply complete! Resources added: XX, changed: 0, destroyed: 0.

Outputs:

api_gateway_invoke_url = "https://xxxxxxxxxx.execute-api.us-east-1.amazonaws.com/v1"
api_key_value = <sensitive>
evidence_bucket = "mission-progress-evidence-123456789012"
frontend_bucket = "mission-progress-frontend-123456789012"
frontend_url = "mission-progress-frontend-123456789012.s3-website-us-east-1.amazonaws.com"
```

### 8.5 Upload Frontend ขึ้น S3

```bash
# ดึงชื่อ S3 Bucket ของ Frontend
FRONTEND_BUCKET=$(terraform output -raw frontend_bucket)

# Upload ไฟล์ frontend ทั้งหมดขึ้น S3
aws s3 sync "../src/frontend/out/" "s3://$FRONTEND_BUCKET/" --delete --cache-control "public, max-age=3600"
```

✅ **ผลลัพธ์ที่คาดหวัง**: แสดงรายการไฟล์ที่อัปโหลด

```bash
# กลับไปที่ root ของโปรเจกต์
cd ..
```

### วิธีลัด — ใช้ Deploy Script (ทำขั้นตอน 7 + 8 รวมกัน)

หรือจะใช้ script สำเร็จรูปที่ทำ Build + Deploy ให้ทั้งหมดในคำสั่งเดียว:

```bash
bash script/deploy.sh
```

> Script นี้จะ: Build Lambda → Build Frontend → Terraform Init → Terraform Apply → Upload Frontend to S3

---

## 9. ขั้นตอนที่ 7 — Seed ข้อมูลตัวอย่าง

หลัง Deploy สำเร็จแล้ว ให้ใส่ข้อมูลตัวอย่างลง DynamoDB:

```bash
bash script/seed-data.sh
```

✅ **ผลลัพธ์ที่คาดหวัง**:

```
=== Seeding DynamoDB with sample data ===
--- Inserting MissionAssignment records ---
  Inserted MSN-001 (DISPATCHED)
  Inserted MSN-002 (EN_ROUTE)
  Inserted MSN-003 (ON_SITE)
  Inserted MSN-004 (NEED_BACKUP)
  Inserted MSN-005 (RESOLVED)

--- Inserting MissionTimeline records ---
  Inserted timeline for MSN-001
  Inserted timeline for MSN-002
  Inserted timeline for MSN-003
  Inserted timeline for MSN-004
  Inserted timeline for MSN-005

=== Seed data complete ===
```

### ข้อมูลตัวอย่างที่ถูกใส่

| Mission ID | Incident ID | Team         | สถานะ       |
| ---------- | ----------- | ------------ | ----------- |
| MSN-001    | INC-001     | TEAM-ALPHA   | DISPATCHED  |
| MSN-002    | INC-002     | TEAM-BRAVO   | EN_ROUTE    |
| MSN-003    | INC-003     | TEAM-CHARLIE | ON_SITE     |
| MSN-004    | INC-004     | TEAM-DELTA   | NEED_BACKUP |
| MSN-005    | INC-005     | TEAM-ECHO    | RESOLVED    |

---

## 10. ขั้นตอนที่ 8 — ดู Output ที่สำคัญ

### 10.1 ดู API URL

```bash
cd terraform
terraform output api_gateway_invoke_url
```

✅ **ผลลัพธ์** (ตัวอย่าง):

```
"https://abc123xyz.execute-api.us-east-1.amazonaws.com/v1"
```

📝 **จดบันทึก URL นี้ไว้** — จะใช้ในขั้นตอนถัดไป

### 10.2 ดู API Key

```bash
terraform output -raw api_key_value
```

✅ **ผลลัพธ์** (ตัวอย่าง):

```
mission-progress-api-key-2024
```

📝 **จดบันทึก API Key นี้ไว้** — จะใช้ใน header ของทุก request

### 10.3 ดู Frontend URL

```bash
terraform output frontend_url
```

✅ **ผลลัพธ์** (ตัวอย่าง):

```
"mission-progress-frontend-123456789012.s3-website-us-east-1.amazonaws.com"
```

📝 เปิดเว็บเบราว์เซอร์แล้วไปที่ `http://<frontend_url>` (ใช้ **http://** ไม่ใช่ https)

```bash
# กลับไปที่ root ของโปรเจกต์
cd ..
```

---

## 11. ขั้นตอนที่ 9 — ทดสอบ API อย่างละเอียด

> **หมายเหตุ**: ส่วนนี้ครอบคลุมการทดสอบทุก Endpoint ทุก Error case ตาม contract_demo1.md และ contract_demo2.md ครบถ้วน
> แต่ละรายการจะระบุ: (1) คำสั่ง curl (2) Lambda/Service ที่เกี่ยวข้อง (3) ผลลัพธ์ที่คาดหวัง

### 11.0 ตั้งค่าตัวแปร (ใน Terminal)

**ต้องทำก่อนรันคำสั่งทดสอบทุกข้อ:**

```bash
# แทนที่ด้วย URL จริงจากขั้นตอน 10.1
export API_URL="https://abc123xyz.execute-api.us-east-1.amazonaws.com/v1"

# แทนที่ด้วย API Key จริงจากขั้นตอน 10.2
export API_KEY="mission-progress-api-key-2024"
```

> 📌 ใช้ `export` เพื่อให้ตัวแปรคงอยู่ตลอด session ไม่ต้องตั้งซ้ำ

### ข้อมูล Seed Data ที่ใช้ทดสอบ

| Mission ID | Incident ID | Team         | สถานะเริ่มต้น | Impact Level |
| ---------- | ----------- | ------------ | ------------- | ------------ |
| MSN-001    | INC-001     | TEAM-ALPHA   | DISPATCHED    | 2            |
| MSN-002    | INC-002     | TEAM-BRAVO   | EN_ROUTE      | 3            |
| MSN-003    | INC-003     | TEAM-CHARLIE | ON_SITE       | 4            |
| MSN-004    | INC-004     | TEAM-DELTA   | NEED_BACKUP   | 4            |
| MSN-005    | INC-005     | TEAM-ECHO    | RESOLVED      | 1            |

---

### 11.1 ทดสอบ Authentication (Lambda: `authorizer`)

> **Lambda ที่ทำงาน:** `mission-progress-authorizer`
> ทุก request ต้องส่ง 2 headers: `x-api-key` และ `X-Rescue-Team-ID`

#### ✅ Test #1 — ส่ง API Key + Team ID ถูกต้อง

```bash
curl -s -H "x-api-key: $API_KEY" -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  "$API_URL/incidents/INC-001" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 200):** ได้รับข้อมูลภารกิจ — ดูรายละเอียดในหัวข้อ 11.3

#### ❌ Test #2 — ไม่ส่ง API Key

```bash
curl -s "$API_URL/incidents/INC-001" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 403):**

```json
{
  "message": "Forbidden"
}
```

#### ❌ Test #3 — ไม่ส่ง X-Rescue-Team-ID (ส่งเฉพาะ API Key)

```bash
curl -s -H "x-api-key: $API_KEY" \
  "$API_URL/incidents/INC-001" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 403):**

```json
{
  "message": "User is not authorized to access this resource"
}
```

---

### 11.2 ทดสอบ GET /incidents — ดูรายการภารกิจของทีม (Lambda: `list-missions`)

> **Lambda ที่ทำงาน:** `mission-progress-list-missions`
> **DynamoDB Table:** MissionAssignment (query ผ่าน GSI `team-index`)
> **Endpoint นี้เพิ่มใน Demo 2**

#### ✅ Test #4 — ดึงภารกิจทั้งหมดของทีม TEAM-ALPHA

```bash
curl -s -H "x-api-key: $API_KEY" -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  "$API_URL/incidents" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 200):**

```json
{
  "team_id": "TEAM-ALPHA",
  "total_missions": 1,
  "missions": [
    {
      "mission_id": "MSN-001",
      "incident_id": "INC-001",
      "rescue_team_id": "TEAM-ALPHA",
      "current_status": "DISPATCHED",
      "latest_impact_level": 2,
      "started_at": "2024-12-01T08:00:00Z",
      "last_updated_at": "2024-12-01T08:00:00Z"
    }
  ]
}
```

> 📌 ระบบจะ return เฉพาะภารกิจของทีมที่ส่งมาใน `X-Rescue-Team-ID` (ป้องกันดูข้อมูลข้ามทีม)

#### ✅ Test #5 — กรองเฉพาะสถานะ ON_SITE

```bash
curl -s -H "x-api-key: $API_KEY" -H "X-Rescue-Team-ID: TEAM-CHARLIE" \
  "$API_URL/incidents?status=ON_SITE" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 200):**

```json
{
  "team_id": "TEAM-CHARLIE",
  "total_missions": 1,
  "missions": [
    {
      "mission_id": "MSN-003",
      "incident_id": "INC-003",
      "rescue_team_id": "TEAM-CHARLIE",
      "current_status": "ON_SITE",
      "latest_impact_level": 4,
      "started_at": "2024-12-01T07:30:00Z",
      "last_updated_at": "2024-12-01T10:00:00Z"
    }
  ]
}
```

#### ✅ Test #6 — ทีมที่ไม่มีภารกิจ (return array ว่าง ไม่ใช่ 404)

```bash
curl -s -H "x-api-key: $API_KEY" -H "X-Rescue-Team-ID: TEAM-NEWBIE" \
  "$API_URL/incidents" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 200):**

```json
{
  "team_id": "TEAM-NEWBIE",
  "total_missions": 0,
  "missions": []
}
```

#### ❌ Test #7 — status filter ไม่ถูกต้อง

```bash
curl -s -H "x-api-key: $API_KEY" -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  "$API_URL/incidents?status=UNKNOWN_STATUS" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 400):**

```json
{
  "error": "INVALID_STATUS",
  "code": "INVALID_STATUS",
  "message": "Invalid status filter: UNKNOWN_STATUS"
}
```

---

### 11.3 ทดสอบ GET /incidents/{incident_id} — ดูรายละเอียดภารกิจ (Lambda: `get-mission`)

> **Lambda ที่ทำงาน:** `mission-progress-get-mission`
> **DynamoDB Tables:** MissionAssignment + MissionTimeline
> **Dependency:** เรียก **IncidentTracking Service** (เจ้าของ: กฤตเมธ ดำทองคำ) ผ่าน HTTP GET
> — ถ้า IncidentTracking ตอบสำเร็จ → `data_source: "full"` (มี description, location, incident_type)
> — ถ้า IncidentTracking ไม่ตอบ (timeout 3 วินาที) → `data_source: "partial"` (Degraded Mode)

#### ✅ Test #8 — ดึงข้อมูลภารกิจที่มีอยู่ (Degraded Mode — IncidentTracking ยังไม่ Deploy)

```bash
curl -s -H "x-api-key: $API_KEY" -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  "$API_URL/incidents/INC-001" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 200 — Degraded Mode):**

```json
{
  "incident_id": "INC-001",
  "mission_id": "MSN-001",
  "rescue_team_id": "TEAM-ALPHA",
  "current_status": "DISPATCHED",
  "latest_impact_level": 2,
  "started_at": "2024-12-01T08:00:00Z",
  "last_updated_at": "2024-12-01T08:00:00Z",
  "timeline": [
    {
      "mission_id": "MSN-001",
      "timestamp": "2024-12-01T08:00:00Z",
      "log_id": "LOG-001",
      "action_type": "STATUS_CHANGE",
      "description": "Mission dispatched to TEAM-ALPHA",
      "performed_by": "SYSTEM"
    }
  ],
  "data_source": "partial"
}
```

> 💡 `data_source: "partial"` = Degraded Mode → ไม่มีฟิลด์ `description`, `location`, `incident_type` เพราะ **IncidentTracking Service** (กฤตเมธ ดำทองคำ) ยังไม่ได้ Deploy หรือ URL ยังเป็น `http://localhost:9999`

#### ✅ Test #8b — ดึงข้อมูลภารกิจ (Full Mode — เมื่อ IncidentTracking Deploy แล้ว)

> **ทดสอบนี้ได้เมื่อ:** เพื่อน (กฤตเมธ) Deploy IncidentTracking Service แล้ว และแก้ `incident_service_url` ใน Terraform เป็น URL จริง แล้วรัน `terraform apply` ใหม่

```bash
curl -s -H "x-api-key: $API_KEY" -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  "$API_URL/incidents/INC-001" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 200 — Full Mode):**

```json
{
  "incident_id": "INC-001",
  "mission_id": "MSN-001",
  "rescue_team_id": "TEAM-ALPHA",
  "current_status": "DISPATCHED",
  "latest_impact_level": 2,
  "started_at": "2024-12-01T08:00:00Z",
  "last_updated_at": "2024-12-01T08:00:00Z",
  "description": "น้ำท่วมหนักบริเวณถนนพหลโยธิน",
  "location": "13.7563,100.5018",
  "incident_type": "FLOOD",
  "timeline": [...],
  "data_source": "full"
}
```

> 💡 สังเกต: มีฟิลด์ `description`, `location`, `incident_type` เพิ่มมา และ `data_source` เปลี่ยนเป็น `"full"`

#### ❌ Test #9 — incident_id ที่ไม่มีในระบบ

```bash
curl -s -H "x-api-key: $API_KEY" -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  "$API_URL/incidents/INC-99999" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 404):**

```json
{
  "error": "INCIDENT_NOT_FOUND",
  "code": "INCIDENT_NOT_FOUND",
  "message": "No mission found for incident: INC-99999"
}
```

---

### 11.4 ทดสอบ POST /incidents/{incident_id}/progress — Happy Path (Lambda: `report-progress`)

> **Lambda ที่ทำงาน:** `mission-progress-report-progress`
> **DynamoDB Tables:** อัปเดต MissionAssignment + เพิ่ม MissionTimeline
> **Async:** Publish events ไปยัง EventBridge → CloudWatch Logs + SQS ของ Service เพื่อน
> **Fallback:** ถ้า EventBridge ล้มเหลว → บันทึกลง EventOutbox (Outbox Pattern)

#### ✅ Test #10 — เปลี่ยนสถานะ DISPATCHED → EN_ROUTE (INC-001)

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"new_status":"EN_ROUTE","note":"กำลังเดินทางไปจุดเกิดเหตุ","current_location":"13.7563,100.5018"}' \
  "$API_URL/incidents/INC-001/progress" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 200):**

```json
{
  "message": "Progress reported successfully",
  "mission_id": "MSN-001",
  "incident_id": "INC-001",
  "old_status": "DISPATCHED",
  "new_status": "EN_ROUTE",
  "updated_at": "2025-..."
}
```

> 📌 **Events ที่ถูก publish:** `MissionStatusChanged` (1 event) → ตรวจสอบใน CloudWatch Logs ดูหัวข้อ 11.8

#### ✅ Test #11 — เปลี่ยนสถานะ EN_ROUTE → ON_SITE (INC-001)

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"new_status":"ON_SITE","note":"ถึงจุดเกิดเหตุแล้ว กำลังประเมินสถานการณ์","current_location":"13.7380,100.5230"}' \
  "$API_URL/incidents/INC-001/progress" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 200):**

```json
{
  "message": "Progress reported successfully",
  "mission_id": "MSN-001",
  "incident_id": "INC-001",
  "old_status": "EN_ROUTE",
  "new_status": "ON_SITE",
  "updated_at": "2025-..."
}
```

#### ✅ Test #12 — เปลี่ยนสถานะ ON_SITE → NEED_BACKUP พร้อม ImpactLevel (INC-003)

> **สำคัญ:** การเปลี่ยนเป็น `NEED_BACKUP` จะ trigger **3 events** พร้อมกัน!

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-CHARLIE" \
  -H "Content-Type: application/json" \
  -d '{"new_status":"NEED_BACKUP","note":"น้ำท่วมสูงกว่าที่คาด ต้องการกำลังเสริม","current_location":"13.7380,100.5230","new_impact_level":4}' \
  "$API_URL/incidents/INC-003/progress" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 200):**

```json
{
  "message": "Progress reported successfully",
  "mission_id": "MSN-003",
  "incident_id": "INC-003",
  "old_status": "ON_SITE",
  "new_status": "NEED_BACKUP",
  "updated_at": "2025-..."
}
```

> 📌 **Events ที่ถูก publish (3 events):**
>
> 1. `MissionStatusChanged` → CloudWatch Logs + IncidentTracking SQS
> 2. `MissionBackupRequested` → CloudWatch Logs + Prioritization SQS
> 3. `ImpactLevelUpdated` → CloudWatch Logs + IncidentTracking SQS + Prioritization SQS
>
> ตรวจสอบทั้ง 3 events ได้ใน CloudWatch Logs ดูหัวข้อ 11.8

#### ✅ Test #13 — เปลี่ยนสถานะ NEED_BACKUP → ON_SITE (INC-004 — กลับมา ON_SITE ได้)

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-DELTA" \
  -H "Content-Type: application/json" \
  -d '{"new_status":"ON_SITE","note":"ได้รับกำลังเสริมแล้ว กลับมาปฏิบัติงาน"}' \
  "$API_URL/incidents/INC-004/progress" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 200):**

```json
{
  "message": "Progress reported successfully",
  "mission_id": "MSN-004",
  "incident_id": "INC-004",
  "old_status": "NEED_BACKUP",
  "new_status": "ON_SITE",
  "updated_at": "2025-..."
}
```

#### ✅ Test #14 — เปลี่ยนสถานะ ON_SITE → RESOLVED พร้อม ImpactLevel (INC-001)

> **สำคัญ:** เมื่อเปลี่ยนเป็น `RESOLVED` จะ trigger event `MissionStatusChanged` ไปยัง **Dispatch SQS** ด้วย (เฉพาะ RESOLVED เท่านั้นที่ส่งไป Dispatch)

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"new_status":"RESOLVED","note":"เหตุการณ์เรียบร้อยแล้ว อพยพผู้ประสบภัยสำเร็จ","new_impact_level":3}' \
  "$API_URL/incidents/INC-001/progress" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 200):**

```json
{
  "message": "Progress reported successfully",
  "mission_id": "MSN-001",
  "incident_id": "INC-001",
  "old_status": "ON_SITE",
  "new_status": "RESOLVED",
  "updated_at": "2025-..."
}
```

> 📌 **Events ที่ถูก publish (2 events):**
>
> 1. `MissionStatusChanged` → CloudWatch Logs + IncidentTracking SQS + **Dispatch SQS** (เพราะ RESOLVED)
> 2. `ImpactLevelUpdated` → CloudWatch Logs + IncidentTracking SQS + Prioritization SQS

#### ✅ Test #14b — ยืนยัน GET หลัง progress: Timeline มี entry ใหม่

```bash
curl -s -H "x-api-key: $API_KEY" -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  "$API_URL/incidents/INC-001" | jq '.timeline'
```

**ผลลัพธ์ที่คาดหวัง:** Timeline จะมีหลาย entries แสดงการเปลี่ยนสถานะทุกครั้ง (DISPATCHED → EN_ROUTE → ON_SITE → RESOLVED)

---

### 11.5 ทดสอบ POST /incidents/{incident_id}/progress — Error Cases

#### ❌ Test #15 — Transition สถานะไม่ถูกต้อง (EN_ROUTE → RESOLVED)

> State Machine ไม่อนุญาตให้ข้ามจาก EN_ROUTE ไป RESOLVED ต้องผ่าน ON_SITE ก่อน

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-BRAVO" \
  -H "Content-Type: application/json" \
  -d '{"new_status":"RESOLVED"}' \
  "$API_URL/incidents/INC-002/progress" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 400):**

```json
{
  "error": "INVALID_STATE_TRANSITION",
  "code": "INVALID_STATE_TRANSITION",
  "message": "Cannot transition from EN_ROUTE to RESOLVED"
}
```

#### ❌ Test #16 — ไม่ส่ง new_status ใน body

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"note":"ทดสอบ"}' \
  "$API_URL/incidents/INC-001/progress" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 400):**

```json
{
  "error": "MISSING_PARAMETER",
  "code": "MISSING_PARAMETER",
  "message": "new_status is required"
}
```

#### ❌ Test #17 — ส่งค่า status ที่ไม่มีในระบบ

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"new_status":"UNKNOWN_STATUS"}' \
  "$API_URL/incidents/INC-001/progress" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 400):**

```json
{
  "error": "INVALID_STATUS",
  "code": "INVALID_STATUS",
  "message": "Invalid status value: UNKNOWN_STATUS"
}
```

#### ❌ Test #18 — incident_id ที่ไม่มีในระบบ

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"new_status":"EN_ROUTE"}' \
  "$API_URL/incidents/INC-99999/progress" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 404):**

```json
{
  "error": "INCIDENT_NOT_FOUND",
  "code": "INCIDENT_NOT_FOUND",
  "message": "No mission found for incident: INC-99999"
}
```

#### ❌ Test #19 — ส่ง JSON body ไม่ถูกต้อง

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d 'invalid-json' \
  "$API_URL/incidents/INC-001/progress" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 400):**

```json
{
  "error": "INVALID_BODY",
  "code": "INVALID_BODY",
  "message": "Invalid request body"
}
```

#### ❌ Test #20 — อัปเดตภารกิจที่ RESOLVED แล้ว (สถานะสุดท้าย — เปลี่ยนต่อไม่ได้)

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ECHO" \
  -H "Content-Type: application/json" \
  -d '{"new_status":"EN_ROUTE"}' \
  "$API_URL/incidents/INC-005/progress" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 400):**

```json
{
  "error": "INVALID_STATE_TRANSITION",
  "code": "INVALID_STATE_TRANSITION",
  "message": "Cannot transition from RESOLVED to EN_ROUTE"
}
```

---

### 11.6 ทดสอบ POST /incidents/{incident_id}/presigned-url — อัปโหลดภาพหลักฐาน (Lambda: `presigned-url`)

> **Lambda ที่ทำงาน:** `mission-progress-presigned-url`
> **DynamoDB Table:** MissionAssignment (ตรวจว่า mission มีอยู่)
> **S3 Bucket:** `mission-progress-evidence-XXXX` (สร้าง presigned PUT URL)
> **Endpoint นี้เพิ่มใน Demo 2**

#### ✅ Test #21 — ขอ presigned URL สำหรับ JPEG

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"file_name":"flood-evidence.jpg","content_type":"image/jpeg"}' \
  "$API_URL/incidents/INC-001/presigned-url" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 200):**

```json
{
  "upload_url": "https://s3.amazonaws.com/mission-progress-evidence-XXXX/evidence/INC-001/TEAM-ALPHA/1718353500-flood-evidence.jpg?...",
  "image_key": "evidence/INC-001/TEAM-ALPHA/1718353500-flood-evidence.jpg",
  "expires_in": 300,
  "message": "Upload URL generated successfully. Use PUT method to upload."
}
```

> 📌 จดค่า `upload_url` และ `image_key` ไว้ — จะใช้ในขั้นตอนถัดไป (Test #25, #26)

#### ✅ Test #22 — ขอ presigned URL สำหรับ PNG

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"file_name":"screenshot.png","content_type":"image/png"}' \
  "$API_URL/incidents/INC-001/presigned-url" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 200):** เหมือน Test #21 แต่ `image_key` จะลงท้ายด้วย `screenshot.png`

#### ❌ Test #23 — content_type ไม่รองรับ (PDF)

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"file_name":"document.pdf","content_type":"application/pdf"}' \
  "$API_URL/incidents/INC-001/presigned-url" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 400):**

```json
{
  "error": "INVALID_CONTENT_TYPE",
  "code": "INVALID_CONTENT_TYPE",
  "message": "content_type must be one of: image/jpeg, image/png, image/webp"
}
```

#### ❌ Test #24a — ไม่ส่ง file_name

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"content_type":"image/jpeg"}' \
  "$API_URL/incidents/INC-001/presigned-url" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 400):**

```json
{
  "error": "MISSING_PARAMETER",
  "code": "MISSING_PARAMETER",
  "message": "file_name is required"
}
```

#### ❌ Test #24b — ไม่ส่ง content_type

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"file_name":"photo.jpg"}' \
  "$API_URL/incidents/INC-001/presigned-url" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 400):**

```json
{
  "error": "MISSING_PARAMETER",
  "code": "MISSING_PARAMETER",
  "message": "content_type is required"
}
```

#### ❌ Test #24c — incident_id ไม่มีในระบบ

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  -H "Content-Type: application/json" \
  -d '{"file_name":"photo.jpg","content_type":"image/jpeg"}' \
  "$API_URL/incidents/INC-99999/presigned-url" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 404):**

```json
{
  "error": "INCIDENT_NOT_FOUND",
  "code": "INCIDENT_NOT_FOUND",
  "message": "No mission found for incident: INC-99999"
}
```

---

### 11.7 ทดสอบ Upload ภาพ + แนบใน Progress (Full Flow)

> **Flow:** ขอ presigned URL → อัปโหลดภาพไป S3 → แนบ image_key ใน report-progress

#### ✅ Test #25 — อัปโหลดภาพผ่าน Presigned URL

**ขั้นตอน 1:** ขอ presigned URL (ใช้ภารกิจที่ยังไม่ RESOLVED เช่น INC-002)

```bash
# ขอ presigned URL
PRESIGN_RESPONSE=$(curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-BRAVO" \
  -H "Content-Type: application/json" \
  -d '{"file_name":"flood-photo.jpg","content_type":"image/jpeg"}' \
  "$API_URL/incidents/INC-002/presigned-url")

echo "$PRESIGN_RESPONSE" | jq .

# ดึงค่า upload_url และ image_key
UPLOAD_URL=$(echo "$PRESIGN_RESPONSE" | jq -r '.upload_url')
IMAGE_KEY=$(echo "$PRESIGN_RESPONSE" | jq -r '.image_key')

echo "Upload URL: $UPLOAD_URL"
echo "Image Key: $IMAGE_KEY"
```

**ขั้นตอน 2:** อัปโหลดภาพ (ต้องมีไฟล์ภาพจริง หรือสร้างไฟล์ทดสอบ)

```bash
# สร้างไฟล์ทดสอบขนาดเล็ก (ถ้าไม่มีรูปจริง)
echo "test-image-data" > /tmp/test-photo.jpg

# อัปโหลดไฟล์ด้วย PUT
curl -X PUT \
  -H "Content-Type: image/jpeg" \
  -T /tmp/test-photo.jpg \
  "$UPLOAD_URL"
```

**ผลลัพธ์ที่คาดหวัง:** HTTP 200 (ไม่มี body — S3 PUT สำเร็จ)

> ⚠️ `Content-Type` ใน PUT ต้องตรงกับ `content_type` ที่ส่งตอนขอ presigned URL

#### ✅ Test #26 — Report Progress พร้อม image_key (แนบภาพหลักฐาน)

```bash
curl -s -X POST \
  -H "x-api-key: $API_KEY" \
  -H "X-Rescue-Team-ID: TEAM-BRAVO" \
  -H "Content-Type: application/json" \
  -d "{\"new_status\":\"ON_SITE\",\"note\":\"ถึงจุดเกิดเหตุแล้ว\",\"image_key\":\"$IMAGE_KEY\"}" \
  "$API_URL/incidents/INC-002/progress" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 200):**

```json
{
  "message": "Progress reported successfully",
  "mission_id": "MSN-002",
  "incident_id": "INC-002",
  "old_status": "EN_ROUTE",
  "new_status": "ON_SITE",
  "updated_at": "2025-..."
}
```

**ยืนยัน:** ดู Timeline จะเห็น `image_key` อยู่ใน entry ล่าสุด

```bash
curl -s -H "x-api-key: $API_KEY" -H "X-Rescue-Team-ID: TEAM-BRAVO" \
  "$API_URL/incidents/INC-002" | jq '.timeline[-1]'
```

**ผลลัพธ์:** Timeline entry ล่าสุดจะมีฟิลด์ `image_key`

---

### 11.8 ตรวจสอบ EventBridge Events ใน CloudWatch Logs

> **ทำไมต้องตรวจ:** เพื่อยืนยันว่า events ถูก publish ไป EventBridge สำเร็จ ซึ่งเป็นส่วน Async ที่ส่งข้อมูลไปยัง Service เพื่อน

#### วิธีที่ 1 — ดูผ่าน AWS Console

1. ไปที่ **AWS Console** → ค้นหา **"CloudWatch"**
2. คลิก **"Logs"** → **"Log groups"**
3. ค้นหา Log Groups ต่อไปนี้:

| Log Group                                             | Event ที่บันทึก          | เงื่อนไขที่ trigger               |
| ----------------------------------------------------- | ------------------------ | --------------------------------- |
| `/aws/events/mission-progress/mission-status-changed` | `MissionStatusChanged`   | ทุกครั้งที่สถานะเปลี่ยน           |
| `/aws/events/mission-progress/backup-requested`       | `MissionBackupRequested` | เฉพาะเมื่อเปลี่ยนเป็น NEED_BACKUP |
| `/aws/events/mission-progress/impact-level-updated`   | `ImpactLevelUpdated`     | เฉพาะเมื่อส่ง new_impact_level    |

4. คลิกที่ Log Group → คลิก **Log stream** ล่าสุด → ดู event payload

#### วิธีที่ 2 — ดูผ่าน AWS CLI

**ตรวจสอบ MissionStatusChanged:**

```bash
aws logs filter-log-events \
  --log-group-name "/aws/events/mission-progress/mission-status-changed" \
  --start-time $(date -d '10 minutes ago' +%s000 2>/dev/null || date -v-10M +%s000) \
  --region us-east-1 | jq '.events[].message' | head -5
```

**ตัวอย่าง Event ที่คาดหวังใน Log:**

```json
{
  "source": "MissionProgressService",
  "detail-type": "MissionStatusChanged",
  "detail": {
    "schema_version": "1.0",
    "mission_id": "MSN-001",
    "incident_id": "INC-001",
    "rescue_team_id": "TEAM-ALPHA",
    "old_status": "DISPATCHED",
    "new_status": "EN_ROUTE",
    "changed_at": "2025-...",
    "changed_by": "TEAM-ALPHA"
  }
}
```

**ตรวจสอบ MissionBackupRequested (ต้องทำ Test #12 ก่อน):**

```bash
aws logs filter-log-events \
  --log-group-name "/aws/events/mission-progress/backup-requested" \
  --start-time $(date -d '10 minutes ago' +%s000 2>/dev/null || date -v-10M +%s000) \
  --region us-east-1 | jq '.events[].message' | head -5
```

**ตัวอย่าง Event:**

```json
{
  "source": "MissionProgressService",
  "detail-type": "MissionBackupRequested",
  "detail": {
    "schema_version": "1.0",
    "mission_id": "MSN-003",
    "incident_id": "INC-003",
    "rescue_team_id": "TEAM-CHARLIE",
    "requested_at": "2025-...",
    "requested_by": "TEAM-CHARLIE",
    "location": "13.7380,100.5230"
  }
}
```

**ตรวจสอบ ImpactLevelUpdated (ต้องทำ Test #12 หรือ #14 ก่อน):**

```bash
aws logs filter-log-events \
  --log-group-name "/aws/events/mission-progress/impact-level-updated" \
  --start-time $(date -d '10 minutes ago' +%s000 2>/dev/null || date -v-10M +%s000) \
  --region us-east-1 | jq '.events[].message' | head -5
```

**ตัวอย่าง Event:**

```json
{
  "source": "MissionProgressService",
  "detail-type": "ImpactLevelUpdated",
  "detail": {
    "schema_version": "1.0",
    "mission_id": "MSN-003",
    "incident_id": "INC-003",
    "rescue_team_id": "TEAM-CHARLIE",
    "old_level": 3,
    "new_level": 4,
    "updated_at": "2025-...",
    "updated_by": "TEAM-CHARLIE"
  }
}
```

> 💡 **สำหรับ Presentation:** แสดง CloudWatch Logs เหล่านี้เพื่อพิสูจน์ว่า events ถูกส่งออกไปจริง → Service เพื่อนสามารถ subscribe ผ่าน SQS ได้

---

### 11.9 ทดสอบ Inbound Event — MissionAssignedEvent (Lambda: `mission-assigned-handler`)

> **Lambda ที่ทำงาน:** `mission-progress-mission-assigned-handler`
> **Event มาจาก:** Dispatch Management Service (เจ้าของ Dispatch Service)
> **การทำงาน:** เมื่อ Dispatch Service มอบหมายภารกิจ → สร้าง mission record อัตโนมัติ (DISPATCHED)
> **Endpoint นี้เพิ่มใน Demo 2 — เป็น Async (ไม่ใช่ REST API)**

#### ✅ Test #27 — จำลองส่ง MissionAssignedEvent ผ่าน AWS CLI

```bash
aws events put-events \
  --region us-east-1 \
  --entries '[{
    "Source": "dispatch-management-service",
    "DetailType": "MissionAssignedEvent",
    "EventBusName": "mission-progress-events",
    "Detail": "{\"mission_id\":\"MSN-TEST-001\",\"rescue_unit_id\":\"TEAM-ALPHA\",\"incident_id\":\"INC-TEST-001\",\"assigned_at\":\"2025-06-14T08:45:00Z\"}"
  }]'
```

**ผลลัพธ์ที่คาดหวัง:**

```json
{
  "FailedEntryCount": 0,
  "Entries": [
    {
      "EventId": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    }
  ]
}
```

**ยืนยัน:** ดึงข้อมูลภารกิจที่เพิ่งสร้าง

```bash
curl -s -H "x-api-key: $API_KEY" -H "X-Rescue-Team-ID: TEAM-ALPHA" \
  "$API_URL/incidents/INC-TEST-001" | jq .
```

**ผลลัพธ์ที่คาดหวัง (HTTP 200):**

```json
{
  "incident_id": "INC-TEST-001",
  "mission_id": "MSN-TEST-001",
  "rescue_team_id": "TEAM-ALPHA",
  "current_status": "DISPATCHED",
  "latest_impact_level": 0,
  "started_at": "2025-06-14T08:45:00Z",
  "last_updated_at": "2025-06-14T08:45:00Z",
  "timeline": [
    {
      "mission_id": "MSN-TEST-001",
      "action_type": "MISSION_ASSIGNED",
      "description": "Mission assigned to TEAM-ALPHA",
      "performed_by": "SYSTEM",
      ...
    }
  ],
  "data_source": "partial"
}
```

#### ✅ Test #28 — ส่ง MissionAssignedEvent ซ้ำ (Idempotent — ไม่ error)

```bash
aws events put-events \
  --region us-east-1 \
  --entries '[{
    "Source": "dispatch-management-service",
    "DetailType": "MissionAssignedEvent",
    "EventBusName": "mission-progress-events",
    "Detail": "{\"mission_id\":\"MSN-TEST-001\",\"rescue_unit_id\":\"TEAM-ALPHA\",\"incident_id\":\"INC-TEST-001\",\"assigned_at\":\"2025-06-14T08:45:00Z\"}"
  }]'
```

**ผลลัพธ์ที่คาดหวัง:** `FailedEntryCount: 0` — event ถูกส่งสำเร็จ แต่ Lambda จะ skip การสร้าง (ไม่ duplicate)

> 📌 **Idempotent:** ใช้ DynamoDB condition `attribute_not_exists(mission_id)` → ถ้า mission_id ซ้ำจะไม่สร้างซ้ำ

---

### 11.10 ทดสอบ Outbox Processor (Lambda: `outbox-processor`)

> **Lambda ที่ทำงาน:** `mission-progress-outbox-processor`
> **การทำงาน:** Lambda scheduled ทุก 5 นาที เพื่อ retry ส่ง events ที่ publish ไม่สำเร็จ (Outbox Pattern)
> **DynamoDB Table:** EventOutbox

#### วิธีตรวจสอบ

```bash
# ดูข้อมูลใน EventOutbox table (ถ้ามี events ค้าง)
aws dynamodb scan --table-name EventOutbox --region us-east-1 | jq '.Items'
```

**ผลลัพธ์ที่คาดหวัง:**

- ถ้า EventBridge ทำงานปกติ → Outbox table จะ **ว่างเปล่า** (`[]`)
- ถ้ามี events ค้าง → จะเห็นรายการ events ที่รอ retry

> 📌 Outbox Processor จะ retry อัตโนมัติ ไม่ต้องทำอะไรเพิ่ม

---

### 11.11 ทดสอบด้วย Postman (GUI — ทางเลือกแทน curl)

ถ้าต้องการใช้ Postman แทน curl:

1. เปิด **Postman** → สร้าง Request ใหม่
2. ตั้ง Headers สำหรับทุก Request:

   | Key                | Value                            |
   | ------------------ | -------------------------------- |
   | `x-api-key`        | `<API_KEY จาก terraform output>` |
   | `X-Rescue-Team-ID` | `TEAM-ALPHA`                     |

3. สำหรับ **GET /incidents/{id}**: Method = GET, URL = `$API_URL/incidents/INC-001`
4. สำหรับ **POST progress**: Method = POST, URL = `$API_URL/incidents/INC-001/progress`, Body → raw → JSON:

```json
{
  "new_status": "EN_ROUTE",
  "note": "กำลังเดินทาง",
  "current_location": "13.7563,100.5018"
}
```

5. สำหรับ **POST presigned-url**: Method = POST, URL = `$API_URL/incidents/INC-001/presigned-url`, Body → raw → JSON:

```json
{
  "file_name": "flood-photo.jpg",
  "content_type": "image/jpeg"
}
```

---

### 11.12 สรุปผลการทดสอบทั้งหมด

| #   | กรณีทดสอบ                                  | Endpoint               | Lambda                         | HTTP | ผลที่คาดหวัง                     |
| --- | ------------------------------------------ | ---------------------- | ------------------------------ | ---- | -------------------------------- |
| 1   | ✅ ส่ง API Key + Team ID ถูกต้อง           | ทุก endpoint           | authorizer                     | 200  | ได้รับข้อมูลตามปกติ              |
| 2   | ❌ ไม่ส่ง API Key                          | ทุก endpoint           | authorizer                     | 403  | `Forbidden`                      |
| 3   | ❌ ไม่ส่ง X-Rescue-Team-ID                 | ทุก endpoint           | authorizer                     | 403  | `User is not authorized...`      |
| 4   | ✅ GET /incidents ดึงภารกิจของทีม          | GET /incidents         | list-missions                  | 200  | รายการภารกิจของทีม               |
| 5   | ✅ GET /incidents กรอง status              | GET /incidents?status= | list-missions                  | 200  | เฉพาะสถานะที่กรอง                |
| 6   | ✅ GET /incidents ทีมไม่มีภารกิจ           | GET /incidents         | list-missions                  | 200  | `missions: []`                   |
| 7   | ❌ GET /incidents status filter ผิด        | GET /incidents?status= | list-missions                  | 400  | `INVALID_STATUS`                 |
| 8   | ✅ GET /incidents/{id} สำเร็จ (Degraded)   | GET /incidents/{id}    | get-mission                    | 200  | `data_source: "partial"`         |
| 8b  | ✅ GET /incidents/{id} สำเร็จ (Full Mode)  | GET /incidents/{id}    | get-mission + IncidentTracking | 200  | `data_source: "full"`            |
| 9   | ❌ GET /incidents/{id} ไม่พบ               | GET /incidents/{id}    | get-mission                    | 404  | `INCIDENT_NOT_FOUND`             |
| 10  | ✅ POST DISPATCHED → EN_ROUTE              | POST progress          | report-progress                | 200  | `Progress reported successfully` |
| 11  | ✅ POST EN_ROUTE → ON_SITE                 | POST progress          | report-progress                | 200  | `Progress reported successfully` |
| 12  | ✅ POST ON_SITE → NEED_BACKUP + Impact     | POST progress          | report-progress                | 200  | + 3 events published             |
| 13  | ✅ POST NEED_BACKUP → ON_SITE              | POST progress          | report-progress                | 200  | `Progress reported successfully` |
| 14  | ✅ POST ON_SITE → RESOLVED + Impact        | POST progress          | report-progress                | 200  | + event → Dispatch SQS           |
| 14b | ✅ GET ยืนยัน Timeline มี entries ครบ      | GET /incidents/{id}    | get-mission                    | 200  | Timeline entries ครบทุกขั้น      |
| 15  | ❌ POST transition ผิด (EN_ROUTE→RESOLVED) | POST progress          | report-progress                | 400  | `INVALID_STATE_TRANSITION`       |
| 16  | ❌ POST ไม่ส่ง new_status                  | POST progress          | report-progress                | 400  | `MISSING_PARAMETER`              |
| 17  | ❌ POST status ไม่มีในระบบ                 | POST progress          | report-progress                | 400  | `INVALID_STATUS`                 |
| 18  | ❌ POST incident_id ไม่มี                  | POST progress          | report-progress                | 404  | `INCIDENT_NOT_FOUND`             |
| 19  | ❌ POST JSON body ไม่ถูกต้อง               | POST progress          | report-progress                | 400  | `INVALID_BODY`                   |
| 20  | ❌ POST RESOLVED → อะไรก็ตาม               | POST progress          | report-progress                | 400  | `INVALID_STATE_TRANSITION`       |
| 21  | ✅ POST presigned-url (JPEG)               | POST presigned-url     | presigned-url                  | 200  | `upload_url` + `image_key`       |
| 22  | ✅ POST presigned-url (PNG)                | POST presigned-url     | presigned-url                  | 200  | `upload_url` + `image_key`       |
| 23  | ❌ POST presigned-url content_type ผิด     | POST presigned-url     | presigned-url                  | 400  | `INVALID_CONTENT_TYPE`           |
| 24a | ❌ POST presigned-url ไม่ส่ง file_name     | POST presigned-url     | presigned-url                  | 400  | `MISSING_PARAMETER`              |
| 24b | ❌ POST presigned-url ไม่ส่ง content_type  | POST presigned-url     | presigned-url                  | 400  | `MISSING_PARAMETER`              |
| 24c | ❌ POST presigned-url incident ไม่มี       | POST presigned-url     | presigned-url                  | 404  | `INCIDENT_NOT_FOUND`             |
| 25  | ✅ Upload ภาพผ่าน presigned URL            | S3 PUT                 | S3 (direct)                    | 200  | อัปโหลดสำเร็จ                    |
| 26  | ✅ POST progress พร้อม image_key           | POST progress          | report-progress                | 200  | Timeline มี image_key            |
| 27  | ✅ MissionAssignedEvent → สร้างภารกิจ      | EventBridge (inbound)  | mission-assigned-handler       | —    | mission DISPATCHED + timeline    |
| 28  | ✅ MissionAssignedEvent ซ้ำ (idempotent)   | EventBridge (inbound)  | mission-assigned-handler       | —    | skip ไม่ error                   |

### ลำดับแนะนำสำหรับ Presentation

> **เพื่อให้ demo ราบรื่น** แนะนำทดสอบตามลำดับนี้:

1. **Test #2, #3** — แสดง Authentication ทำงาน (403)
2. **Test #1, #4** — แสดงว่าส่ง Header ถูกต้องแล้วผ่าน (200)
3. **Test #8** — แสดง GET mission detail (Degraded Mode)
4. **Test #10 → #11** — แสดง Full Flow: DISPATCHED → EN_ROUTE → ON_SITE
5. **Test #21** — แสดง Presigned URL
6. **Test #25, #26** — แสดง Upload ภาพ + แนบใน progress (image_key)
7. **Test #12** — แสดง NEED_BACKUP + ImpactLevel (trigger 3 events)
8. **Test #14** — แสดง RESOLVED (trigger event ไป Dispatch SQS)
9. **Test #14b** — แสดง Timeline ครบทุกขั้นตอน
10. **Test #15, #16, #17, #19, #20** — แสดง Error handling ทุกแบบ
11. **CloudWatch Logs** (หัวข้อ 11.8) — แสดง events ที่ถูก publish
12. **Test #27** — แสดง Inbound MissionAssignedEvent จาก Dispatch
13. **Frontend** (หัวข้อ 12) — แสดง Dashboard + Mission Detail + Upload

---

## 12. ขั้นตอนที่ 10 — ใช้งาน Frontend (Web UI)

### 12.1 เปิดหน้าเว็บ

1. เปิดเว็บเบราว์เซอร์
2. ไปที่ URL ของ Frontend (จากขั้นตอน 10.3):

```
http://mission-progress-frontend-123456789012.s3-website-us-east-1.amazonaws.com
```

> ⚠️ ใช้ **http://** ไม่ใช่ https://

### 12.2 หน้า Login (ตั้งค่าการเชื่อมต่อ)

จะเห็นหน้าฟอร์มให้กรอกข้อมูล 3 ช่อง:

1. **API URL** — ใส่ API Gateway URL จากขั้นตอน 10.1 เช่น:

   ```
   https://abc123xyz.execute-api.us-east-1.amazonaws.com/v1
   ```

2. **API Key** — ใส่ API Key จากขั้นตอน 10.2 เช่น:

   ```
   mission-progress-api-key-2024
   ```

3. **Team ID** — เลือกจาก dropdown หรือกรอกเอง เช่น:
   - `TEAM-ALPHA`
   - `TEAM-BRAVO`
   - `TEAM-CHARLIE`
   - `TEAM-DELTA`
   - `TEAM-ECHO`

4. กดปุ่ม **"เข้าสู่ระบบ"** หรือ **"Connect"**

### 12.3 หน้า Dashboard

หลังล็อกอินสำเร็จ จะเห็น:

- **สรุปจำนวนภารกิจตามสถานะ** — การ์ดแสดงจำนวน DISPATCHED, EN_ROUTE, ON_SITE, NEED_BACKUP, RESOLVED
- **รายการภารกิจ** — ตารางแสดงรายละเอียดแต่ละภารกิจ
- **ตัวกรองสถานะ** — กดเลือกสถานะเพื่อกรองรายการ
- **ปุ่มรีเฟรช** — กดเพื่อโหลดข้อมูลใหม่

### 12.4 หน้ารายละเอียดภารกิจ

คลิกที่ภารกิจในตาราง จะเข้าสู่หน้ารายละเอียดที่แสดง:

- ข้อมูลภารกิจ (Mission ID, Incident ID, Team, สถานะ)
- Timeline การปฏิบัติงาน
- ปุ่มเปลี่ยนสถานะ (ตาม State Machine)
- ฟอร์มอัปโหลดหลักฐานภาพ

---

## 13. ขั้นตอนที่ 11 — ตรวจสอบทรัพยากรบน AWS Console

หลังจาก Deploy แล้ว สามารถเข้าไปดูทรัพยากรที่สร้างขึ้นบน AWS Console ได้:

### 13.1 เปิด AWS Console

1. กลับไปที่หน้า **AWS Academy Learner Lab**
2. คลิกที่ **"AWS" 🟢** (ตัวหนังสือ "AWS" ที่มีวงกลมสีเขียว) → จะเปิด AWS Console ในหน้าต่างใหม่

### 13.2 ดู DynamoDB Tables

1. ที่ช่องค้นหาด้านบน (Search bar) พิมพ์ **"DynamoDB"** แล้วคลิก **"DynamoDB"** ในผลลัพธ์
2. ที่เมนูด้านซ้าย คลิก **"Tables"**
3. จะเห็น 3 ตาราง:
   - **MissionAssignment** — เก็บข้อมูลภารกิจ
   - **MissionTimeline** — เก็บ Timeline การปฏิบัติงาน
   - **EventOutbox** — เก็บ Event ที่รอส่ง
4. คลิกที่ชื่อตาราง เช่น **"MissionAssignment"** → คลิก **"Explore table items"** → จะเห็นข้อมูลที่ seed ไว้

### 13.3 ดู Lambda Functions

1. ที่ช่องค้นหาด้านบน พิมพ์ **"Lambda"** แล้วคลิก **"Lambda"**
2. ที่เมนูด้านซ้าย คลิก **"Functions"**
3. จะเห็น Lambda Functions ที่ขึ้นต้นด้วย `mission-progress-`:
   - `mission-progress-report-progress`
   - `mission-progress-get-mission`
   - `mission-progress-authorizer`
   - `mission-progress-outbox-processor`
   - `mission-progress-presigned-url`
   - `mission-progress-list-missions`
   - `mission-progress-mission-assigned-handler`
4. คลิกที่ชื่อ Function → ไปที่แท็บ **"Monitor"** → คลิก **"View CloudWatch logs"** → ดูล็อกการทำงาน

### 13.4 ดู API Gateway

1. ที่ช่องค้นหาด้านบน พิมพ์ **"API Gateway"** แล้วคลิก **"API Gateway"**
2. จะเห็น API ชื่อ **"mission-progress-api"**
3. คลิกเข้าไป จะเห็น:
   - **Resources**: แสดงโครงสร้าง API paths (`/incidents`, `/incidents/{incident_id}`, `/incidents/{incident_id}/progress`, etc.)
   - **Stages**: คลิก **"v1"** จะเห็น **Invoke URL** ด้านบน
   - **Authorizers**: คลิกจะเห็น Lambda Authorizer

### 13.5 ดู EventBridge

1. ที่ช่องค้นหาด้านบน พิมพ์ **"EventBridge"** แล้วคลิก **"Amazon EventBridge"**
2. ที่เมนูด้านซ้าย คลิก **"Event buses"** → จะเห็น **"mission-progress-events"**
3. คลิก **"Rules"** ที่เมนูด้านซ้าย → จะเห็น 3 rules:
   - `mission-status-changed-rule` — จับ event เมื่อสถานะภารกิจเปลี่ยน
   - `backup-requested-rule` — จับ event เมื่อขอ Backup
   - `impact-level-updated-rule` — จับ event เมื่ออัปเดต Impact Level

### 13.6 ดู CloudWatch Logs (Events ที่ถูก Publish)

1. ที่ช่องค้นหาด้านบน พิมพ์ **"CloudWatch"** แล้วคลิก **"CloudWatch"**
2. ที่เมนูด้านซ้าย คลิก **"Logs"** → **"Log groups"**
3. จะเห็น Log Groups:
   - `/aws/events/mission-progress/mission-status-changed`
   - `/aws/events/mission-progress/backup-requested`
   - `/aws/events/mission-progress/impact-level-updated`
   - `/aws/lambda/mission-progress-report-progress`
   - `/aws/lambda/mission-progress-get-mission`
   - ฯลฯ
4. คลิกที่ Log Group → คลิก **Log stream** ล่าสุด → ดูรายละเอียด event/log

> 💡 **Tip**: หลังจากทดสอบ POST report-progress แล้ว ให้มาดูที่ CloudWatch Logs ของ Event จะเห็น event ที่ถูก publish เช่น `MissionStatusChanged`

### 13.7 ดู S3 Buckets

1. ที่ช่องค้นหาด้านบน พิมพ์ **"S3"** แล้วคลิก **"S3"**
2. จะเห็น 2 buckets:
   - `mission-progress-frontend-XXXX` — เก็บไฟล์ Frontend (HTML, JS, CSS)
   - `mission-progress-evidence-XXXX` — เก็บหลักฐานภาพจากหน้างาน
3. คลิก **"mission-progress-frontend-XXXX"** → คลิกแท็บ **"Properties"** → เลื่อนลงจะเห็น **"Static website hosting"** เปิดอยู่พร้อม URL

---

## 14. ขั้นตอนที่ 12 — Cleanup (ลบทรัพยากรทั้งหมด)

> ⚠️ **สำคัญมาก**: เมื่อทดสอบเสร็จแล้ว **ต้องลบทรัพยากรทั้งหมด** เพื่อไม่ให้เสีย Budget ของ Learner Lab

### 14.1 ตรวจสอบ AWS Credentials ก่อน Destroy

```bash
aws sts get-caller-identity
```

✅ ต้องได้ผลลัพธ์เป็น JSON (ไม่ใช่ error)

> ⚠️ ถ้าได้ error → กลับไปขั้นตอน 3.4 เพื่อดึง Credentials ใหม่ เพราะ Terraform Destroy ต้องใช้ AWS Credentials ที่ valid

### 14.2 รัน Destroy Script

```bash
bash script/destroy.sh
```

Script นี้จะทำสิ่งต่อไปนี้:

1. ลบไฟล์ทั้งหมดใน S3 Buckets (Frontend + Evidence) — ต้องลบก่อนจึงจะลบ Bucket ได้
2. รัน `terraform destroy -auto-approve` เพื่อลบทรัพยากรทั้งหมด

✅ **ผลลัพธ์ที่คาดหวัง**:

```
=== Destroying MissionProgress Service ===
--- Emptying S3 buckets ---
delete: s3://mission-progress-frontend-XXXX/index.html
delete: s3://mission-progress-frontend-XXXX/...
--- Terraform destroy ---
...
Destroy complete! Resources: XX destroyed.

=== Destroy complete ===
```

### 14.3 ตรวจสอบว่าลบสำเร็จ

```bash
cd terraform
terraform state list
```

✅ **ผลลัพธ์ที่คาดหวัง**: ไม่แสดงรายการอะไรเลย (หมายถึงไม่มี resource เหลือ)

### วิธี Destroy ด้วยมือ (ถ้า Script ไม่ทำงาน)

ถ้า destroy script มีปัญหา สามารถทำเองได้:

```bash
cd terraform

# 1. ลบไฟล์ใน S3 Buckets
aws s3 rm "s3://$(terraform output -raw frontend_bucket)" --recursive
aws s3 rm "s3://$(terraform output -raw evidence_bucket)" --recursive

# 2. Terraform Destroy
terraform destroy -auto-approve
```

### 14.4 ตรวจสอบบน AWS Console

1. กลับไปที่ **AWS Console** (คลิก "AWS" 🟢 ที่ Learner Lab)
2. ตรวจสอบแต่ละ service ว่าไม่มีทรัพยากรเหลือ:
   - **DynamoDB** → Tables → ไม่มี MissionAssignment, MissionTimeline, EventOutbox
   - **Lambda** → Functions → ไม่มี mission-progress-\*
   - **API Gateway** → APIs → ไม่มี mission-progress-api
   - **S3** → Buckets → ไม่มี mission-progress-\*
   - **EventBridge** → Event buses → ไม่มี mission-progress-events

### 14.5 หยุด Lab Session

1. กลับไปที่หน้า **AWS Academy Learner Lab**
2. คลิกปุ่ม **"End Lab"** (ปุ่มสีแดง) ด้านบน
3. จะมี popup ถามยืนยัน → คลิก **"Yes"**
4. วงกลมจะเปลี่ยนเป็น 🔴 **สีแดง** — หมายถึง Lab ถูกหยุดแล้ว

> 💡 **หมายเหตุ**: แม้จะไม่กด End Lab ระบบจะหยุดอัตโนมัติเมื่อหมดเวลา session แต่แนะนำให้กด End Lab ด้วยตนเองเพื่อหยุดการคิด budget ทันที

---

## 15. Appendix: State Machine & API Reference

### State Machine (การเปลี่ยนสถานะที่อนุญาต)

```
DISPATCHED ──→ EN_ROUTE ──→ ON_SITE ──→ RESOLVED
                                │
                                ▼
                          NEED_BACKUP ──→ RESOLVED
                                │
                                └──→ ON_SITE
```

| สถานะเดิม (From) | สถานะใหม่ที่เปลี่ยนได้ (To) |
| :--------------- | :-------------------------- |
| DISPATCHED       | EN_ROUTE                    |
| EN_ROUTE         | ON_SITE                     |
| ON_SITE          | NEED_BACKUP, RESOLVED       |
| NEED_BACKUP      | ON_SITE, RESOLVED           |
| RESOLVED         | ❌ (สถานะสุดท้าย)           |

### API Endpoints

| Method | Path                                     | คำอธิบาย                    |
| ------ | ---------------------------------------- | --------------------------- |
| GET    | `/incidents`                             | รายการภารกิจทั้งหมด         |
| GET    | `/incidents?status=ON_SITE`              | กรองภารกิจตามสถานะ          |
| GET    | `/incidents/{incident_id}`               | รายละเอียดภารกิจ + Timeline |
| POST   | `/incidents/{incident_id}/progress`      | อัปเดตสถานะภารกิจ           |
| POST   | `/incidents/{incident_id}/presigned-url` | ขอ URL สำหรับอัปโหลดภาพ     |

### Headers ที่ต้องส่งทุก Request

| Header             | ค่า                             | จำเป็น |
| ------------------ | ------------------------------- | ------ |
| `x-api-key`        | API Key จาก Terraform output    | ✅     |
| `X-Rescue-Team-ID` | ชื่อทีม เช่น `TEAM-ALPHA`       | ✅     |
| `Content-Type`     | `application/json` (เฉพาะ POST) | ✅\*   |

### POST /incidents/{incident_id}/progress — Request Body

```json
{
  "new_status": "EN_ROUTE",
  "note": "หมายเหตุเพิ่มเติม",
  "new_impact_level": 3,
  "current_location": "13.75,100.50",
  "image_key": "evidence/INC-001/TEAM-ALPHA/1718353500-photo.jpg"
}
```

| ฟิลด์              | จำเป็น | คำอธิบาย                                              |
| ------------------ | ------ | ----------------------------------------------------- |
| `new_status`       | ✅     | สถานะใหม่ (ตาม State Machine)                         |
| `note`             | ❌     | หมายเหตุเพิ่มเติม                                     |
| `current_location` | ❌     | พิกัด GPS                                             |
| `new_impact_level` | ❌     | ระดับผลกระทบ (1-4) → trigger ImpactLevelUpdated event |
| `image_key`        | ❌     | S3 key ภาพหลักฐาน (ได้จาก presigned-url endpoint)     |

### POST /incidents/{incident_id}/presigned-url — Request Body

```json
{
  "file_name": "flood-evidence.jpg",
  "content_type": "image/jpeg"
}
```

| ฟิลด์          | จำเป็น | คำอธิบาย                                           |
| -------------- | ------ | -------------------------------------------------- |
| `file_name`    | ✅     | ชื่อไฟล์ภาพ                                        |
| `content_type` | ✅     | MIME type: `image/jpeg`, `image/png`, `image/webp` |

### Error Codes ทั้งหมด

| HTTP Status | Error Code                 | สาเหตุ                                     | Endpoints ที่เกี่ยว                            |
| ----------- | -------------------------- | ------------------------------------------ | ---------------------------------------------- |
| 400         | `MISSING_PARAMETER`        | ไม่ส่ง parameter ที่จำเป็น                 | ทั้งหมด                                        |
| 400         | `INVALID_BODY`             | JSON body ไม่ถูกต้อง                       | POST progress, POST presigned-url              |
| 400         | `INVALID_STATUS`           | new_status / status filter ไม่ถูกต้อง      | POST progress, GET /incidents                  |
| 400         | `INVALID_STATE_TRANSITION` | เปลี่ยนสถานะไม่ตรงกฎ State Machine         | POST progress                                  |
| 400         | `INVALID_CONTENT_TYPE`     | content_type ไม่รองรับ (ต้องเป็น image/\*) | POST presigned-url                             |
| 403         | —                          | ไม่ส่ง x-api-key หรือ X-Rescue-Team-ID     | ทั้งหมด                                        |
| 404         | `INCIDENT_NOT_FOUND`       | ไม่พบภารกิจสำหรับ incident_id ที่ระบุ      | GET mission, POST progress, POST presigned-url |
| 500         | `INTERNAL_ERROR`           | เกิดข้อผิดพลาดภายในระบบ                    | ทั้งหมด                                        |
| 500         | `PRESIGN_FAILED`           | ไม่สามารถสร้าง presigned URL ได้           | POST presigned-url                             |

### CORS Headers

| Header                         | ค่า                                       |
| ------------------------------ | ----------------------------------------- |
| `Access-Control-Allow-Origin`  | `*`                                       |
| `Access-Control-Allow-Methods` | `GET,POST,OPTIONS`                        |
| `Access-Control-Allow-Headers` | `Content-Type,x-api-key,X-Rescue-Team-ID` |

---

## 16. Troubleshooting (แก้ปัญหาที่พบบ่อย)

### ❌ `aws sts get-caller-identity` → ExpiredTokenException

**สาเหตุ**: AWS Credentials หมดอายุ (Lab session หมดเวลา)

**วิธีแก้**:

1. กลับไปที่ AWS Academy Learner Lab
2. ถ้าวงกลมเป็น 🔴 สีแดง → คลิก **"Start Lab"** ใหม่ รอจนเป็น 🟢 สีเขียว
3. คลิก **"AWS Details"** → คลิก **"Show"** ข้าง AWS CLI
4. คัดลอก Credentials ใหม่ วางทับในไฟล์ `~/.aws/credentials`
5. ทดสอบอีกครั้ง

### ❌ `terraform apply` → Error: error configuring Terraform AWS Provider

**สาเหตุ**: AWS Credentials ไม่ถูกต้อง หรือไม่มีไฟล์ credentials

**วิธีแก้**: ทำขั้นตอน 4 (ตั้งค่า AWS CLI) ใหม่

### ❌ `terraform apply` → Error acquiring the state lock

**สาเหตุ**: มีคน (หรือ process อื่น) กำลังรัน Terraform อยู่

**วิธีแก้**:

```bash
# ปลดล็อก (ใช้ Lock ID จาก error message)
terraform force-unlock <LOCK_ID>
```

### ❌ `bash script/build.sh` → go: command not found

**สาเหตุ**: ไม่ได้ติดตั้ง Go หรือ PATH ไม่ถูกต้อง

**วิธีแก้**:

```bash
# ตรวจสอบว่า Go ติดตั้งแล้ว
which go

# ถ้าไม่พบ → ติดตั้ง Go ตามขั้นตอน 2.2
```

### ❌ `bash script/build.sh` → npm: command not found

**สาเหตุ**: ไม่ได้ติดตั้ง Node.js/npm

**วิธีแก้**:

```bash
# ตรวจสอบ
which node && which npm

# ถ้าไม่พบ → ติดตั้ง Node.js ตามขั้นตอน 2.2
```

### ❌ `terraform apply` → Error: creating S3 Bucket: BucketAlreadyExists

**สาเหตุ**: Bucket ชื่อนี้มีคนอื่นใช้อยู่แล้วใน AWS (ชื่อ S3 bucket ต้อง unique ทั่วโลก)

**วิธีแก้**: แก้ไข `terraform/variables.tf` เปลี่ยนค่า `project_name` ให้ unique มากขึ้น

### ❌ curl ได้ `{"message": "Unauthorized"}`

**สาเหตุ**: ส่ง Header `x-api-key` หรือ `X-Rescue-Team-ID` ไม่ถูกต้อง หรือไม่ได้ส่ง

**วิธีแก้**:

- ตรวจสอบว่าส่ง header ทั้ง 2 ตัว
- ตรวจสอบค่า API Key ว่าตรงกับ `terraform output -raw api_key_value`

### ❌ curl ได้ `{"message": "Internal server error"}`

**สาเหตุ**: Lambda function มี error

**วิธีแก้**:

1. ไปที่ AWS Console → Lambda → คลิก function ที่เกี่ยวข้อง
2. ไปที่แท็บ **"Monitor"** → คลิก **"View CloudWatch logs"**
3. คลิก Log stream ล่าสุด → ดู error message

### ❌ Frontend เปิดไม่ได้ (404 หรือ AccessDenied)

**สาเหตุ**: ไม่ได้ upload ไฟล์ Frontend ขึ้น S3 หรือ URL ผิด

**วิธีแก้**:

```bash
# ตรวจสอบว่ามีไฟล์ใน S3 หรือไม่
aws s3 ls "s3://$(cd terraform && terraform output -raw frontend_bucket)/"

# ถ้าว่าง → Upload ใหม่
cd terraform
FRONTEND_BUCKET=$(terraform output -raw frontend_bucket)
aws s3 sync "../src/frontend/out/" "s3://$FRONTEND_BUCKET/" --delete
cd ..
```

### ❌ `terraform destroy` ค้าง หรือล้มเหลว

**สาเหตุ**: S3 Bucket ยังมีไฟล์อยู่ (ลบ Bucket ที่มีไฟล์ไม่ได้)

**วิธีแก้**:

```bash
cd terraform
# ลบไฟล์ใน bucket ก่อน
aws s3 rm "s3://$(terraform output -raw frontend_bucket)" --recursive
aws s3 rm "s3://$(terraform output -raw evidence_bucket)" --recursive
# แล้วรัน destroy ใหม่
terraform destroy -auto-approve
```

### ❌ `bash script/seed-data.sh` → ResourceNotFoundException

**สาเหตุ**: ยังไม่ได้ Deploy (DynamoDB tables ยังไม่ถูกสร้าง)

**วิธีแก้**: ทำขั้นตอน 8 (Deploy) ก่อน แล้วค่อยรัน seed-data

---

> 📝 **สรุปลำดับขั้นตอนทั้งหมด**:
>
> 1. เตรียมเครื่องมือ (Go, Node.js, Terraform, AWS CLI)
> 2. Start AWS Learner Lab + ดึง Credentials
> 3. ตั้งค่า AWS CLI (`~/.aws/credentials`)
> 4. Clone โปรเจกต์
> 5. ติดตั้ง Dependencies (Go + Node.js)
> 6. Build (`bash script/build.sh`)
> 7. Deploy (`terraform apply` + upload frontend) หรือ (`bash script/deploy.sh`)
> 8. Seed ข้อมูลตัวอย่าง (`bash script/seed-data.sh`)
> 9. ทดสอบ API ด้วย curl / Postman
> 10. ใช้งาน Frontend (Web UI)
> 11. ตรวจสอบทรัพยากรบน AWS Console
> 12. **Cleanup** (`bash script/destroy.sh` + End Lab)

-
