import { useState } from 'react'
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
  Switch,
  NavLink,
  Modal,
  Button,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { IconCopy, IconCheck, IconDownload, IconRefresh } from '@tabler/icons-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api, openInBrowser } from '../api/client'
import { useT } from '../i18n'

export function About() {
  const { t } = useT()
  const queryClient = useQueryClient()
  const [checking, setChecking] = useState(false)
  const [enableAutoModalOpen, setEnableAutoModalOpen] = useState(false)

  const { data: aboutInfo } = useQuery({
    queryKey: ['about'],
    queryFn: api.getAbout,
  })

  const { data: settings } = useQuery({
    queryKey: ['settings'],
    queryFn: api.getSettings,
  })

  const { data: updateResult } = useQuery({
    queryKey: ['update-check'],
    queryFn: api.getUpdateCheck,
  })

  const autoCheckEnabled = settings?.check_updates === 'true'

  const handleManualCheck = async () => {
    setChecking(true)
    try {
      const result = await api.triggerUpdateCheck()
      queryClient.setQueryData(['update-check'], result)
      if (!autoCheckEnabled) {
        setEnableAutoModalOpen(true)
      }
    } catch {
      notifications.show({
        title: t('common.error'),
        message: t('update.check_failed'),
        color: 'red',
      })
    } finally {
      setChecking(false)
    }
  }

  const handleEnableAutoCheck = async (enable: boolean) => {
    setEnableAutoModalOpen(false)
    if (enable) {
      await api.updateSettings({ check_updates: 'true' })
      queryClient.invalidateQueries({ queryKey: ['settings'] })
    }
  }

  const handleAutoCheckToggle = async (checked: boolean) => {
    await api.updateSettings({ check_updates: checked ? 'true' : 'false' })
    queryClient.invalidateQueries({ queryKey: ['settings'] })
  }

  if (!aboutInfo) return null

  return (
    <Stack gap="lg">
      <Group gap="sm">
        <Title order={2}>TidyBill</Title>
        <Badge variant="light" color="blue" size="lg">v{aboutInfo.version}</Badge>
      </Group>

      {/* Update check */}
      <Paper p="md" radius="md" withBorder>
        {updateResult?.checked_at ? (
          updateResult.available ? (
            <NavLink
              label={t('update.available')}
              leftSection={<IconDownload size={20} />}
              onClick={() => openInBrowser(updateResult.release_url)}
              active
              color="blue"
              styles={{ root: { borderRadius: 'var(--mantine-radius-sm)' } }}
            />
          ) : (
            <Text size="sm" c="dimmed" ta="center" py="xs">
              {t('update.up_to_date')}
            </Text>
          )
        ) : (
          <NavLink
            label={checking ? t('update.checking') : t('update.check_manually')}
            leftSection={checking ? <IconRefresh size={20} /> : <IconCheck size={20} />}
            onClick={handleManualCheck}
            disabled={checking}
            styles={{ root: { borderRadius: 'var(--mantine-radius-sm)' } }}
          />
        )}
        <Tooltip
          label={t('update.check_tooltip')}
          multiline
          w={350}
          withArrow
          events={{ hover: true, focus: true, touch: true }}
        >
          <Switch
            mt="md"
            label={t('update.settings_label')}
            description={t('update.settings_desc')}
            checked={autoCheckEnabled}
            onChange={(e) => handleAutoCheckToggle(e.currentTarget.checked)}
          />
        </Tooltip>
      </Paper>

      {/* About */}
      <Paper p="md" radius="md" withBorder>
        <Text size="sm" mb="xs">{t('about.description')}</Text>
        <Text size="sm" c="dimmed">{t('about.opensource')}</Text>
      </Paper>

      {/* Issues */}
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

      {/* Support */}
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
            <Code>{aboutInfo.monero_address}</Code>
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
            <Code>{aboutInfo.bitcoin_address}</Code>
          </Group>
        </Stack>
      </Paper>

      <Modal
        opened={enableAutoModalOpen}
        onClose={() => setEnableAutoModalOpen(false)}
        title={t('update.enable_auto_title')}
        centered
      >
        <Text mb="lg">{t('update.enable_auto_message')}</Text>
        <Group justify="flex-end">
          <Button variant="default" onClick={() => handleEnableAutoCheck(false)}>
            {t('update.enable_auto_no')}
          </Button>
          <Button onClick={() => handleEnableAutoCheck(true)}>
            {t('update.enable_auto_yes')}
          </Button>
        </Group>
      </Modal>
    </Stack>
  )
}
