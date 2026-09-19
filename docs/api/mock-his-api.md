# Mock HIS API

Base path: `/api/v1`

## Health

`GET /health`

## Get visit

`GET /api/v1/visits/{visitId}`

Example:

```json
{
  "visitId": "VISIT-001",
  "patientRef": "PATIENT-DEMO-001",
  "status": "ACTIVE",
  "steps": [
    {"sequence": 1, "serviceCode": "REGISTRATION", "status": "COMPLETED"},
    {"sequence": 2, "serviceCode": "SCREENING", "status": "COMPLETED"},
    {"sequence": 3, "serviceCode": "DOCTOR", "status": "COMPLETED"},
    {"sequence": 4, "serviceCode": "LAB", "status": "READY"},
    {"sequence": 5, "serviceCode": "PHARMACY", "status": "PENDING"}
  ]
}
```

Unknown visits return HTTP 404.
