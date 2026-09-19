import { createFileRoute } from '@tanstack/react-router'
import { StaffPlaceholder } from './-Placeholder'

export const Route = createFileRoute('/staff/patients')({
  component: () => (
    <StaffPlaceholder
      title="ผู้ป่วยวันนี้"
      note="รายชื่อผู้ป่วยและขั้นตอนปัจจุบันจะมาจาก features/visit เมื่อเชื่อม API จริงในขั้นถัดไป"
    />
  ),
})
