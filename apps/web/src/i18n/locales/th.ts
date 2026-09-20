/**
 * Thai catalog — the canonical message set (ADR-0012). Every other locale is
 * typed against this shape, so a missing key in `en.ts` is a compile error
 * (and the parity test in `../i18n.test.ts` fails `npm run test` first).
 *
 * Keys are named after what they are for, grouped by feature prefix
 * (`step.*`, `sp.*`, later `journey.*`, `navigate.*`, `auth.*`).
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
} satisfies Record<string, string>

export type Catalog = typeof th
export type MessageKey = keyof Catalog
