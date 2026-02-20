import { useEffect, useState } from 'react'
import { Center, Loader, Stack, Text } from '@mantine/core'
import { initApiBase, api } from '../api/client'
import { applyZoom } from '../utils/zoom'

export function ApiGate({ children }: { children: React.ReactNode }) {
  const [ready, setReady] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    initApiBase()
      .then(async () => {
        // Apply saved zoom level on startup
        try {
          const settings = await api.getSettings()
          const scale = parseFloat(settings.ui_scale || '1') || 1
          if (scale !== 1) await applyZoom(scale)
        } catch { /* ignore — settings will load later */ }
        setReady(true)
      })
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
