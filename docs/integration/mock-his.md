# Mock HIS

## Purpose

Mock HIS is a deterministic fake upstream hospital system used for the hackathon and automated development. It is not a second implementation of CarePath business logic.

## Responsibilities

Mock HIS provides upstream facts such as:

- patient/visit reference
- visit status
- ordered/required service sequence
- service codes
- simple queue/status data where needed for demonstration

It does **not** calculate indoor routes or own CarePath floor plans.

## Demo visit

The starter project contains one sample visit:

```text
VISIT-001
1. Registration   COMPLETED
2. Screening      COMPLETED
3. Doctor          COMPLETED
4. Laboratory     READY
5. Pharmacy       PENDING
6. Complete       PENDING
```

The CarePath API fetches this upstream shape and converts service codes such as `LAB` into internal service points such as `LAB-01`.

## Production replacement

```text
CarePath Core
   -> HIS Port
      -> MockHISAdapter       (local/demo)
      -> RealHISAdapter       (production)
```

No journey-domain module should import a mock-specific type.
