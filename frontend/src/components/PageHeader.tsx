import type { ReactNode } from 'react'

export function PageHeader({ eyebrow, title, children }: { eyebrow?: string; title: string; children?: ReactNode }) {
  return (
    <header className="mb-4">
      {eyebrow && <div className="td-eyebrow mb-1">{eyebrow}</div>}
      <h1 className="h2 mb-2">{title}</h1>
      {children && <div className="text-body-secondary">{children}</div>}
    </header>
  )
}
