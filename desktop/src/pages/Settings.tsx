import {
  Title,
  Text,
  Paper,
  Stack,
  Select,
  Switch,
  Group,
  Badge,
  Loader,
  Center,
  TextInput,
  Button,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useEffect } from 'react'
import { api } from '../api/client'
import { useT } from '../i18n'

const langOptions = [
  { value: 'cs', label: 'Cestina' },
  { value: 'sk', label: 'Slovencina' },
  { value: 'en', label: 'English' },
]

export function Settings() {
  const queryClient = useQueryClient()
  const { t, setLang } = useT()

  const { data: settings, isLoading } = useQuery({
    queryKey: ['settings'],
    queryFn: api.getSettings,
  })

  const [dirLogos, setDirLogos] = useState('')
  const [dirPdfs, setDirPdfs] = useState('')
  const [dirPreviews, setDirPreviews] = useState('')

  useEffect(() => {
    if (settings) {
      setDirLogos(settings.dir_logos || '')
      setDirPdfs(settings.dir_pdfs || '')
      setDirPreviews(settings.dir_previews || '')
    }
  }, [settings])

  const updateMutation = useMutation({
    mutationFn: api.updateSettings,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings'] })
      notifications.show({ title: t('notify.settings_saved'), message: t('notify.settings_saved_msg'), color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  if (isLoading) {
    return <Center h={300}><Loader /></Center>
  }

  const comingSoonBadge = <Badge size="xs" variant="light" color="gray">{t('settings.coming_soon')}</Badge>

  return (
    <Stack gap="lg">
      <div>
        <Title order={2}>{t('settings.title')}</Title>
        <Text c="dimmed" size="sm">{t('settings.subtitle')}</Text>
      </div>

      <Paper p="md" radius="md" withBorder>
        <Text fw={500} mb="md">{t('settings.general')}</Text>
        <Stack gap="md">
          <Select
            label={t('settings.language')}
            data={langOptions}
            value={settings?.language || 'cs'}
            onChange={(v) => { if (v) setLang(v as 'cs' | 'sk' | 'en') }}
            w={300}
          />
          <Group gap="xs">
            <Select
              label={t('settings.default_currency')}
              data={['CZK', 'EUR', 'USD']}
              defaultValue="CZK"
              w={300}
              disabled
            />
            {comingSoonBadge}
          </Group>
          <Group gap="xs">
            <Select
              label={t('settings.date_format')}
              data={['DD.MM.YYYY', 'YYYY-MM-DD', 'MM/DD/YYYY']}
              defaultValue="DD.MM.YYYY"
              w={300}
              disabled
            />
            {comingSoonBadge}
          </Group>
        </Stack>
      </Paper>

      <Paper p="md" radius="md" withBorder>
        <Text fw={500} mb="md">{t('settings.directories')}</Text>
        <Stack gap="md">
          <TextInput
            label={t('settings.dir_logos')}
            placeholder={t('settings.dir_placeholder')}
            value={dirLogos}
            onChange={(e) => setDirLogos(e.currentTarget.value)}
            w={500}
          />
          <TextInput
            label={t('settings.dir_pdfs')}
            placeholder={t('settings.dir_placeholder')}
            value={dirPdfs}
            onChange={(e) => setDirPdfs(e.currentTarget.value)}
            w={500}
          />
          <TextInput
            label={t('settings.dir_previews')}
            placeholder={t('settings.dir_placeholder')}
            value={dirPreviews}
            onChange={(e) => setDirPreviews(e.currentTarget.value)}
            w={500}
          />
          <Button
            w={200}
            onClick={() => updateMutation.mutate({
              dir_logos: dirLogos,
              dir_pdfs: dirPdfs,
              dir_previews: dirPreviews,
            })}
            loading={updateMutation.isPending}
          >
            {t('settings.save_directories')}
          </Button>
        </Stack>
      </Paper>

      <Paper p="md" radius="md" withBorder>
        <Group gap="xs" mb="md">
          <Text fw={500}>{t('settings.invoices')}</Text>
          {comingSoonBadge}
        </Group>
        <Stack gap="md">
          <Select
            label={t('settings.default_vat')}
            data={['0%', '12%', '21%']}
            defaultValue="21%"
            w={300}
            disabled
          />
          <Select
            label={t('settings.default_due')}
            data={['7', '14', '30', '60']}
            defaultValue="14"
            w={300}
            disabled
          />
          <Group>
            <Switch label={t('settings.auto_number')} defaultChecked disabled />
          </Group>
        </Stack>
      </Paper>

      <Paper p="md" radius="md" withBorder>
        <Group gap="xs" mb="md">
          <Text fw={500}>{t('settings.pdf_output')}</Text>
          {comingSoonBadge}
        </Group>
        <Stack gap="md">
          <Select
            label={t('settings.default_template')}
            data={['Classic', 'Modern', 'Minimal']}
            defaultValue="Classic"
            w={300}
            disabled
          />
          <Group>
            <Switch label={t('settings.include_qr')} defaultChecked disabled />
          </Group>
          <Group>
            <Switch label={t('settings.include_logo')} defaultChecked disabled />
          </Group>
        </Stack>
      </Paper>
    </Stack>
  )
}
