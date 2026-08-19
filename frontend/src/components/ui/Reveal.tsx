import { useEffect, useRef, useState, type ReactNode } from 'react'
import clsx from 'clsx'

/**
 * Fades a section in the first time it is scrolled into view.
 *
 * Wrapped is read by scrolling from top to bottom, one statement at a time, and the
 * reveal is what makes it read that way instead of as a wall of numbers. It happens
 * once and never reverses - a section that fades back out while scrolling up would be
 * a distraction rather than an effect.
 *
 * Anyone who has asked for reduced motion gets the content immediately.
 */
export function Reveal({ children, className }: { children: ReactNode; className?: string }) {
  const ref = useRef<HTMLDivElement>(null)
  const [shown, setShown] = useState(false)

  useEffect(() => {
    const element = ref.current
    if (!element) return

    if (
      typeof IntersectionObserver === 'undefined' ||
      window.matchMedia('(prefers-reduced-motion: reduce)').matches
    ) {
      setShown(true)
      return
    }

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setShown(true)
          observer.disconnect()
        }
      },
      { rootMargin: '0px 0px -15% 0px' },
    )
    observer.observe(element)

    // Content must never end up permanently invisible because an observer did not
    // fire - a print, a screenshot, or a page that is never scrolled would otherwise
    // show blank bands where the numbers should be. The effect is a nicety; the
    // content is not.
    const fallback = window.setTimeout(() => setShown(true), 2000)

    return () => {
      observer.disconnect()
      window.clearTimeout(fallback)
    }
  }, [])

  return (
    <div
      ref={ref}
      className={clsx(
        'transition-[opacity,transform] duration-700 ease-out motion-reduce:transition-none',
        shown ? 'translate-y-0 opacity-100' : 'translate-y-4 opacity-0',
        className,
      )}
    >
      {children}
    </div>
  )
}
