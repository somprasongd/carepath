# Shared Contracts

This package contains version-controlled API contracts that may be consumed by frontend, backend, tests, and integration tooling.

- `openapi/carepath.yaml` — the CarePath API the web app talks to
- `openapi/mock-his.yaml` — the Mock HIS API CarePath's HIS adapter talks to

Changing these files is an integration-boundary change (ADR-0006): check
`apps/api` and `apps/mock-his` for drift, and regenerate the web types.

## Frontend type generation

`apps/web/src/api/schema.d.ts` is generated from `openapi/carepath.yaml`:

```bash
npm run gen:api     # from the repo root (or apps/web)
```

The script pins `openapi-typescript` + `typescript@5` in an isolated npx run
because the app itself builds with TypeScript 7, whose package no longer
exposes the JS compiler API the generator uses. Commit the generated file.
