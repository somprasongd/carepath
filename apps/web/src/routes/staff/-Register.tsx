import { useState } from 'react'
import {
  Button,
  Card,
  ChoiceCards,
  Meta,
  ReadoutField,
  SearchIcon,
  SectionTitle,
  TextField,
} from '@/design-system'
import { patientLookup, pathwayTemplates } from '@/mocks/demo-data'
import { TaskHeader } from './-TaskHeader'

/**
 * เจ้าหน้าที่ · ลงทะเบียนผู้ป่วย (FR-13, FR-14, US-13) — look a patient up by
 * HN, assign a Care Pathway Template, and preview the ordered VisitStep list
 * CarePath will generate. No registration endpoint exists yet, so "ค้นหา" and
 * "ลงทะเบียน" only drive this screen's own state — see queries.ts in
 * features/visit for the pattern this will move to once one lands.
 */
export function Register() {
  const [hn, setHn] = useState('0012345')
  const [looked, setLooked] = useState(false)
  const [templateId, setTemplateId] = useState(pathwayTemplates[0].id)

  const template = pathwayTemplates.find((t) => t.id === templateId) ?? pathwayTemplates[0]

  return (
    <div className="@container flex h-dvh flex-col bg-neutral">
      <TaskHeader role="เจ้าหน้าที่ลงทะเบียน" />

      <main className="flex-1 overflow-y-auto px-gutter py-8 @7xl:px-gutter-desktop">
        <div className="mx-auto grid max-w-[1240px] grid-cols-1 items-start gap-6 @5xl:grid-cols-[400px_1fr]">
          <Card radius="lg" padding="xl">
            <SectionTitle>ค้นหาผู้ป่วย</SectionTitle>
            <Meta className="-mt-2 mb-5">ค้นหาด้วยเลขประจำตัวผู้ป่วย (HN) จาก HIS</Meta>

            <TextField
              label="เลขประจำตัวผู้ป่วย (HN)"
              value={hn}
              onChange={(e) => setHn(e.target.value)}
              trailing={
                <button
                  type="button"
                  aria-label="ค้นหา"
                  onClick={() => setLooked(true)}
                  className="flex size-11 shrink-0 cursor-pointer items-center justify-center rounded-md border-0 bg-secondary text-surface"
                >
                  <SearchIcon />
                </button>
              }
            />

            <div className="mt-5 border-t border-line pt-5">
              {looked ? (
                <div className="flex flex-col gap-4">
                  <ReadoutField label="ชื่อ-สกุล">{patientLookup.name}</ReadoutField>
                  <div className="grid grid-cols-2 gap-4">
                    <ReadoutField label="อายุ">{patientLookup.age}</ReadoutField>
                    <ReadoutField label="สิทธิการรักษา">{patientLookup.coverage}</ReadoutField>
                  </div>
                  <ReadoutField label="นัดหมายวันนี้">{patientLookup.appointment}</ReadoutField>
                </div>
              ) : (
                <p className="m-0 font-sans text-body-sm text-ink-muted">
                  กรอกเลข HN แล้วกดค้นหาเพื่อดึงข้อมูลผู้ป่วยจาก HIS
                </p>
              )}
            </div>
          </Card>

          <div className="flex flex-col gap-5">
            <Card radius="lg" padding="xl">
              <SectionTitle>เลือกแผนการดูแล (Pathway Template)</SectionTitle>
              <Meta className="-mt-2 mb-4">
                ระบบจะสร้างลำดับขั้นตอนของผู้ป่วยให้อัตโนมัติตามแผนที่เลือก
              </Meta>
              <ChoiceCards
                label="แผนการดูแล"
                value={templateId}
                onValueChange={setTemplateId}
                options={pathwayTemplates.map((t) => ({
                  value: t.id,
                  title: t.name,
                  description: t.description,
                  meta: `${t.steps.length} ขั้นตอน`,
                }))}
              />
            </Card>

            <Card radius="lg" padding="xl">
              <SectionTitle>เส้นทางที่ระบบจะสร้าง — {template.name}</SectionTitle>
              <Meta className="-mt-2 mb-4">ผู้ป่วยจะเห็นขั้นตอนนี้ทันทีที่เปิด CarePath</Meta>

              <ol className="m-0 mb-6 flex list-none flex-col p-0">
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

              <Button variant="secondary" block>
                ลงทะเบียนและสร้างเส้นทาง
              </Button>
            </Card>
          </div>
        </div>
      </main>
    </div>
  )
}
