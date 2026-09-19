import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "cn"
import { Slot } from "radix-ui"
import * as React from "react"

/*
 * shadcn's Badge with CarePath's pill set. Two rules from DESIGN.md are baked
 * in: a status badge never uses an alert red/green (success and warning tints
 * carry the meaning instead), and a zone chip is always dark `ink` text
 * because every zone tint is pale. The zone fill itself is supplied by the
 * caller — see ZoneChip.
 */
const badgeVariants = cva(
  "inline-flex w-fit shrink-0 items-center justify-center whitespace-nowrap",
  {
    variants: {
      variant: {
        routable:
          "rounded-full px-2.5 py-1 font-sans text-caption font-bold bg-success-tint text-success",
        blocked:
          "rounded-full px-2.5 py-1 font-sans text-caption font-bold bg-warning-tint text-warning",
        busy: "rounded-full px-2.5 py-1 font-sans text-caption bg-warning-tint text-warning",
        quiet: "rounded-full px-2.5 py-1 font-sans text-caption bg-zone-support text-ink-muted",
        zone: "gap-1.5 rounded-full px-2.5 py-1 font-sans text-caption text-ink",
        queue:
          "rounded-sm px-2.5 py-[5px] font-code text-[14px] font-bold bg-primary-tint text-primary",
        ref: "rounded-full border border-line px-3 py-[5px] font-code text-label-code font-medium bg-surface text-ink-muted",
      },
    },
    defaultVariants: {
      variant: "quiet",
    },
  }
)

function Badge({
  className,
  variant,
  asChild = false,
  ...props
}: React.ComponentProps<"span"> &
  VariantProps<typeof badgeVariants> & { asChild?: boolean }) {
  const Comp = asChild ? Slot.Root : "span"

  return (
    <Comp
      data-slot="badge"
      data-variant={variant}
      className={cn(badgeVariants({ variant }), className)}
      {...props}
    />
  )
}

export { Badge, badgeVariants }
