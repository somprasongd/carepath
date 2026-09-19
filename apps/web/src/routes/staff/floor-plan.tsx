import { createFileRoute } from '@tanstack/react-router'
import { StaffPlaceholder } from './-Placeholder'

export const Route = createFileRoute('/staff/floor-plan')({
  component: () => (
    <StaffPlaceholder
      title="ผังอาคาร"
      note="ผัง SVG และกราฟนำทางอยู่ใน packages/floorplans แล้ว หน้านี้จะเรนเดอร์ผังและโหนดนำทางเมื่อเชื่อม features/hospitalmap"
    />
  ),
})
