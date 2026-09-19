# CarePath API - Initial Contract

Base path: `/api/v1`

## Health

`GET /health`

## Visit

`GET /api/v1/visits/{visitId}`

Returns a normalized CarePath view of the visit and its steps.

## Next destination

`GET /api/v1/visits/{visitId}/next`

Returns the next actionable visit step and its service point/place mapping.

## Route

`GET /api/v1/navigation/route?from={nodeId}&to={nodeId}`

Returns route node IDs/coordinates for overlay on the floor plan.

## Location

`POST /api/v1/locations/resolve`

Normalizes provider input such as QR into a CarePath location result.

The starter code implements health and visit passthrough first; the remaining routes are intentionally documented as the next build slices.
