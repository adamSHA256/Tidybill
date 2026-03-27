import { useState, useRef } from 'react'
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
  PasswordInput,
} from '@mantine/core'
import { DateInput } from '@mantine/dates'
import { notifications } from '@mantine/notifications'
import { IconDownload, IconUpload, IconCloudOff, IconAlertCircle, IconInfoCircle, IconKey } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { api, isTauri, isMobileDevice, shareFile, type ExportFilters, type ImportReport } from '../api/client'
import { useT } from '../i18n'
import { useIsMobile } from '../hooks/useIsMobile'

export function SyncPage() {
  const { t } = useT()
  const isMobile = useIsMobile()

  // Export state
  const [exporting, setExporting] = useState(false)
  const [filterModalOpen, setFilterModalOpen] = useState(false)
  const [filterSupplierIds, setFilterSupplierIds] = useState<string[]>([])
  const [filterSkipPaidYears, setFilterSkipPaidYears] = useState<number | ''>('')
  const [filterDateFrom, setFilterDateFrom] = useState<Date | null>(null)
  const [filterDateTo, setFilterDateTo] = useState<Date | null>(null)
  const [filterExcludeSettings, setFilterExcludeSettings] = useState(false)
  const [encryptExport, setEncryptExport] = useState(false)
  const [exportPassphrase, setExportPassphrase] = useState('')
  const [exportPassphraseConfirm, setExportPassphraseConfirm] = useState('')
  const [generatedMnemonic, setGeneratedMnemonic] = useState<string | null>(null)
  const [generatingMnemonic, setGeneratingMnemonic] = useState(false)

  // Import state
  const [importMode, setImportMode] = useState('merge')
  const [importing, setImporting] = useState(false)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [previewModalOpen, setPreviewModalOpen] = useState(false)
  const [previewReport, setPreviewReport] = useState<ImportReport | null>(null)
  const [importResult, setImportResult] = useState<ImportReport | null>(null)
  const [resultModalOpen, setResultModalOpen] = useState(false)
  const [importPassphrase, setImportPassphrase] = useState('')
  const [fileEncrypted, setFileEncrypted] = useState(false)
  const selectedFileRef = useRef<File | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

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

  const validatePassphrase = (pass: string): string | null => {
    if (pass.length < 8) return t('backup.passphrase_too_short')
    if (!/[^a-zA-Z0-9]/.test(pass)) return t('backup.passphrase_needs_special')
    return null
  }

  const passphraseValid = !encryptExport || (
    !validatePassphrase(exportPassphrase) &&
    exportPassphrase === exportPassphraseConfirm
  )

  const handleGenerateMnemonic = async () => {
    setGeneratingMnemonic(true)
    try {
      const { mnemonic } = await api.generateMnemonic()
      setExportPassphrase(mnemonic)
      setExportPassphraseConfirm(mnemonic)
      setGeneratedMnemonic(mnemonic)
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err)
      notifications.show({ title: t('common.error'), message, color: 'red' })
    } finally {
      setGeneratingMnemonic(false)
    }
  }

  // Returns the actual saved path/filename, or null if cancelled
  const triggerDownload = async (blob: Blob, filename: string): Promise<string | null> => {
    // On desktop Tauri, show native file save dialog
    if (isTauri() && !isMobileDevice()) {
      try {
        const { save } = await import('@tauri-apps/plugin-dialog')
        const { downloadDir } = await import('@tauri-apps/api/path')
        let defaultPath = filename
        try {
          const dlDir = await downloadDir()
          defaultPath = `${dlDir}/${filename}`
        } catch { /* fallback to just filename */ }
        const filePath = await save({
          defaultPath,
          filters: [{ name: 'TidyBill Backup', extensions: ['tidybill'] }],
        })
        if (!filePath) return null // user cancelled

        const { writeFile } = await import('@tauri-apps/plugin-fs')
        const arrayBuffer = await blob.arrayBuffer()
        await writeFile(filePath, new Uint8Array(arrayBuffer))
        return filePath
      } catch (err) {
        console.error('Native save dialog failed, falling back to download:', err)
      }
    }
    // Fallback: browser-style download
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.click()
    URL.revokeObjectURL(url)
    return filename
  }

  const handleExportAll = async () => {
    if (encryptExport && exportPassphrase !== exportPassphraseConfirm) {
      notifications.show({ title: t('common.error'), message: t('backup.passphrase_mismatch'), color: 'red' })
      return
    }
    setExporting(true)
    try {
      const passphrase = encryptExport ? exportPassphrase : undefined
      const filename = `tidybill-backup-${new Date().toISOString().split('T')[0]}.tidybill`
      if (isTauri() && isMobileDevice()) {
        const result = await api.exportBackupToFile(undefined, passphrase)
        await shareFile(result.path, result.filename)
        notifications.show({ title: t('backup.export_success'), message: '', color: 'green' })
      } else {
        const blob = await api.exportBackup(undefined, passphrase)
        const savedPath = await triggerDownload(blob, filename)
        if (!savedPath) return // user cancelled
        // Show the actual saved path (may differ from default filename)
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
    if (encryptExport && exportPassphrase !== exportPassphraseConfirm) {
      notifications.show({ title: t('common.error'), message: t('backup.passphrase_mismatch'), color: 'red' })
      return
    }
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

      const passphrase = encryptExport ? exportPassphrase : undefined
      const filename = `tidybill-backup-${new Date().toISOString().split('T')[0]}.tidybill`
      if (isTauri() && isMobileDevice()) {
        const result = await api.exportBackupToFile(filters, passphrase)
        await shareFile(result.path, result.filename)
        notifications.show({ title: t('backup.export_success'), message: '', color: 'green' })
      } else {
        const blob = await api.exportBackup(filters, passphrase)
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
    // Reset the input so the same file can be re-selected
    e.target.value = ''

    // Check if encrypted by reading first 6 bytes
    const header = await file.slice(0, 6).arrayBuffer()
    const magic = new TextDecoder().decode(new Uint8Array(header).slice(0, 5))
    const isEncrypted = magic === 'TBILL'
    setFileEncrypted(isEncrypted)
    selectedFileRef.current = file

    if (isEncrypted) {
      // Don't auto-preview — need passphrase first
      setImportPassphrase('')
      return
    }

    setPreviewLoading(true)
    try {
      const report = await api.previewImport(file, undefined, importMode)
      setPreviewReport(report)
      setPreviewModalOpen(true)
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err)
      notifications.show({ title: t('common.error'), message, color: 'red' })
    } finally {
      setPreviewLoading(false)
    }
  }

  const handlePreviewEncrypted = async () => {
    if (!selectedFileRef.current) return
    setPreviewLoading(true)
    try {
      const report = await api.previewImport(selectedFileRef.current, importPassphrase, importMode)
      setPreviewReport(report)
      setPreviewModalOpen(true)
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err)
      notifications.show({ title: t('common.error'), message, color: 'red' })
    } finally {
      setPreviewLoading(false)
    }
  }

  const handleImportConfirm = async () => {
    if (!selectedFileRef.current) return
    setImporting(true)
    setPreviewModalOpen(false)
    try {
      const passphrase = fileEncrypted ? importPassphrase : undefined
      const result = await api.importBackup(selectedFileRef.current, importMode, passphrase)
      setImportResult(result)
      setResultModalOpen(true)
      notifications.show({ title: t('backup.import_success'), message: '', color: 'green' })
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err)
      notifications.show({ title: t('common.error'), message, color: 'red' })
    } finally {
      setImporting(false)
    }
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

  return (
    <Container size="sm" py="xl">
      <Title order={isMobile ? 3 : 2} mb="lg">{t('backup.title')}</Title>

      {/* Export section */}
      <Paper p="md" radius="md" withBorder>
        <Stack gap="md">
          <Text fw={500}>{t('backup.export_title')}</Text>
          <Text c="dimmed" size="sm">{t('backup.export_desc')}</Text>
          <Group>
            <Button
              leftSection={<IconDownload size={16} />}
              onClick={handleExportAll}
              loading={exporting}
              disabled={!passphraseValid}
            >
              {t('backup.export_all')}
            </Button>
            <Button
              variant="light"
              onClick={() => setFilterModalOpen(true)}
              disabled={exporting || !passphraseValid}
            >
              {t('backup.export_filtered')}
            </Button>
          </Group>
          <Switch
            label={t('backup.encrypt')}
            checked={encryptExport}
            onChange={(e) => {
              setEncryptExport(e.currentTarget.checked)
              if (!e.currentTarget.checked) {
                setGeneratedMnemonic(null)
              }
            }}
          />
          {encryptExport && (
            <Stack gap="xs">
              <Group align="end">
                <PasswordInput
                  label={t('backup.passphrase')}
                  description={t('backup.passphrase_rules')}
                  value={exportPassphrase}
                  onChange={(e) => {
                    setExportPassphrase(e.currentTarget.value)
                    setGeneratedMnemonic(null)
                  }}
                  error={exportPassphrase ? validatePassphrase(exportPassphrase) : undefined}
                  style={{ flex: 1 }}
                />
                <Button
                  variant="light"
                  size="sm"
                  leftSection={<IconKey size={16} />}
                  onClick={handleGenerateMnemonic}
                  loading={generatingMnemonic}
                >
                  {t('backup.generate_mnemonic')}
                </Button>
              </Group>
              <PasswordInput
                label={t('backup.passphrase_confirm')}
                value={exportPassphraseConfirm}
                onChange={(e) => setExportPassphraseConfirm(e.currentTarget.value)}
                error={exportPassphrase !== exportPassphraseConfirm && exportPassphraseConfirm ? t('backup.passphrase_mismatch') : undefined}
              />
              {generatedMnemonic && (
                <Alert icon={<IconAlertCircle size={16} />} color="yellow">
                  <Text size="sm" fw={500} mb="xs">{t('backup.mnemonic_warning')}</Text>
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '4px 16px' }}>
                    {generatedMnemonic.split(' ').map((word, i) => (
                      <Text key={i} size="sm" ff="monospace">{i + 1}. {word}</Text>
                    ))}
                  </div>
                </Alert>
              )}
              <Text c="dimmed" size="xs">{t('backup.encrypt_warning')}</Text>
            </Stack>
          )}
        </Stack>
      </Paper>

      {/* Import section */}
      <Paper p="md" radius="md" withBorder mt="md">
        <Stack gap="md">
          <Text fw={500}>{t('backup.import_title')}</Text>
          <Text c="dimmed" size="sm">{t('backup.import_desc')}</Text>

          <Radio.Group value={importMode} onChange={setImportMode}>
            <Stack gap="xs">
              <Radio value="merge" label={t('backup.import_mode_merge')} description={t('backup.import_mode_merge_desc')} />
              <Radio value="replace" label={t('backup.import_mode_replace')} description={t('backup.import_mode_replace_desc')} />
              <Radio value="force" label={t('backup.import_mode_force')} description={t('backup.import_mode_force_desc')} />
            </Stack>
          </Radio.Group>

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
          {fileEncrypted && selectedFileRef.current && (
            <Stack gap="xs">
              <Alert icon={<IconAlertCircle size={16} />} color="blue">
                {t('backup.file_encrypted')}
              </Alert>
              <PasswordInput
                label={t('backup.import_passphrase')}
                value={importPassphrase}
                onChange={(e) => setImportPassphrase(e.currentTarget.value)}
              />
              <Button onClick={handlePreviewEncrypted} disabled={!importPassphrase} loading={previewLoading}>
                {t('backup.import_decrypt_preview')}
              </Button>
            </Stack>
          )}
        </Stack>
      </Paper>

      {/* Sync (coming soon) section */}
      <Paper p="md" radius="md" withBorder mt="md">
        <Stack gap="sm">
          <Group gap="xs">
            <IconCloudOff size={20} style={{ opacity: 0.5 }} />
            <Text fw={500} c="dimmed">{t('backup.sync_title')}</Text>
          </Group>
          <Text c="dimmed" size="sm">{t('backup.sync_coming_soon')}</Text>
        </Stack>
      </Paper>

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
