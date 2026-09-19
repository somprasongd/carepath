import { createFileRoute } from '@tanstack/react-router'
import { DesignSystemPage } from './-design/DesignSystemPage'

/** The living version of DESIGN.md — see routes/-design/. */
export const Route = createFileRoute('/design')({
  component: DesignSystemPage,
})
