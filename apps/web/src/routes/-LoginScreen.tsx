import {
  Button,
  Card,
  ChoiceCards,
  PageTitle,
  QueueIcon,
  RegistrationIcon,
  ServicePointsIcon,
  OverviewIcon,
  TextField,
} from '@/design-system'
import { staffRoleOptions, type StaffRoleId } from '@/mocks/demo-data'
import { useState } from 'react'

const roleIcon: Record<StaffRoleId, React.ReactNode> = {
  registration: <RegistrationIcon />,
  'service-point': <QueueIcon />,
  admin: <ServicePointsIcon />,
  executive: <OverviewIcon />,
}

/**
 * เข้าสู่ระบบ · เลือกบทบาท (FR-18) — the gate every staff/admin/executive
 * function sits behind. No backend auth exists yet: username/password are
 * inert, and the four cards are the roles docs/requirements/use-case-diagram.md
 * names for everyone except the patient and their relative, who enter through
 * LINE and never see this screen.
 */
export function LoginScreen({ onSignIn }: { onSignIn: (landing: string) => void }) {
  const [roleId, setRoleId] = useState<StaffRoleId>('registration')
  const role = staffRoleOptions.find((r) => r.id === roleId) ?? staffRoleOptions[0]

  return (
    <div className="flex min-h-dvh items-center justify-center bg-neutral px-gutter py-12">
      <Card radius="lg" padding="xl" className="w-full max-w-[560px]">
        <div className="mb-9 flex items-baseline gap-2">
          <span className="font-code text-[20px] font-bold text-ink">CarePath</span>
          <span className="font-sans text-body-sm text-ink-muted">คอนโซลเจ้าหน้าที่</span>
        </div>

        <PageTitle className="mb-1.5">เข้าสู่ระบบ</PageTitle>
        <p className="mt-0 mb-7 font-sans text-body-sm text-ink-muted">
          ใช้บัญชีที่โรงพยาบาลออกให้ — สิทธิ์การเข้าถึงเป็นไปตามบทบาทที่เลือกด้านล่าง
        </p>

        <div className="mb-7 grid grid-cols-1 gap-4 @sm:grid-cols-2">
          <TextField label="ชื่อผู้ใช้" placeholder="เช่น somchai.r" autoComplete="username" />
          <TextField
            label="รหัสผ่าน"
            type="password"
            placeholder="••••••••"
            autoComplete="current-password"
          />
        </div>

        <div className="mb-2.5 font-sans text-caption font-semibold text-ink">เลือกบทบาทของคุณ</div>
        <ChoiceCards
          label="บทบาท"
          columns={2}
          value={roleId}
          onValueChange={(v) => setRoleId(v as StaffRoleId)}
          options={staffRoleOptions.map((r) => ({
            value: r.id,
            title: r.label,
            description: r.scope,
            icon: roleIcon[r.id],
          }))}
          className="mb-6"
        />

        <Button variant="secondary" block onClick={() => onSignIn(role.landing)}>
          เข้าสู่ระบบในฐานะ{role.label}
        </Button>

        <p className="mt-4 mb-0 text-center font-sans text-[11px] text-ink-muted">
          ต้นแบบสาธิตเท่านั้น — ไม่มีการยืนยันตัวตนจริง
        </p>
      </Card>
    </div>
  )
}
