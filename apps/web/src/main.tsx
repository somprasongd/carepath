import { RouterProvider, createRouter } from '@tanstack/react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import React from 'react'
import ReactDOM from 'react-dom/client'
import { LocaleProvider } from './i18n'
import { resolveInitialLocale } from './i18n/initial-locale'
import { routeTree } from './routeTree.gen'
import './styles/index.css'

const router = createRouter({ routeTree })

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // A hospital visit moves on the scale of minutes, not seconds; the
      // user's own actions (retry, navigation) are the real refresh triggers.
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

function render(lineLanguage: string | undefined) {
  // Resolved once per boot, after liff.init() has run in LINE mode (see
  // bootstrap) — so a foreign patient's first paint is already English:
  // stored choice → LINE language → Thai (i18n/initial-locale.ts).
  const initialLocale = resolveInitialLocale(lineLanguage)
  ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
      <LocaleProvider initialLocale={initialLocale}>
        <QueryClientProvider client={queryClient}>
          <RouterProvider router={router} />
        </QueryClientProvider>
      </LocaleProvider>
    </React.StrictMode>,
  )
}

/**
 * LINE's OAuth redirect lands on `/` carrying `code`/`state` query params the
 * LIFF SDK needs to complete its token exchange. `routes/index.tsx`'s
 * `beforeLoad` redirect to `/patient/journey` doesn't preserve them, so if
 * `liff.init()` only ran later (inside the `/patient` layout, per
 * `auth/LiffAuthProvider.tsx`), the router would already have rewritten the
 * URL and dropped them first — `isLoggedIn()` would then never see the
 * completed login and `liff.login()` would fire again, looping forever.
 * Calling `init()` here, before the router ever mounts, gives LIFF first
 * look at the raw URL. `liff.init()` is safe to call again later (it's
 * idempotent), which is what LiffAuthProvider still does to read status.
 */
async function bootstrap() {
  const liffId = import.meta.env.VITE_LIFF_ID
  if (import.meta.env.VITE_AUTH_MODE === 'line' && liffId) {
    try {
      const { default: liff } = await import('@line/liff')
      await liff.init({ liffId })
      // Read here, after init, so the very first render can resolve the
      // boot locale (#94) — there is no flash of Thai for an English LINE
      // user. getLanguage is only meaningful post-init.
      render(liff.getLanguage())
      return
    } catch {
      // Swallowed here — LiffAuthProvider re-runs init() and surfaces the
      // error through the normal loading/error UI states.
    }
  }
  render(undefined)
}

void bootstrap()
