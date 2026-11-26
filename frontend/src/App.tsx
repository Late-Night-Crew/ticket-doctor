import { lazy } from 'react'
import { Route, Routes } from 'react-router'
import { AppLayout } from './components/AppLayout'
import Home from './pages/Home'

// Lazy-load everything past the home page (NFR-2: small first load on 3G).
const Builder = lazy(() => import('./pages/Builder'))
const Analysis = lazy(() => import('./pages/Analysis'))
const Wizard = lazy(() => import('./pages/Wizard'))
const Results = lazy(() => import('./pages/Results'))
const Glossary = lazy(() => import('./pages/Glossary'))
const NotFound = lazy(() => import('./pages/NotFound'))
const Styleguide = import.meta.env.DEV ? lazy(() => import('./pages/Styleguide')) : null

export default function App() {
  return (
    <Routes>
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
