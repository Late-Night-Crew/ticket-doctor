/// <reference types="vitest/config" />
import * as purgeModule from '@fullhuman/postcss-purgecss'
import type { PurgeCSSUserDefinedOptions } from '@fullhuman/postcss-purgecss'
import type { AcceptedPlugin } from 'postcss'
import react from '@vitejs/plugin-react'
import { rm } from 'node:fs/promises'
import { resolve } from 'node:path'
import { defineConfig, loadEnv, type Plugin } from 'vite'

// CommonJS package whose types declare an ES default export: normalise both shapes.
type PurgePlugin = (opts: Omit<PurgeCSSUserDefinedOptions, 'css'>) => AcceptedPlugin
const purgeCSS = ((purgeModule as unknown as { default?: PurgePlugin }).default ??
  (purgeModule as unknown as PurgePlugin)) as PurgePlugin

/** The MSW worker is only needed when mocks are on; keep it out of real production builds. */
function dropMockWorker(useMocks: boolean): Plugin {
  let outDir = 'dist'
  return {
    name: 'drop-mock-worker',
    apply: 'build',
    configResolved(c) {
      outDir = resolve(c.root, c.build.outDir)
    },
    async closeBundle() {
      if (!useMocks) await rm(resolve(outDir, 'mockServiceWorker.js'), { force: true })
    },
  }
}

export default defineConfig(({ command, mode }) => ({
  // VITE_API_PROXY=http://localhost:8080 sends /api to a local Go API, so the
  // app and API share an origin as on Render (needed for the admin cookie).
  server: {
    proxy: loadEnv(mode, process.cwd(), 'VITE_').VITE_API_PROXY
      ? { '/api': loadEnv(mode, process.cwd(), 'VITE_').VITE_API_PROXY }
      : undefined,
  },
  // Read .env files too, so a mock-enabled preview deploy keeps its worker.
  plugins: [react(), dropMockWorker(loadEnv(mode, process.cwd(), 'VITE_').VITE_USE_MOCKS === 'true')],
  css: {
    preprocessorOptions: {
      scss: {
        // Bootstrap 5.3 still uses Sass @import and global functions.
        quietDeps: true,
        silenceDeprecations: ['import', 'global-builtin', 'color-functions', 'if-function'],
      },
    },
    // Production only: drop Bootstrap rules no component uses (NFR-2, slow 3G).
    postcss:
      command === 'build'
        ? {
            plugins: [
              purgeCSS({
                content: ['./index.html', './src/**/*.{ts,tsx}', './node_modules/react-bootstrap/esm/**/*.js'],
                // Bootstrap's class names include ":" and "/" in some utilities.
                defaultExtractor: (content: string) => content.match(/[\w-/:%.]+(?<!:)/g) ?? [],
                safelist: {
                  // Classes added at runtime by react-bootstrap transitions.
                  standard: ['show', 'showing', 'hiding', 'fade', 'collapsing', 'modal-open', /^modal/, /^offcanvas/, /^btn-close/],
                  // Our classes (some built dynamically, e.g. td-risk--${band}), theme and ARIA state selectors.
                  greedy: [/^td-/, /data-bs-theme/, /aria-/],
                },
                variables: false,
                keyframes: true,
              }),
            ],
          }
        : undefined,
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    css: false,
  },
}))
