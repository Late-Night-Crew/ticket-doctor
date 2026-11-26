import { useEffect, useState } from 'react'
import { IconMoon, IconSun } from './Icons'

type Theme = 'light' | 'dark'
const KEY = 'td.theme'

function initialTheme(): Theme {
  try {
    const saved = localStorage.getItem(KEY)
    if (saved === 'light' || saved === 'dark') return saved
  } catch {
    /* ignore */
  }
  return window.matchMedia?.('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export function ThemeToggle() {
  const [theme, setTheme] = useState<Theme>(initialTheme)

  useEffect(() => {
    document.documentElement.setAttribute('data-bs-theme', theme)
    try {
      localStorage.setItem(KEY, theme)
    } catch {
      /* ignore */
    }
  }, [theme])

  const next = theme === 'dark' ? 'light' : 'dark'
  return (
    <button
      type="button"
      className="btn btn-link td-theme-toggle text-body-secondary p-2"
      onClick={() => setTheme(next)}
      aria-label={`Switch to ${next} theme`}
    >
      {theme === 'dark' ? <IconSun width={20} height={20} /> : <IconMoon width={20} height={20} />}
    </button>
  )
}
