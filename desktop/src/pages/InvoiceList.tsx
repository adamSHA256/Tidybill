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
import { IconSearch, IconPlus } from '@tabler/icons-react'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api, formatMoney, formatDate } from '../api/client'
import { useT } from '../i18n'

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

  if (isLoading) {
    return <Center h={300}><Loader /></Center>
  }

  const supplierOptions = [
    { value: '', label: t('invoice.all_suppliers') },
    ...(suppliers || []).map((s) => ({ value: s.id, label: s.name })),
  ]

  const filtered = (invoices || []).filter((inv) => {
    if (search) {
      const q = search.toLowerCase()
      return inv.invoice_number.toLowerCase().includes(q) ||
        (inv.customer?.name || '').toLowerCase().includes(q) ||
        (inv.supplier?.name || '').toLowerCase().includes(q)
    }
    return true
  })

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
                <Table.Th>{t('invoice.number')}</Table.Th>
                <Table.Th>{t('invoice.supplier')}</Table.Th>
                <Table.Th>{t('invoice.customer')}</Table.Th>
                <Table.Th>{t('invoice.issue_date')}</Table.Th>
                <Table.Th>{t('invoice.due_date')}</Table.Th>
                <Table.Th>{t('invoice.amount')}</Table.Th>
                <Table.Th>{t('invoice.status')}</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {filtered.map((inv) => {
                const isOverdue = inv.status !== 'paid' && inv.status !== 'cancelled' &&
                  new Date(inv.due_date) < new Date()
                return (
                  <Table.Tr key={inv.id} style={{ cursor: 'pointer' }} onClick={() => navigate(`/invoices/${inv.id}`)}>
                    <Table.Td fw={600} ff="monospace" fz="sm">{inv.invoice_number}</Table.Td>
                    <Table.Td fz="sm">{inv.supplier?.name || '—'}</Table.Td>
                    <Table.Td fz="sm">{inv.customer?.name || '—'}</Table.Td>
                    <Table.Td fz="sm">{formatDate(inv.issue_date)}</Table.Td>
                    <Table.Td fz="sm" c={isOverdue ? 'red' : undefined}>{formatDate(inv.due_date)}</Table.Td>
                    <Table.Td fz="sm" fw={600}>{formatMoney(inv.total)}</Table.Td>
                    <Table.Td>
                      <Badge color={statusColors[inv.status]} size="sm" variant="light">
                        {t(`status.${inv.status}`)}
                      </Badge>
                    </Table.Td>
                  </Table.Tr>
                )
              })}
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
