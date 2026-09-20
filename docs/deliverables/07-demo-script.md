# 7. Demo Script — สคริปต์สาธิตระบบ (ซ้อมจริงแล้ว)

สคริปต์สาธิตสำหรับ Hackathon presentation (deliverable ประกอบ [Presentation Slides](06-presentation-slides.md) ช่วง Live demo 2:50 นาที) — เขียนตาม issue #41 ให้ทุกขั้นมี **expected result** ที่จับตาได้จริง และมีแผน **recovery/fallback** เมื่อ integration มีปัญหากลางการสาธิต

ทุก expected result ในเอกสารนี้ **ยืนยันจากการซ้อมจริงบน docker compose stack วันที่ 2026-09-20** (ผ่านทั้ง UI และ API) และ happy path ทั้งเส้นถูก pin ไว้ด้วย E2E test ระดับ API ที่ [`apps/api/internal/e2e`](../../apps/api/internal/e2e/happy_path_test.go) ซึ่งรันใน CI ทุกครั้ง

## เรื่องที่เล่า (demo story)

ผู้ป่วย **สมหญิง รักษ์ดี (VISIT-002)** เดินทางมาคลินิกอายุรกรรม แพทย์สั่งเอกซเรย์ปอด ต้องกลับไปพบแพทย์ซ้ำหลังอ่านผล แล้วจบด้วยรับยา + ชำระเงิน — ครบทุกชนิดขั้นตอนของ Care Graph (ลงทะเบียน → พบแพทย์ → ตรวจพิเศษ → รอบสอง → เงิน/ยา) ข้อมูล seed นี้มากับ Mock HIS ทุกครั้งที่ reset (issue #39)

ผู้ชมจะเห็น 4 สิ่ง: **journey timeline บนมือถือ** · **หน้าจออัปเดตสด ๆ โดยไม่ต้อง refresh** · **เส้นทางในผังอาคารจากตำแหน่งปัจจุบัน** · **ฝั่งเจ้าหน้าที่คุมขั้นตอนได้**

บทบาท: **driver** (คนขับ/คลิก) + **narrator** (คนเล่า) — ตามที่แบ่งใน [Slides §Live demo](06-presentation-slides.md)

## Checklist เตรียม environment ก่อน demo

ทำเป็นลิสต์เช็ก 10 นาทีก่อนขึ้นเวที:

1. **Reset ข้อมูลให้เป็น seed จริงทุกครั้ง** (สำคัญสุด — ซ้อมมาทั้งวันจะเต็มไปด้วย state ค้าง):
   ```bash
   docker compose down -v
   POSTGRES_PORT=5433 docker compose up -d --build   # ถ้าเครื่องไม่มี postgres ท้องถิ่นแย่ง 5432 ตัด POSTGRES_PORT=5433 ออก
   ```
   ใช้เวลา ~1 นาที — เริ่มทำก่อนแม่สอยเสื้อ
2. **รอพร้อมแล้วยืนยัน seed**:
   ```bash
   curl -s localhost:8080/health          # {"status":"ok","service":"carepath-api"}
   curl -s localhost:8090/api/v1/demo/visits   # ต้องเห็น VISIT-001 และ VISIT-002 (ORD-002:PLACED)
   ```
3. **แท็บที่ต้องเปิดค้างไว้ก่อนเริ่ม** (เตรียมให้ครบ อย่าค้นหา URL กลางสาธิต):
   - **แท็บ A — มือถือผู้ป่วย** (เปิด DevTools responsive หรือจอมือถือจริง): `http://localhost:5173/patient/journey?visit=VISIT-002`
   - **แท็บ B — staff console**: `http://localhost:5173/login` → เข้าสู่ระบบ `staff / demo` (หรือ `admin / demo`) → ไปที่ **ผู้ป่วย** (`/staff/patients`)
   - **แท็บ C — Mock HIS console**: `http://localhost:8090/console`
   - (สำรอง) terminal พร้อมคำสั่ง curl ของ Beat 3 วางไว้
4. **ล็อกอิน staff ให้เสร็จก่อนเริ่ม** — ถ้า stack ถูก rebuild ระหว่างวัน token เดิมตาย (JWT_SECRET สุ่มใหม่ทุก boot เมื่อไม่ได้ตั้ง) ต้องล็อกอินใหม่ หรือตั้ง `JWT_SECRET=...` ใน `.env` ก่อน `up` เพื่อให้ token อยู่รอดข้าม rebuild
5. **อัดวิดีโอ backup** ของ sequence นี้ทั้งเส้นไว้ในเครื่อง (brief §12 กำหนด) — ถ้าเน็ตหลุดให้เล่นวิดีโอแทน

## บท demo step-by-step

### Beat 1 — เริ่มจาก LINE/entry (0:20)

เข้าแอปผู้ป่วยผ่าน **LINE LIFF** (กรณีตั้ง `LINE_CHANNEL_ID` + LIFF URL จริง — การเข้าผ่าน LINE ถูกเดินไว้แล้วใน issue #15/#16) หรือ **URL ตรง** สำหรับ demo ในเครื่อง: เปิดแท็บ A

Expected:
- หัวหน้า "การมาโรงพยาบาลของคุณวันนี้" + ชื่อบัญชี "เข้าสู่ระบบด้วยไลน์ · Demo User" (นอก LINE แอป fallback เป็น demo identity อัตโนมัติ)
- "ความคืบหน้า · เสร็จแล้ว 1 จาก 4 ขั้นตอน"
- Timeline: **ลงทะเบียน** (เสร็จสิ้นแล้ว) → **พบแพทย์ · อายุรกรรม** (ขั้นปัจจุบัน · OPD-NS-01) → **เอกซเรย์** → **ชำระเงิน**
- ปุ่ม CTA: "นำทางไปพบแพทย์ · อายุรกรรม"

### Beat 2 — เปิดหน้านำทาง (0:20)

กด CTA "นำทางไปพบแพทย์"

Expected: หน้า "เส้นทางไปพบแพทย์ · ชั้น 1" แสดงผังชั้น 1 (อาคาร I-13) มีจุดหมายปักอยู่ แต่**ยังไม่มีเส้นทาง** — ข้อความ "เส้นทางจะปรากฏเมื่อทราบตำแหน่งปัจจุบันของคุณ — สแกน QR ที่จุดบริการเพื่อเริ่มนำทาง" (จุดขายเรื่อง: ในอาคารไม่มี GPS เราจึงต้องรู้จุดยืนก่อน)

### Beat 3 — รายงานตำแหน่งปัจจุบัน (0:30)

จำลองผู้ป่วยเดินมาถึงโซนสาธาระของชั้น 1 ด้วย **Zigbee simulator** (terminal สำรอง):

```bash
curl -X POST localhost:8080/api/v1/demo/zigbee/location \
  -H 'Content-Type: application/json' \
  -d '{"visitId":"VISIT-002","floorId":"I-1301","zone":"PUBLIC","confidence":0.92}'
# → {"nodeId":"I-1301/node-cashier","zone":"PUBLIC","source":"ZIGBEE",...}
```

แล้ว reload แท็บ A (หน้านำทาง)

Expected:
- ป้าย **"คุณอยู่ที่นี่"** ปรากฏบนผัง + **เส้นทางสี** ลากจากจุดยืนไปจุดหมาย (จุดบริการพยาบาล OPD)
- แถบล่าง: "ตำแหน่งปัจจุบัน · ชั้น 1 · โซน PUBLIC · Zigbee"

ทางเลือกถ้า curl ไม่สะดวกบนเวที — **QR**: ที่จุดบริการมี QR เขียน place id (เช่น `CASHIER-01`) ยิงผ่าน contract endpoint ปกติ:
```bash
curl -X POST localhost:8080/api/v1/journeys/VISIT-002/location \
  -H 'Content-Type: application/json' -d '{"source":"QR","raw":"CASHIER-01"}'
```

### Beat 4 — ถึงคิวแล้ว หน้าจออัปเดตสด (0:40)

สลับไป**แท็บ B (staff console → ผู้ป่วย → VISIT-002)** กด **"เริ่มขั้นตอน"** ที่ พบแพทย์ · อายุรกรรม

สลับกลับ**แท็บ A โดยไม่ refresh** — narrator บอกผู้ชมว่า "รอ 15 วินาที"

Expected (ภายใน ~15 วินาที หน้าอัปเดตเอง):
- "ความคืบหน้า · เสร็จแล้ว 1 จาก **5** ขั้นตอน" (เพิ่มขั้น "รอผลตรวจ")
- CTA เปลี่ยนเป็น **"นำทางไปเอกซเรย์"** — Care Graph รู้แล้วว่าขั้นถัดไปคืออะไร และ destination เปลี่ยนตาม

หมายเหตุเทคนิค: staff สั่งผ่าน transition API (`POST /api/v1/journeys/VISIT-002/steps/CLINIC:MED:1/transition` — stepKey ต้องส่ง colon ตรง ๆ ห้าม encode เป็น `%3A`) การเปลี่ยนสถานะบันทึกลง audit trail ทุกครั้ง

### Beat 5 — Mock HIS: ตรวจเอกซเรย์เสร็จ → next destination เปลี่ยน (0:40)

สลับไป**แท็บ C (Mock HIS console)** หรือ terminal — นี่คือ "HIS แจ้งผลตรวจ":
```bash
curl -X POST localhost:8090/api/v1/demo/orders/ORD-002/performed -H 'Content-Type: application/json' -d '{}'
curl -X POST localhost:8090/api/v1/demo/orders/ORD-002/resulted  -H 'Content-Type: application/json' -d '{}'
```

กลับแท็บ A (ไม่ refresh)

Expected (~20 วินาที — ingest 5 วิ + poll 15 วิ):
- "ความคืบหน้า · เสร็จแล้ว **2** จาก 5 ขั้นตอน" · เอกซเรย์ติด "เสร็จสิ้นแล้ว"
- ขั้นใหม่ **"กลับไปพบแพทย์ · อายุรกรรม"** สถานะพร้อม
- CTA เปลี่ยนเป็น **"นำทางไปกลับไปพบแพทย์"** — ครบ AC "จบด้วย next destination หลัง service complete"

### Beat 6–7 — ปิดท้าย: รอบสอง ยา และเงิน (0:30 — ทำถ้าเวลาเหลือ)

Beat ตัดทอนได้ตามเวลาบนเวที (พฤติกรรมเดียวกับ Beat 4–5) ไล่ตามลำดับนี้เท่านั้น:

1. staff เริ่มขั้น "กลับไปพบแพทย์" (แท็บ B) → Mock HIS สั่งยา: `POST localhost:8090/api/v1/demo/visits/VISIT-002/orders` ด้วย body `{"orderType":"DRUG","orderName":"ยาหลังเอกซเรย์","orderedByClinic":"MED"}` → หน้า A เพิ่มขั้น **รับยา**
2. ปิด encounter: `POST localhost:8090/api/v1/demo/visits/VISIT-002/clinics/MED/complete-encounter` → staff กดเสร็จขั้น "รอผลตรวจ" → CTA กลายเป็น **นำทางไปชำระเงิน** (จุด CASHIER-01)
3. ชำระเงิน → รับยา → `POST localhost:8090/api/v1/demo/visits/VISIT-002/complete` → ทุกขั้นเสร็จสิ้น timeline จบสวย

> ลำดับสำคัญ: อย่า "เสร็จ" ขั้นพบแพทย์รอบแรกก่อนรอบสองจะเริ่ม — planner ถือว่า visit จบเฉพาะคลินิกแล้วตัดรอบกลับทิ้ง (semantics ตาม ADR-0009)

### Beat 8 — แชร์ความคืบหน้าให้ญาติ (0:30 — ทำได้ทุกจังหวะ ไม่ต้องรอท้าย)

1. แท็บ A (ผู้ป่วย): กดปุ่มรอง **"แชร์ความคืบหน้าให้ญาติ"** ใต้ timeline → sheet เด้งขึ้นพร้อมลิงก์ `/shared#…` และเวลาหมดอายุ ("ใช้ได้ถึง …") → กด **คัดลอกลิงก์**
2. เปิดแท็บ C (incognito = มือถือญาติ) วางลิงก์ → เห็นแค่ขั้นตอนปัจจุบัน + จุดบริการ + ชั้น **ไม่มีชื่อผู้ป่วย ไม่มีรหัสใด ๆ** — ชี้ให้เห็นว่าเป็นข้อมูลหยาบตาม ADR-0011
3. staff ปิดขั้น (แท็บ B) → ภายใน ~15 วินาที แท็บ C ขยับเอง — ญาติรู้ว่ามารับตอนไหน
4. ปิดด้วย "หยุดแชร์" บนแท็บ A → แท็บ C เปลี่ยนเป็น "ลิงก์นี้หมดอายุแล้ว — ขอลิงก์ใหม่จากผู้ป่วยได้เลย" ใน refetch ถัดไป

> เกร็ด: token อยู่ใน URL fragment (`#` หลัง `/shared`) — เบราว์เซอร์ไม่ส่ง fragment ไปที่ server ใด ๆ เลย จึงไม่หลุดเข้า log

## Recovery / fallback เมื่อมีปัญหากลาง demo

| อาการ | สาเหตุ | วิธีแก้บนเวที |
|---|---|---|
| แท็บ A ไม่อัปเดตหลังสั่ง HIS/staff | polling (HIS ingest 5 วิ + refetch 15 วิ) | รอจนครบ ~25 วินาที แล้ว reload หน้า 1 ครั้ง — ยังไม่ขยับค่อยไปดู row ถัดไป |
| ปุ่ม staff กดแล้วเด้ง "เปลี่ยนสถานะไม่สำเร็จ (401)" | token ตาย (stack rebuild ระหว่างวัน + ไม่ได้ตั้ง JWT_SECRET) | ล็อกอิน staff ใหม่ (`staff / demo`) แล้วกดซ้ำ — state ไม่หาย |
| ปุ่ม "แชร์ความคืบหน้า" กดไม่ได้ พร้อมข้อความ "ยังขอสิทธิ์แชร์ไม่สำเร็จ" | หน้าผู้ป่วยขอ session ไม่สำเร็จ (API ล่มชั่วคราว หรือ `ALLOW_DEMO_AUTH=false`) | reload หน้าผู้ป่วย 1 ครั้ง — หน้า journey ใช้ได้ตามปกติตลอด แค่ปุ่มแชร์รอ session กลับมา |
| เส้นทาง/จุดยืนไม่ขึ้นบนผัง | ยังไม่มี location observation | ยิง curl Zigbee ใหม่ (Beat 3) → reload · ตรวจด้วย `GET localhost:8080/api/v1/journeys/VISIT-002/location` |
| สั่ง Mock HIS แล้ว journey ไม่ขยับเลย | restart Mock HIS **ไม่ใช่** reset — event id ถูก dedupe ไปแล้ว | ใช้ console สร้าง order/event ใหม่ หรือ reset จริง (ดู row ล่าง) |
| ทุกอย่างพัง / อยากเริ่มใหม่ | — | `docker compose down -v && POSTGRES_PORT=5433 docker compose up -d` (~1 นาที) คืน seed ต้นฉบับ — รายละเอียดหัวข้อ "Reset ข้อมูล demo" ใน [README](../../README.md) |
| API ตาย/เน็ตหลุด | สภาพเวที | เล่นวิดีโอ backup (เตรียมไว้ตาม brief §12) — API จริง error เป็น envelope สวย ๆ ไม่ใช่ stack trace (NFR-10) |

## ของจริงกับของจำลอง — บอกผู้ชมตามจริง

- **Zigbee คือ simulator endpoint** (`/api/v1/demo/zigbee/location`) จำลอง positioning service — hardware จริงเป็น phase 2 ตาม ADR-0004 โดยผ่าน provider interface เดียวกัน
- **LINE คือ LIFF จริง** เมื่อตั้งค่า channel — ในเครื่อง demo ใช้ demo identity (แสดงเป็น "Demo User") เพราะเปิดในเบราว์เซอร์ปกติ
- **Mock HIS แสดงตัวตนชัด ๆ** ที่ :8090/console — สลับเป็น HIS จริงทีหลังผ่าน adapter เดิม ไม่แก้ business logic (ADR-0005)
- happy path ทั้งเส้นมี E2E test คุ้มอยู่ใน CI (`apps/api/internal/e2e`) — สิ่งที่สาธิตคือสิ่งที่ test ผ่านทุกวัน

## เอกสารเกี่ยวข้อง

- [Presentation Slides](06-presentation-slides.md) — ตำแหน่ง demo ในลำดับการนำเสนอ
- [README §ตัวอย่างการใช้งาน / Reset ข้อมูล demo](../../README.md) — seed ทั้งสอง visit และวิธี reset
- [Mock HIS](../integration/mock-his.md) — รายการ demo API ทั้งหมด
- [QR and Zigbee Location](../integration/location-zigbee.md) — location providers
- [Test Result](05-test-result.md) — TC ที่ครอบ flow นี้
