import { useState } from 'react'
import { Modal, Stack, Table, Text, Button, PasswordInput, Radio, Group, Alert, Loader, Center } from '@mantine/core'
import { IconAlertCircle } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { notifications } from '@mantine/notifications'
import { api, type CloudBlobRef, type ImportReport } from '../api/client'
import { useT } from '../i18n'

interface CloudRestoreModalProps {
  opened: boolean
  transportId: string | null
  onClose: () => void
}

export function CloudRestoreModal({ opened, transportId, onClose }: CloudRestoreModalProps) {
  const { t } = useT()
  const [selected, setSelected] = useState<CloudBlobRef | null>(null)
  const [passphrase, setPassphrase] = useState('')
  const [mode, setMode] = useState('merge')
  const [previewing, setPreviewing] = useState(false)
  const [importing, setImporting] = useState(false)
  const [previewReport, setPreviewReport] = useState<ImportReport | null>(null)
  const [error, setError] = useState<string | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['cloud-list', transportId],
    queryFn: () => transportId ? api.cloud.list(transportId).then((r) => r.blobs) : Promise.resolve([]),
    enabled: opened && transportId !== null,
  })

  const blobs = data || []

  const handleClose = () => {
    setSelected(null)
    setPassphrase('')
    setMode('merge')
    setPreviewReport(null)
    setError(null)
    onClose()
  }

  const handlePreview = async () => {
    if (!selected || !transportId) return
    setPreviewing(true)
    setError(null)
    try {
      const report = await api.cloud.downloadPreview(transportId, selected.id, passphrase || undefined, mode)
      setPreviewReport(report)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setPreviewing(false)
    }
  }

  const handleImport = async () => {
    if (!selected || !transportId) return
    setImporting(true)
    setError(null)
    try {
      await api.cloud.downloadApply(transportId, selected.id, passphrase || undefined, mode)
      notifications.show({
        title: t('cloud.restore.success'),
        message: selected.filename,
        color: 'green',
      })
      handleClose()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setImporting(false)
    }
  }

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / 1024 / 1024).toFixed(1)} MB`
  }

  return (
    <Modal
      opened={opened}
      onClose={handleClose}
      title={t('cloud.restore_action')}
      size="lg"
    >
      <Stack gap="md">
        {error && (
          <Alert color="red" icon={<IconAlertCircle size={16} />}>
            {error}
          </Alert>
        )}

        {isLoading ? (
          <Center><Loader size="sm" /></Center>
        ) : blobs.length === 0 ? (
          <Text c="dimmed" size="sm">{t('cloud.restore.empty')}</Text>
        ) : (
          <Table highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>File</Table.Th>
                <Table.Th>Date</Table.Th>
                <Table.Th>Size</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {blobs.map((blob) => (
                <Table.Tr
                  key={blob.id}
                  style={{ cursor: 'pointer', background: selected?.id === blob.id ? 'var(--mantine-color-blue-light)' : undefined }}
                  onClick={() => setSelected(blob)}
                >
                  <Table.Td>{blob.filename}</Table.Td>
                  <Table.Td>{new Date(blob.modified_at).toLocaleDateString()}</Table.Td>
                  <Table.Td>{formatSize(blob.size)}</Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        )}

        {selected && (
          <Stack gap="md">
            {(selected.encrypted || !selected.encrypted) && (
              <PasswordInput
                label={t('cloud.restore.encrypted_prompt')}
                value={passphrase}
                onChange={(e) => setPassphrase(e.currentTarget.value)}
                placeholder="Passphrase (if encrypted)"
              />
            )}

            <Radio.Group value={mode} onChange={setMode} label="Import mode">
              <Stack gap="xs" mt="xs">
                <Radio value="merge" label={t('cloud.restore.mode_merge')} />
                <Radio value="replace" label={t('cloud.restore.mode_replace')} />
                <Radio value="force" label={t('cloud.restore.mode_force')} />
              </Stack>
            </Radio.Group>

            {previewReport && (
              <Alert color="blue">
                Preview: {previewReport.summary.to_insert} insert, {previewReport.summary.to_update} update, {previewReport.summary.to_skip} skip
              </Alert>
            )}

            <Group>
              <Button variant="light" onClick={handlePreview} loading={previewing}>
                {t('cloud.restore.preview')}
              </Button>
              <Button onClick={handleImport} loading={importing}>
                {t('cloud.restore.apply')}
              </Button>
            </Group>
          </Stack>
        )}
      </Stack>
    </Modal>
  )
}
