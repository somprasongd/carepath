import { fileURLToPath } from 'node:url'
import tailwindcss from '@tailwindcss/vite'
import { tanstackRouter } from '@tanstack/router-plugin/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  server: {
    proxy: {
      // Same-origin API in dev: the browser only ever talks to the Vite
      // server, which forwards /api to the backend — so one origin (and one
      // tunnel, for phone testing) serves the app. docker-compose points
      // API_PROXY_TARGET at the api service; native runs default to a local
      // API on :8080.
      '/api': {
        target: process.env.API_PROXY_TARGET ?? 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  plugins: [
    // Must run before the React plugin: it generates src/routeTree.gen.ts from
    // the files in src/routes. Files whose name starts with `-` are excluded
    // from routing, which is how a route colocates its own components.
    tanstackRouter({ target: 'react', autoCodeSplitting: true }),
    react(),
    tailwindcss(),
  ],
  resolve: {
    // `@/…` is what shadcn/ui generates into its components — keep it in sync
    // with `paths` in tsconfig.json and `aliases` in components.json.
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
})
