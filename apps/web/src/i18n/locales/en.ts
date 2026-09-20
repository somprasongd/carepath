import type { Catalog } from './th'

/**
 * English catalog (ADR-0012). Typed as the Thai catalog's shape: a missing
 * key here is a compile error, so an untranslated string breaks the build
 * instead of leaking Thai onto an English screen.
 */
export const en: Catalog = {
  'step.title.REGISTRATION': 'Check-in',
  'step.title.LAB': 'Blood draw',
  'step.title.XRAY': 'X-ray',
  'step.title.EKG': 'EKG',
  'step.title.ULTRASOUND': 'Ultrasound',
  'step.title.CASHIER': 'Payment',
  'step.title.PHARMACY': 'Medication pickup',

  'step.clinic.MED': 'Internal Medicine',
  'step.clinic.SURG': 'Surgery',
  'step.seeDoctor': 'See the doctor',
  'step.seeDoctorAgain': 'Return to the doctor',

  'sp.REGISTRATION': 'Registration',
  'sp.LAB': 'Laboratory',
  'sp.ORDERTYPE:LAB': 'Laboratory',
  'sp.XRAY': 'X-ray',
  'sp.ORDERTYPE:XRAY': 'X-ray',
  'sp.PHARMACY': 'Pharmacy',
  'sp.CASHIER': 'Cashier',
  'sp.CLINIC:MED': 'Internal Medicine',
  'sp.DOCTOR': 'Consultation room',
}
