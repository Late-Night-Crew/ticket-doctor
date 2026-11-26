/** Last-5 form, e.g. W-D-L-W-W (FR-2.4). */
export function FormString({ form }: { form?: string }) {
  if (!form) return null
  const parts = form.split('-')
  return (
    <span className="td-form" aria-label={`Last ${parts.length} results: ${form}`}>
      {parts.map((r, i) => (
        <span key={i} className={r.toLowerCase()}>
          {r}
          {i < parts.length - 1 && <span className="d">-</span>}
        </span>
      ))}
    </span>
  )
}
