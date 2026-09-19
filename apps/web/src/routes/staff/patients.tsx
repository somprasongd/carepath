import { createFileRoute } from '@tanstack/react-router'
import { Patients } from './-Patients'

export const Route = createFileRoute('/staff/patients')({
  component: Patients,
})
