import { Link } from 'react-router'
import { COPY } from '../config/copy'

export default function NotFound() {
  return (
    <div className="text-center py-5">
      <h1 className="h3">{COPY.notFound.title}</h1>
      <p className="text-body-secondary">{COPY.notFound.body}</p>
      <Link to="/" className="btn btn-primary">
        {COPY.notFound.back}
      </Link>
    </div>
  )
}
