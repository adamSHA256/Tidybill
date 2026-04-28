import type { ReactNode } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { Box, UnstyledButton, Text, ActionIcon } from '@mantine/core'
import {
  IconDashboard,
  IconFileInvoice,
  IconPlus,
  IconDots,
  IconHelp,
} from '@tabler/icons-react'
import { useT } from '../i18n'

const tabs = [
  { key: 'nav.dashboard', icon: IconDashboard, path: '/', anchor: 'm-tab-dashboard' },
  { key: 'nav.invoices', icon: IconFileInvoice, path: '/invoices', anchor: 'm-tab-invoices' },
  { key: 'nav.new_invoice', icon: IconPlus, path: '/invoices/new', anchor: 'm-tab-new-invoice' },
  { key: 'nav.more', icon: IconDots, path: '/more', anchor: 'm-tab-more' },
]

export function MobileShell({ children }: { children: ReactNode }) {
  const location = useLocation()
  const navigate = useNavigate()
  const { t } = useT()

  const getActiveTab = () => {
    const path = location.pathname
    if (path === '/') return '/'
    if (path === '/invoices/new') return '/invoices/new'
    if (path.startsWith('/invoices')) return '/invoices'
    return '/more'
  }

  const activeTab = getActiveTab()

  return (
    <Box style={{
      minHeight: '100vh',
      paddingBottom: 64,
    }}>
      <Box style={{ position: 'fixed', top: 12, right: 12, zIndex: 90 }}>
        <ActionIcon variant="light" size="lg" onClick={() => navigate('/help')} aria-label={t('tour.help_aria')}>
          <IconHelp size={20} />
        </ActionIcon>
      </Box>
      <Box p="md">
        {children}
      </Box>

      <Box
        style={{
          position: 'fixed',
          bottom: 0,
          left: 0,
          right: 0,
          height: 64,
          display: 'flex',
          alignItems: 'center',
          borderTop: '1px solid var(--mantine-color-default-border)',
          backgroundColor: 'var(--mantine-color-body)',
          zIndex: 100,
        }}
      >
        {tabs.map((tab) => {
          const active = activeTab === tab.path
          return (
            <UnstyledButton
              key={tab.path}
              data-tour={tab.anchor}
              onClick={() => navigate(tab.path)}
              style={{
                flex: 1,
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                gap: 2,
                paddingTop: 8,
                paddingBottom: 8,
                color: active
                  ? 'var(--mantine-primary-color-filled)'
                  : 'var(--mantine-color-dimmed)',
              }}
            >
              <tab.icon size={22} stroke={active ? 2 : 1.5} />
              <Text size="xs" fw={active ? 600 : 400}>
                {t(tab.key)}
              </Text>
            </UnstyledButton>
          )
        })}
      </Box>
    </Box>
  )
}
