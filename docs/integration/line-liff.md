# LINE OA / LIFF Integration

## Target flow

```text
LINE OA
  -> open LIFF application
  -> LINE Login / identity verification
  -> CarePath session
  -> active visit
  -> current/next step
  -> navigation
```

## Scope: patients only

This flow is the **patient** identity path. Staff and admins never pass through
LINE — they log in with a username and password CarePath itself stores, and get
a JWT access token plus a rotating refresh token instead of a session token
(see [ADR-0010](../adr/0010-staff-auth-jwt-argon2.md)). The two mechanisms stay
separate: a LINE session token opens no staff endpoint, and a staff token opens
no patient session.

## MVP simplification

The starter frontend can run as a normal browser application. LINE LIFF bootstrap can be added behind an identity/session adapter so local browser development remains easy.

## Production considerations

- Validate LINE-issued identity tokens on the backend.
- Do not trust profile fields supplied only by the browser.
- Avoid placing patient identifiers directly in query strings.
