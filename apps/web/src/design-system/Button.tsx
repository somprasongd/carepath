import type { ButtonHTMLAttributes, ReactNode } from 'react'

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

export function Button({
  variant = 'primary',
  href,
  block = false,
  className,
  children,
  type = 'button',
  ...rest
}: ButtonProps) {
  const classes = ['cp-btn', `cp-btn--${variant}`, block ? 'cp-btn--block' : '', className ?? '']
    .filter(Boolean)
    .join(' ')

  if (href) {
    return (
      <a className={classes} href={href}>
        {children}
      </a>
    )
  }

  return (
    <button className={classes} type={type} {...rest}>
      {children}
    </button>
  )
}
