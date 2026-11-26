// @vitest-environment node
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join } from 'node:path'

/** RG-5: no "sure win", "banker" or "guaranteed" anywhere in the app. */
const BANNED = [/sure[\s-]?win/i, /\bbankers?\b/i, /guarantee(d)?\b(?!s)/i]

function files(dir: string): string[] {
  return readdirSync(dir).flatMap((name: string) => {
    const p = join(dir, name)
    if (statSync(p).isDirectory()) return files(p)
    return /\.(tsx?|scss|css|html)$/.test(name) && !name.endsWith('.test.ts') ? [p] : []
  })
}

describe('responsible-gambling copy (RG-5)', () => {
  const root = join(process.cwd(), 'src')
  it.each(files(root).map((f) => [f.slice(root.length + 1), f]))('%s has no banned wording', (_rel, file) => {
    const text = readFileSync(file, 'utf8')
    for (const re of BANNED) expect(text).not.toMatch(re)
  })
})
