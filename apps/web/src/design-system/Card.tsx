import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from 'cn'
import type { HTMLAttributes, ReactNode } from 'react'

/*
 * Written by hand rather than taken from shadcn: shadcn's Card is a
 * seven-part header/content/footer structure, and CarePath's is a plain
 * surface with two shape grammars. It also carries no shadow — DESIGN.md
 * reserves shadow for the sticky bar and the bottom sheet, the only two
 * things that genuinely float above scrolling content.
 */
const cardVariants = cva('border border-line bg-surface', {
  variants: {
    radius: {
      /* Patient register: soft. */
      lg: 'rounded-lg',
      /* Staff register: flatter, closer to a printed board. */
      md: 'rounded-md',
      /* The current step — the card the patient is "holding" right now. */
      ticket: 'rounded-ticket',
    },
    tone: {
      surface: '',
      attention: 'border-primary-edge bg-primary-tint',
    },
    padding: {
      none: 'p-0',
      md: 'p-3',
      lg: 'p-4',
      xl: 'p-5',
    },
  },
  defaultVariants: {
    radius: 'lg',
    tone: 'surface',
    padding: 'lg',
  },
})

export type CardProps = HTMLAttributes<HTMLDivElement> &
  VariantProps<typeof cardVariants> & {
    children: ReactNode
  }

export function Card({ radius, tone, padding, className, children, ...rest }: CardProps) {
  return (
    <div className={cn(cardVariants({ radius, tone, padding }), className)} {...rest}>
      {children}
    </div>
  )
}

export function Divider() {
  return <hr className="m-0 h-px border-0 bg-line" />
}
