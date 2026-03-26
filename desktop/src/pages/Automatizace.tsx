import { useState, useEffect } from 'react'
import { Container, Paper, Stack, Text, TextInput, Textarea, Button, Title, Divider } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../api/client'
import { useT } from '../i18n'
import { PlaceholderChips } from '../components/PlaceholderChips'
import { useNavigate } from 'react-router-dom'

export function Automatizace() {
  const { t } = useT()
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  const { data: settings } = useQuery({
    queryKey: ['settings'],
    queryFn: api.getSettings,
  })

  const [emailSubject, setEmailSubject] = useState('Faktura ((number))')
  const [emailBody, setEmailBody] = useState('Dobrý den,\n\nv příloze zasílám fakturu č. ((number)) na částku ((total)).\nSplatnost: ((due_date)).\n\nS pozdravem\n((supplier))')
  const [copySubject, setCopySubject] = useState('TidyBill - ((subject))')

  useEffect(() => {
    if (settings) {
      if (settings['email.default_subject']) setEmailSubject(settings['email.default_subject'])
      if (settings['email.default_body']) setEmailBody(settings['email.default_body'])
      if (settings['email.copy_subject']) setCopySubject(settings['email.copy_subject'])

      // Persist defaults to DB if they don't exist yet (so they appear in exports)
      const missing: Record<string, string> = {}
      if (!settings['email.default_subject']) missing['email.default_subject'] = 'Faktura ((number))'
      if (!settings['email.default_body']) missing['email.default_body'] = 'Dobrý den,\n\nv příloze zasílám fakturu č. ((number)) na částku ((total)).\nSplatnost: ((due_date)).\n\nS pozdravem\n((supplier))'
      if (!settings['email.copy_subject']) missing['email.copy_subject'] = 'TidyBill - ((subject))'
      if (Object.keys(missing).length > 0) {
        api.updateSettings(missing).then(() => {
          queryClient.invalidateQueries({ queryKey: ['settings'] })
        })
      }
    }
  }, [settings]) // eslint-disable-line react-hooks/exhaustive-deps

  const saveMutation = useMutation({
    mutationFn: () => api.updateSettings({ 'email.default_subject': emailSubject, 'email.default_body': emailBody, 'email.copy_subject': copySubject }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings'] })
      notifications.show({ title: t('email.template_saved'), message: t('email.template_saved_msg'), color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  return (
    <Container size="sm" py="xl">
      <Title order={2} mb="lg">{t('email.automatizace_title')}</Title>

      <Paper p="md" radius="md" withBorder>
        <Stack gap="md">
          <Text c="dimmed" size="sm">{t('email.automatizace_desc')}</Text>

          <TextInput
            label={t('email.default_subject_label')}
            value={emailSubject}
            onChange={(e) => setEmailSubject(e.currentTarget.value)}
          />
          <PlaceholderChips onInsert={(p) => setEmailSubject(prev => prev + ' ' + p)} />

          <Textarea
            label={t('email.default_body_label')}
            value={emailBody}
            onChange={(e) => setEmailBody(e.currentTarget.value)}
            minRows={5}
            autosize
          />
          <PlaceholderChips onInsert={(p) => setEmailBody(prev => prev + p)} />

          <Divider my="sm" />

          <TextInput
            label={t('email.copy_subject_label')}
            description={t('email.copy_subject_desc')}
            value={copySubject}
            onChange={(e) => setCopySubject(e.currentTarget.value)}
          />

          <Text c="dimmed" size="xs">{t('email.placeholder_hint')}</Text>

          <Button w={200} onClick={() => saveMutation.mutate()} loading={saveMutation.isPending}>
            {t('email.save_template')}
          </Button>
        </Stack>
      </Paper>

      <Paper p="md" radius="md" withBorder mt="md">
        <Stack gap="sm">
          <Text c="dimmed" size="sm">{t('email.smtp_in_supplier')}</Text>
          <Button variant="light" onClick={() => navigate('/suppliers')}>
            {t('email.goto_suppliers')}
          </Button>
        </Stack>
      </Paper>
    </Container>
  )
}
