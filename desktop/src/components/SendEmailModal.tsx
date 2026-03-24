import { useState, useEffect } from 'react'
import { Modal, Stack, TextInput, Textarea, Button, Group, Text, Alert, Switch } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../api/client'
import { useT } from '../i18n'
import { useIsMobile } from '../hooks/useIsMobile'
import { IconMail, IconAlertCircle, IconPaperclip } from '@tabler/icons-react'

interface Props {
  invoiceId: string
  opened: boolean
  onClose: () => void
}

export function SendEmailModal({ invoiceId, opened, onClose }: Props) {
  const { t } = useT()
  const isMobile = useIsMobile()
  const queryClient = useQueryClient()

  const [to, setTo] = useState('')
  const [subject, setSubject] = useState('')
  const [body, setBody] = useState('')
  const [sendCopy, setSendCopy] = useState(true)

  // Load email preview (pre-filled data)
  const { data: preview } = useQuery({
    queryKey: ['email-preview', invoiceId],
    queryFn: () => api.getEmailPreview(invoiceId),
    enabled: opened,
  })

  // Populate form from preview
  useEffect(() => {
    if (preview) {
      setTo(preview.to)
      setSubject(preview.subject)
      setBody(preview.body)
    }
  }, [preview])

  const handleSend = () => {
    if (!to || !subject) return

    // Close modal immediately and show persistent "sending" notification
    onClose()

    const sendingNotifId = 'sending-' + invoiceId
    notifications.show({
      id: sendingNotifId,
      title: t('email.sending'),
      message: t('email.sending_hint'),
      color: 'blue',
      loading: true,
      autoClose: false,
      withCloseButton: false,
    })

    api.sendInvoiceEmail(invoiceId, { to, subject, body, send_copy: sendCopy })
      .then(() => {
        notifications.hide(sendingNotifId)
        notifications.show({
          title: t('email.sent_success'),
          message: t('email.sent_success_msg'),
          color: 'green',
        })
        queryClient.invalidateQueries({ queryKey: ['invoice', invoiceId] })
        queryClient.invalidateQueries({ queryKey: ['invoices'] })
        queryClient.invalidateQueries({ queryKey: ['email-preview', invoiceId] })
      })
      .catch((err: Error) => {
        notifications.hide(sendingNotifId)
        const message = err.message === 'TIMEOUT'
          ? t('email.send_timeout')
          : err.message
        notifications.show({
          title: t('email.sent_error'),
          message,
          color: 'red',
        })
      })
  }

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={t('email.send_title')}
      size="lg"
      fullScreen={isMobile}
    >
      <Stack gap="md">
        <TextInput
          label={t('email.send_recipient')}
          value={to}
          onChange={(e) => setTo(e.currentTarget.value)}
          required
        />
        <TextInput
          label={t('email.send_subject')}
          value={subject}
          onChange={(e) => setSubject(e.currentTarget.value)}
          required
        />
        <Textarea
          label={t('email.send_body')}
          value={body}
          onChange={(e) => setBody(e.currentTarget.value)}
          minRows={6}
          autosize
        />

        {preview?.pdf_filename && (
          <Group gap="xs">
            <IconPaperclip size={14} />
            <Text size="sm" c="dimmed">{preview.pdf_filename}</Text>
          </Group>
        )}

        <Switch
          label={t('email.send_copy')}
          checked={sendCopy}
          onChange={(e) => setSendCopy(e.currentTarget.checked)}
        />

        {preview?.already_sent_at && (
          <Alert icon={<IconAlertCircle size={16} />} color="yellow" variant="light">
            {t('email.already_sent_warning')} ({new Date(preview.already_sent_at).toLocaleString()})
          </Alert>
        )}

        <Group justify="space-between" mt="md">
          <Button variant="default" onClick={onClose}>
            {t('wizard.back')}
          </Button>
          <Button
            leftSection={<IconMail size={16} />}
            onClick={handleSend}
            disabled={!to || !subject}
          >
            {t('email.send_button')}
          </Button>
        </Group>
      </Stack>
    </Modal>
  )
}
