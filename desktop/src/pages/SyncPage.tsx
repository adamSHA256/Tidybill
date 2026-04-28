import { useState, useRef, useEffect } from 'react'
import {
  Container,
  Paper,
  Stack,
  Text,
  Title,
  Button,
  Group,
  Modal,
  Radio,
  Table,
  NumberInput,
  Switch,
  MultiSelect,
  Alert,
  Loader,
  Center,
  Tooltip,
} from '@mantine/core'
import { DateInput } from '@mantine/dates'
import { notifications } from '@mantine/notifications'
import { IconDownload, IconUpload, IconAlertCircle, IconInfoCircle, IconCloudUpload, IconShieldLock } from '@tabler/icons-react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { api, isTauri, isMobileDevice, shareFile, type ExportFilters, type ImportReport, type CloudTransportInfo, type CloudBlobRef } from '../api/client'
import { useT } from '../i18n'
import { useIsMobile } from '../hooks/useIsMobile'
import { CloudSyncPanel } from '../components/CloudSyncPanel'
import { ConnectCloudModal } from '../components/ConnectCloudModal'

function translateBackendError(msg: string, t: (key: string) => string): string {
  if (msg.includes('wrong passphrase or corrupted')) return t('error.wrong_passphrase')
  if (msg.includes('passphrase required')) return t('error.passphrase_required')
  if (msg.includes('passphrase must be at least')) return t('error.passphrase_too_short')
  return msg
}

function getTransportLabel(id: string, t: (key: string) => string): string {
  if (id === 'local') return t('cloud.local.label')
  if (id === 'gdrive') return t('cloud.gdrive.label')
  if (id.startsWith('rclone:')) {
    const backend = id.replace('rclone:', '')
    return t(`cloud.rclone.${backend}.label`) || id
  }
  return id
}

export function SyncPage() {
  const { t } = useT()
  const isMobile = useIsMobile()
  const navigate = useNavigate()

  // Export state
  const [exporting, setExporting] = useState(false)
  const [exportDestination, setExportDestination] = useState('local')
  const [filterModalOpen, setFilterModalOpen] = useState(false)
  const [filterSupplierIds, setFilterSupplierIds] = useState<string[]>([])
  const [filterSkipPaidYears, setFilterSkipPaidYears] = useState<number | ''>('')
  const [filterDateFrom, setFilterDateFrom] = useState<Date | null>(null)
  const [filterDateTo, setFilterDateTo] = useState<Date | null>(null)
  const [filterExcludeSettings, setFilterExcludeSettings] = useState(false)
  const [encryptExport, setEncryptExport] = useState(false)

  // Import state
  const [importSource, setImportSource] = useState('local')
  const [importMode, setImportMode] = useState('merge')
  const [importing, setImporting] = useState(false)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [previewModalOpen, setPreviewModalOpen] = useState(false)
  const [previewReport, setPreviewReport] = useState<ImportReport | null>(null)
  const [importResult, setImportResult] = useState<ImportReport | null>(null)
  const [resultModalOpen, setResultModalOpen] = useState(false)
  const selectedFileRef = useRef<File | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Cloud import state
  const [selectedCloudBlob, setSelectedCloudBlob] = useState<CloudBlobRef | null>(null)
  const [cloudImportError, setCloudImportError] = useState<'not_configured' | 'wrong_key' | null>(null)
  const [isCloudImportFlow, setIsCloudImportFlow] = useState(false)

  // Cloud sync state
  const [connectOpen, setConnectOpen] = useState(false)
  const qc = useQueryClient()

  const { data: transports, isLoading: transportsLoading } = useQuery({
    queryKey: ['cloud-transports'],
    queryFn: () => api.cloud.transports().then((r) => r.transports),
    refetchInterval: 5_000,
    placeholderData: () => {
      try {
        const cached = localStorage.getItem('cloud-transports-cache')
        return cached ? (JSON.parse(cached) as CloudTransportInfo[]) : undefined
      } catch {
        return undefined
      }
    },
  })

  useEffect(() => {
    if (transports && transports.length > 0) {
      try {
        localStorage.setItem('cloud-transports-cache', JSON.stringify(transports))
      } catch { /* ignore */ }
    }
  }, [transports])

  const { data: settings } = useQuery({
    queryKey: ['settings'],
    queryFn: api.getSettings,
  })

  const { data: masterKeyStatus } = useQuery({
    queryKey: ['master-key-status'],
    queryFn: api.masterKey.status,
    refetchInterval: false,
  })
  const masterKeyConfigured = masterKeyStatus?.configured ?? false

  // Cloud blob list for Import panel (fetched when a cloud source is selected)
  const { data: cloudBlobList, isLoading: cloudBlobsLoading } = useQuery({
    queryKey: ['cloud-list', importSource],
    queryFn: () => api.cloud.list(importSource).then((r) => r.blobs),
    enabled: importSource !== 'local',
  })

  const disconnect = useMutation({
    mutationFn: async (transportId: string) => {
      if (transportId === 'gdrive') {
        return api.cloud.gdriveDisconnect()
      }
      const backend = transportId.replace('rclone:', '')
      return api.cloud.rcloneDisconnect(backend)
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['cloud-transports'] })
    },
  })

  const { data: suppliers } = useQuery({
    queryKey: ['suppliers'],
    queryFn: api.getSuppliers,
  })

  const supplierOptions = (suppliers || []).map((s) => ({
    value: s.id,
    label: s.name,
  }))

  const formatDateStr = (d: Date): string => {
    return d.toISOString().split('T')[0]
  }

  // Returns the actual saved path/filename, or null if cancelled
  const triggerDownload = async (blob: Blob, filename: string): Promise<string | null> => {
    if (isTauri() && !isMobileDevice()) {
      try {
        const { save } = await import('@tauri-apps/plugin-dialog')
        const { downloadDir } = await import('@tauri-apps/api/path')
        let defaultPath = filename
        try {
          // Prefer user-configured backup dir, fall back to system downloads
          const baseDir = settings?.default_backup_dir || await downloadDir()
          defaultPath = `${baseDir}/${filename}`
        } catch { /* fallback to just filename */ }
        const filePath = await save({
          defaultPath,
          filters: [{ name: 'TidyBill Backup', extensions: ['tidybill'] }],
        })
        if (!filePath) return null

        const { writeFile } = await import('@tauri-apps/plugin-fs')
        const arrayBuffer = await blob.arrayBuffer()
        await writeFile(filePath, new Uint8Array(arrayBuffer))
        return filePath
      } catch (err) {
        console.error('Native save dialog failed, falling back to download:', err)
      }
    }
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.click()
    URL.revokeObjectURL(url)
    return filename
  }

  const handleExportAll = async () => {
    setExporting(true)
    try {
      if (exportDestination !== 'local') {
        await api.cloud.upload(exportDestination)
        notifications.show({ title: t('cloud.upload.success'), message: '', color: 'green' })
        return
      }
      const filename = `tidybill-backup-${new Date().toISOString().split('T')[0]}.tidybill`
      if (isTauri() && isMobileDevice()) {
        const result = await api.exportBackupToFile(undefined, undefined, encryptExport || undefined)
        await shareFile(result.path, result.filename)
        notifications.show({ title: t('backup.export_success'), message: '', color: 'green' })
      } else {
        const blob = await api.exportBackup(undefined, undefined, encryptExport || undefined)
        const savedPath = await triggerDownload(blob, filename)
        if (!savedPath) return
        const savedName = savedPath.includes('/') ? savedPath.split('/').pop() : savedPath
        notifications.show({ title: t('backup.export_success'), message: savedName || '', color: 'green' })
      }
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err)
      notifications.show({ title: t('common.error'), message, color: 'red' })
    } finally {
      setExporting(false)
    }
  }

  const handleExportFiltered = async () => {
    setExporting(true)
    setFilterModalOpen(false)
    try {
      const filters: ExportFilters = {}
      if (filterSupplierIds.length > 0) filters.supplier_ids = filterSupplierIds
      if (typeof filterSkipPaidYears === 'number' && filterSkipPaidYears > 0) {
        filters.skip_paid_older_than_years = filterSkipPaidYears
      }
      if (filterDateFrom) filters.date_from = formatDateStr(filterDateFrom)
      if (filterDateTo) filters.date_to = formatDateStr(filterDateTo)
      if (filterExcludeSettings) filters.exclude_settings = true

      const filename = `tidybill-backup-${new Date().toISOString().split('T')[0]}.tidybill`
      if (isTauri() && isMobileDevice()) {
        const result = await api.exportBackupToFile(filters, undefined, encryptExport || undefined)
        await shareFile(result.path, result.filename)
        notifications.show({ title: t('backup.export_success'), message: '', color: 'green' })
      } else {
        const blob = await api.exportBackup(filters, undefined, encryptExport || undefined)
        const savedPath = await triggerDownload(blob, filename)
        if (!savedPath) return
        const savedName = savedPath.includes('/') ? savedPath.split('/').pop() : savedPath
        notifications.show({ title: t('backup.export_success'), message: savedName || '', color: 'green' })
      }
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err)
      notifications.show({ title: t('common.error'), message, color: 'red' })
    } finally {
      setExporting(false)
    }
  }

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    e.target.value = ''

    selectedFileRef.current = file
    setPreviewLoading(true)
    try {
      const report = await api.previewImport(file, undefined, importMode)
      setPreviewReport(report)
      setIsCloudImportFlow(false)
      setPreviewModalOpen(true)
    } catch (err: unknown) {
      const raw = err instanceof Error ? err.message : String(err)
      notifications.show({ title: t('common.error'), message: translateBackendError(raw, t), color: 'red' })
    } finally {
      setPreviewLoading(false)
    }
  }

  const handleCloudBlobPreview = async () => {
    if (!selectedCloudBlob || importSource === 'local') return
    setCloudImportError(null)
    setPreviewLoading(true)
    try {
      const report = await api.cloud.downloadPreview(
        importSource,
        selectedCloudBlob.id,
        undefined,
        importMode,
      )
      setPreviewReport(report)
      setIsCloudImportFlow(true)
      setPreviewModalOpen(true)
    } catch (err: unknown) {
      const raw = err instanceof Error ? err.message : String(err)
      if (raw.includes('passphrase required') || raw.includes('master_key_not_configured')) {
        setCloudImportError('not_configured')
      } else if (raw.includes('wrong passphrase or corrupted')) {
        setCloudImportError('wrong_key')
      } else {
        notifications.show({ title: t('common.error'), message: raw, color: 'red' })
      }
    } finally {
      setPreviewLoading(false)
    }
  }

  const handleImportConfirm = async () => {
    setImporting(true)
    setPreviewModalOpen(false)
    try {
      if (isCloudImportFlow && selectedCloudBlob) {
        const result = await api.cloud.downloadApply(
          importSource,
          selectedCloudBlob.id,
          undefined,
          importMode,
        )
        setImportResult(result)
        setResultModalOpen(true)
        notifications.show({ title: t('backup.import_success'), message: '', color: 'green' })
      } else {
        if (!selectedFileRef.current) return
        const result = await api.importBackup(selectedFileRef.current, importMode, undefined)
        setImportResult(result)
        setResultModalOpen(true)
        notifications.show({ title: t('backup.import_success'), message: '', color: 'green' })
      }
    } catch (err: unknown) {
      const raw = err instanceof Error ? err.message : String(err)
      notifications.show({ title: t('common.error'), message: translateBackendError(raw, t), color: 'red' })
    } finally {
      setImporting(false)
    }
  }

  const formatBlobSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / 1024 / 1024).toFixed(1)} MB`
  }

  const tableLabels: Record<string, string> = {
    suppliers: t('backup.table_suppliers'),
    bank_accounts: t('backup.table_bank_accounts'),
    customers: t('backup.table_customers'),
    invoices: t('backup.table_invoices'),
    invoice_items: t('backup.table_invoice_items'),
    items: t('backup.table_items'),
    customer_items: t('backup.table_customer_items'),
    pdf_templates: t('backup.table_pdf_templates'),
    settings: t('backup.table_settings'),
    vat_rates: t('backup.table_vat_rates'),
    smtp_configs: t('backup.table_smtp_configs'),
  }

  const renderReportTable = (report: ImportReport) => {
    const tableNames = report.details ? Object.keys(report.details) : []
    return (
      <Table striped highlightOnHover>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>{t('backup.table_header_table')}</Table.Th>
            <Table.Th ta="right">{t('backup.table_header_inserted')}</Table.Th>
            <Table.Th ta="right">{t('backup.table_header_updated')}</Table.Th>
            <Table.Th ta="right">{t('backup.table_header_skipped')}</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {tableNames.map((name) => {
            const row = report.details[name]
            return (
              <Table.Tr key={name}>
                <Table.Td>{tableLabels[name] || name}</Table.Td>
                <Table.Td ta="right">{row.insert}</Table.Td>
                <Table.Td ta="right">{row.update}</Table.Td>
                <Table.Td ta="right">{row.skip}</Table.Td>
              </Table.Tr>
            )
          })}
        </Table.Tbody>
        <Table.Tfoot>
          <Table.Tr fw={700}>
            <Table.Td fw={700}>Total</Table.Td>
            <Table.Td ta="right" fw={700}>{report.summary.to_insert}</Table.Td>
            <Table.Td ta="right" fw={700}>{report.summary.to_update}</Table.Td>
            <Table.Td ta="right" fw={700}>{report.summary.to_skip}</Table.Td>
          </Table.Tr>
        </Table.Tfoot>
      </Table>
    )
  }

  const connectedTransports = (transports ?? []).filter((tr) => tr.status.connected)

  return (
    <Container size="sm" py="xl">
      <Title order={isMobile ? 3 : 2} mb="lg">{t('backup.title')}</Title>

      {!masterKeyConfigured && (
        <Alert icon={<IconShieldLock size={16} />} color="yellow" mb="md">
          <Group gap="xs" wrap="nowrap">
            <span>{t('banner.no_master_key')}</span>
            <Button
              size="compact-xs"
              variant="subtle"
              color="yellow"
              onClick={() => navigate('/settings#master-key')}
            >
              {t('banner.no_master_key_action')}
            </Button>
          </Group>
        </Alert>
      )}

      {/* Export section */}
      <Paper p="md" radius="md" withBorder>
        <Stack gap="md">
          <Text fw={500}>{t('backup.export_title')}</Text>
          <Text c="dimmed" size="sm">{t('backup.export_desc')}</Text>

          {/* Destination picker */}
          <Radio.Group
            label={t('backup.dest_label')}
            value={exportDestination}
            onChange={setExportDestination}
          >
            <Stack gap="xs" mt="xs">
              <Radio value="local" label={t('backup.dest_local')} />
              {connectedTransports.map((tr) => (
                <Radio
                  key={tr.id}
                  value={tr.id}
                  label={getTransportLabel(tr.id, t)}
                />
              ))}
            </Stack>
          </Radio.Group>

          <Group>
            <Button
              leftSection={exportDestination === 'local' ? <IconDownload size={16} /> : <IconCloudUpload size={16} />}
              onClick={handleExportAll}
              loading={exporting}
              disabled={encryptExport && !masterKeyConfigured}
            >
              {exportDestination === 'local'
                ? t('backup.export_all')
                : t('cloud.upload_action')}
            </Button>
            {exportDestination === 'local' && (
              <Button
                variant="light"
                onClick={() => setFilterModalOpen(true)}
                disabled={exporting || (encryptExport && !masterKeyConfigured)}
              >
                {t('backup.export_filtered')}
              </Button>
            )}
          </Group>

          <Switch
            label={t('backup.encrypt')}
            checked={encryptExport}
            onChange={(e) => setEncryptExport(e.currentTarget.checked)}
          />
          {encryptExport && !masterKeyConfigured && (
            <Alert icon={<IconAlertCircle size={16} />} color="yellow">
              {t('backup.encrypt_master_disabled')}
            </Alert>
          )}
        </Stack>
      </Paper>

      {/* Import section */}
      <Paper p="md" radius="md" withBorder mt="md">
        <Stack gap="md">
          <Text fw={500}>{t('backup.import_title')}</Text>
          <Text c="dimmed" size="sm">{t('backup.import_desc')}</Text>

          {/* Source picker */}
          <Radio.Group
            label={t('backup.source_label')}
            value={importSource}
            onChange={(val) => {
              setImportSource(val)
              setSelectedCloudBlob(null)
              setCloudImportError(null)
              selectedFileRef.current = null
            }}
          >
            <Stack gap="xs" mt="xs">
              <Radio value="local" label={t('backup.source_local')} />
              {connectedTransports.map((tr) => (
                <Radio
                  key={tr.id}
                  value={tr.id}
                  label={getTransportLabel(tr.id, t)}
                />
              ))}
            </Stack>
          </Radio.Group>

          <Radio.Group value={importMode} onChange={setImportMode}>
            <Stack gap="xs">
              <Radio value="merge" label={t('backup.import_mode_merge')} description={t('backup.import_mode_merge_desc')} />
              <Radio value="replace" label={t('backup.import_mode_replace')} description={t('backup.import_mode_replace_desc')} />
              <Radio value="force" label={t('backup.import_mode_force')} description={t('backup.import_mode_force_desc')} />
            </Stack>
          </Radio.Group>

          {/* Local import */}
          {importSource === 'local' && (
            <>
              <input
                type="file"
                ref={fileInputRef}
                accept=".tidybill"
                style={{ display: 'none' }}
                onChange={handleFileSelect}
              />
              <Button
                leftSection={previewLoading ? <Loader size={16} /> : <IconUpload size={16} />}
                variant="light"
                onClick={() => fileInputRef.current?.click()}
                loading={previewLoading || importing}
              >
                {t('backup.import_select')}
              </Button>
            </>
          )}

          {/* Cloud import */}
          {importSource !== 'local' && (
            <Stack gap="sm">
              {cloudBlobsLoading ? (
                <Center><Loader size="sm" /></Center>
              ) : !cloudBlobList || cloudBlobList.length === 0 ? (
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
                    {cloudBlobList.map((blob) => (
                      <Table.Tr
                        key={blob.id}
                        style={{
                          cursor: 'pointer',
                          background: selectedCloudBlob?.id === blob.id ? 'var(--mantine-color-blue-light)' : undefined,
                        }}
                        onClick={() => setSelectedCloudBlob(blob)}
                      >
                        <Table.Td>{blob.filename}</Table.Td>
                        <Table.Td>{new Date(blob.modified_at).toLocaleDateString()}</Table.Td>
                        <Table.Td>{formatBlobSize(blob.size)}</Table.Td>
                      </Table.Tr>
                    ))}
                  </Table.Tbody>
                </Table>
              )}

              {selectedCloudBlob && (
                <Stack gap="xs">
                  {cloudImportError === 'not_configured' && (
                    <Alert icon={<IconShieldLock size={16} />} color="yellow">
                      <Group gap="xs" wrap="nowrap">
                        <span>{t('cloud.restore.error_not_configured')}</span>
                        <Button size="compact-xs" variant="subtle" color="yellow" onClick={() => navigate('/settings#master-key')}>
                          {t('banner.no_master_key_action')}
                        </Button>
                      </Group>
                    </Alert>
                  )}
                  {cloudImportError === 'wrong_key' && (
                    <Alert icon={<IconAlertCircle size={16} />} color="red">
                      <Group gap="xs" wrap="nowrap">
                        <span>{t('cloud.restore.error_wrong_key')}</span>
                        <Button size="compact-xs" variant="subtle" color="red" onClick={() => navigate('/settings#master-key')}>
                          {t('master_key.change_btn')}
                        </Button>
                      </Group>
                    </Alert>
                  )}
                  <Button
                    variant="light"
                    leftSection={previewLoading ? <Loader size={16} /> : <IconDownload size={16} />}
                    onClick={handleCloudBlobPreview}
                    loading={previewLoading || importing}
                  >
                    {t('backup.cloud_load')}
                  </Button>
                </Stack>
              )}
            </Stack>
          )}
        </Stack>
      </Paper>

      {/* Cloud Sync section */}
      <Paper p="md" withBorder mt="md">
        <Stack gap="md">
          <Group justify="space-between">
            <Title order={4}>{t('cloud.title')}</Title>
            <Button leftSection={<IconCloudUpload size={16} />}
                    onClick={() => setConnectOpen(true)}>
              {t('cloud.connect_button')}
            </Button>
          </Group>
          <CloudSyncPanel
            transports={transports ?? []}
            isLoading={transportsLoading}
            onDisconnect={(id) => disconnect.mutate(id)}
          />
        </Stack>
      </Paper>

      <ConnectCloudModal
        opened={connectOpen}
        onClose={() => setConnectOpen(false)}
        onConnected={() => qc.invalidateQueries({ queryKey: ['cloud-transports'] })}
      />

      {/* Export filter modal */}
      <Modal
        opened={filterModalOpen}
        onClose={() => setFilterModalOpen(false)}
        title={t('backup.export_filtered')}
        size="md"
      >
        <Stack gap="md">
          <MultiSelect
            label={t('backup.filter_supplier')}
            data={supplierOptions}
            value={filterSupplierIds}
            onChange={setFilterSupplierIds}
            clearable
          />
          <Group align="end" gap="xs">
            <NumberInput
              label={t('backup.filter_skip_paid')}
              min={0}
              max={99}
              value={filterSkipPaidYears}
              onChange={(v) => setFilterSkipPaidYears(typeof v === 'number' ? v : '')}
              w={200}
            />
            <Text size="sm" pb={8}>{t('backup.filter_years')}</Text>
          </Group>
          {!isMobile && (
            <Group grow>
              <DateInput
                label={t('backup.filter_date_from')}
                value={filterDateFrom}
                onChange={(v) => setFilterDateFrom(v ? new Date(v) : null)}
                clearable
              />
              <DateInput
                label={t('backup.filter_date_to')}
                value={filterDateTo}
                onChange={(v) => setFilterDateTo(v ? new Date(v) : null)}
                clearable
              />
            </Group>
          )}
          <Group gap={4}>
            <Switch
              label={t('backup.filter_exclude_settings')}
              checked={filterExcludeSettings}
              onChange={(e) => setFilterExcludeSettings(e.currentTarget.checked)}
            />
            <Tooltip label={t('backup.filter_exclude_settings_hint')} multiline w={300} withArrow events={{ hover: true, focus: true, touch: true }}>
              <IconInfoCircle size={14} style={{ opacity: 0.5, cursor: 'help' }} />
            </Tooltip>
          </Group>
          <Group justify="flex-end">
            <Button variant="default" onClick={() => setFilterModalOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button onClick={handleExportFiltered} loading={exporting}>
              {t('backup.export_all')}
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Preview modal */}
      <Modal
        opened={previewModalOpen}
        onClose={() => setPreviewModalOpen(false)}
        title={t('backup.import_preview_title')}
        size="lg"
      >
        <Stack gap="md">
          {previewReport && (
            <>
              {renderReportTable(previewReport)}
              {previewReport.warnings && previewReport.warnings.length > 0 && (
                <Alert icon={<IconAlertCircle size={16} />} color="yellow">
                  {previewReport.warnings.map((w, i) => (
                    <Text key={i} size="sm">{w.description}</Text>
                  ))}
                </Alert>
              )}
            </>
          )}
          <Group justify="flex-end">
            <Button variant="default" onClick={() => setPreviewModalOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button onClick={handleImportConfirm} loading={importing}>
              {t('backup.import_confirm')}
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Result modal */}
      <Modal
        opened={resultModalOpen}
        onClose={() => setResultModalOpen(false)}
        title={t('backup.import_success')}
        size="lg"
      >
        <Stack gap="md">
          {importResult && (
            <>
              {renderReportTable(importResult)}
              {importResult.warnings && importResult.warnings.length > 0 && (
                <Alert icon={<IconAlertCircle size={16} />} color="yellow">
                  {importResult.warnings.map((w, i) => (
                    <Text key={i} size="sm">{typeof w === 'string' ? w : w.description}</Text>
                  ))}
                </Alert>
              )}
              {(importMode === 'force' || importMode === 'replace') && importResult.details?.smtp_configs && (importResult.details.smtp_configs.insert > 0 || importResult.details.smtp_configs.update > 0) && (
                <Alert icon={<IconAlertCircle size={16} />} color="orange">
                  <Text size="sm">{t('backup.import_smtp_warning')}</Text>
                </Alert>
              )}
            </>
          )}
          <Group justify="flex-end">
            <Button onClick={() => setResultModalOpen(false)}>
              OK
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Invisible loading overlay for import */}
      {importing && (
        <Center style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.3)', zIndex: 1000 }}>
          <Loader size="xl" />
        </Center>
      )}
    </Container>
  )
}
