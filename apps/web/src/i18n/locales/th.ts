/**
 * Thai catalog — the canonical message set (ADR-0012). Every other locale is
 * typed against this shape, so a missing key in `en.ts` is a compile error
 * (and the parity test in `../i18n.test.ts` fails `npm run test` first).
 *
 * Keys are named after what they are for, grouped by feature prefix
 * (`step.*`, `sp.*`, `journey.*`, `navigate.*`, `auth.*`).
 */
export const th = {
  // Step titles — keyed by the step `kind` (ADR-0009 step identity).
  'step.title.REGISTRATION': 'ลงทะเบียน',
  'step.title.LAB': 'เจาะเลือด',
  'step.title.XRAY': 'เอกซเรย์',
  'step.title.EKG': 'ตรวจคลื่นไฟฟ้าหัวใจ',
  'step.title.ULTRASOUND': 'อัลตราซาวด์',
  'step.title.CASHIER': 'ชำระเงิน',
  'step.title.PHARMACY': 'รับยา',

  // CLINIC steps name their clinic by code; the name rides as a title suffix.
  'step.clinic.MED': 'อายุรกรรม',
  'step.clinic.SURG': 'ศัลยกรรม',
  'step.seeDoctor': 'พบแพทย์',
  'step.seeDoctorAgain': 'กลับไปพบแพทย์',

  // Rail meta lines under a step title (features/visit/journey.ts stepMeta).
  'step.meta.cancelled': 'ยกเลิกแล้ว',
  'step.meta.completed': 'เสร็จสิ้นแล้ว',
  'step.meta.waitingResult': 'รอผลตรวจ',
  'step.meta.ready': 'พร้อมให้บริการ',
  'step.meta.pending': 'รอดำเนินการ',

  // Service point names — keyed by ServicePoint.code (contract §ServicePoint);
  // unmapped codes fall back to the server's `name` (ADR-0012 §1).
  'sp.REGISTRATION': 'จุดลงทะเบียน',
  'sp.LAB': 'ห้องเจาะเลือด',
  'sp.ORDERTYPE:LAB': 'ห้องเจาะเลือด',
  'sp.XRAY': 'ห้องเอกซเรย์',
  'sp.ORDERTYPE:XRAY': 'ห้องเอกซเรย์',
  'sp.PHARMACY': 'จุดจ่ายยา',
  'sp.CASHIER': 'จุดชำระเงิน',
  'sp.CLINIC:MED': 'อายุรกรรม',
  'sp.DOCTOR': 'ห้องตรวจโรค',

  // Shared composition.
  'common.floor': 'ชั้น {code}',
  // The น. marker rides the catalog so the one clock formatter
  // (i18n/time.ts) never embeds Thai script itself.
  'common.clock': '{time} น.',

  // Language switch (i18n/LanguageToggle.tsx, #94). Each option names its
  // language in its own script — the Thai endonym stays 'ไทย' in every
  // locale, which is also why the thai-guard excludes only locales/th.ts.
  'common.language': 'ภาษา',
  'common.language.switch': 'สลับภาษา',
  'common.locale.th': 'ไทย',
  'common.locale.en': 'EN',

  // Large-text mode (preferences/LargeTextToggle.tsx, #99/FR-20) — the
  // accessible name of the A+ button; "A+" itself reads in every language.
  'prefs.largeText': 'ตัวหนังสือใหญ่',

  // LINE LIFF login gate (auth/LoginGate.tsx).
  'auth.checking': 'กำลังตรวจสอบสิทธิ์การเข้าใช้งาน…',
  'auth.redirecting': 'กำลังนำท่านไปยังหน้าเข้าสู่ระบบ LINE…',
  'auth.errorTitle': 'เข้าสู่ระบบไม่สำเร็จ',
  'auth.errorUnknown': 'เกิดข้อผิดพลาดที่ไม่ทราบสาเหตุ',
  'auth.retry': 'ลองอีกครั้ง',

  // VN front door (routes/patient/-VisitEntryScreen.tsx) and the slip-link
  // exchange / no-visit doors (routes/patient/-VisitLinkExchange.tsx).
  'entry.role': 'ผู้ป่วย',
  'entry.title': 'ค้นหาการนัดหมาย',
  'entry.lead': 'กรอกหมายเลขการรักษา (VN) จากสลิกของโรงพยาบาลเพื่อดูแผนการรักษาของคุณ',
  'entry.vnLabel': 'หมายเลขการรักษา (VN)',
  'entry.vnPlaceholder': 'เช่น VISIT-001',
  'entry.required': 'กรุณากรอกหมายเลขการรักษา',
  'entry.submit': 'ดูแผนการรักษา',
  'entry.exchanging': 'กำลังเปิดการรับบริการ…',
  'entry.exchangingLead': 'กรุณารอสักครู่ ระบบกำลังแสดงแผนการรักษาของท่าน',
  'entry.linkInvalidTitle': 'ลิงก์ไม่ถูกต้องหรือหมดอายุ',
  'entry.linkInvalidLead': 'กรุณาสแกน QR บนใบนำทางที่โรงพยาบาลออกให้อีกครั้ง',
  'entry.linkErrorTitle': 'เปิดการรับบริการไม่สำเร็จ',
  'entry.linkErrorLead': 'ระบบไม่ตอบสนอง กรุณาลองอีกครั้ง',
  'entry.noVisitTitle': 'เปิดแผนการรักษาจากใบนำทาง',
  'entry.noVisitLead': 'สแกน QR บนใบนำทางที่โรงพยาบาลออกให้ เพื่อดูแผนการรักษาของท่าน',

  // Patient journey home (routes/patient/-JourneyScreen.tsx).
  'journey.title': 'การมาโรงพยาบาลของคุณวันนี้',
  'journey.lead': 'ติดตามขั้นตอนของคุณ แล้วไปยังจุดบริการถัดไปได้จากปุ่มด้านล่าง',
  'journey.signedInWithLine': 'เข้าสู่ระบบด้วยไลน์ · {name}',
  'journey.loading': 'กำลังโหลดขั้นตอนของคุณ…',
  'journey.retry': 'ลองใหม่',
  'journey.error.notFound': 'ไม่พบข้อมูลการมาโรงพยาบาลของคุณ',
  'journey.error.hisUnavailable': 'ระบบข้อมูลของโรงพยาบาลไม่พร้อมใช้งาน',
  'journey.error.unreachable': 'เชื่อมต่อระบบไม่สำเร็จ',
  'journey.progress': 'ความคืบหน้า · เสร็จแล้ว {done} จาก {total} ขั้นตอน',
  'journey.nextStep': 'ขั้นตอนถัดไป',
  'journey.navigateCta': 'นำทางไป{title}',
  'journey.shareWithFamily': 'แชร์ความคืบหน้าให้ญาติ',
  'journey.shareDemoHint': 'โหมดสาธิต: ยังขอสิทธิ์แชร์ไม่สำเร็จ',
  'journey.notifyToggleLabel': 'แจ้งเตือนเมื่อใกล้ถึงคิว — หนึ่งครั้งต่อขั้นตอน ผ่าน LINE',
  'journey.notifyOn': 'เปิดแจ้งเตือน',
  'journey.notifyOff': 'ปิดแจ้งเตือน',
  'journey.recommendReason.nearest': 'แนะนำจุดที่ใกล้จากตำแหน่งล่าสุดของคุณ',
  'journey.recommendReason.planOrder':
    'แนะนำตามลำดับขั้นตอนของแผน — สแกน QR หน้าจุดบริการเพื่อให้ระบบเลือกจุดที่ใกล้ที่สุดจากตำแหน่งคุณ',

  // Next-step queue card (features/visit/components/QueueCard.tsx, FR-17).
  'queue.title': 'คิวขั้นตอนถัดไป',
  'queue.peopleAhead': 'มีผู้รออยู่ข้างหน้า {count} คน',
  'queue.youAreNext': 'คิวของคุณถึงแล้ว',
  'queue.estimate': 'เวลารอโดยประมาณ {minutes} นาที',
  'queue.noData': 'ยังไม่มีข้อมูลเวลารอของจุดบริการนี้',

  // End-of-visit summary card (features/visit/components/VisitOutcomeCard.tsx).
  'outcome.cancelled.label': 'การมาโรงพยาบาลนี้',
  'outcome.cancelled.title': 'ถูกยกเลิก',
  'outcome.cancelled.body': 'หากคุณมีข้อสงสัย โปรดติดต่อเจ้าหน้าที่ที่จุดบริการ',
  'outcome.completed.title': 'เสร็จสิ้นทุกขั้นตอนแล้ว',
  'outcome.completed.body': 'ขอบคุณที่ใช้บริการวันนี้ ขอให้คุณมีสุขภาพแข็งแรง',

  // Navigate screen chrome (routes/patient/-NavigateScreen.tsx).
  'navigate.back': 'ย้อนกลับไปหน้าเส้นทาง',
  'navigate.loadingTitle': 'กำลังโหลดจุดหมาย…',
  'navigate.noDestinationTitle': 'จุดบริการของคุณ',
  'navigate.pendingNotice': 'กำลังโหลดจุดหมายของคุณ…',
  'navigate.noDestinationNotice': 'ยังไม่มีจุดบริการถัดไปในการมาโรงพยาบาลครั้งนี้',
  'navigate.unsupportedNotice':
    'ระบบยังไม่รองรับเส้นทางในอาคารสำหรับจุดบริการนี้ — โปรดถามเจ้าหน้าที่ที่จุดรับลงทะเบียน',
  'navigate.waitingLocation':
    'เส้นทางจะปรากฏเมื่อทราบตำแหน่งปัจจุบันของคุณ — สแกน QR ที่จุดบริการเพื่อเริ่มนำทาง',
  'navigate.askStaffIfLost': 'แจ้งเจ้าหน้าที่หากหลงทาง',
  'navigate.scanCta': 'สแกน QR ที่จุดบริการ',
  'navigate.pickInstead': 'หรือเลือกจุดที่คุณอยู่เอง',
  'navigate.updateLocation': 'สแกน QR / เลือกจุด เพื่ออัปเดทตำแหน่ง',
  'navigate.avoidStairs': 'เลี่ยงบันได (ใช้ลิฟต์)',
  'navigate.voice.toggle': 'เสียงนำทาง',

  // Destination plan text (features/floorplan/destination.ts).
  'navigate.routeTitle': 'เส้นทางไป{name}',
  'navigate.subtitle': '{floor} · {name} · {place}',

  // Current-location line and turn cues (features/navigation/route.ts).
  'navigate.locationNowAt': 'ตำแหน่งปัจจุบัน · {floor}',
  'navigate.zone': 'โซน {zone}',
  'navigate.source.QR': 'สแกน QR',
  'navigate.source.ZIGBEE': 'Zigbee',
  'navigate.source.MANUAL': 'ระบุเอง',
  'navigate.cue.followLine': 'เดินตามเส้นสายส้มบนผัง',
  'navigate.cue.elevator': 'ใช้ลิฟต์ไป{floor}',
  'navigate.cue.stairs': 'ใช้บันไดไป{floor}',
  'navigate.cue.arrive': 'ถึง{name} — จุดหมายของคุณ',

  // Spoken guidance (#108, FR-25): เสียงอ่าน cue ชุดเดียวกับหน้าจอ ยกเว้น
  // ประโยคถึงจุดหมายที่ตัดชื่อจุดบริการออก — ข้อความที่พูดได้ต้องมาจาก
  // กลุ่มนี้ + ชื่อชั้นเท่านั้น (กฎความเป็นส่วนตัวใน features/navigation/speech.ts).
  'navigate.voice.intro': 'เส้นทางไปจุดหมายของคุณ',
  'navigate.cue.arriveUnnamed': 'ถึงจุดหมายของคุณแล้ว',

  // The assumed-origin fallback line (features/navigation/origin.ts).
  'navigate.assumedLocation': 'ตำแหน่งโดยประมาณ · หลังขั้นตอน{step}',

  // Floor plan map controls (design-system/FloorPlanMap.tsx labels).
  'map.viewFullFloor': 'ดูทั้งชั้น',
  'map.viewRoute': 'ดูเส้นทาง',
  'map.viewDestination': 'ดูจุดหมาย',
  'map.youAreHere': 'คุณอยู่ที่นี่',
  'map.planAria': 'ผัง{floor} — จุดหมาย {name}',
  'map.routeAria': 'ผัง{floor} — เส้นทางจากตำแหน่งปัจจุบันไป{name}',
  'map.zoomIn': 'ซูมเข้า',
  'map.zoomOut': 'ซูมออก',
  // The plan is fetched rather than bundled (ADR-0015), so the map card has
  // states the rest of the screen does not. Both say plainly that only the
  // picture is missing — the directions below it still work.
  'map.loading': 'กำลังโหลดผังอาคาร… คำบอกทางด้านล่างใช้ได้ตามปกติ',
  'map.unavailable': 'แสดงผังอาคารไม่ได้ตอนนี้ ใช้คำบอกทางด้านล่างแทนได้',

  // Location report overlay (routes/patient/-ScanOverlay.tsx).
  'scan.title.camera': 'สแกน QR ที่จุดบริการ',
  'scan.title.pick': 'เลือกจุดที่คุณอยู่',
  'scan.close': 'ปิด',
  'scan.cameraHint': 'ชี้กล้องไปที่ QR หน้าจุดบริการ เพื่อบอกตำแหน่งปัจจุบันของคุณ',
  'scan.cameraDenied': 'เปิดกล้องไม่ได้ — อนุญาตให้เบราว์เซอร์ใช้กล้อง หรือเลือกจุดเองแทน',
  'scan.cameraUnsupported': 'เบราว์เซอร์นี้ไม่รองรับการสแกน — เลือกจุดที่คุณอยู่แทนได้',
  'scan.switchToPick': 'เลือกจุดที่คุณอยู่เอง',
  'scan.switchToCamera': 'สแกน QR แทน',
  'scan.reportInvalid': 'ไม่พบจุดบริการจากข้อมูลนี้ — ลองอีกครั้ง หรือเลือกจุดเอง',
  'scan.reportFailed': 'ส่งตำแหน่งไม่สำเร็จ — ลองอีกครั้ง',
  'scan.loadingPlaces': 'กำลังโหลดจุดบริการ…',
  'scan.noPlaces': 'ยังไม่มีจุดบริการที่ระบุตำแหน่งได้',
  'scan.floor': 'ชั้น {floor}',

  // Wayfinding-anchor kinds (features/floorplan/graphs.ts; staff QR sheet).
  'anchor.kind.ELEVATOR': 'ลิฟต์',
  'anchor.kind.STAIRS': 'บันได',
  'anchor.kind.ENTRANCE': 'ทางเข้า',

  // Journey rail captions (design-system/JourneyRail.tsx labels).
  'rail.currentStep': 'ขั้นตอนปัจจุบัน',
  'rail.nextStep': 'ขั้นตอนถัดไป',
  'rail.queue': 'คิวที่ {number}',

  // Share sheet, patient side (features/share/components/ShareSheet.tsx).
  'share.title': 'แชร์ความคืบหน้าให้ญาติ',
  'share.privacyNote': 'ญาติเปิดลิงก์นี้เห็นเฉพาะขั้นตอนที่กำลังอยู่ ไม่เห็นข้อมูลอื่นของคุณ',
  'share.creating': 'กำลังสร้างลิงก์…',
  'share.stopOldFirst': 'หยุดแชร์ลิงก์เดิมก่อน',
  'share.stopping': 'กำลังหยุดแชร์…',
  'share.recreate': 'ลองสร้างใหม่',
  'share.copied': 'คัดลอกแล้ว',
  'share.copyLink': 'คัดลอกลิงก์',
  'share.stop': 'หยุดแชร์',
  'share.stoppedNote': 'หยุดแชร์ลิงก์เดิมแล้ว — ลิงก์ด้านบนคือลิงก์ใหม่',
  'share.validUntil': 'ใช้ได้ถึง {time}',
  'share.error.tooMany': 'มีลิงก์ที่ยังใช้ได้มากเกินไป — หยุดแชร์ลิงก์เดิมก่อนแล้วลองใหม่',
  'share.error.signInFirst': 'เข้าสู่ระบบใหม่ก่อนจึงจะแชร์ได้',
  'share.error.generic': 'สร้างลิงก์ไม่สำเร็จ — ลองอีกครั้ง',
  'share.close': 'ปิด',

  // Shared view, relative side (features/share/shared-view.ts, routes/-SharedScreen.tsx).
  'shared.title': 'ติดตามการรักษา',
  'shared.checkingLink': 'กำลังตรวจสอบลิงก์…',
  'shared.autoRefresh': 'หน้านี้อัปเดตอัตโนมัติทุก 15 วินาที',
  'shared.done': 'ผู้ป่วยเสร็จการรักษาแล้ว — กลับบ้านได้เลย',
  'shared.expired': 'ลิงก์นี้หมดอายุแล้ว — ขอลิงก์ใหม่จากผู้ป่วยได้เลย',
  'shared.invalid': 'เปิดหน้านี้ด้วยลิงก์ที่ผู้ป่วยแชร์มาเท่านั้น — ขอลิงก์จากผู้ป่วยได้เลย',
  'shared.unreachable': 'เชื่อมต่อไม่สำเร็จ — ลองปิดแล้วเปิดหน้านี้ใหม่อีกครั้ง',
  'shared.status.waiting': 'กำลังรอคิว',
  'shared.status.inService': 'กำลังรับบริการ',
  'shared.status.done': 'เสร็จเรียบร้อย',
  'shared.status.awaiting': 'รออัปเดต',
  'shared.updatedAt': 'อัปเดตล่าสุด {time}',
  'shared.validUntil': 'ใช้ได้ถึง {time}',
  'shared.preparingNextStep': 'ระหว่างเตรียมขั้นตอนถัดไป',
  'shared.awaitingServicePoint': 'รอยืนยันจุดบริการ',
} satisfies Record<string, string>

export type Catalog = typeof th
export type MessageKey = keyof Catalog
