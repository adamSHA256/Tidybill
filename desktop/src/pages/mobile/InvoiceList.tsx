import {
  Title,
  Text,
  Paper,
  Group,
  Badge,
  TextInput,
  Select,
  Stack,
  Loader,
  Center,
  ActionIcon,
} from '@mantine/core'
import { IconSearch, IconPlus } from '@tabler/icons-react'
import { useState, useMemo, useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api, formatMoney, formatDate, type Invoice } from '../../api/client'
import { useT } from '../../i18n'

type SortField = 'invoice_number' | 'issue_date' | 'created_at' | 'due_date' | 'total'
type SortDir = 'asc' | 'desc'

const DEFAULT_SORT_FIELD: SortField = 'created_at'

const statusColors: Record<string, string> = {
  draft: 'gray',
  created: 'blue',
  sent: 'yellow',
  paid: 'green',
  overdue: 'red',
  partially_paid: 'orange',
  cancelled: 'dimmed',
}

export function MobileInvoiceList() {
  const [searchParams] = useSearchParams()
  const [filter, setFilter] = useState<string | null>(searchParams.get('status') || 'all')
  const [search, setSearch] = useState('')
  const [supplierFilter, setSupplierFilter] = useState<string | null>(null)
  const [sortField, setSortField] = useState<SortField>(DEFAULT_SORT_FIELD)
  const [sortDir, setSortDir] = useState<SortDir>('desc')
  const navigate = useNavigate()
  const { t } = useT()

  const { data: settings } = useQuery({
    queryKey: ['settings'],
    queryFn: api.getSettings,
  })

  useEffect(() => {
    if (settings?.invoice_default_sort) {
      const saved = settings.invoice_default_sort as SortField
      setSortField(saved)
      setSortDir('desc')
    }
  }, [settings?.invoice_default_sort])

  const { data: suppliers } = useQuery({
    queryKey: ['suppliers'],
    queryFn: api.getSuppliers,
  })

  const statusValue = filter === 'all' ? undefined : filter ?? undefined
  const { data: invoices, isLoading } = useQuery({
    queryKey: ['invoices', statusValue, supplierFilter],
    queryFn: () => api.getInvoices({
      ...(statusValue ? { status: statusValue } : {}),
      ...(supplierFilter ? { supplier_id: supplierFilter } : {}),
    }),
  })

  const filtered = useMemo(() => {
    let result = (invoices || []).filter((inv) => {
      if (search) {
        const q = search.toLowerCase()
        return inv.invoice_number.toLowerCase().includes(q) ||
          (inv.customer?.name || '').toLowerCase().includes(q)
      }
      return true
    })

    if (sortField) {
      const cmp = (a: Invoice, b: Invoice): number => {
        switch (sortField) {
          case 'invoice_number': return a.invoice_number.localeCompare(b.invoice_number)
          case 'issue_date': return a.issue_date.localeCompare(b.issue_date)
          case 'created_at': return a.created_at.localeCompare(b.created_at)
          case 'due_date': return a.due_date.localeCompare(b.due_date)
          case 'total': return a.total - b.total
        }
      }
      result = [...result].sort((a, b) => sortDir === 'asc' ? cmp(a, b) : cmp(b, a))
    }

    return result
  }, [invoices, search, sortField, sortDir])

  if (isLoading) {
    return <Center h={300}><Loader /></Center>
  }

  const statusOptions = [
    { value: 'all', label: t('invoice.filter_all') },
    { value: 'draft', label: t('invoice.filter_draft') },
    { value: 'sent', label: t('invoice.filter_sent') },
    { value: 'paid', label: t('invoice.filter_paid') },
    { value: 'overdue', label: t('invoice.filter_overdue') },
    { value: 'unpaid', label: t('invoice.filter_unpaid') },
  ]

  const supplierOptions = [
    { value: '', label: t('invoice.all_suppliers') },
    ...(suppliers || []).map((s) => ({ value: s.id, label: s.name })),
  ]

  return (
    <Stack gap="md">
      <Group justify="space-between" align="center">
        <Title order={3}>{t('invoice.title')}</Title>
        <ActionIcon
          variant="filled"
          size="lg"
          radius="xl"
          onClick={() => navigate('/invoices/new')}
          aria-label={t('invoice.new')}
        >
          <IconPlus size={20} />
        </ActionIcon>
      </Group>

      <Stack gap="xs">
        <TextInput
          placeholder={t('invoice.search')}
          leftSection={<IconSearch size={16} />}
          value={search}
          onChange={(e) => setSearch(e.currentTarget.value)}
        />
        <Group grow>
          <Select
            data={statusOptions}
            value={filter}
            onChange={setFilter}
            allowDeselect={false}
          />
          {(suppliers || []).length > 1 && (
            <Select
              placeholder={t('invoice.filter_supplier')}
              data={supplierOptions}
              value={supplierFilter || ''}
              onChange={(v) => setSupplierFilter(v || null)}
              clearable
            />
          )}
        </Group>
      </Stack>

      {filtered.length === 0 ? (
        <Text c="dimmed" size="sm" ta="center" py="xl">
          {(invoices || []).length === 0 ? t('invoice.no_invoices') : t('invoice.no_match')}
        </Text>
      ) : (
        <Stack gap="xs">
          {filtered.map((inv) => (
            <Paper
              key={inv.id}
              p="sm"
              radius="md"
              withBorder
              style={{ cursor: 'pointer' }}
              onClick={() => navigate(`/invoices/${inv.id}`)}
            >
              <Group justify="space-between" align="center" mb={4}>
                <Text fw={600} ff="monospace" size="sm">
                  {inv.invoice_number}
                </Text>
                <Badge color={statusColors[inv.status]} size="sm" variant="light">
                  {t(`status.${inv.status}`)}
                </Badge>
              </Group>

              <Text size="sm" c="dimmed" mb={4}>
                {inv.customer?.name || '\u2014'}
              </Text>

              <Group justify="space-between" align="center">
                <Text size="sm">{formatDate(inv.issue_date)}</Text>
                <Text size="sm" fw={600}>{formatMoney(inv.total, inv.currency)}</Text>
              </Group>

              <Text size="xs" c="dimmed" mt={2}>
                {t('invoice.due_date')}: {formatDate(inv.due_date)}
              </Text>
            </Paper>
          ))}
        </Stack>
      )}

      {filtered.length > 0 && (
        <Text size="sm" c="dimmed" ta="center">
          {t('invoice.showing').replace('{count}', String(filtered.length))}
        </Text>
      )}
    </Stack>
  )
}
