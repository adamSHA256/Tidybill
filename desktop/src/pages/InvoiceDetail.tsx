import {
  Title,
  Text,
  Paper,
  Group,
  Stack,
  Table,
  Badge,
  Button,
  Divider,
  SimpleGrid,
  Textarea,
  Menu,
  Loader,
  Center,
  Modal,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import {
  IconArrowLeft,
  IconFileTypePdf,
  IconTrash,
  IconCheck,
  IconChevronDown,
  IconNotes,
} from '@tabler/icons-react'
import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, formatMoney, formatDate, type InvoiceStatus } from '../api/client'
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

const allStatuses: InvoiceStatus[] = ['draft', 'created', 'sent', 'paid', 'overdue', 'partially_paid', 'cancelled']

export function InvoiceDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { t } = useT()

  const [notesOpen, setNotesOpen] = useState(false)
  const [internalNotes, setInternalNotes] = useState('')
  const [deleteOpen, setDeleteOpen] = useState(false)

  const { data: invoice, isLoading } = useQuery({
    queryKey: ['invoice', id],
    queryFn: () => api.getInvoice(id!),
    enabled: !!id,
  })

  const statusMutation = useMutation({
    mutationFn: (status: string) => api.updateInvoiceStatus(id!, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['invoice', id] })
      queryClient.invalidateQueries({ queryKey: ['invoices'] })
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] })
      notifications.show({ title: t('notify.status_updated'), message: t('notify.status_updated_msg'), color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const notesMutation = useMutation({
    mutationFn: (notes: string) => api.updateInvoiceNotes(id!, notes),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['invoice', id] })
      notifications.show({ title: t('notify.notes_saved'), message: t('notify.notes_saved_msg'), color: 'green' })
      setNotesOpen(false)
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const pdfMutation = useMutation({
    mutationFn: () => api.generatePDF(id!),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['invoice', id] })
      notifications.show({ title: t('notify.pdf_generated'), message: data.path, color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: () => api.deleteInvoice(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['invoices'] })
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] })
      notifications.show({ title: t('notify.invoice_deleted'), message: t('notify.invoice_deleted_msg'), color: 'green' })
      navigate('/invoices')
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  if (isLoading) {
    return <Center h={300}><Loader /></Center>
  }

  if (!invoice) {
    return (
      <Center h={300}>
        <Stack align="center">
          <Text c="dimmed">{t('invoice.not_found')}</Text>
          <Button variant="light" onClick={() => navigate('/invoices')}>{t('invoice.back_to_list')}</Button>
        </Stack>
      </Center>
    )
  }

  const isOverdue = invoice.status !== 'paid' && invoice.status !== 'cancelled' &&
    new Date(invoice.due_date) < new Date()

  return (
    <Stack gap="lg">
      <Group justify="space-between">
        <Group>
          <Button variant="subtle" leftSection={<IconArrowLeft size={16} />}
            onClick={() => navigate('/invoices')} color="gray">
            {t('common.back')}
          </Button>
          <div>
            <Group gap="sm">
              <Title order={2} ff="monospace">{invoice.invoice_number}</Title>
              <Badge color={statusColors[invoice.status]} size="lg" variant="light">
                {t(`status.${invoice.status}`)}
              </Badge>
              {isOverdue && invoice.status !== 'overdue' && (
                <Badge color="red" size="lg" variant="filled">{t('status.overdue')}</Badge>
              )}
            </Group>
          </div>
        </Group>
        <Group>
          <Button variant="light" leftSection={<IconFileTypePdf size={16} />}
            onClick={() => pdfMutation.mutate()} loading={pdfMutation.isPending}>
            {t('invoice.generate_pdf')}
          </Button>
          <Menu shadow="md" width={200}>
            <Menu.Target>
              <Button variant="light" rightSection={<IconChevronDown size={14} />}>
                {t('invoice.change_status')}
              </Button>
            </Menu.Target>
            <Menu.Dropdown>
              {allStatuses.map((s) => (
                <Menu.Item key={s} onClick={() => statusMutation.mutate(s)}
                  disabled={s === invoice.status}
                  color={statusColors[s]}>
                  {t(`status.${s}`)}
                </Menu.Item>
              ))}
            </Menu.Dropdown>
          </Menu>
          {invoice.status !== 'paid' && (
            <Button color="green" leftSection={<IconCheck size={16} />}
              onClick={() => statusMutation.mutate('paid')}
              loading={statusMutation.isPending}>
              {t('invoice.mark_paid')}
            </Button>
          )}
        </Group>
      </Group>

      <SimpleGrid cols={{ base: 1, md: 2 }}>
        <Paper p="md" radius="md" withBorder>
          <Text fw={500} mb="md">{t('invoice.detail_title')}</Text>
          <Stack gap="xs">
            <Group justify="space-between">
              <Text size="sm" c="dimmed">{t('invoice.invoice_number')}</Text>
              <Text size="sm" fw={600} ff="monospace">{invoice.invoice_number}</Text>
            </Group>
            <Group justify="space-between">
              <Text size="sm" c="dimmed">{t('invoice.variable_symbol')}</Text>
              <Text size="sm" fw={600} ff="monospace">{invoice.variable_symbol}</Text>
            </Group>
            <Group justify="space-between">
              <Text size="sm" c="dimmed">{t('invoice.issue_date')}</Text>
              <Text size="sm">{formatDate(invoice.issue_date)}</Text>
            </Group>
            <Group justify="space-between">
              <Text size="sm" c="dimmed">{t('invoice.due_date')}</Text>
              <Text size="sm" c={isOverdue ? 'red' : undefined} fw={isOverdue ? 600 : undefined}>
                {formatDate(invoice.due_date)}
              </Text>
            </Group>
            <Group justify="space-between">
              <Text size="sm" c="dimmed">{t('invoice.currency')}</Text>
              <Text size="sm">{invoice.currency}</Text>
            </Group>
            {invoice.pdf_path && (
              <Group justify="space-between">
                <Text size="sm" c="dimmed">{t('invoice.pdf')}</Text>
                <Text size="xs" c="blue" ff="monospace" truncate style={{ maxWidth: 200 }}>
                  {invoice.pdf_path}
                </Text>
              </Group>
            )}
          </Stack>
        </Paper>

        <Paper p="md" radius="md" withBorder>
          <Text fw={500} mb="md">{t('invoice.customer_section')}</Text>
          {invoice.customer ? (
            <Stack gap="xs">
              <Text size="sm" fw={600}>{invoice.customer.name}</Text>
              <Text size="sm" c="dimmed">
                ICO: {invoice.customer.ico}
                {invoice.customer.dic && ` | DIC: ${invoice.customer.dic}`}
              </Text>
              <Text size="sm" c="dimmed">
                {[invoice.customer.street, invoice.customer.city, invoice.customer.zip].filter(Boolean).join(', ')}
              </Text>
              {invoice.customer.email && (
                <Text size="sm" c="dimmed">{invoice.customer.email}</Text>
              )}
            </Stack>
          ) : (
            <Text size="sm" c="dimmed">{t('invoice.customer_not_available')}</Text>
          )}
        </Paper>
      </SimpleGrid>

      <Paper p="md" radius="md" withBorder>
        <Text fw={500} mb="md">{t('invoice.items_section')}</Text>
        <Table>
          <Table.Thead>
            <Table.Tr>
              <Table.Th style={{ width: '40%' }}>{t('invoice.description')}</Table.Th>
              <Table.Th style={{ width: '10%' }}>{t('invoice.qty')}</Table.Th>
              <Table.Th style={{ width: '10%' }}>{t('invoice.unit')}</Table.Th>
              <Table.Th style={{ width: '15%' }}>{t('invoice.unit_price')}</Table.Th>
              <Table.Th style={{ width: '10%' }}>{t('invoice.vat_pct')}</Table.Th>
              <Table.Th style={{ width: '15%' }} ta="right">{t('invoice.total_col')}</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {(invoice.items || []).map((item) => (
              <Table.Tr key={item.id}>
                <Table.Td>
                  <Text size="sm">{item.description}</Text>
                </Table.Td>
                <Table.Td fz="sm">{item.quantity}</Table.Td>
                <Table.Td fz="sm">{item.unit}</Table.Td>
                <Table.Td fz="sm">{formatMoney(item.unit_price)}</Table.Td>
                <Table.Td fz="sm">{item.vat_rate}%</Table.Td>
                <Table.Td fz="sm" fw={600} ta="right">{formatMoney(item.subtotal)}</Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>

        <Divider my="md" />

        <Stack gap={4} align="end" pr="xl">
          <Group>
            <Text size="sm" c="dimmed" w={180} ta="right">{t('invoice.subtotal')}</Text>
            <Text size="sm" fw={600} w={120} ta="right">{formatMoney(invoice.subtotal)}</Text>
          </Group>
          <Group>
            <Text size="sm" c="dimmed" w={180} ta="right">{t('invoice.vat')}</Text>
            <Text size="sm" fw={600} w={120} ta="right">{formatMoney(invoice.vat_total)}</Text>
          </Group>
          <Divider w={300} />
          <Group>
            <Text size="lg" fw={700} w={180} ta="right">{t('invoice.total')}</Text>
            <Text size="lg" fw={700} w={120} ta="right">{formatMoney(invoice.total, invoice.currency)}</Text>
          </Group>
        </Stack>
      </Paper>

      {invoice.notes && (
        <Paper p="md" radius="md" withBorder>
          <Text fw={500} mb="xs">{t('invoice.notes_section')}</Text>
          <Text size="sm" c="dimmed">{invoice.notes}</Text>
        </Paper>
      )}

      <Paper p="md" radius="md" withBorder>
        <Group justify="space-between" mb="xs">
          <Text fw={500}>{t('invoice.internal_notes')}</Text>
          <Button variant="light" size="xs" leftSection={<IconNotes size={14} />}
            onClick={() => { setInternalNotes(invoice.internal_notes || ''); setNotesOpen(true) }}>
            {t('common.edit')}
          </Button>
        </Group>
        {invoice.internal_notes ? (
          <Text size="sm" c="dimmed" style={{ whiteSpace: 'pre-wrap' }}>{invoice.internal_notes}</Text>
        ) : (
          <Text size="sm" c="dimmed" fs="italic">{t('invoice.no_internal_notes')}</Text>
        )}
      </Paper>

      <Divider />

      <Group justify="end">
        <Button color="red" variant="light" leftSection={<IconTrash size={16} />}
          onClick={() => setDeleteOpen(true)}>
          {t('invoice.delete')}
        </Button>
      </Group>

      <Modal opened={notesOpen} onClose={() => setNotesOpen(false)} title={t('invoice.internal_notes_title')}>
        <Stack gap="md">
          <Textarea
            placeholder={t('invoice.internal_notes_placeholder')}
            value={internalNotes}
            onChange={(e) => setInternalNotes(e.currentTarget.value)}
            minRows={4}
            autosize
          />
          <Group justify="end">
            <Button variant="default" onClick={() => setNotesOpen(false)}>{t('common.cancel')}</Button>
            <Button onClick={() => notesMutation.mutate(internalNotes)}
              loading={notesMutation.isPending}>{t('common.save')}</Button>
          </Group>
        </Stack>
      </Modal>

      <Modal opened={deleteOpen} onClose={() => setDeleteOpen(false)} title={t('invoice.delete_title')}>
        <Stack gap="md">
          <Text size="sm" dangerouslySetInnerHTML={{ __html: t('invoice.delete_confirm').replace('{number}', invoice.invoice_number) }} />
          <Group justify="end">
            <Button variant="default" onClick={() => setDeleteOpen(false)}>{t('common.cancel')}</Button>
            <Button color="red" onClick={() => deleteMutation.mutate()}
              loading={deleteMutation.isPending}>{t('common.delete')}</Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  )
}
