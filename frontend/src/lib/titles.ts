import { COPY } from '../config/copy'

const TITLES: [prefix: string, title: string][] = [
  ['/admin/login', 'Admin sign-in'],
  ['/admin/results', 'Enter results · Admin'],
  ['/admin/thresholds', 'Rule thresholds · Admin'],
  ['/admin', 'Dashboard · Admin'],
  ['/ticket/analysis', 'Ticket analysis'],
  ['/ticket', COPY.nav.build],
  ['/wizard', COPY.wizard.entryTitle],
  ['/results', COPY.nav.results],
  ['/glossary', COPY.nav.glossary],
  ['/dev/styleguide', 'Styleguide'],
]

export function titleFor(pathname: string): string {
  if (pathname === '/') return `${COPY.appName}: ${COPY.tagline}`
  const match = TITLES.find(([prefix]) => pathname === prefix || pathname.startsWith(`${prefix}/`))
  return `${match ? match[1] : COPY.notFound.title} · ${COPY.appName}`
}
