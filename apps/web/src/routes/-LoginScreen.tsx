import { ApiError } from '@/api/client'
import { useStaffAuth } from '@/auth/StaffAuthContext'
import { Button, Card, PageTitle, TextField } from '@/design-system'
import { useState, type FormEvent } from 'react'

/**
 * เข้าสู่ระบบ (FR-18) — the staff console's front door. Username/password
 * against the real /api/v1/auth/login (ADR-0010); failure reasons are
 * deliberately uniform server-side, so the form shows one message for every
 * rejected sign-in. Landing depends on the roles the token carries.
 */
export function LoginScreen() {
  const { login } = useStaffAuth()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string | undefined>()

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setPending(true)
    setError(undefined)
    try {
      const identity = await login(username.trim(), password)
      window.location.assign(landingFor(identity))
    } catch (cause) {
      setError(
        cause instanceof ApiError && cause.status === 401
          ? 'ชื่อผู้ใช้หรือรหัสผ่านไม่ถูกต้อง'
          : 'เข้าสู่ระบบไม่ได้ — ลองอีกครั้ง',
      )
      setPending(false)
    }
  }

  return (
    <div className="@container h-full w-full">
      <div className="flex h-full items-center justify-center overflow-y-auto bg-neutral px-gutter py-12">
        <Card radius="lg" padding="xl" className="w-full max-w-[440px]">
          <div className="mb-9 flex items-baseline gap-2">
            <span className="font-code text-[20px] font-bold text-ink">CarePath</span>
            <span className="font-sans text-body-sm text-ink-muted">คอนโซลเจ้าหน้าที่</span>
          </div>

          <PageTitle className="mb-1.5">เข้าสู่ระบบ</PageTitle>
          <p className="mt-0 mb-7 font-sans text-body-sm text-ink-muted">
            ใช้บัญชีที่โรงพยาบาลออกให้ — สิทธิ์การเข้าถึงเป็นไปตามบทบาทของบัญชี
          </p>

          <form onSubmit={submit}>
            <div className="mb-5 grid grid-cols-1 gap-4">
              <TextField
                label="ชื่อผู้ใช้"
                placeholder="เช่น somchai.r"
                autoComplete="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
              />
              <TextField
                label="รหัสผ่าน"
                type="password"
                placeholder="••••••••"
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>

            {error && (
              <p role="alert" className="mb-4 font-sans text-body-sm text-warning">
                {error}
              </p>
            )}

            <Button type="submit" variant="secondary" block disabled={pending}>
              {pending ? 'กำลังเข้าสู่ระบบ…' : 'เข้าสู่ระบบ'}
            </Button>
          </form>

          <p className="mt-4 mb-0 text-center font-sans text-[11px] text-ink-muted">
            บัญชีสาธิต — admin/demo (ผู้ดูแล) · staff/demo (เจ้าหน้าที่) · exec/demo (ผู้บริหาร)
          </p>
        </Card>
      </div>
    </div>
  )
}

/** Landing per role: ADMIN → service-point management, EXECUTIVE → the overview dashboard, STAFF → the queue console. */
function landingFor(identity: { roles: string[] }): string {
  if (identity.roles.includes('ADMIN')) return '/staff/service-points'
  if (identity.roles.includes('EXECUTIVE')) return '/staff/overview'
  return '/staff/queue'
}
