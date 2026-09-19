import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "cn"
import { Slot } from "radix-ui"
import * as React from "react"

/*
 * shadcn's Button, with CarePath's variants in place of the stock set. The
 * rules encoded here come from DESIGN.md and are not style preferences:
 *
 * - `primary` is orange with DARK INK text, never white — white measures
 *   2.6:1 on this orange, ink ~5.5:1. It is the single patient CTA per
 *   screen, never a staff action.
 * - `secondary` (blue fill) is the staff primary; `ghost` the staff secondary.
 * - Every size clears 44px. Patients and staff both tap these while walking,
 *   so shadcn's stock h-9 (36px) is not an option here.
 */
const buttonVariants = cva(
  "inline-flex shrink-0 items-center justify-center gap-2 font-code whitespace-nowrap transition-colors outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-secondary disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        primary: "bg-primary text-ink rounded-lg text-button hover:bg-primary/90",
        secondary:
          "bg-secondary text-surface rounded-md text-button hover:bg-secondary/90",
        ghost:
          "bg-surface text-secondary border border-secondary rounded-sm text-caption font-bold hover:bg-secondary-tint",
      },
      size: {
        /* The patient CTA is taller than the 44px floor on purpose. */
        lg: "min-h-[50px] px-5",
        md: "min-h-[44px] px-4",
        sm: "min-h-[44px] px-3",
      },
      block: {
        true: "flex w-full",
      },
    },
    defaultVariants: {
      variant: "primary",
      size: "lg",
    },
  }
)

function Button({
  className,
  variant,
  size,
  block,
  asChild = false,
  ...props
}: React.ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean
  }) {
  const Comp = asChild ? Slot.Root : "button"

  return (
    <Comp
      data-slot="button"
      data-variant={variant}
      className={cn(buttonVariants({ variant, size, block, className }))}
      {...props}
    />
  )
}

export { Button, buttonVariants }
