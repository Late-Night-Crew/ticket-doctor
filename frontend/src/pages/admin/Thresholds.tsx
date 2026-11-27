import { useThresholds, type Threshold } from '../../api/admin'
import { PageSpinner } from '../../components/AppLayout'
import { PageHeader } from '../../components/PageHeader'
import { pct } from '../../lib/format'

const show = (t: Threshold, v: number) => (t.unit === 'percent' ? pct(v) : v.toString())

/** The rule thresholds in use (NFR-6), read-only. */
export default function Thresholds() {
  const { data, isPending, isError, error } = useThresholds()
  if (isPending) return <PageSpinner />
  if (isError) return <div className="alert alert-danger">Couldn't load the thresholds: {error.message}</div>

  return (
    <>
      <PageHeader eyebrow="Admin" title="Rule thresholds">
        The numbers behind every estimate, risk band and suggestion. To change one, edit <code>rules:</code> in the
        backend config and release, so every change is reviewed and recorded. Check the track record before and after.
      </PageHeader>

      <div className="td-surface p-0 overflow-hidden">
        <div className="table-responsive">
          <table className="table align-middle mb-0">
            <thead>
              <tr>
                <th scope="col">Rule</th>
                <th scope="col" className="text-end">
                  In use
                </th>
                <th scope="col" className="text-end">
                  SRS default
                </th>
              </tr>
            </thead>
            <tbody>
              {data.map((t) => {
                const changed = t.value !== t.default
                return (
                  <tr key={t.key}>
                    <td>
                      <div className="fw-semibold">
                        {t.label}
                        {changed && <span className="badge text-bg-warning ms-2">changed</span>}
                      </div>
                      <div className="small text-body-secondary">{t.description}</div>
                      <code className="small">{t.key}</code>
                    </td>
                    <td className="text-end td-num fw-semibold">{show(t, t.value)}</td>
                    <td className="text-end td-num text-body-secondary">{show(t, t.default)}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </div>
    </>
  )
}
