import { useState } from 'react'
import {
  Button,
  Card,
  ChoiceCards,
  Code,
  InfoNote,
  Meta,
  PageTitle,
  SectionTitle,
} from '@/design-system'
import { pathwayTemplates } from '@/mocks/demo-data'
import { TaskHeader } from './-TaskHeader'

/**
 * เจ้าหน้าที่ · แผนการดูแล (Pathway Template) — FR-13, FR-14, US-13.
 *
 * Per ADR-0008 the HIS is the sole system of record for opening a visit and
 * ordering services; CarePath never writes there. This screen used to look a
 * patient up and "register" their visit — it doesn't anymore. It only reads
 * static Pathway Template config and lets staff copy the resulting
 * service-code checklist to type into the HIS themselves when they open the
 * visit there. There is nothing here to wire to a backend: the projected
 * `VisitStep` list a patient actually sees comes from the HIS events
 * apps/api's `journey` module ingests, once that step is placed.
 */
export function PathwayTemplates() {
  const [templateId, setTemplateId] = useState(pathwayTemplates[0].id)
  const [copied, setCopied] = useState(false)

  const template = pathwayTemplates.find((t) => t.id === templateId) ?? pathwayTemplates[0]
  const codeList = template.steps.map((s) => s.code).join(',')

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(codeList)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // Clipboard access can be denied (permissions, insecure context); the
      // codes are still visible to copy by hand, so this fails silently.
    }
  }

  return (
    <div className="@container flex h-dvh flex-col bg-neutral">
      <TaskHeader role="เจ้าหน้าที่ลงทะเบียน" />

      <main className="flex-1 overflow-y-auto px-gutter py-8 @7xl:px-gutter-desktop">
        <div className="mx-auto max-w-[1240px]">
          <PageTitle>แผนการดูแล (Pathway Template)</PageTitle>
          <Meta className="mt-1.5 mb-5">
            เลือกแผนการดูแลเพื่อดูรายการรหัสบริการที่ต้องกรอกตอนเปิด visit ในระบบ HIS
          </Meta>

          <div className="mb-6 max-w-[720px]">
            <InfoNote>
              หน้านี้ไม่ได้เปิด visit หรือส่งข้อมูลไปที่ HIS — เป็นรายการอ้างอิงให้เจ้าหน้าที่กรอกในระบบ
              HIS ด้วยตนเองเท่านั้น (การเปิด visit และสั่ง order เป็นหน้าที่ของ HIS)
            </InfoNote>
          </div>

          <div className="grid grid-cols-1 gap-6 @5xl:grid-cols-2">
            <Card radius="lg" padding="xl">
              <SectionTitle>เลือกแผนการดูแล</SectionTitle>
              <ChoiceCards
                label="แผนการดูแล"
                value={templateId}
                onValueChange={setTemplateId}
                options={pathwayTemplates.map((t) => ({
                  value: t.id,
                  title: t.name,
                  description: t.description,
                  meta: `${t.steps.length} รายการ`,
                }))}
              />
            </Card>

            <Card radius="lg" padding="xl">
              <SectionTitle>รหัสบริการที่ต้องสั่ง — {template.name}</SectionTitle>
              <Meta className="-mt-2 mb-4">เรียงตามลำดับ ใช้กรอกในระบบ HIS ตอนเปิด visit</Meta>

              <ol className="m-0 mb-5 flex list-none flex-col p-0">
                {template.steps.map((step, i) => (
                  <li key={step.code} className="flex gap-3">
                    <div className="flex w-[22px] shrink-0 flex-col items-center">
                      <span className="flex size-[22px] shrink-0 items-center justify-center rounded-full border border-line bg-neutral font-code text-[11px] font-bold text-ink-muted">
                        {i + 1}
                      </span>
                      {i < template.steps.length - 1 && (
                        <span className="mt-0.5 min-h-5 w-0.5 flex-1 bg-line" />
                      )}
                    </div>
                    <div className="pb-5">
                      <div className="font-sans text-body-md font-semibold text-ink">
                        {step.title}
                      </div>
                      <div className="mt-0.5 font-sans text-caption font-normal text-ink-muted">
                        {step.code}
                      </div>
                    </div>
                  </li>
                ))}
              </ol>

              <div className="mb-3 overflow-x-auto rounded-md border border-line bg-neutral px-3 py-2.5">
                <Code className="whitespace-nowrap">{codeList}</Code>
              </div>

              <Button variant="secondary" block onClick={() => void copy()}>
                {copied ? 'คัดลอกแล้ว ✓' : 'คัดลอกรายการรหัสบริการ'}
              </Button>
            </Card>
          </div>
        </div>
      </main>
    </div>
  )
}
