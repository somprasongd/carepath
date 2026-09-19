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

## MVP simplification

The starter frontend can run as a normal browser application. LINE LIFF bootstrap can be added behind an identity/session adapter so local browser development remains easy.

## Production considerations

- Validate LINE-issued identity tokens on the backend.
- Do not trust profile fields supplied only by the browser.
- Avoid placing patient identifiers directly in query strings.
