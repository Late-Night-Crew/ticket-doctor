import { PageHeader } from '../components/PageHeader'
import { GLOSSARY } from '../config/glossary'

export default function Glossary() {
  return (
    <>
      <PageHeader eyebrow="Glossary" title="Betting terms in plain words" />
      <dl className="td-surface p-3 mb-0">
        {GLOSSARY.map((g) => (
          <div key={g.id} id={g.id} className="py-2 border-bottom">
            <dt>{g.term}</dt>
            <dd className="mb-0 text-body-secondary">{g.body}</dd>
          </div>
        ))}
      </dl>
    </>
  )
}
