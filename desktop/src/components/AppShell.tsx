import type { ReactNode } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import {
  AppShell as MantineAppShell,
  NavLink,
  Group,
  ActionIcon,
  useMantineColorScheme,
  Divider,
  Badge,
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
} from '@tabler/icons-react'
import { useT } from '../i18n'
import tidybillLogo from '../assets/tidybill_logo.svg'

const navKeys = [
  { key: 'nav.dashboard', icon: IconDashboard, path: '/' },
  { key: 'nav.invoices', icon: IconFileInvoice, path: '/invoices' },
  { key: 'nav.customers', icon: IconUsers, path: '/customers' },
  { key: 'nav.suppliers', icon: IconBuildingStore, path: '/suppliers' },
]

const toolKeys = [
  { key: 'nav.templates', icon: IconTemplate, path: '/templates' },
  { key: 'nav.items', icon: IconPackage, path: '/items' },
]

export function AppShell({ children }: { children: ReactNode }) {
  const location = useLocation()
  const navigate = useNavigate()
  const { colorScheme, toggleColorScheme } = useMantineColorScheme()
  const { t } = useT()

  return (
    <MantineAppShell
      navbar={{ width: 260, breakpoint: 'sm' }}
      padding="md"
    >
      <MantineAppShell.Navbar p="md">
        <MantineAppShell.Section>
          <Group justify="space-between" mb="md">
            <img src={tidybillLogo} alt="TidyBill" height={36} />
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

        <MantineAppShell.Section grow>
          {navKeys.map((item) => (
            <NavLink
              key={item.path}
              label={t(item.key)}
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

          <Divider my="sm" label={t('nav.future')} labelPosition="left" />

          <NavLink
            label={t('nav.template_designer')}
            leftSection={<IconTemplate size={18} />}
            disabled
            rightSection={<Badge size="xs" variant="light" color="gray">{t('nav.later')}</Badge>}
          />
        </MantineAppShell.Section>

        <MantineAppShell.Section>
          <Divider mb="sm" />
          <NavLink
            label={t('nav.settings')}
            leftSection={<IconSettings size={18} />}
            active={location.pathname === '/settings'}
            onClick={() => navigate('/settings')}
          />
        </MantineAppShell.Section>
      </MantineAppShell.Navbar>

      <MantineAppShell.Main>
        {children}
      </MantineAppShell.Main>
    </MantineAppShell>
  )
}
