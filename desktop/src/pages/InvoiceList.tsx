import {
  Title,
  Text,
  Paper,
  Group,
  Table,
  Badge,
  Button,
  TextInput,
  Select,
  SegmentedControl,
  Stack,
  Loader,
  Center,
} from '@mantine/core'
import { IconSearch, IconPlus, IconArrowsSort, IconSortAscending, IconSortDescending } from '@tabler/icons-react'
import { useState, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api, formatMoney, formatDate, type Invoice } from '../api/client'
import { useT } from '../i18n'

type SortField = 'invoice_number' | 'issue_date' | 'created_at' | 'total'
type SortDir = 'asc' | 'desc'

const statusColors: Record<string, string> = {
  draft: 'gray',
  created: 'blue',
  sent: 'yellow',
  paid: 'green',
  overdue: 'red',
  partially_paid: 'orange',
  cancelled: 'dimmed',
}

export function InvoiceList() {
  const [filter, setFilter] = useState('all')
  const [search, setSearch] = useState('')
  const [supplierFilter, setSupplierFilter] = useState<string | null>(null)
  const [sortField, setSortField] = useState<SortField | null>('created_at')
  const [sortDir, setSortDir] = useState<SortDir>('desc')
  const navigate = useNavigate()
  const { t } = useT()

  const { data: suppliers } = useQuery({
    queryKey: ['suppliers'],
    queryFn: api.getSuppliers,
  })

  const { data: invoices, isLoading } = useQuery({
    queryKey: ['invoices', filter === 'all' ? undefined : filter, supplierFilter],
    queryFn: () => api.getInvoices({
      ...(filter !== 'all' ? { status: filter } : {}),
      ...(supplierFilter ? { supplier_id: supplierFilter } : {}),
    }),
  })

  const toggleSort = (field: SortField) => {
    if (sortField === field) {
      if (sortDir === 'asc') setSortDir('desc')
      else { setSortField(null); setSortDir('asc') }
    } else {
      setSortField(field)
      setSortDir('asc')
    }
  }

  const SortIcon = ({ field }: { field: SortField }) => {
    if (sortField !== field) return <IconArrowsSort size={14} style={{ opacity: 0.3 }} />
    return sortDir === 'asc' ? <IconSortAscending size={14} /> : <IconSortDescending size={14} />
  }

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

  const supplierOptions = [
    { value: '', label: t('invoice.all_suppliers') },
    ...(suppliers || []).map((s) => ({ value: s.id, label: s.name })),
  ]

  return (
    <Stack gap="lg">
      <Group justify="space-between">
        <Title order={2}>{t('invoice.title')}</Title>
        <Button leftSection={<IconPlus size={16} />} onClick={() => navigate('/invoices/new')}>
          {t('invoice.new')}
        </Button>
      </Group>

      <Group>
        <SegmentedControl
          value={filter}
          onChange={setFilter}
          data={[
            { label: t('invoice.filter_all'), value: 'all' },
            { label: t('invoice.filter_draft'), value: 'draft' },
            { label: t('invoice.filter_sent'), value: 'sent' },
            { label: t('invoice.filter_paid'), value: 'paid' },
            { label: t('invoice.filter_overdue'), value: 'overdue' },
          ]}
        />
        <TextInput
          placeholder={t('invoice.search')}
          leftSection={<IconSearch size={16} />}
          value={search}
          onChange={(e) => setSearch(e.currentTarget.value)}
          style={{ flex: 1, maxWidth: 300 }}
        />
        {(suppliers || []).length > 1 && (
          <Select
            placeholder={t('invoice.filter_supplier')}
            data={supplierOptions}
            value={supplierFilter || ''}
            onChange={(v) => setSupplierFilter(v || null)}
            clearable
            w={200}
          />
        )}
      </Group>

      <Paper p="md" radius="md" withBorder>
        {filtered.length === 0 ? (
          <Text c="dimmed" size="sm" ta="center" py="xl">
            {(invoices || []).length === 0 ? t('invoice.no_invoices') : t('invoice.no_match')}
          </Text>
        ) : (
          <Table>
            <Table.Thead>
              <Table.Tr>
                <Table.Th style={{ cursor: 'pointer' }} onClick={() => toggleSort('invoice_number')}>
                  <Group gap={4} wrap="nowrap">{t('invoice.number')} <SortIcon field="invoice_number" /></Group>
                </Table.Th>
                <Table.Th>{t('invoice.customer')}</Table.Th>
                <Table.Th style={{ cursor: 'pointer' }} onClick={() => toggleSort('issue_date')}>
                  <Group gap={4} wrap="nowrap">{t('invoice.issue_date')} <SortIcon field="issue_date" /></Group>
                </Table.Th>
                <Table.Th style={{ cursor: 'pointer' }} onClick={() => toggleSort('created_at')}>
                  <Group gap={4} wrap="nowrap">{t('invoice.created_at')} <SortIcon field="created_at" /></Group>
                </Table.Th>
                <Table.Th style={{ cursor: 'pointer' }} onClick={() => toggleSort('total')}>
                  <Group gap={4} wrap="nowrap">{t('invoice.amount')} <SortIcon field="total" /></Group>
                </Table.Th>
                <Table.Th>{t('invoice.status')}</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {filtered.map((inv) => (
                  <Table.Tr key={inv.id} style={{ cursor: 'pointer' }} onClick={() => navigate(`/invoices/${inv.id}`)}>
                    <Table.Td fw={600} ff="monospace" fz="sm">{inv.invoice_number}</Table.Td>
                    <Table.Td fz="sm">{inv.customer?.name || '—'}</Table.Td>
                    <Table.Td fz="sm">{formatDate(inv.issue_date)}</Table.Td>
                    <Table.Td fz="sm">{formatDate(inv.created_at)}</Table.Td>
                    <Table.Td fz="sm" fw={600}>{formatMoney(inv.total)}</Table.Td>
                    <Table.Td>
                      <Badge color={statusColors[inv.status]} size="sm" variant="light">
                        {t(`status.${inv.status}`)}
                      </Badge>
                    </Table.Td>
                  </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        )}

        {filtered.length > 0 && (
          <Group justify="space-between" mt="md">
            <Text size="sm" c="dimmed">{t('invoice.showing').replace('{count}', String(filtered.length))}</Text>
          </Group>
        )}
      </Paper>
    </Stack>
  )
}
