import { useEffect, useState } from 'react'

type NextStep = {
  sequence: number
  status: string
  servicePoint?: {
    id: string
    code: string
    name: string
    placeId: string
  }
}

type Visit = {
  visitId: string
  patientRef: string
  status: string
  next?: NextStep
}

const apiBase = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

export function App() {
  const [visit, setVisit] = useState<Visit | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    fetch(`${apiBase}/api/v1/visits/VISIT-001`)
      .then((r) => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        return r.json()
      })
      .then(setVisit)
      .catch((e) => setError(String(e)))
  }, [])

  return (
    <main className="page">
      <section className="hero">
        <div className="eyebrow">CarePath</div>
        <h1>เส้นทางการรับบริการของคุณ</h1>
        <p>ตัวอย่าง MVP ที่อ่าน Visit จาก Mock HIS และแปลงขั้นตอนถัดไปเป็นจุดบริการใน CarePath</p>
        <p>
          <a href="/design">ดูระบบดีไซน์และหน้าจออ้างอิงที่ /design →</a>
        </p>
      </section>

      {error && <div className="card error">เชื่อมต่อ API ไม่สำเร็จ: {error}</div>}

      {visit && (
        <section className="card">
          <div className="meta">Visit: {visit.visitId}</div>
          <h2>ขั้นตอนถัดไป</h2>
          {visit.next?.servicePoint ? (
            <>
              <div className="destination">{visit.next.servicePoint.name}</div>
              <div className="meta">Place: {visit.next.servicePoint.placeId}</div>
              <button>นำทางไปจุดบริการ</button>
            </>
          ) : (
            <p>ยังไม่มีจุดบริการถัดไป</p>
          )}
        </section>
      )}
    </main>
  )
}
