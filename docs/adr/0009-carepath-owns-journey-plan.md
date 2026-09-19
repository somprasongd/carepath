# ADR-0009: CarePath Owns the Journey Plan; the HIS Provides Clinical Facts

- Status: Accepted
- Date: 2026-09-19
- Amends: [ADR-0008](0008-his-canonical-event-contract.md) (§1, §2, §3, §6, §7)

## Context

ADR-0008 assumed the HIS can hand CarePath an ordered list of service steps
(`Visit.steps[] = {sequence, serviceCode, status}`) and owns their status.
Reviewing the real OPD flow with the hospital showed that assumption does not
hold: **the HIS has no concept of an ordered patient journey.** What it does
know, and can emit, is:

- the visit — HN, VN, patient name, walk-in vs appointment;
- which clinic(s) the patient is assigned to (one visit may involve more than one);
- orders — order name, order type (`LAB`, `XRAY`, `EKG`, `US`, `DRUG`), the
  ordering clinic, and the timestamp the order was placed;
- the lifecycle of an order (placed → performed → resulted);
- when a doctor finishes examining a patient at a clinic.

The actual OPD flow those facts have to be turned into:

1. always start at registration;
2. appointment patients with pre-visit orders go to LAB / X-Ray first, then the clinic;
3. appointment patients without orders go straight to the assigned clinic;
4. mid-consultation the doctor may order more investigations — the patient does
   them and **comes back to the same doctor**;
5. the doctor may or may not prescribe drugs;
6. every patient goes to the cashier, whatever their insurance and even with no
   orders at all;
7. if drugs were prescribed, the pharmacy comes after the cashier.

Nothing in that list is derivable from a `serviceCode` list the HIS does not
produce. Either CarePath derives the journey itself, or there is no journey.

## Decision

### 1. Ownership moves

| Concern | Owner |
| --- | --- |
| Visit, clinic assignment, orders, order results, encounter completion | **HIS** (system of record) |
| Journey plan: which steps exist, their order, their status | **CarePath** |
| Service point ↔ place binding, navigation | **CarePath** (unchanged) |

This supersedes ADR-0008 §2. CarePath no longer proxies step-status transitions
to the HIS: a step is a CarePath concept and most steps have no HIS counterpart.
Where a counterpart does exist (an order being fulfilled), CarePath may notify
the HIS, but the journey never depends on that write succeeding.

### 2. Inbound canonical facts replace the step-level event set

ADR-0008's `service.requested / started / completed / cancelled` events are
retired. The canonical inbound events become:

| Event | Payload |
| --- | --- |
| `visit.opened` | `patientRef` (HN), `patientName`, `visitType: WALKIN\|APPOINTMENT`, `clinics[]`, `openedAt` |
| `visit.updated` | `clinics[]`, `status` |
| `visit.closed` | `status` |
| `order.placed` | `orderRef`, `orderType`, `orderName`, `orderedByClinic`, `orderedAt` |
| `order.performed` | `orderRef`, `performedAt` — the procedure is done (blood drawn, film taken) |
| `order.resulted` | `orderRef`, `resultedAt` — the result is reported and readable by the doctor |
| `order.cancelled` | `orderRef` |
| `encounter.completed` | `clinicCode`, `completedAt` — this doctor is finished with this patient for this round |

The envelope (`eventId`, `occurredAt`, `visitId`, `patientRef`, `type`,
`payload`) and consumer-side idempotency on `eventId` are unchanged from
ADR-0008 §3. A real HIS adapter maps whatever it has onto these facts
(ADR-0005); `encounter.completed`, for example, may come from the doctor
closing the visit note, from the clinic worklist status, or from staff
confirming in CarePath.

### 3. The plan is a pure function, recomputed on every fact

`plan(visitFacts) -> []Step` is deterministic and side-effect free. It runs
again on every inbound event; the result is **diffed** against the stored plan
rather than appended to. Ordering is the sort key `(clinicIndex, phase, orderedAt)`:

| phase | step | exists when |
| --- | --- | --- |
| 0 | `REG` | always; `COMPLETED` at `visit.opened` — opening the visit *is* registration |
| 10 | one step per order type ordered before the clinic | orders with `orderedAt <= openedAt` |
| 20 | `CLINIC:<code>#1` | one per assigned clinic |
| 30 | one step per order type ordered during the encounter | `order.placed` while the encounter is open |
| 40 | `CLINIC:<code>#n+1` | per §4 below |
| 90 | `CASHIER` | always, **exactly once per visit** regardless of how many clinics |
| 95 | `PHARMACY` | a `DRUG` order exists |

Orders of the same type placed in the same round collapse into one step (five
lab tests = one trip to the lab); the step carries `orderRefs[]`.

### 4. Returning to the same doctor is inferred, and confirmable

A diagnostic order (`LAB`, `XRAY`, `EKG`, `US`) placed while a clinic's
encounter is still open implies the doctor intends to read the result:
the planner emits the investigation steps and a follow-up
`CLINIC:<code>#n+1` step after them. A `DRUG` order never implies a return.

`encounter.completed` closes the round: any follow-up clinic step still in a
non-started state is dropped from the plan. Clinic staff can force either
outcome from the service-point console ("send for more tests, then come back"
vs "finished with this patient"), which is the override when the inference is
wrong and the fallback when a HIS cannot emit `encounter.completed`.

### 5. Result reporting gates the return, not the trip

The patient's investigation step completes at `order.performed` — once blood is
drawn the patient should leave the lab. The follow-up clinic step is
`WAITING` until **every** diagnostic order of that round is `order.resulted`,
then becomes actionable. `WAITING` is added to the canonical step statuses
(`PENDING | WAITING | READY | STARTED | COMPLETED | CANCELLED`) because it is a
state the patient must see and understand ("waiting for lab results"), not an
internal detail.

### 6. Steps in the same phase are not ordered against each other

Steps sharing a sort key are all `READY` at once: a patient with a lab and an
X-Ray before their clinic may do either first. The journey therefore exposes
`actionable[]` plus one `recommended` step (nearest, or shortest queue); the
singular `next` of the ADR-0008-era contract is retired.

### 7. Step identity is a stable key, not a sequence number

Because replanning renumbers, `sequence` is display order only. Each step
carries a stable `stepKey` (`CLINIC:MED:2`, `LAB:ORD-118`, `CASHIER`), which is
what the API, the UI, and the transition command address. The diff rule:
a step that is `COMPLETED`, `STARTED` or `CANCELLED` is history and is never
removed or reordered out of its place; only `PENDING` / `WAITING` steps may be
withdrawn when the facts no longer justify them.

### 8. The patient's name may be stored

ADR-0008 §5 required `patientRef` to carry no PHI. The staff console needs a
name to call a patient, so CarePath stores the display name on the journey
visit. It is the minimum: name only, no other demographics, lifecycle bound to
the visit projection, never exposed on patient-facing endpoints beyond the
patient's own visit. Non-functional requirements record it as PHI.

### 9. Mock HIS becomes a fact driver

Mock HIS stops modelling steps. Its console drives the flow the way the real
world does: open a visit (HN/VN/name/type/clinics), place an order (name +
type + ordering clinic), mark it performed, report its result, complete an
encounter, close the visit — each emitting the canonical event above. The
existing step-transition endpoint is removed with the step-level events.

## Consequences

### Positive
- The journey now matches the real OPD flow, including the return-to-doctor
  loop that no HIS field describes.
- The planner is pure and deterministic, so the hard part of the domain is unit
  testable without a database or a HIS.
- Onboarding a real HIS is a mapping exercise onto eight facts, none of which
  requires the HIS to understand journeys.
- `WAITING` plus result gating gives the patient an honest "waiting for results"
  state instead of a silent gap.

### Trade-offs
- The ordering rules are hospital policy living in CarePath code; a different
  hospital means different rules. Mitigated by keeping them in one pure
  package, driven by the pathway-template configuration of FR-13.
- The return-to-doctor rule is an inference. It is wrong when a doctor orders an
  investigation for a future visit and the HIS sends no `encounter.completed`;
  staff override covers it.
- Breaking changes across the board: contracts (both files), the journey
  projection and its schema, the transition endpoint's addressing and meaning,
  the Mock HIS console, and the patient and staff screens.
- CarePath now holds PHI (a name), which it previously did not.
