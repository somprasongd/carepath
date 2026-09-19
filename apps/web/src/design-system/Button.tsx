import type { ButtonHTMLAttributes, ReactNode } from 'react'
import { Button as UIButton } from './ui/button'

export type ButtonVariant = 'primary' | 'secondary' | 'ghost'

export type ButtonProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'type'> & {
  /**
   * `primary` is the patient CTA — orange, dark ink, one per screen, never a
   * staff action. `secondary` is the staff primary. `ghost` is staff secondary.
   */
  variant?: ButtonVariant
  /** Renders an anchor instead of a button. */
  href?: string
  block?: boolean
  type?: 'button' | 'submit'
  children: ReactNode
}

/** Each variant carries its own height — see the size rules in ui/button.tsx. */
const sizeFor = { primary: 'lg', secondary: 'md', ghost: 'sm' } as const

export function Button({
  variant = 'primary',
  href,
  block = false,
  className,
  children,
  type = 'button',
  ...rest
}: ButtonProps) {
  if (href) {
    return (
      <UIButton asChild variant={variant} size={sizeFor[variant]} block={block} className={className}>
        <a href={href}>{children}</a>
      </UIButton>
    )
  }

  return (
    <UIButton
      type={type}
      variant={variant}
      size={sizeFor[variant]}
      block={block}
      className={className}
      {...rest}
    >
      {children}
    </UIButton>
  )
}

/** A text-only action — the quiet sibling of `ghost`. */
export function LinkButton({
  children,
  onClick,
  className,
}: {
  children: ReactNode
  onClick?: () => void
  className?: string
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`cursor-pointer border-0 bg-none p-0 text-caption font-bold text-secondary no-underline ${className ?? ''}`}
    >
      {children}
    </button>
  )
}
