import { Route, Routes } from 'react-router'
import WizardEntry from './wizard/WizardEntry'
import WizardFlow from './wizard/WizardFlow'

/** Ticket wizard (FR-8): /wizard, /wizard/beginner, /wizard/guided. */
export default function Wizard() {
  return (
    <Routes>
      <Route index element={<WizardEntry />} />
      <Route path="beginner" element={<WizardFlow key="beginner" path="beginner" />} />
      <Route path="guided" element={<WizardFlow key="guided" path="guided" />} />
    </Routes>
  )
}
