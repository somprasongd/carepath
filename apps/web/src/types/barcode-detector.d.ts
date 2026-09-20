// Minimal ambient types for the Barcode Detection API (Chromium, Safari 17+;
// not in TypeScript's DOM lib yet). Only the surface useQrScanner touches is
// declared — detected-barcode formats are narrowed to QR alone.
interface BarcodeDetectorOptions {
  formats?: string[]
}

interface DetectedBarcode {
  rawValue: string
}

declare class BarcodeDetector {
  constructor(options?: BarcodeDetectorOptions)
  static getSupportedFormats(): Promise<string[]>
  detect(source: CanvasImageSource): Promise<DetectedBarcode[]>
}
