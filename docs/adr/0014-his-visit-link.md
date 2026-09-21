# ADR-0014: HIS-Minted Visit Link — the Slip Is the Credential Handoff

- Status: Accepted (amended 2026-09-20 — see Amendment)
- Date: 2026-09-20
- Related: [ADR-0010](0010-staff-auth-jwt-argon2.md) (identity kinds; patient sessions), [ADR-0011](0011-visit-share-link.md) (third credential; hash-only storage; token never in logs), [ADR-0005](0005-his-adapter-and-mock-his.md) (HIS behind a port), [ADR-0008](0008-his-canonical-event-contract.md) (event-driven HIS integration), #96 (journey session guard), #136

## Context

Layer 1 of the journey guard (#96) made the patient session *claim* a visit
before it can read the journey — but the claim credential was still the visit
number itself: `POST /journeys/{visitId}/claim` with the VN typed in
(`?visit=VISIT-004` on the URL). A VN is printed on registration slips and
called out loud at counters; it is guessable (`VISIT-004`), observable on
paper, and belongs to a sequence anyone can walk. In production the VN-entry
screen therefore had to go (issue #136 decision 2: demo deployments only).

What replaces it has to work *without the patient doing anything*: the
hospital already hands the patient a navigation slip (ใบนำทาง) at
registration. The natural move is that the slip carries the credential.

Constraints inherited from earlier ADRs:

- ADR-0011: credentials are never stored raw (sha256 only) and never placed
  in URL paths or query strings (the request logger writes `c.Path()`).
- ADR-0010: the patient *session* stands for a verified identity; whatever
  the slip carries must not be a second identity, only a way to *grant the
  session a claim* on exactly one visit.
- ADR-0005/0008: CarePath does not read the HIS database; everything the
  HIS knows arrives as REST calls or canonical events. Minting a link is
  CarePath asking nothing of the HIS — it is the **first inbound HIS→CarePath
  call** (until now the HIS only *received* projections or commands).

## Decision

Add a **visit link** minted by CarePath on the HIS's request, printed on the
slip as a QR, and redeemed by the patient's browser exactly once into a
session claim. New module `internal/visitlink`, table
`carepath.visit_access_token`, contract 0.21.0.

1. **The link is a claim grant, not an identity.** It does one thing:
   `POST /api/v1/journeys/claim` with `{token}` gives the calling session's
   identity a claim on the token's visit — the same claim #96 requires. It is
   accepted nowhere else, and nothing else is accepted on that route. The
   session remains the only thing that reads journeys.

2. **Deterministic HMAC token, hash-only storage.**
   `token = base64url(HMAC-SHA256(VISIT_LINK_SECRET, "visit-link:{visitId}:{version}"))[:16]`
   (22 chars). Only `sha256(token)` and the `version` land in the database.
   Determinism buys three things for free: **idempotent mint** (the HIS can
   reprint a slip and get the same link), **no raw-token round-trip through
   storage**, and **break-glass revocation of every link at once** by
   rotating the secret. Redeem verifies by recomputing the HMAC
   (`hmac.Equal`) against the stored version — a hash-lookup followed by a
   proof, so a leaked database alone cannot redeem anything.

3. **`rotate=true` is the revoke-and-reprint.** Each mint with `rotate=true`
   bumps the version, which changes the HMAC input and kills the old token.
   Reprint after a lost slip = mint with rotate; there is no admin revocation
   surface (#136 decision 3) — the hospital workflow owns the paper.

4. **Token in the URL fragment, never the query.** The minted URL is
   `{PATIENT_APP_BASE_URL}/patient/journey#vt={token}`. Fragments are not
   sent to servers and never appear in access logs; the web app lifts `#vt=`
   into sessionStorage and `history.replaceState`s it away *before*
   `liff.init()` (the LIFF OAuth redirect would drop the fragment). Query
   strings were rejected for the same reason ADR-0011 rejected path tokens:
   `?visit=` in logs is how this whole hole opened.

5. **Lifetime is lazy and policy-driven, not a stored expiry.** The table
   stores no `expires_at`; redeem re-checks the visit's *current* projected
   state: ACTIVE links stay live; COMPLETED links live on for
   `VISIT_LINK_COMPLETED_GRACE` (default **30 minutes** from
   `completed_at` — #136 decision 1: the patient should still see "จบ
   การรับบริการแล้ว" and the summary while walking out); CANCELLED dies
   immediately. Because the check is lazy, the HIS needs no extra "revoke"
   call — the completion event it already emits is the signal. A
   **minted claim outlives its token**: the guard checks claims, not links,
   so a session that redeemed in-grace keeps reading its journey after the
   grace window closes. Only *new* redemptions stop.

6. **Fail-closed registration on two secrets.** The mint route
   (`POST /api/v1/his/visits/{visitId}/patient-link?format=url|qr`,
   QR = PNG for direct slip printing) and the whole module register only when
   `HIS_API_KEY` **and** `VISIT_LINK_SECRET` are both set. Unset means the
   surface does not exist (404), not that it is unprotected. The mint route
   is guarded by `X-HIS-API-Key` compared with `subtle.ConstantTimeCompare` —
   the key *mints* a credential, it is not one, which is why it is a fourth
   security scheme (`hisApiKey`) in the contract and never accepted anywhere
   else.

7. **The demo VN entry stays demo-only.** `POST /journeys/{visitId}/claim`
   (naming the VN) remains registered only under `ALLOW_DEMO_AUTH`, and the
   web VN-entry screen only renders when `VITE_AUTH_MODE=demo`. In
   production the slip link is the only front door. Mock HIS plays the real
   HIS here: its console QR proxies the CarePath mint server-side
   (`CAREPATH_HIS_API_KEY` never reaches the browser), falling back to the
   legacy `?visit=` QR when the key is unset, so local dev works with zero
   configuration and the demo can show both modes.

## Consequences

- The VN never appears in a patient-facing URL again; guessing VNs yields
  only uniform 404s (unchanged from #96).
- Copying the slip link shares *access to the claim grant*, so the link is
  single-purpose and short-lived like the paper it rides on; the 30-minute
  completed-grace bounds exposure of a forwarded link without an expiry
  column.
- Secret rotation (`VISIT_LINK_SECRET`) invalidates all outstanding links at
  once — cheap for patients (next slip works), acceptable as break-glass.
- The HIS integration gains its first inbound REST dependency (the mint
  call); a real HIS adapter that cannot call out can instead print the
  CarePath URL from a batch mint, but the contract surface stays the same.
- e2e coverage: `internal/e2e/visitlink_test.go` walks mint (401/404/
  idempotent/rotate) and the full patient lifecycle including stranger-404,
  rotated-out-404, in-grace redemption, and post-grace "claim outlives
  token" — under distinct LINE identities, since demo sessions all share one
  fixed identity.

## Amendment (2026-09-20): the QR-only hospital, and retiring claim-by-name

Layer 2 shipped with a gap the first production slip exposed: redeeming the
token required a *pre-existing* session, and the only configured session
source was demo (`ALLOW_DEMO_AUTH`) — so production ran with the demo door
open, and `?visit=VISIT-002` was readable by editing the URL (claim-by-name
was still mounted). Two product decisions follow (issue #136):

1. **Entry is two first-class modes, not one plus a demo fallback.** A
   hospital *with* a LINE Official Account sends patients through LINE LIFF
   (`source "line"`); a hospital *without* one hands the patient nothing but
   the printed slip — and the slip alone must carry them in. The token
   therefore gained a second redemption path: `POST /auth/session` with
   `source "visit-token"` **bootstraps the session itself** (contract
   0.22.0). `session` defines a `VisitTokenResolver` port (it cannot import
   `visitlink` — that would cycle through `RequireSession`); main.go injects
   the visitlink service. The minted identity is visit-scoped and throwaway
   (`{source: "visit", externalId: "visit:{vn}"}`, no DisplayName —
   ADR-0012), the session is TTL-bound like any other, and the server
   claims for that identity *before* persisting the session (a failure
   costs a retry, never an orphaned session that can never claim). This
   narrows §1 — the link is a claim grant first, and an identity *only* in
   the sense of a visit-scoped session identity that dies with the session;
   the durable grant is still the claim.
2. **A live mint retires claim-by-name everywhere, demo deployments
   included.** main.go registers `POST /journeys/{visitId}/claim` only when
   `ALLOW_DEMO_AUTH` **and not** the visit-link env are set: with the mint
   live, knowing a visit id must stop being a credential, or the §7 demo
   exception re-opens the very hole layer 2 exists to close. The VN entry
   screen consequently closes on any deployment with the mint configured;
   demo deployments *without* the mint keep today's behavior unchanged.

Web exchange rule: the slip screen waits for the auth provider to settle
(the provider overwrites the API bearer when it finishes), then redeems
into an existing LINE identity's claim but bootstraps a visit-token session
in every other case (demo mode, LIFF failure, no bearer). The client also
treats a non-journey 404 from the claim route as retired-route (remembered,
not retried) — distinct from "journey not found", which stays retryable
while the projection may still land.

e2e: `TestVisitLinkQRBootstrap` walks the anonymous bootstrap end to end
(no Authorization header at all), the visit-scoped read, the neighbor-404
(`?visit=` guessing stays dead with the mint live), and rotation killing
the bootstrap path.
