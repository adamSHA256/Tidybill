import type { ReactNode } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import {
  AppShell as MantineAppShell,
  NavLink,
  Group,
  ActionIcon,
  useMantineColorScheme,
  Divider,
  Button,
} from '@mantine/core'
import {
  IconDashboard,
  IconFileInvoice,
  IconUsers,
  IconBuildingStore,
  IconSettings,
  IconSun,
  IconMoon,
  IconTemplate,
  IconPackage,
  IconPlus,
  IconInfoCircle,
  IconMail,
  IconDatabaseExport,
} from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'
import { useT } from '../i18n'

function TidyBillLogo({ colorScheme }: { colorScheme: string }) {
  const tidyFill = colorScheme === 'dark' ? '#ffffff' : '#1B2B3A'
  return (
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 330 100" width="330" height="100" style={{ height: 56, width: 'auto' }}>
      <g transform="translate(0, 2)">
        <path d="M0 6 C0 2.69 2.69 0 6 0 L40 0 C43.31 0 46 2.69 46 6 L46 70 L38.5 63.5 L31 70 L23 63.5 L15.5 70 L8 63.5 L0 70 Z"
              fill={tidyFill} />
        <line x1="9" y1="16" x2="37" y2="16" stroke="#4A9E8E" strokeWidth="2.5" strokeLinecap="round"/>
        <line x1="9" y1="27" x2="31" y2="27" stroke="#4A9E8E" strokeWidth="2.5" strokeLinecap="round" opacity="0.6"/>
        <line x1="9" y1="38" x2="34" y2="38" stroke="#4A9E8E" strokeWidth="2.5" strokeLinecap="round" opacity="0.35"/>
        <circle cx="37" cy="52" r="10.5" fill="#4A9E8E"/>
        <polyline points="31.5,52 35,55.5 43,47.5" stroke="white" strokeWidth="2" fill="none" strokeLinecap="round" strokeLinejoin="round"/>
      </g>
      <text x="60" y="52" fontFamily="system-ui, -apple-system, 'Segoe UI', sans-serif" fontWeight="700" fontSize="42" fill={tidyFill} letterSpacing="-1">
        Tidy<tspan fill="#4A9E8E">Bill</tspan>
      </text>
    </svg>
  )
}

const navKeys = [
  { key: 'nav.dashboard', icon: IconDashboard, path: '/' },
  { key: 'nav.invoices', icon: IconFileInvoice, path: '/invoices' },
  { key: 'nav.customers', icon: IconUsers, path: '/customers' },
  { key: 'nav.suppliers', pluralKey: 'nav.suppliers_plural', icon: IconBuildingStore, path: '/suppliers' },
]

const toolKeys = [
  { key: 'nav.templates', icon: IconTemplate, path: '/templates' },
  { key: 'nav.items', icon: IconPackage, path: '/items' },
  { key: 'nav.automatizace', icon: IconMail, path: '/automatizace' },
  { key: 'nav.sync', icon: IconDatabaseExport, path: '/sync' },
]

export function AppShell({ children }: { children: ReactNode }) {
  const location = useLocation()
  const navigate = useNavigate()
  const { colorScheme, toggleColorScheme } = useMantineColorScheme()
  const { t } = useT()

  const { data: suppliers } = useQuery({
    queryKey: ['suppliers'],
    queryFn: api.getSuppliers,
  })
  const supplierCount = (suppliers || []).length

  const getNavLabel = (item: typeof navKeys[number]) => {
    if (item.pluralKey && supplierCount > 1) {
      return t(item.pluralKey)
    }
    return t(item.key)
  }

  return (
    <MantineAppShell
      navbar={{ width: 260, breakpoint: 'sm' }}
      padding="md"
      styles={{ navbar: { height: '100%' } }}
    >
      <MantineAppShell.Navbar p="md">
        <MantineAppShell.Section>
          <Group justify="space-between" wrap="nowrap" mb="md" pl={10} pr={16}>
            <TidyBillLogo colorScheme={colorScheme} />
            <ActionIcon
              variant="subtle"
              onClick={toggleColorScheme}
              size="lg"
            >
              {colorScheme === 'dark' ? <IconSun size={18} /> : <IconMoon size={18} />}
            </ActionIcon>
          </Group>

          <Button
            fullWidth
            leftSection={<IconPlus size={16} />}
            mb="lg"
            onClick={() => navigate('/invoices/new')}
          >
            {t('nav.new_invoice')}
          </Button>
        </MantineAppShell.Section>

        <MantineAppShell.Section grow style={{ overflow: 'auto' }}>
          {navKeys.map((item) => (
            <NavLink
              key={item.path}
              label={getNavLabel(item)}
              leftSection={<item.icon size={18} />}
              active={location.pathname === item.path}
              onClick={() => navigate(item.path)}
              mb={4}
            />
          ))}

          <Divider my="sm" label={t('nav.tools')} labelPosition="left" />

          {toolKeys.map((item) => (
            <NavLink
              key={item.path}
              label={t(item.key)}
              leftSection={<item.icon size={18} />}
              active={location.pathname === item.path}
              onClick={() => navigate(item.path)}
              mb={4}
            />
          ))}

        </MantineAppShell.Section>

        <MantineAppShell.Section>
          <Divider mb="sm" />
          <NavLink
            label={t('nav.settings')}
            leftSection={<IconSettings size={18} />}
            active={location.pathname === '/settings'}
            onClick={() => navigate('/settings')}
          />
          <NavLink
            label={t('about.title')}
            leftSection={<IconInfoCircle size={18} />}
            active={location.pathname === '/about'}
            onClick={() => navigate('/about')}
          />
        </MantineAppShell.Section>
      </MantineAppShell.Navbar>

      <MantineAppShell.Main>
        {children}
      </MantineAppShell.Main>
    </MantineAppShell>
  )
}
