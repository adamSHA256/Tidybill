import { useNavigate } from 'react-router-dom'
import {
  NavLink,
  Stack,
  Title,
  Divider,
  Group,
  ActionIcon,
  useMantineColorScheme,
} from '@mantine/core'
import {
  IconUsers,
  IconBuildingStore,
  IconTemplate,
  IconPackage,
  IconSettings,
  IconSun,
  IconMoon,
  IconInfoCircle,
  IconMail,
  IconDatabaseExport,
} from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'
import { useT } from '../i18n'

export function MorePage() {
  const navigate = useNavigate()
  const { colorScheme, toggleColorScheme } = useMantineColorScheme()
  const { t } = useT()

  const { data: suppliers } = useQuery({
    queryKey: ['suppliers'],
    queryFn: api.getSuppliers,
  })
  const supplierCount = (suppliers || []).length

  const mainItems = [
    { key: 'nav.customers', icon: IconUsers, path: '/customers' },
    {
      key: supplierCount > 1 ? 'nav.suppliers_plural' : 'nav.suppliers',
      icon: IconBuildingStore,
      path: '/suppliers',
    },
  ]

  const toolItems = [
    { key: 'nav.templates', icon: IconTemplate, path: '/templates' },
    { key: 'nav.items', icon: IconPackage, path: '/items' },
  ]

  return (
    <Stack>
      <Group justify="space-between">
        <Title order={3}>TidyBill</Title>
        <ActionIcon variant="subtle" onClick={toggleColorScheme} size="lg">
          {colorScheme === 'dark' ? <IconSun size={20} /> : <IconMoon size={20} />}
        </ActionIcon>
      </Group>

      {mainItems.map((item) => (
        <NavLink
          key={item.path}
          label={t(item.key)}
          leftSection={<item.icon size={20} />}
          onClick={() => navigate(item.path)}
        />
      ))}

      <Divider label={t('nav.tools')} labelPosition="left" />

      {toolItems.map((item) => (
        <NavLink
          key={item.path}
          label={t(item.key)}
          leftSection={<item.icon size={20} />}
          onClick={() => navigate(item.path)}
        />
      ))}

      <Divider />

      <NavLink
        label={t('nav.automatizace')}
        leftSection={<IconMail size={20} />}
        onClick={() => navigate('/automatizace')}
      />
      <NavLink
        label={t('nav.sync')}
        leftSection={<IconDatabaseExport size={20} />}
        onClick={() => navigate('/sync')}
      />
      <NavLink
        label={t('nav.settings')}
        leftSection={<IconSettings size={20} />}
        onClick={() => navigate('/settings')}
      />
      <NavLink
        label={t('about.title')}
        leftSection={<IconInfoCircle size={20} />}
        onClick={() => navigate('/about')}
      />
    </Stack>
  )
}
