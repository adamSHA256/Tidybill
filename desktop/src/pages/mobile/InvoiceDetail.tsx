import {
  Title,
  Text,
  Paper,
  Group,
  Stack,
  Badge,
  Button,
  Divider,
  Textarea,
  Menu,
  Loader,
  Center,
  Modal,
  Tooltip,
  Box,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import {
  IconFileTypePdf,
  IconTrash,
  IconCheck,
  IconNotes,
  IconEdit,
  IconCopy,
  IconInfoCircle,
  IconDots,
} from '@tabler/icons-react'
import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, formatMoney, formatDate, openInBrowser, openFolder, type InvoiceStatus } from '../../api/client'
import { useT } from '../../i18n'
import { useIsMobile } from '../../hooks/useIsMobile'

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

export function MobileInvoiceDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { t } = useT()
  const isMobile = useIsMobile()

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
      openInBrowser(data.path).catch((err: Error) =>
        notifications.show({ title: t('common.error'), message: err.message, color: 'red' }))
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

  const translateUnit = (unit: string) => {
    const k = 'unit.' + unit
    const tr = t(k)
    return tr !== k ? tr : unit
  }

  return (
    <>
      <Stack gap="md" pb={80}>
        {/* Header: title + status badge */}
        <Group justify="space-between" align="center">
          <Group gap="sm" style={{ flexWrap: 'wrap' }}>
            <Title order={3} ff="monospace">{invoice.invoice_number}</Title>
            <Badge color={statusColors[invoice.status]} size="lg" variant="light">
              {t(`status.${invoice.status}`)}
            </Badge>
            {isOverdue && invoice.status !== 'overdue' && (
              <Badge color="red" size="lg" variant="filled">{t('status.overdue')}</Badge>
            )}
          </Group>
        </Group>

        {/* Invoice details card */}
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
            <Group justify="space-between">
              <Text size="sm" c="dimmed">{t('invoice.pdf')}</Text>
              {invoice.pdf_path ? (
                <Group gap="xs">
                  <Button variant="light" size="compact-xs" onClick={() => {
                    openInBrowser(invoice.pdf_path).catch((err: Error) =>
                      notifications.show({ title: t('common.error'), message: err.message, color: 'red' }))
                  }}>
                    {t('invoice.open_pdf')}
                  </Button>
                  <Button variant="light" size="compact-xs" color="gray" onClick={() => {
                    openFolder(invoice.pdf_path).catch((err: Error) =>
                      notifications.show({ title: t('common.error'), message: err.message, color: 'red' }))
                  }}>
                    {t('invoice.open_folder')}
                  </Button>
                </Group>
              ) : (
                <Button variant="light" size="compact-xs" leftSection={<IconFileTypePdf size={14} />}
                  onClick={() => pdfMutation.mutate()} loading={pdfMutation.isPending}>
                  {t('invoice.generate_pdf')}
                </Button>
              )}
            </Group>
          </Stack>
        </Paper>

        {/* Customer card */}
        <Paper p="md" radius="md" withBorder>
          <Text fw={500} mb="md">{t('invoice.customer_section')}</Text>
          {invoice.customer ? (
            <Stack gap="xs">
              <Text size="sm" fw={600}>{invoice.customer.name}</Text>
              <Text size="sm" c="dimmed">
                ICO: {invoice.customer.ico}
                {invoice.customer.dic && ` | DIC: ${invoice.customer.dic}`}
                {invoice.customer.ic_dph && ` | IČ DPH: ${invoice.customer.ic_dph}`}
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

        {/* Items section */}
        <Paper p="md" radius="md" withBorder>
          <Text fw={500} mb="md">{t('invoice.items_section')}</Text>
          <Stack gap="xs">
            {(invoice.items || []).map((item) => (
              <Paper key={item.id} p="sm" radius="sm" withBorder>
                <Text size="sm" fw={500} mb={4}>{item.description}</Text>
                <Text size="sm" c="dimmed">
                  {item.quantity} {translateUnit(item.unit)} × {formatMoney(item.unit_price, invoice.currency)}  ({item.vat_rate}%)
                </Text>
                <Text size="sm" fw={600} ta="right">
                  {formatMoney(item.subtotal, invoice.currency)}
                </Text>
              </Paper>
            ))}
          </Stack>

          <Divider my="md" />

          {/* Totals */}
          <Stack gap={4} align="end">
            <Group>
              <Text size="sm" c="dimmed" w={140} ta="right">{t('invoice.subtotal')}</Text>
              <Text size="sm" fw={600} w={120} ta="right">{formatMoney(invoice.subtotal, invoice.currency)}</Text>
            </Group>
            <Group>
              <Text size="sm" c="dimmed" w={140} ta="right">{t('invoice.vat')}</Text>
              <Text size="sm" fw={600} w={120} ta="right">{formatMoney(invoice.vat_total, invoice.currency)}</Text>
            </Group>
            <Divider w={260} />
            <Group>
              <Text size="lg" fw={700} w={140} ta="right">{t('invoice.total')}</Text>
              <Text size="lg" fw={700} w={120} ta="right">{formatMoney(invoice.total, invoice.currency)}</Text>
            </Group>
          </Stack>
        </Paper>

        {/* Notes */}
        {invoice.notes && (
          <Paper p="md" radius="md" withBorder>
            <Text fw={500} mb="xs">{t('invoice.notes_section')}</Text>
            <Text size="sm" c="dimmed">{invoice.notes}</Text>
          </Paper>
        )}

        {/* Internal notes */}
        <Paper p="md" radius="md" withBorder>
          <Group justify="space-between" mb="xs">
            <Group gap={4}>
              <Text fw={500}>{t('invoice.internal_notes')}</Text>
              <Tooltip label={t('invoice.internal_notes_hint')} multiline w={300} withArrow>
                <IconInfoCircle size={14} style={{ opacity: 0.5, cursor: 'help' }} />
              </Tooltip>
            </Group>
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
      </Stack>

      {/* Sticky bottom action bar */}
      <Box
        style={{
          position: 'fixed',
          bottom: 64,
          left: 0,
          right: 0,
          background: 'var(--mantine-color-body)',
          borderTop: '1px solid var(--mantine-color-default-border)',
          padding: '8px 16px',
          zIndex: 99,
        }}
      >
        <Group justify="space-between">
          {invoice.status !== 'paid' ? (
            <Button
              color="green"
              leftSection={<IconCheck size={16} />}
              onClick={() => statusMutation.mutate('paid')}
              loading={statusMutation.isPending}
              style={{ flex: 1 }}
            >
              {t('invoice.mark_paid')}
            </Button>
          ) : (
            <div style={{ flex: 1 }} />
          )}

          <Menu shadow="md" width={220} position="top-end">
            <Menu.Target>
              <Button variant="default" leftSection={<IconDots size={16} />}>
                {t('common.more') ?? 'More'}
              </Button>
            </Menu.Target>
            <Menu.Dropdown>
              <Menu.Item
                leftSection={<IconEdit size={16} />}
                onClick={() => navigate(`/invoices/${id}/edit`)}
              >
                {t('invoice.edit')}
              </Menu.Item>
              <Menu.Item
                leftSection={<IconCopy size={16} />}
                onClick={() => navigate('/invoices/new', {
                  state: {
                    duplicateFrom: {
                      invoice_number: invoice.invoice_number,
                      supplier_id: invoice.supplier_id,
                      customer_id: invoice.customer_id,
                      bank_account_id: invoice.bank_account_id,
                      currency: invoice.currency,
                      payment_method: invoice.payment_method,
                      notes: invoice.notes,
                      items: (invoice.items || []).map((item) => ({
                        item_id: item.item_id,
                        description: item.description,
                        quantity: item.quantity,
                        unit: item.unit,
                        unit_price: item.unit_price,
                        vat_rate: item.vat_rate,
                      })),
                    },
                  },
                })}
              >
                {t('invoice.duplicate')}
              </Menu.Item>
              <Menu.Item
                leftSection={<IconFileTypePdf size={16} />}
                onClick={() => pdfMutation.mutate()}
                disabled={pdfMutation.isPending}
              >
                {t('invoice.generate_pdf')}
              </Menu.Item>

              <Menu.Divider />
              <Menu.Label>{t('invoice.change_status')}</Menu.Label>
              {allStatuses.map((s) => (
                <Menu.Item
                  key={s}
                  onClick={() => statusMutation.mutate(s)}
                  disabled={s === invoice.status}
                  color={statusColors[s]}
                >
                  {t(`status.${s}`)}
                </Menu.Item>
              ))}

              <Menu.Divider />
              <Menu.Item
                leftSection={<IconTrash size={16} />}
                color="red"
                onClick={() => setDeleteOpen(true)}
              >
                {t('invoice.delete')}
              </Menu.Item>
            </Menu.Dropdown>
          </Menu>
        </Group>
      </Box>

      {/* Internal notes modal */}
      <Modal opened={notesOpen} onClose={() => setNotesOpen(false)} title={t('invoice.internal_notes_title')} fullScreen={isMobile}>
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

      {/* Delete confirmation modal */}
      <Modal opened={deleteOpen} onClose={() => setDeleteOpen(false)} title={t('invoice.delete_title')} fullScreen={isMobile}>
        <Stack gap="md">
          <Text size="sm" dangerouslySetInnerHTML={{ __html: t('invoice.delete_confirm').replace('{number}', invoice.invoice_number) }} />
          <Group justify="end">
            <Button variant="default" onClick={() => setDeleteOpen(false)}>{t('common.cancel')}</Button>
            <Button color="red" onClick={() => deleteMutation.mutate()}
              loading={deleteMutation.isPending}>{t('common.delete')}</Button>
          </Group>
        </Stack>
      </Modal>
    </>
  )
}
