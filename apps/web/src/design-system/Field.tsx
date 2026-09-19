import { cn } from 'cn'
import { Label } from 'radix-ui'
import { useId, type InputHTMLAttributes, type ReactNode } from 'react'

/*
 * Written by hand rather than taken from shadcn: shadcn ships Input and Label
 * as separate unstyled pieces and leaves the label/hint/error wiring to each
 * form. CarePath has few forms and they are all filled in at a counter with
 * someone waiting, so the label, the 44px target and the hint are one
 * component that cannot be assembled wrong.
 */

export type TextFieldProps = Omit<InputHTMLAttributes<HTMLInputElement>, 'id' | 'className'> & {
  label: string
  /** One line under the field: what to type, or where the value comes from. */
  hint?: ReactNode
  /** An action that belongs to the field itself, e.g. a search button. */
  trailing?: ReactNode
  className?: string
}

export function TextField({ label, hint, trailing, className, ...input }: TextFieldProps) {
  const id = useId()

  return (
    <div className={cn('flex flex-col gap-1.5', className)}>
      <Label.Root htmlFor={id} className="font-sans text-caption font-semibold text-ink">
        {label}
      </Label.Root>
      <div className="flex gap-2">
        <input
          id={id}
          className="h-11 min-w-0 flex-1 rounded-md border border-line bg-surface px-3 font-sans text-body-md text-ink outline-none placeholder:text-ink-muted focus-visible:border-secondary focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-secondary"
          {...input}
        />
        {trailing}
      </div>
      {hint && <p className="m-0 font-sans text-caption font-normal text-ink-muted">{hint}</p>}
    </div>
  )
}

/** A read-back of something the system already knows — not an editable field. */
export function ReadoutField({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <span className="font-sans text-caption font-semibold text-ink-muted">{label}</span>
      <span className="font-sans text-body-md text-ink">{children}</span>
    </div>
  )
}
