# User Stories

Each story tags the MoSCoW requirement it satisfies (see [Product Requirements](product-requirements.md) and the [hackathon brief](โจทย์%20Hackathon%201%20-%20โรงพยาบาล%20CarePath.docx) §5) so scope stays traceable end to end. See [Use Case Diagram](use-case-diagram.md) for how these stories map to actors and use cases.

## Patient

### US-01 Open CarePath from LINE — *Must (M4)*
As a patient, I want to open CarePath from LINE OA so that I can continue my hospital journey without installing another app.

### US-02 See my current and next step — *Must (M4)*
As a patient, I want to see my current and next service step so that I understand what I need to do.

### US-03 Navigate to the next service — *Must (M5, M6)*
As a patient, I want CarePath to show where the next service is and how to walk there so that I do not need to ask staff for directions.

### US-04 Establish my current location — *Should (S2)*
As a patient, I want to scan a nearby QR code so that CarePath can calculate a route from a known location.

### US-09 Get an accessible route — *Should (S5)*
As an elderly patient or wheelchair user, I want larger text and a route that avoids stairs so that I can navigate safely on my own.

### US-10 Get notified before my queue comes up — *Should (S6)*
As a patient, I want to be notified when my queue is approaching so that I can rest elsewhere without fear of missing my turn.

### US-11 Use CarePath in my own language — *Should (S4)*
As a foreign patient, I want to switch the app to English so that I can understand my journey and navigation instructions.

## Relative

### US-12 Track a patient's progress remotely — *Could (C2)*
As a patient's relative, I want to follow which step the patient is currently on from my own phone via a time-limited link so that I can arrive to pick them up on time.

## Registration / screening staff

### US-13 Register a visit and assign a pathway template — *Must (M3)*
As registration/screening staff, I want to register a patient's visit for the day and assign a Care Pathway Template so that the system automatically generates that patient's ordered visit steps.

## Service-point staff

### US-14 Call the queue and record step completion — *Must (M7)*
As staff at a service point (exam room, lab, X-ray, pharmacy), I want to call the next queue ticket and mark a step as arrived or completed so that the patient automatically advances to their next step.

### US-15 Insert an unplanned step — *Must (M7)*
As staff at a service point, I want to send a patient to an additional step that was not part of the original plan (e.g. an extra test) so that the patient's visit plan matches what actually happens, without breaking the existing before/after ordering.

## Hospital admin

### US-05 Configure service-point mapping — *Must (M1)*
As hospital staff, I want a logical service such as LAB or PHARMACY mapped to a physical place so that workflow changes are independent from the floor-plan design.

### US-06 Update floor/navigation data — *Must (M1, M6)*
As hospital staff or an administrator, I want floor and route data maintained independently from clinical workflow so that facility changes (e.g. a room moved to another building) do not require rewriting care pathways or produce incorrect routes.

### US-16 Create and edit care pathway templates — *Must (M2)*
As a system administrator, I want to create and edit Care Pathway Templates (e.g. "returning diabetic patient") — their steps, order, and prerequisite conditions — so that registration staff can assign a consistent, reusable plan to each patient.

### US-17 Manage user roles and access — *Must (M8)*
As a system administrator, I want to manage user accounts and role-based permissions so that each user (patient, staff, admin, executive) sees and can do only what their role allows.

## Hospital executive

### US-18 View bottlenecks and average wait time — *Should (S7)*
As a hospital executive, I want to see which service points are bottlenecks, their average wait time, and the average total time patients spend in the hospital so that I can allocate staff where they are needed most.

## Integration / IT

### US-07 Develop without a production HIS — *Must (supports M3–M7)*
As a developer, I want Mock HIS to provide deterministic visits and service states so that the CarePath demo and tests do not depend on hospital production systems.

### US-08 Replace Mock HIS with a real adapter — *Non-functional (maintainability)*
As an integration engineer, I want CarePath core to consume a stable HIS port so that a real HIS connector can replace Mock HIS without changing journey/navigation logic.

## Optional / stretch (Could Have)

### US-19 Automatic re-sequencing on abnormal queue length — *Could (C1)*
As a patient, I want the system to reorder my remaining steps when one service point has an unusually long queue, without violating any before/after prerequisite, so that I spend less time waiting overall.

### US-20 Voice-guided navigation — *Could (C3)*
As an elderly patient who finds reading difficult, I want navigation instructions read aloud so that I can follow the route without needing to read text.

### US-21 Suggested amenities along the route — *Could (C4)*
As a patient, I want the app to point out a nearby restroom, waiting area, or food stall on the way to my next step so that I can take care of other needs without going out of my way.
