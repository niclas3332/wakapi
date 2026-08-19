import type { ReactNode, SelectHTMLAttributes, InputHTMLAttributes, ButtonHTMLAttributes } from 'react'
import { useId } from 'react'
import clsx from 'clsx'

const controlClasses =
  'w-full rounded border border-ink-700 bg-ink-900 px-3 py-2 text-sm text-fg placeholder:text-fg-subtle transition-colors focus:border-ink-600 disabled:opacity-50'

/**
 * A labelled control.
 *
 * The label is always a real <label> bound to its input, and the hint sits under the
 * control rather than inside the placeholder - a placeholder disappears exactly when
 * someone starts typing, which is when they need the hint most.
 */
export function Field({
  label,
  hint,
  children,
  className,
}: {
  label: string
  hint?: ReactNode
  children: (id: string) => ReactNode
  className?: string
}) {
  const id = useId()
  return (
    <div className={clsx('space-y-1.5', className)}>
      <label htmlFor={id} className="block text-xs font-medium text-fg-muted">
        {label}
      </label>
      {children(id)}
      {hint && <p className="text-xs text-fg-subtle">{hint}</p>}
    </div>
  )
}

export function TextInput({ className, ...props }: InputHTMLAttributes<HTMLInputElement>) {
  return <input {...props} className={clsx(controlClasses, className)} />
}

export function Select({ className, ...props }: SelectHTMLAttributes<HTMLSelectElement>) {
  return <select {...props} className={clsx(controlClasses, className)} />
}

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: 'primary' | 'secondary' | 'danger'
}

export function Button({ variant = 'secondary', className, ...props }: ButtonProps) {
  return (
    <button
      {...props}
      className={clsx(
        'rounded px-3 py-1.5 text-sm font-medium transition-colors disabled:opacity-50',
        variant === 'primary' && 'bg-accent-strong text-white hover:bg-accent',
        variant === 'secondary' &&
          'border border-ink-700 text-fg-muted hover:border-ink-600 hover:text-fg',
        // Destructive controls are outlined rather than filled: a solid red button
        // invites the click it is warning about.
        variant === 'danger' && 'border border-danger/50 text-danger hover:bg-danger/10',
        className,
      )}
    />
  )
}

/**
 * A boolean setting as a row: label and explanation on the left, switch on the right.
 * The whole row is the label, so the hit target is the row and not just the switch.
 */
export function Toggle({
  label,
  hint,
  checked,
  onChange,
  disabled,
}: {
  label: string
  hint?: ReactNode
  checked: boolean
  onChange: (value: boolean) => void
  disabled?: boolean
}) {
  return (
    <label
      className={clsx(
        'flex items-start gap-3 rounded px-2 py-2 transition-colors',
        disabled ? 'opacity-50' : 'cursor-pointer hover:bg-ink-800',
      )}
    >
      <input
        type="checkbox"
        checked={checked}
        disabled={disabled}
        onChange={(event) => onChange(event.target.checked)}
        className="mt-0.5 h-4 w-4 shrink-0 accent-[var(--color-accent-strong)]"
      />
      <span className="min-w-0">
        <span className="block text-sm text-fg">{label}</span>
        {hint && <span className="mt-0.5 block text-xs text-fg-subtle">{hint}</span>}
      </span>
    </label>
  )
}

/**
 * The outcome of the last action on a card, shown where the action happened.
 *
 * Deliberately not a floating toast: a toast appears away from the control and
 * disappears before it can be re-read, which for a settings screen is exactly wrong.
 */
export function ActionStatus({
  state,
  message,
}: {
  state: 'idle' | 'pending' | 'success' | 'error'
  message?: string
}) {
  if (state === 'idle' || !message) {
    return state === 'pending' ? <span className="text-xs text-fg-subtle">Saving…</span> : null
  }
  return (
    <span
      role="status"
      className={clsx('text-xs', state === 'error' ? 'text-danger' : 'text-fg-muted')}
    >
      {message}
    </span>
  )
}

/** Footer row of a settings card: status on the left, actions on the right. */
export function FormActions({ children, status }: { children: ReactNode; status?: ReactNode }) {
  return (
    <div className="mt-4 flex flex-wrap items-center gap-3 border-t border-ink-700 pt-3">
      {status}
      <div className="ml-auto flex items-center gap-2">{children}</div>
    </div>
  )
}
