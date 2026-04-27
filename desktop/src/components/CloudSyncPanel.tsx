import { Stack, Text, Group, Button, Badge, Paper } from '@mantine/core'
import { IconCloudUpload, IconDownload, IconUnlink } from '@tabler/icons-react'
import { useT } from '../i18n'
import type { CloudTransportInfo } from '../api/client'

interface CloudSyncPanelProps {
  transports: CloudTransportInfo[]
  isLoading?: boolean
  onUpload: (transportId: string) => void
  onRestore: (transportId: string) => void
  onDisconnect: (transportId: string) => void
}

export function CloudSyncPanel({ transports, isLoading, onUpload, onRestore, onDisconnect }: CloudSyncPanelProps) {
  const { t } = useT()

  if (transports.length === 0) {
    return (
      <Text c="dimmed" size="sm">{t('cloud.not_connected')}</Text>
    )
  }

  return (
    <Stack gap="xs">
      {transports.map((tr) => (
        <Paper key={tr.id} p="sm" withBorder radius="sm">
          <Group justify="space-between" wrap="nowrap">
            <Stack gap={2}>
              <Group gap="xs">
                <Text fw={500} size="sm">
                  {getTransportLabel(tr.id, t)}
                </Text>
                {isLoading ? (
                  <Badge color="yellow" size="xs">{t('cloud.status.connecting')}</Badge>
                ) : tr.status.connected ? (
                  <Badge color="green" size="xs">{t('cloud.status.connected')}</Badge>
                ) : (
                  <Badge color="red" size="xs">{t('cloud.status.disconnected')}</Badge>
                )}
              </Group>
              {tr.status.account_label && (
                <Text size="xs" c="dimmed">{tr.status.account_label}</Text>
              )}
              {tr.status.detail && (
                <Text size="xs" c="red">{tr.status.detail}</Text>
              )}
            </Stack>
            <Group gap="xs" wrap="nowrap">
              <Button
                size="xs"
                variant="light"
                leftSection={<IconCloudUpload size={14} />}
                onClick={() => onUpload(tr.id)}
                disabled={!tr.status.connected}
              >
                {t('cloud.upload_action')}
              </Button>
              <Button
                size="xs"
                variant="light"
                leftSection={<IconDownload size={14} />}
                onClick={() => onRestore(tr.id)}
                disabled={!tr.status.connected}
              >
                {t('cloud.restore_action')}
              </Button>
              {tr.id !== 'local' && (
                <Button
                  size="xs"
                  variant="subtle"
                  color="red"
                  leftSection={<IconUnlink size={14} />}
                  onClick={() => onDisconnect(tr.id)}
                >
                  {t('cloud.disconnect_action')}
                </Button>
              )}
            </Group>
          </Group>
        </Paper>
      ))}
    </Stack>
  )
}

function getTransportLabel(id: string, t: (key: string) => string): string {
  if (id === 'local') return t('cloud.local.label')
  if (id === 'gdrive') return t('cloud.gdrive.label')
  if (id.startsWith('rclone:')) {
    const backend = id.replace('rclone:', '')
    const key = `cloud.rclone.${backend}.label`
    return t(key) || id
  }
  return id
}
