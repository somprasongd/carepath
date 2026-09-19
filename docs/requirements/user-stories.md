# Initial User Stories

## Patient

### US-01 Open CarePath from LINE
As a patient, I want to open CarePath from LINE OA so that I can continue my hospital journey without installing another app.

### US-02 See my current and next step
As a patient, I want to see my current and next service step so that I understand what I need to do.

### US-03 Navigate to the next service
As a patient, I want CarePath to show where the next service is and how to walk there so that I do not need to ask staff for directions.

### US-04 Establish my current location
As a patient, I want to scan a nearby QR code so that CarePath can calculate a route from a known location.

## Hospital staff

### US-05 Configure service-point mapping
As hospital staff, I want a logical service such as LAB or PHARMACY mapped to a physical place so that workflow changes are independent from the floor-plan design.

### US-06 Update floor/navigation data
As hospital staff or an administrator, I want floor and route data maintained independently from clinical workflow so that facility changes do not require rewriting care pathways.

## Integration / IT

### US-07 Develop without a production HIS
As a developer, I want Mock HIS to provide deterministic visits and service states so that the CarePath demo and tests do not depend on hospital production systems.

### US-08 Replace Mock HIS with a real adapter
As an integration engineer, I want CarePath core to consume a stable HIS port so that a real HIS connector can replace Mock HIS without changing journey/navigation logic.
