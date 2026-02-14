import { useEffect, useRef, useState } from 'react'
import { Overlay, Center, Stack, Text, Loader } from '@mantine/core'
import { IconPlugConnectedX } from '@tabler/icons-react'
import { checkHealth } from '../api/client'
import { useT } from '../i18n'

const POLL_INTERVAL = 5000
const FAILURE_THRESHOLD = 2

export function ApiHealthGuard({ children }: { children: React.ReactNode }) {
  const [disconnected, setDisconnected] = useState(false)
  const failCount = useRef(0)
  const { t } = useT()

  useEffect(() => {
    const interval = setInterval(async () => {
      const ok = await checkHealth()
      if (ok) {
        failCount.current = 0
        setDisconnected(false)
      } else {
        failCount.current++
        if (failCount.current >= FAILURE_THRESHOLD) {
          setDisconnected(true)
        }
      }
    }, POLL_INTERVAL)

    return () => clearInterval(interval)
  }, [])

  return (
    <>
      {children}
      {disconnected && (
        <Overlay fixed zIndex={1000} backgroundOpacity={0.85} color="#000">
          <Center h="100vh">
            <Stack align="center" gap="md">
              <IconPlugConnectedX size={64} color="var(--mantine-color-red-6)" />
              <Text size="xl" fw={700} c="white">
                {t('health.disconnected_title')}
              </Text>
              <Text c="dimmed" ta="center" maw={400}>
                {t('health.disconnected_message')}
              </Text>
              <Loader color="white" size="sm" mt="md" />
            </Stack>
          </Center>
        </Overlay>
      )}
    </>
  )
}
