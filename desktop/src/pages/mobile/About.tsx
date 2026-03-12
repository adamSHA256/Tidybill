import {
  Stack,
  Title,
  Text,
  Badge,
  Group,
  Anchor,
  Code,
  ActionIcon,
  Tooltip,
  Paper,
  CopyButton,
} from '@mantine/core'
import { IconCopy, IconCheck, IconInfoCircle } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { api, openInBrowser } from '../../api/client'
import { useT } from '../../i18n'

export function MobileAbout() {
  const { t } = useT()
  const { data: aboutInfo } = useQuery({
    queryKey: ['about'],
    queryFn: api.getAbout,
  })

  if (!aboutInfo) return null

  return (
    <Stack gap="md">
      <Group gap="sm">
        <Title order={3}>TidyBill</Title>
        <Badge variant="light" color="blue" size="lg">v{aboutInfo.version}</Badge>
      </Group>

      <Paper p="md" radius="md" withBorder>
        <Text size="sm" mb="xs">{t('about.description')}</Text>
        <Text size="sm" c="dimmed">{t('about.opensource')}</Text>
      </Paper>

      <Paper p="md" radius="md" withBorder>
        <Text fw={500} size="sm" mb={4}>{t('about.issues_title')}</Text>
        <Anchor
          size="sm"
          onClick={(e: React.MouseEvent) => { e.preventDefault(); openInBrowser(aboutInfo.github_issues_url) }}
          style={{ cursor: 'pointer' }}
        >
          {t('about.issues_link')}
        </Anchor>
      </Paper>

      <Paper p="md" radius="md" withBorder>
        <Text fw={500} size="sm" mb={4}>{t('about.support_title')}</Text>
        <Text size="sm" c="dimmed" mb="sm">{t('about.support_desc')}</Text>
        <Stack gap="xs">
          <Group gap="xs">
            <Text size="sm" fw={500} w={100}>Monero (XMR)</Text>
            <CopyButton value={aboutInfo.monero_address.replace(/^<|>$/g, '')}>
              {({ copied, copy }) => (
                <Tooltip label={copied ? t('about.copied') : t('common.copy')} events={{ hover: true, focus: true, touch: true }}>
                  <ActionIcon variant="subtle" color={copied ? 'teal' : 'gray'} onClick={copy}>
                    {copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
                  </ActionIcon>
                </Tooltip>
              )}
            </CopyButton>
            <Code style={{ wordBreak: 'break-all', fontSize: 10 }}>{aboutInfo.monero_address}</Code>
          </Group>
          <Group gap="xs">
            <Text size="sm" fw={500} w={100}>Bitcoin (BTC)</Text>
            <CopyButton value={aboutInfo.bitcoin_address.replace(/^<|>$/g, '')}>
              {({ copied, copy }) => (
                <Tooltip label={copied ? t('about.copied') : t('common.copy')} events={{ hover: true, focus: true, touch: true }}>
                  <ActionIcon variant="subtle" color={copied ? 'teal' : 'gray'} onClick={copy}>
                    {copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
                  </ActionIcon>
                </Tooltip>
              )}
            </CopyButton>
            <Code style={{ wordBreak: 'break-all', fontSize: 10 }}>{aboutInfo.bitcoin_address}</Code>
          </Group>
        </Stack>
      </Paper>
    </Stack>
  )
}
