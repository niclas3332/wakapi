import type { ReactNode } from 'react'
import clsx from 'clsx'

type CardProps = {
  title?: ReactNode
  /** Small note beside the title, e.g. how many entries are shown. */
  meta?: ReactNode
  /** Controls aligned to the right of the header. */
  actions?: ReactNode
  className?: string
  bodyClassName?: string
  children: ReactNode
}

export function Card({ title, meta, actions, className, bodyClassName, children }: CardProps) {
  return (
    <section
      className={clsx(
        'rounded-lg border border-ink-700 bg-ink-850 shadow-[0_1px_0_0_rgba(255,255,255,0.03)_inset]',
        className,
      )}
    >
      {(title || actions) && (
        <header className="flex items-center gap-3 border-b border-ink-700 px-4 py-3">
          <h2 className="text-sm font-semibold text-fg">{title}</h2>
          {meta && <span className="eyebrow truncate">{meta}</span>}
          {actions && <div className="ml-auto flex items-center gap-2">{actions}</div>}
        </header>
      )}
      <div className={clsx('p-4', bodyClassName)}>{children}</div>
    </section>
  )
}
