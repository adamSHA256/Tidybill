import { useState } from 'react'
import { Stack, Button, Text, Alert, Loader, Center } from '@mantine/core'
import { IconBrandGoogle, IconAlertCircle } from '@tabler/icons-react'
import { useT } from '../../i18n'
import { api, openInBrowser, type CloudTransportInfo } from '../../api/client'

interface GDriveStepProps {
  transports: CloudTransportInfo[]
  onClose: () => void
  onConnected: () => void
}

export function GDriveStep({ transports, onClose, onConnected }: GDriveStepProps) {
  const { t } = useT()
  const [waiting, setWaiting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleConnect = async () => {
    setError(null)
    setWaiting(true)
    try {
      const { auth_url } = await api.cloud.gdriveConnect()
      await openInBrowser(auth_url)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : String(err))
      setWaiting(false)
    }
  }

  // Check if connected (parent refetches transports every 5s)
  const gdrive = transports.find((t) => t.id === 'gdrive')
  if (gdrive?.status.connected) {
    onConnected()
    return null
  }

  return (
    <Stack gap="md">
      {!waiting ? (
        <>
          <Text size="sm">{t('cloud.gdrive.label')}</Text>
          {error && (
            <Alert color="red" icon={<IconAlertCircle size={16} />}>
              {error}
            </Alert>
          )}
          <Button
            leftSection={<IconBrandGoogle size={16} />}
            onClick={handleConnect}
          >
            {t('cloud.gdrive.open_browser')}
          </Button>
        </>
      ) : (
        <Stack gap="md">
          <Text size="sm">{t('cloud.gdrive.waiting')}</Text>
          <Center><Loader size="sm" /></Center>
          <Button variant="subtle" onClick={onClose}>Cancel</Button>
        </Stack>
      )}
    </Stack>
  )
}
