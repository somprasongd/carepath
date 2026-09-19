import { fileURLToPath } from 'node:url'
import tailwindcss from '@tailwindcss/vite'
import { tanstackRouter } from '@tanstack/router-plugin/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
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
