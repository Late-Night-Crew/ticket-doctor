import { lazy, Suspense, type ReactNode } from 'react'
import { Route, Routes } from 'react-router'
import { AppLayout, PageSpinner } from './components/AppLayout'
import Home from './pages/Home'

// Lazy-load everything past the home page (NFR-2: small first load on 3G).
const Builder = lazy(() => import('./pages/Builder'))
const Analysis = lazy(() => import('./pages/Analysis'))
const Wizard = lazy(() => import('./pages/Wizard'))
const Results = lazy(() => import('./pages/Results'))
const Glossary = lazy(() => import('./pages/Glossary'))
const NotFound = lazy(() => import('./pages/NotFound'))
// The admin area (B6) is its own app shell and its own chunks: public visitors never download it.
const AdminLayout = lazy(() => import('./pages/admin/AdminLayout'))
const AdminLogin = lazy(() => import('./pages/admin/Login'))
const AdminDashboard = lazy(() => import('./pages/admin/Dashboard'))
const AdminResults = lazy(() => import('./pages/admin/ResultEntry'))
const AdminThresholds = lazy(() => import('./pages/admin/Thresholds'))
const Styleguide = import.meta.env.DEV ? lazy(() => import('./pages/Styleguide')) : null

export default function App() {
  return (
    <Routes>
      <Route path="admin/login" element={<Lazy><AdminLogin /></Lazy>} />
      <Route path="admin" element={<Lazy><AdminLayout /></Lazy>}>
        <Route index element={<AdminDashboard />} />
        <Route path="results" element={<AdminResults />} />
        <Route path="thresholds" element={<AdminThresholds />} />
        <Route path="*" element={<NotFound />} />
      </Route>
      <Route element={<AppLayout />}>
        <Route index element={<Home />} />
        <Route path="ticket" element={<Builder />} />
        <Route path="ticket/analysis" element={<Analysis />} />
        <Route path="wizard/*" element={<Wizard />} />
        <Route path="results" element={<Results />} />
        <Route path="glossary" element={<Glossary />} />
        {Styleguide && <Route path="dev/styleguide" element={<Styleguide />} />}
        <Route path="*" element={<NotFound />} />
      </Route>
    </Routes>
  )
}

/** Pages outside the public layout need their own loading fallback. */
function Lazy({ children }: { children: ReactNode }) {
  return <Suspense fallback={<PageSpinner />}>{children}</Suspense>
}
