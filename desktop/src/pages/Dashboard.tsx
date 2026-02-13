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

  const recentInvoices = (invoices || []).slice(0, 5)
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
        <Paper p="md" radius="md" withBorder>
          <Text size="xs" c="dimmed">{t('dashboard.total_revenue')}</Text>
          <Title order={2} mt={4}>{formatMoney(stats?.total_revenue_month || 0)}</Title>
        </Paper>

        <Paper p="md" radius="md" withBorder>
          <Text size="xs" c="dimmed">{t('dashboard.unpaid_invoices')}</Text>
          <Title order={2} mt={4} c={stats?.unpaid_count ? 'red' : undefined}>
            {stats?.unpaid_count || 0}
          </Title>
          {(stats?.unpaid_amount || 0) > 0 && (
            <Text size="xs" c="red" mt={4}>{t('dashboard.outstanding').replace('{amount}', formatMoney(stats!.unpaid_amount))}</Text>
          )}
        </Paper>

        <Paper p="md" radius="md" withBorder>
          <Text size="xs" c="dimmed">{t('dashboard.active_customers')}</Text>
          <Title order={2} mt={4}>{stats?.active_customers || 0}</Title>
        </Paper>

        <Paper p="md" radius="md" withBorder>
          <Text size="xs" c="dimmed">{t('dashboard.invoices_month')}</Text>
          <Title order={2} mt={4}>{stats?.invoices_this_month || 0}</Title>
        </Paper>
      </SimpleGrid>

      <SimpleGrid cols={{ base: 1, md: 2 }}>
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
                  <Table.Tr key={inv.id}>
                    <Table.Td fw={600} ff="monospace" fz="sm">{inv.invoice_number}</Table.Td>
                    <Table.Td fz="sm">{inv.customer?.name || '—'}</Table.Td>
                    <Table.Td fz="sm">{formatMoney(inv.total)}</Table.Td>
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

        <Stack gap="md">
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

          {overdueInvoices.length > 0 && (
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
                    <div key={inv.id}>
                      <Text size="sm" fw={500}>{inv.invoice_number} — {inv.customer?.name || '—'}</Text>
                      <Text size="xs" c="red">{formatMoney(inv.total)} — {t('dashboard.days_overdue').replace('{days}', String(daysOverdue))}</Text>
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
