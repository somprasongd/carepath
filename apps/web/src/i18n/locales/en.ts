import type { Catalog } from './th'

/**
 * English catalog (ADR-0012). Typed as the Thai catalog's shape: a missing
 * key here is a compile error, so an untranslated string breaks the build
 * instead of leaking Thai onto an English screen. Patient-facing wording
 * follows DESIGN.md: plain language a worried first-time visitor can read,
 * no hospital domain vocabulary.
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

  'step.meta.cancelled': 'Cancelled',
  'step.meta.completed': 'Completed',
  'step.meta.waitingResult': 'Waiting for results',
  'step.meta.ready': 'Ready for you',
  'step.meta.pending': 'Upcoming',

  'sp.REGISTRATION': 'Registration',
  'sp.LAB': 'Laboratory',
  'sp.ORDERTYPE:LAB': 'Laboratory',
  'sp.XRAY': 'X-ray',
  'sp.ORDERTYPE:XRAY': 'X-ray',
  'sp.PHARMACY': 'Pharmacy',
  'sp.CASHIER': 'Cashier',
  'sp.CLINIC:MED': 'Internal Medicine',
  'sp.DOCTOR': 'Consultation room',

  'common.floor': 'Floor {code}',
  'common.clock': '{time}',

  'common.language': 'Language',
  'common.locale.th': 'ไทย',
  'common.locale.en': 'EN',

  'prefs.largeText': 'Large text',

  'auth.checking': 'Checking your access…',
  'auth.redirecting': 'Taking you to LINE sign-in…',
  'auth.errorTitle': 'Sign-in failed',
  'auth.errorUnknown': 'Something went wrong. Please try again.',
  'auth.retry': 'Try again',

  'entry.role': 'Patient',
  'entry.title': 'Find your appointment',
  'entry.lead': 'Enter the visit number (VN) from your hospital slip to see your care plan.',
  'entry.vnLabel': 'Visit number (VN)',
  'entry.vnPlaceholder': 'e.g. VISIT-001',
  'entry.required': 'Please enter your visit number',
  'entry.submit': 'View my care plan',

  'journey.title': 'Your hospital visit today',
  'journey.lead': 'Follow your steps and use the button below to reach your next service point.',
  'journey.signedInWithLine': 'Signed in with LINE · {name}',
  'journey.loading': 'Loading your steps…',
  'journey.retry': 'Try again',
  'journey.error.notFound': 'We could not find a visit with that number.',
  'journey.error.hisUnavailable': 'The hospital record system is unavailable right now.',
  'journey.error.unreachable': 'We could not reach the service.',
  'journey.progress': 'Progress · {done} of {total} steps done',
  'journey.nextStep': 'Next step',
  'journey.navigateCta': 'Take me to {title}',
  'journey.shareWithFamily': 'Share progress with family',
  'journey.shareDemoHint': 'Demo mode: sharing is not available yet',
  'journey.recommendReason.nearest': 'Picked as the closest to your last known location',
  'journey.recommendReason.planOrder':
    'Picked by your care plan’s order — scan a QR at any service point so we can pick the closest one',

  // Next-step queue card (features/visit/components/QueueCard.tsx, FR-17).
  'queue.title': 'Queue at your next step',
  'queue.peopleAhead': '{count} waiting ahead of you',
  'queue.youAreNext': 'You are next',
  'queue.estimate': 'Estimated wait about {minutes} minutes',
  'queue.noData': 'No wait-time data for this point yet',

  'outcome.cancelled.label': 'This visit',
  'outcome.cancelled.title': 'was cancelled',
  'outcome.cancelled.body': 'If you have any questions, please ask the staff at a service point.',
  'outcome.completed.title': 'All steps completed',
  'outcome.completed.body': 'Thank you for visiting today — wishing you good health.',

  'navigate.back': 'Back to your journey',
  'navigate.loadingTitle': 'Finding your destination…',
  'navigate.noDestinationTitle': 'Your service point',
  'navigate.pendingNotice': 'Looking up where you need to go…',
  'navigate.noDestinationNotice': 'There is no next service point in this visit yet.',
  'navigate.unsupportedNotice':
    'Indoor directions to this service point are not available yet — please ask at the registration desk.',
  'navigate.waitingLocation':
    'Directions appear once we know where you are — scan a QR code at a service point to start.',
  'navigate.askStaffIfLost': 'Tell the staff if you get lost',
  'navigate.scanCta': 'Scan the QR at the service point',
  'navigate.pickInstead': 'Or pick where you are',
  'navigate.updateLocation': 'Scan QR / pick a point to update your location',
  'navigate.avoidStairs': 'Avoid stairs (use the lift)',

  'navigate.routeTitle': 'Directions to {name}',
  'navigate.subtitle': '{floor} · {name} · {place}',

  'navigate.locationNowAt': 'Current location · {floor}',
  'navigate.zone': 'Zone {zone}',
  'navigate.source.QR': 'QR scan',
  'navigate.source.ZIGBEE': 'Zigbee',
  'navigate.source.MANUAL': 'Set manually',
  'navigate.cue.followLine': 'Follow the orange line on the map',
  'navigate.cue.elevator': 'Take the elevator to {floor}',
  'navigate.cue.stairs': 'Take the stairs to {floor}',
  'navigate.cue.arrive': 'Arrive at {name} — your destination',

  'navigate.assumedLocation': 'Approximate · after {step}',

  'map.viewFullFloor': 'View full floor',
  'map.viewRoute': 'View the route',
  'map.viewDestination': 'View the destination',
  'map.youAreHere': 'You are here',
  'map.planAria': 'Floor plan, {floor} — destination {name}',
  'map.routeAria': 'Floor plan, {floor} — route from your location to {name}',
  'map.zoomIn': 'Zoom in',
  'map.zoomOut': 'Zoom out',

  'scan.title.camera': 'Scan the service-point QR',
  'scan.title.pick': 'Pick where you are',
  'scan.close': 'Close',
  'scan.cameraHint':
    'Point the camera at the QR by the service point to set your current location',
  'scan.cameraDenied':
    'Camera unavailable — allow camera access, or pick your point instead',
  'scan.cameraUnsupported': 'This browser cannot scan — pick where you are instead',
  'scan.switchToPick': 'Pick where you are',
  'scan.switchToCamera': 'Scan a QR instead',
  'scan.reportInvalid':
    'No service point matches this code — try again, or pick your point',
  'scan.reportFailed': 'Could not send your location — try again',
  'scan.loadingPlaces': 'Loading service points…',
  'scan.noPlaces': 'No locatable service points yet',
  'scan.floor': 'Floor {floor}',

  'anchor.kind.ELEVATOR': 'Elevator',
  'anchor.kind.STAIRS': 'Stairs',
  'anchor.kind.ENTRANCE': 'Entrance',

  'rail.currentStep': 'Current step',
  'rail.nextStep': 'Next step',
  'rail.queue': 'Queue {number}',

  'share.title': 'Share progress with family',
  'share.privacyNote':
    'Family who open this link see only the current step — nothing else about you.',
  'share.creating': 'Creating the link…',
  'share.stopOldFirst': 'Stop the old links first',
  'share.stopping': 'Stopping…',
  'share.recreate': 'Try again',
  'share.copied': 'Copied',
  'share.copyLink': 'Copy link',
  'share.stop': 'Stop sharing',
  'share.stoppedNote': 'The old links are stopped — the link above is a new one.',
  'share.validUntil': 'Valid until {time}',
  'share.error.tooMany': 'Too many links are still active — stop the old ones and try again.',
  'share.error.signInFirst': 'Please sign in again before sharing.',
  'share.error.generic': 'We could not create the link — please try again.',
  'share.close': 'Close',

  'shared.title': 'Treatment follow-up',
  'shared.checkingLink': 'Checking the link…',
  'shared.autoRefresh': 'This page updates itself every 15 seconds',
  'shared.done': 'Treatment is finished — ready to go home.',
  'shared.expired': 'This link has expired — please ask the patient for a new one.',
  'shared.invalid': 'This page opens only with the link the patient shared — ask them for it.',
  'shared.unreachable': 'We could not connect — try closing and reopening this page.',
  'shared.status.waiting': 'Waiting',
  'shared.status.inService': 'Being seen',
  'shared.status.done': 'Done',
  'shared.status.awaiting': 'Awaiting update',
  'shared.updatedAt': 'Updated {time}',
  'shared.validUntil': 'Valid until {time}',
  'shared.preparingNextStep': 'Preparing the next step',
  'shared.awaitingServicePoint': 'Confirming the service point',
}
