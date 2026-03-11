import {
  Title,
  Text,
  SimpleGrid,
  Paper,
  Group,
  Stack,
  Table,
  Badge,
  Button,
  Alert,
  Loader,
  Center,
} from '@mantine/core'
import { IconPlus, IconAlertTriangle } from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api, formatMoney, type Invoice } from '../api/client'
import { useT } from '../i18n'

interface DashboardWidgets {
  revenue: boolean
  unpaid: boolean
  customers: boolean
  invoices_month: boolean
  overdue: boolean
  recent: boolean
  quick_actions: boolean
}

const defaultWidgets: DashboardWidgets = {
  revenue: true,
  unpaid: true,
  customers: true,
  invoices_month: true,
  overdue: true,
  recent: true,
  quick_actions: true,
}

function parseWidgets(raw?: string): DashboardWidgets {
  if (!raw) return { ...defaultWidgets }
  try {
    return { ...defaultWidgets, ...JSON.parse(raw) }
  } catch {
    return { ...defaultWidgets }
  }
}

const statusColors: Record<string, string> = {
  draft: 'gray',
  created: 'blue',
  sent: 'yellow',
  paid: 'green',
  overdue: 'red',
  partially_paid: 'orange',
  cancelled: 'dimmed',
}

export function Dashboard() {
  const navigate = useNavigate()
  const { t } = useT()

  const { data: settings } = useQuery({
    queryKey: ['settings'],
    queryFn: api.getSettings,
  })

  const widgets = parseWidgets(settings?.dashboard_widgets)

  const { data: stats, isLoading: statsLoading } = useQuery({
    queryKey: ['dashboard-stats'],
    queryFn: api.getDashboardStats,
  })

  const { data: invoices, isLoading: invoicesLoading } = useQuery({
    queryKey: ['invoices'],
    queryFn: () => api.getInvoices(),
  })

  if (statsLoading || invoicesLoading) {
    return <Center h={300}><Loader /></Center>
  }

  const recentInvoices = [...(invoices || [])]
    .sort((a, b) => b.created_at.localeCompare(a.created_at))
    .slice(0, 5)
  const overdueInvoices = (invoices || []).filter(
    (inv: Invoice) => {
      if (inv.status === 'paid' || inv.status === 'cancelled') return false
      return new Date(inv.due_date) < new Date()
    }
  )

  return (
    <Stack gap="lg">
      <div>
        <Title order={2}>{t('dashboard.title')}</Title>
        <Text c="dimmed" size="sm">{t('dashboard.subtitle')}</Text>
      </div>

      <SimpleGrid cols={{ base: 1, xs: 2, md: 4 }}>
        {widgets.revenue && (
          <Paper p="md" radius="md" withBorder>
            <Text size="xs" c="dimmed">{t('dashboard.total_revenue')}</Text>
            {(stats?.revenue_by_currency || []).length > 0 ? (
              (stats!.revenue_by_currency).map((rc: { currency: string; amount: number }) => (
                <Title order={2} mt={4} key={rc.currency}>{formatMoney(rc.amount, rc.currency)}</Title>
              ))
            ) : (
              <Title order={2} mt={4}>0</Title>
            )}
          </Paper>
        )}

        {widgets.unpaid && (
          <Paper p="md" radius="md" withBorder>
            <Text size="xs" c="dimmed">{t('dashboard.unpaid_invoices')}</Text>
            <Title order={2} mt={4} c={stats?.unpaid_count ? 'red' : undefined}>
              {stats?.unpaid_count || 0}
            </Title>
            {(stats?.unpaid_by_currency || []).length > 0 && (
              <Stack gap={0} mt={4}>
                {(stats!.unpaid_by_currency).map((uc: { currency: string; amount: number }) => (
                  <Text size="xs" c="red" key={uc.currency}>{t('dashboard.outstanding').replace('{amount}', formatMoney(uc.amount, uc.currency))}</Text>
                ))}
              </Stack>
            )}
          </Paper>
        )}

        {widgets.customers && (
          <Paper p="md" radius="md" withBorder>
            <Text size="xs" c="dimmed">{t('dashboard.active_customers')}</Text>
            <Title order={2} mt={4}>{stats?.active_customers || 0}</Title>
          </Paper>
        )}

        {widgets.invoices_month && (
          <Paper p="md" radius="md" withBorder>
            <Text size="xs" c="dimmed">{t('dashboard.invoices_month')}</Text>
            <Title order={2} mt={4}>{stats?.invoices_this_month || 0}</Title>
          </Paper>
        )}
      </SimpleGrid>

      <SimpleGrid cols={{ base: 1, md: 2 }}>
        {widgets.recent && (
          <Paper p="md" radius="md" withBorder>
            <Group justify="space-between" mb="md">
              <Text fw={500}>{t('dashboard.recent_invoices')}</Text>
              <Button variant="subtle" size="xs" onClick={() => navigate('/invoices')}>
                {t('dashboard.view_all')}
              </Button>
            </Group>

            {recentInvoices.length === 0 ? (
              <Text c="dimmed" size="sm" ta="center" py="xl">{t('dashboard.no_invoices')}</Text>
            ) : (
              <Table>
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th>{t('invoice.number')}</Table.Th>
                    <Table.Th>{t('invoice.customer')}</Table.Th>
                    <Table.Th>{t('invoice.amount')}</Table.Th>
                    <Table.Th>{t('invoice.status')}</Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {recentInvoices.map((inv) => (
                    <Table.Tr key={inv.id} style={{ cursor: 'pointer' }} onClick={() => navigate(`/invoices/${inv.id}`)}>
                      <Table.Td fw={600} ff="monospace" fz="sm">{inv.invoice_number}</Table.Td>
                      <Table.Td fz="sm">{inv.customer?.name || '—'}</Table.Td>
                      <Table.Td fz="sm">{formatMoney(inv.total, inv.currency)}</Table.Td>
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
          </Paper>
        )}

        <Stack gap="md">
          {widgets.quick_actions && (
            <Paper p="md" radius="md" withBorder>
              <Text fw={500} mb="md">{t('dashboard.quick_actions')}</Text>
              <Stack gap="sm">
                <Button fullWidth leftSection={<IconPlus size={16} />} onClick={() => navigate('/invoices/new')}>
                  {t('dashboard.create_invoice')}
                </Button>
                <Button fullWidth variant="default" onClick={() => navigate('/customers')}>
                  {t('dashboard.manage_customers')}
                </Button>
              </Stack>
            </Paper>
          )}

          {widgets.overdue && overdueInvoices.length > 0 && (
            <Alert
              variant="light"
              color="red"
              title={t('dashboard.overdue_title')}
              icon={<IconAlertTriangle size={18} />}
            >
              <Stack gap="xs">
                {overdueInvoices.map((inv) => {
                  const daysOverdue = Math.floor(
                    (Date.now() - new Date(inv.due_date).getTime()) / (1000 * 60 * 60 * 24)
                  )
                  return (
                    <div key={inv.id} style={{ cursor: 'pointer' }} onClick={() => navigate(`/invoices/${inv.id}`)}>
                      <Text size="sm" fw={500} td="underline">{inv.invoice_number} — {inv.customer?.name || '—'}</Text>
                      <Text size="xs" c="red">{formatMoney(inv.total, inv.currency)} — {t('dashboard.days_overdue').replace('{days}', String(daysOverdue))}</Text>
                    </div>
                  )
                })}
              </Stack>
            </Alert>
          )}
        </Stack>
      </SimpleGrid>
    </Stack>
  )
}
