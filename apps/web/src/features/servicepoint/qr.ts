import qrcode from 'qrcode-generator'

/**
 * The payload of a place's location sticker QR — exactly the string the API's
 * QR provider parses (a place reference only, never patient data; full URLs
 * like https://host/location/REG-01 also scan, but the bare path is
 * host-independent and stable across deployments).
 */
export function locationQrPayload(placeId: string): string {
  return `location/${placeId}`
}

/**
 * One encoded payload as SVG geometry: every dark module as a unit-square
 * subpath, so a single <path> paints the whole code at any size with
 * crispEdges. `moduleCount` is the code's side length in modules; callers add
 * the 4-module quiet zone around it (viewBox −4 … count+4).
 */
export function qrModules(payload: string): { path: string; moduleCount: number } {
  const qr = qrcode(0, 'M')
  qr.addData(payload)
  qr.make()
  const count = qr.getModuleCount()
  let path = ''
  for (let row = 0; row < count; row++) {
    for (let col = 0; col < count; col++) {
      if (qr.isDark(row, col)) path += `M${col} ${row}h1v1h-1z`
    }
  }
  return { path, moduleCount: count }
}
