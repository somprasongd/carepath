# 6. Presentation Slides — Outline (สไลด์และการนำเสนอ)

**Deliverable 6 of 6** per the hackathon brief §10: present 7 minutes per team (live demo included) + 3 minutes Q&A. This is a markdown outline for the team to turn into actual slides — content is pulled from the other deliverables, not invented fresh, so the story stays consistent end to end.

Team: **OST-1** — นายสมประสงค์ ดำยศ, นายปฐมพงศ์ พรนราดล. Presenter assignments below are a starting split between the two of you — swap freely to match who actually built/knows each part best.

## Timing budget (7:00 total)

| # | Slide | Time | Cumulative |
|---|---|---|---|
| 1 | Title | 0:10 | 0:10 |
| 2 | Problem | 0:45 | 0:55 |
| 3 | Solution — what CarePath does | 0:30 | 1:25 |
| 4 | Who it's for & scope (Must/Should/Could) | 0:30 | 1:55 |
| 5 | Architecture & key design decisions | 0:45 | 2:40 |
| — | **Live demo** | 2:50 | 5:30 |
| 6 | Data model highlights | 0:20 | 5:50 |
| 7 | Quality — testing & a real bug we caught | 0:30 | 6:20 |
| 8 | Status & what's next | 0:30 | 6:50 |
| 9 | Team & thank you | 0:10 | 7:00 |

Rehearse with a timer — 7:00 is a hard stop per the brief, and criterion 4 (20 pts) explicitly scores "demos smoothly within 7 minutes."

## Slide-by-slide outline

### 1. Title (0:10) — presenter: สมประสงค์

- **CarePath** — ระบบนำทางผู้ป่วยข้ามอาคารและติดตามขั้นตอนการรักษาในโรงพยาบาล
- ทีม **OST-1** · นายสมประสงค์ ดำยศ · นายปฐมพงศ์ พรนราดล
- โจทย์ Hackathon ที่ 1 — วิทยาลัยการคอมพิวเตอร์ ม.สงขลานครินทร์ ภูเก็ต

### 2. Problem (0:45) — presenter: ปฐมพงศ์

On-slide (keep to 3 bullets, say the rest):
- ผู้ป่วยหลงทางข้ามอาคาร/ชั้น โดยเฉพาะผู้สูงอายุ ผู้ใช้รถเข็น และผู้มาครั้งแรก
- ไม่รู้ว่าเหลืออีกกี่ขั้นตอน ต้องรออีกนานแค่ไหน ขั้นตอนถัดไปอยู่ที่ใด
- เจ้าหน้าที่เสียเวลาตอบคำถามเรื่องเส้นทางแทนที่จะดูแลผู้ป่วย

Say but don't put on slide: the two-part problem statement — (1) what's next, in what order, how many steps left, and (2) how to physically walk there (no GPS indoors).

### 3. Solution (0:30) — presenter: สมประสงค์

- CarePath = "ผู้ช่วยนำทางส่วนตัว" ตลอดการมาโรงพยาบาลหนึ่งครั้ง
- แยกสองเรื่องออกจากกันโดยเจตนา: **Care Graph** (ขั้นตอนถัดไปคืออะไร) กับ **Navigation Graph** (จะไปถึงอย่างไร) เชื่อมกันผ่าน ServicePoint → Place
- ระบุตำแหน่งไม่ใช้ GPS: สแกน QR เป็นหลัก, เลือกจากรายการเป็นทางเลือกสำรอง

### 4. Who it's for & scope (0:30) — presenter: ปฐมพงศ์

- 6 actors: ผู้ป่วย/ญาติ, เจ้าหน้าที่เวชระเบียน, เจ้าหน้าที่ประจำจุดบริการ, ผู้ดูแลระบบ, ผู้บริหาร — full detail: [Requirement Specification §1.2](01-requirement-specification.md#12-actors)
- Must/Should/Could scope: show the MoSCoW table from [Requirement Specification §1.3](01-requirement-specification.md#13-scope-moscow-brief-5) — say clearly which of M1–M8 are demoed live today vs. designed-only (see slide 8)

### 5. Architecture & key design decisions (0:45) — presenter: สมประสงค์

- One diagram: [System Architecture](../architecture/system-architecture.md) flowchart (Channels → API → Core modules → Data / HIS)
- 3 decisions worth 15 seconds each, because they're the ones a judge will probe:
  1. **CarePath never reads the HIS database directly** — Mock HIS today, a real HIS adapter later, without touching journey/navigation code ([ADR-0005](../adr/0005-his-adapter-and-mock-his.md))
  2. **Care Graph and Navigation Graph are separate models** — a clinical workflow change and a room move never touch the same code ([ADR-0002](../adr/0002-separate-care-and-navigation-graphs.md))
  3. **Location is provider-abstracted** — QR today, Zigbee later, same interface ([ADR-0004](../adr/0004-location-provider-abstraction.md))

### Live demo (2:50) — driver: ปฐมพงศ์, narrator: สมประสงค์

Script (each step is something already verified working — see [Test Result](05-test-result.md) TC-01, TC-02, TC-12, TC-13):

1. **(0:30)** Open the web app on `VISIT-001` — point at the journey timeline: registration ✓, screening ✓, doctor ✓, **now: blood draw at Laboratory·LAB-01**, next: pharmacy.
2. **(0:30)** Call the underlying API live in a second tab/terminal — `GET /api/v1/visits/VISIT-001/next` — show the judges the raw JSON resolving to a real `ServicePoint`, to prove it's computed, not hardcoded on the frontend.
3. **(0:40)** Trigger the "not yet built" path on purpose: tap "navigate to blood draw." Show the clear fallback message instead of a crash — this demonstrates NFR-10 (no raw errors) and is honest about current scope rather than hiding it.
4. **(0:40)** Kill Mock HIS live (or show a pre-recorded clip of it) to show the 502 handling: API stays up, client gets `"upstream service unavailable"`, not a stack trace — this is the real bug-fix story from testing (see slide 7).
5. **(0:30)** Show the ER diagram / SQL script briefly on screen (don't read it line by line) — say "this is the schema for the full Must-Have scope; today's demo exercises the visit and service-point tables live."

Keep a **local recording of this exact sequence** as a backup (brief §12 explicitly asks for one) — narrate over it if the venue's network drops.

### 6. Data model highlights (0:20) — presenter: ปฐมพงศ์

- One diagram only (pick the busiest-looking one for visual impact — Group A, hospital map, from [ER Diagram §2.3](02-er-diagram-database.md#23-er-diagrams))
- One line: "two schemas, `his` and `carepath` — the HIS boundary from slide 5 is enforced at the database level too, not just in code"

### 7. Quality — testing & a real bug we caught (0:30) — presenter: สมประสงค์

- 46 automated tests passing (Go + Vitest) across all three services
- Tell the TC-05 story in one sentence: "while writing our test cases, we found the API was leaking a raw connection error to the client when the HIS was down — we fixed it and added a regression test the same day." (Full writeup: [Test Result TC-05](05-test-result.md#test-cases).) This is a stronger signal to judges than an all-green table with no story behind it.

### 8. Status & what's next (0:30) — presenter: ปฐมพงศ์

Be honest here — criterion 3 rewards working Must-Have over broken breadth:

- **Working today**: visit/journey display, next-step resolution, service-point mapping, HIS adapter boundary, error handling — all live-demoed above
- **Designed, not yet built**: registration + pathway templates (M2/M3), staff queue console (M7), login/RBAC (M8) — schema, screens, and for M8 the full contract and decision record are designed ([ER Diagram](02-er-diagram-database.md), [Prototype](03-prototype-wireframe.md#33-known-gap--screens-not-yet-designed), [ADR-0010](../adr/0010-staff-auth-jwt-argon2.md): argon2id passwords, JWT access + rotating refresh tokens, `STAFF`/`ADMIN` roles), implementation is next
- One sentence on why: "we prioritized getting the core loop — what's next, how do I get there — fully correct and tested, over partial coverage of everything."

### 9. Team & thank you (0:10) — presenter: สมประสงค์ & ปฐมพงศ์

- นายสมประสงค์ ดำยศ — `[what they built]`
- นายปฐมพงศ์ พรนราดล — `[what they built]`
- ขอบคุณครับ — พร้อมตอบคำถาม

## Anticipated Q&A (3:00)

Split questions between the two of you so both get airtime (criterion 4: "สมาชิกทุกคนมีส่วนร่วม").

| Likely question | Suggested answer | Who |
|---|---|---|
| "ทำไมยังไม่มีหน้าจอลงทะเบียนผู้ป่วย/คิว?" (Why no registration/queue screen yet?) | We prioritized the navigation core loop first; the schema and screens for M2/M3/M7 are fully designed ([ER Diagram](02-er-diagram-database.md), [Prototype gap list](03-prototype-wireframe.md#33-known-gap--screens-not-yet-designed)) — it's a build-order choice, not an oversight. | สมประสงค์ |
| "ถ้าเปลี่ยนเป็น HIS จริงต้องแก้โค้ดตรงไหน?" (What changes if you swap in a real HIS?) | Only the adapter in `internal/his` — journey and navigation code depend on a stable internal contract, never the vendor's API directly (ADR-0005, ADR-0008). | ปฐมพงศ์ |
| "เส้นทางเดินคำนวณจากอะไร ไม่ใช่ฝังตายตัวใช่ไหม?" (Is the route hardcoded?) | No — it's Dijkstra over a `nav_node`/`nav_edge` graph in Postgres ([ER Diagram §2.4](02-er-diagram-database.md#24-design-questions-the-brief-raises-7)); the navigation module itself isn't wired to the UI yet, which the demo showed honestly with the fallback message. | สมประสงค์ |
| "ข้อมูลผู้ป่วยปลอดภัยแค่ไหน?" (How secure is patient data?) | Passwords are one-way hashed with argon2id and never stored or logged in plaintext; staff sessions use a 15-minute signed access token plus a revocable, single-use refresh token, so a leaked credential has a short and cuttable life ([ADR-0010](../adr/0010-staff-auth-jwt-argon2.md)). Queries are parameterized (verified — see [Test Result TC-06](05-test-result.md#test-cases)), and PHI stays off public-facing screens like the queue-call display (NFR-03, NFR-11). | ปฐมพงศ์ |
| "ทำไม demo ถึง login ด้วย admin/demo ได้?" (Why does the demo log in with admin/demo?) | Those are seed accounts for the hackathon demo, and their hash is in a public repository by design. The pre-deployment checklist in ADR-0010 §12 removes them, sets a real `JWT_SECRET` from a secret store, and requires TLS before anyone outside the team can reach the system. | ปฐมพงศ์ |
| "ทดสอบระบบยังไงบ้าง?" (How did you test it?) | 46 automated tests plus manual test cases against the live system — [Test Result](05-test-result.md) documents all of them, including a real bug we found and fixed. | สมประสงค์ |
| "ทำไมใช้ Mock HIS แทน HIS จริง?" (Why Mock HIS instead of a real HIS?) | The brief explicitly allows/expects this (§3, §5.1 M9-ish, §8) — a hackathon can't get real hospital system access; Mock HIS implements the same contract a real adapter would (ADR-0005). | ปฐมพงศ์ |

## Backup plan (brief §12)

- Record the full live-demo script above as a screen capture **before** presentation day.
- Keep Mock HIS's fixed demo data (`VISIT-001`) — don't reset or reseed it right before presenting.
- If the network fails: narrate the recording instead of apologizing for it; the brief explicitly expects a backup plan, so having one is a plus, not a fallback to be embarrassed about.
