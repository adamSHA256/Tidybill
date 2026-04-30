import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { MantineProvider, createTheme } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { ModalsProvider } from '@mantine/modals'
import { DatesProvider } from '@mantine/dates'
import dayjs from 'dayjs'
import customParseFormat from 'dayjs/plugin/customParseFormat'
import 'dayjs/locale/cs'
import 'dayjs/locale/sk'

// Without this, dayjs ignores valueFormat ("DD.MM.YYYY") when parsing typed
// input, falls back to native Date which interprets dot-separated dates as
// MM.DD.YYYY — so "03.02.2026" (Feb 3) becomes March 2.
dayjs.extend(customParseFormat)
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from 'react-router-dom'
import { router } from './router'
import { I18nProvider } from './i18n'
import { ApiGate } from './components/ApiGate'

import '@mantine/core/styles.css'
import '@mantine/dates/styles.css'
import '@mantine/notifications/styles.css'

const theme = createTheme({
  primaryColor: 'tidybill',
  colors: {
    tidybill: [
      '#edf8f6',
      '#d8ede9',
      '#aedad2',
      '#7cc7b9',
      '#56b6a4',
      '#4A9E8E',
      '#3d8a7b',
      '#337568',
      '#296056',
      '#1f4c44',
    ],
  },
  fontFamily: 'system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
  defaultRadius: 'md',
})

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30 * 1000,
      retry: 1,
    },
  },
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <MantineProvider theme={theme} defaultColorScheme="light">
      <Notifications position="top-right" autoClose={5000} />
      <ApiGate>
        <QueryClientProvider client={queryClient}>
          <ModalsProvider>
            <DatesProvider settings={{ locale: 'cs' }}>
              <I18nProvider>
                <RouterProvider router={router} />
              </I18nProvider>
            </DatesProvider>
          </ModalsProvider>
        </QueryClientProvider>
      </ApiGate>
    </MantineProvider>
  </StrictMode>,
)
