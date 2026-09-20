import { cn } from 'cn'
import { RadioGroup } from 'radix-ui'
import type { ReactNode } from 'react'

/*
 * A radio group whose options are full cards. Built on Radix's RadioGroup
 * rather than a row of buttons so arrow-key roving, `aria-checked` and the
 * single tab stop come for free — this is the control a registration clerk
 * uses dozens of times a shift.
 *
 * Staff register: `md` radius, 1px line, no shadow. The selected card is
 * marked in node blue, never orange — orange is the patient's route.
 */

export type Choice = {
  value: string
  title: string
  description?: ReactNode
  /** A short figure on the right, e.g. "5 steps". */
  meta?: ReactNode
  icon?: ReactNode
}

export type ChoiceCardsProps = {
  /** Describes the group for screen readers, e.g. a care-plan group. */
  label: string
  options: Choice[]
  value: string
  onValueChange: (value: string) => void
  /** Two columns from the console breakpoint up. */
  columns?: 1 | 2
  className?: string
}

export function ChoiceCards({
  label,
  options,
  value,
  onValueChange,
  columns = 1,
  className,
}: ChoiceCardsProps) {
  return (
    <RadioGroup.Root
      aria-label={label}
      value={value}
      onValueChange={onValueChange}
      className={cn('grid gap-2', columns === 2 && '@3xl:grid-cols-2', className)}
    >
      {options.map((option) => (
        <RadioGroup.Item
          key={option.value}
          value={option.value}
          className="group flex min-h-11 cursor-pointer items-start gap-3 rounded-md border border-line bg-surface p-3 text-left outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-secondary data-[state=checked]:border-secondary data-[state=checked]:bg-secondary-tint"
        >
          <span
            aria-hidden="true"
            className="mt-0.5 flex size-[18px] shrink-0 items-center justify-center rounded-full border-2 border-line group-data-[state=checked]:border-secondary"
          >
            <span className="size-2 rounded-full bg-transparent group-data-[state=checked]:bg-secondary" />
          </span>

          {option.icon && (
            <span className="mt-px shrink-0 text-ink-muted group-data-[state=checked]:text-secondary">
              {option.icon}
            </span>
          )}

          <span className="min-w-0 flex-1">
            <span className="block font-sans text-body-md font-bold text-ink">{option.title}</span>
            {option.description && (
              <span className="mt-0.5 block font-sans text-caption font-normal text-ink-muted">
                {option.description}
              </span>
            )}
          </span>

          {option.meta && (
            <span className="shrink-0 font-code text-label-code text-ink-muted">{option.meta}</span>
          )}
        </RadioGroup.Item>
      ))}
    </RadioGroup.Root>
  )
}
