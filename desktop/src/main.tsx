import { StrictMode, useEffect, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { MantineProvider, createTheme, Center, Loader, Stack, Text } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { ModalsProvider } from '@mantine/modals'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import App from './App'
import { I18nProvider } from './i18n'
import { initApiBase } from './api/client'

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
      staleTime: 5 * 60 * 1000,
      retry: 1,
    },
  },
})

function ApiGate({ children }: { children: React.ReactNode }) {
  const [ready, setReady] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    initApiBase()
      .then(() => setReady(true))
      .catch((err) => setError(err.message))
  }, [])

  if (error) {
    return (
      <Center h="100vh">
        <Stack align="center" gap="md">
          <Text size="xl" fw={700} c="red">Failed to connect to backend</Text>
          <Text c="dimmed">{error}</Text>
        </Stack>
      </Center>
    )
  }

  if (!ready) {
    return (
      <Center h="100vh">
        <Loader size="lg" />
      </Center>
    )
  }

  return <>{children}</>
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <MantineProvider theme={theme} defaultColorScheme="light">
      <Notifications position="top-right" />
      <ApiGate>
        <QueryClientProvider client={queryClient}>
          <ModalsProvider>
            <I18nProvider>
              <BrowserRouter>
                <App />
              </BrowserRouter>
            </I18nProvider>
          </ModalsProvider>
        </QueryClientProvider>
      </ApiGate>
    </MantineProvider>
  </StrictMode>,
)
