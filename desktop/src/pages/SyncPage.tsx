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
} from '@mantine/core'
import { DateInput } from '@mantine/dates'
import { notifications } from '@mantine/notifications'
import { IconDownload, IconUpload, IconCloudOff, IconAlertCircle, IconInfoCircle } from '@tabler/icons-react'
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

  // Import state
  const [importMode, setImportMode] = useState('merge')
  const [importing, setImporting] = useState(false)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [previewModalOpen, setPreviewModalOpen] = useState(false)
  const [previewReport, setPreviewReport] = useState<ImportReport | null>(null)
  const [importResult, setImportResult] = useState<ImportReport | null>(null)
  const [resultModalOpen, setResultModalOpen] = useState(false)
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

  const triggerDownload = (blob: Blob, filename: string) => {
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.click()
    URL.revokeObjectURL(url)
  }

  const handleExportAll = async () => {
    setExporting(true)
    try {
      if (isTauri() && isMobileDevice()) {
        const result = await api.exportBackupToFile()
        await shareFile(result.path, result.filename)
      } else {
        const blob = await api.exportBackup()
        triggerDownload(blob, `tidybill-backup-${new Date().toISOString().split('T')[0]}.tidybill`)
      }
      notifications.show({ title: t('backup.export_success'), message: '', color: 'green' })
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

      if (isTauri() && isMobileDevice()) {
        const result = await api.exportBackupToFile(filters)
        await shareFile(result.path, result.filename)
      } else {
        const blob = await api.exportBackup(filters)
        triggerDownload(blob, `tidybill-backup-${new Date().toISOString().split('T')[0]}.tidybill`)
      }
      notifications.show({ title: t('backup.export_success'), message: '', color: 'green' })
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

    selectedFileRef.current = file
    setPreviewLoading(true)
    try {
      const report = await api.previewImport(file)
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
      const result = await api.importBackup(selectedFileRef.current, importMode)
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
                <Table.Td>{name}</Table.Td>
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
            >
              {t('backup.export_all')}
            </Button>
            <Button
              variant="light"
              onClick={() => setFilterModalOpen(true)}
              disabled={exporting}
            >
              {t('backup.export_filtered')}
            </Button>
          </Group>
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
