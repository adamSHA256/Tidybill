import { useState, useEffect } from 'react'
import { Stack, TextInput, NumberInput, Switch, Select, Button, Group, Text, PasswordInput, Divider, Tooltip } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, type SmtpConfig, type SmtpConfigInput } from '../api/client'
import { useT } from '../i18n'
import { IconPlugConnected, IconInfoCircle } from '@tabler/icons-react'

const providerPresets: Record<string, { host: string; port: number; starttls: boolean }> = {
  gmail: { host: 'smtp.gmail.com', port: 587, starttls: true },
  outlook: { host: 'smtp.office365.com', port: 587, starttls: true },
  yahoo: { host: 'smtp.mail.yahoo.com', port: 587, starttls: true },
  icloud: { host: 'smtp.mail.me.com', port: 587, starttls: true },
  seznam: { host: 'smtp.seznam.cz', port: 465, starttls: false },
  protonmail: { host: 'smtp.protonmail.ch', port: 587, starttls: true },
  custom: { host: '', port: 587, starttls: true },
}

interface Props {
  supplierId: string
  supplierName: string
  supplierEmail?: string
}

export function SmtpConfigForm({ supplierId, supplierName, supplierEmail }: Props) {
  const { t } = useT()
  const queryClient = useQueryClient()

  // Form state
  const [provider, setProvider] = useState<string>('custom')
  const [host, setHost] = useState('')
  const [port, setPort] = useState<number>(587)
  const [starttls, setStarttls] = useState(true)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [fromName, setFromName] = useState('')
  const [fromEmail, setFromEmail] = useState('')
  const [enabled, setEnabled] = useState(true)

  // Load existing config
  const { data: configData } = useQuery({
    queryKey: ['smtp-config', supplierId],
    queryFn: () => api.getSmtpConfig(supplierId),
  })

  // Load all suppliers for copy-from feature
  const { data: suppliers } = useQuery({
    queryKey: ['suppliers'],
    queryFn: api.getSuppliers,
  })

  const existingConfig = configData && 'id' in configData ? configData as SmtpConfig : null

  // Sync form state from loaded config
  useEffect(() => {
    if (existingConfig) {
      setHost(existingConfig.host)
      setPort(existingConfig.port)
      setStarttls(existingConfig.use_starttls)
      setUsername(existingConfig.username)
      setPassword('')
      setFromName(existingConfig.from_name)
      setFromEmail(existingConfig.from_email)
      setEnabled(existingConfig.enabled)

      // Detect provider from host
      const detected = Object.entries(providerPresets).find(
        ([key, preset]) => key !== 'custom' && preset.host === existingConfig.host
      )
      setProvider(detected ? detected[0] : 'custom')
    } else if (configData && !('id' in configData)) {
      // No existing config — prefill from supplier info
      if (supplierName) setFromName(supplierName)
      if (supplierEmail) setFromEmail(supplierEmail)
    }
  }, [existingConfig, configData, supplierName, supplierEmail])

  const handleProviderChange = (value: string | null) => {
    const key = value || 'custom'
    setProvider(key)
    const preset = providerPresets[key]
    if (preset) {
      setHost(preset.host)
      setPort(preset.port)
      setStarttls(preset.starttls)
    }
  }

  const buildInput = (): SmtpConfigInput => ({
    host,
    port,
    username,
    password: password || undefined,
    from_name: fromName,
    from_email: fromEmail,
    use_starttls: starttls,
    enabled,
  })

  // Save mutation
  const saveMutation = useMutation({
    mutationFn: () => api.upsertSmtpConfig(supplierId, buildInput()),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['smtp-config', supplierId] })
      notifications.show({
        title: t('email.smtp_saved'),
        message: t('email.smtp_saved_msg'),
        color: 'green',
      })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  // Test connection mutation
  const testMutation = useMutation({
    mutationFn: () => api.testSmtpConnection(supplierId, buildInput()),
    onSuccess: () => {
      notifications.show({
        title: t('email.smtp_test'),
        message: t('email.smtp_test_success'),
        color: 'green',
      })
    },
    onError: (err: Error) => {
      notifications.show({
        title: t('email.smtp_test_failed'),
        message: err.message === 'TIMEOUT' ? t('email.send_timeout') : err.message,
        color: 'red',
      })
    },
  })

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: () => api.deleteSmtpConfig(supplierId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['smtp-config', supplierId] })
      setHost('')
      setPort(587)
      setStarttls(true)
      setUsername('')
      setPassword('')
      setFromName('')
      setFromEmail('')
      setEnabled(true)
      setProvider('custom')
      notifications.show({
        title: t('email.smtp_deleted'),
        message: '',
        color: 'green',
      })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  // Copy from another supplier
  const handleCopyFrom = (fromSupplierId: string | null) => {
    if (!fromSupplierId) return
    api.getSmtpConfig(fromSupplierId).then((data) => {
      if (data && 'id' in data) {
        const src = data as SmtpConfig
        setHost(src.host)
        setPort(src.port)
        setStarttls(src.use_starttls)
        setUsername(src.username)
        setPassword('') // password is never copied
        setFromName(src.from_name)
        setFromEmail(src.from_email)
        setEnabled(src.enabled)

        const detected = Object.entries(providerPresets).find(
          ([key, preset]) => key !== 'custom' && preset.host === src.host
        )
        setProvider(detected ? detected[0] : 'custom')
      }
    })
  }

  const otherSuppliers = (suppliers || []).filter((s) => s.id !== supplierId)

  return (
    <form onSubmit={(e) => e.preventDefault()} autoComplete="off">
    <Stack gap="md">
      <Group gap="xs">
        <IconPlugConnected size={16} />
        <Text size="sm" fw={600}>{t('email.smtp_title')}</Text>
        <Tooltip label={t('email.smtp_tooltip')} multiline w={300} withArrow events={{ hover: true, focus: true, touch: true }}>
          <IconInfoCircle size={14} style={{ opacity: 0.5, cursor: 'help' }} />
        </Tooltip>
      </Group>

      {otherSuppliers.length > 0 && (
        <Select
          label={
            <Group gap={4}>
              <span>{t('email.smtp_copy_from')}</span>
              <Tooltip label={t('email.smtp_copy_tooltip')} multiline w={300} withArrow events={{ hover: true, focus: true, touch: true }}>
                <IconInfoCircle size={14} style={{ opacity: 0.5, cursor: 'help' }} />
              </Tooltip>
            </Group>
          }
          data={otherSuppliers.map((s) => ({ value: s.id, label: s.name }))}
          onChange={handleCopyFrom}
          placeholder="..."
          clearable
          size="sm"
        />
      )}

      <Select
        label={t('email.smtp_provider')}
        data={[
          { value: 'gmail', label: 'Gmail' },
          { value: 'outlook', label: 'Outlook' },
          { value: 'yahoo', label: 'Yahoo' },
          { value: 'icloud', label: 'iCloud' },
          { value: 'seznam', label: 'Seznam' },
          { value: 'protonmail', label: 'ProtonMail' },
          { value: 'custom', label: t('email.smtp_custom') },
        ]}
        value={provider}
        onChange={handleProviderChange}
        allowDeselect={false}
        size="sm"
      />

      <Group grow>
        <TextInput
          label={t('email.smtp_host')}
          value={host}
          onChange={(e) => setHost(e.currentTarget.value)}
          size="sm"
        />
        <NumberInput
          label={t('email.smtp_port')}
          value={port}
          onChange={(v) => setPort(typeof v === 'number' ? v : 587)}
          min={1}
          max={65535}
          size="sm"
        />
      </Group>

      <Switch
        label={
          <Group gap={4}>
            <span>{t('email.smtp_starttls')}</span>
            <Tooltip label={t('email.smtp_starttls_tooltip')} multiline w={300} withArrow events={{ hover: true, focus: true, touch: true }}>
              <IconInfoCircle size={14} style={{ opacity: 0.5, cursor: 'help' }} />
            </Tooltip>
          </Group>
        }
        checked={starttls}
        onChange={(e) => setStarttls(e.currentTarget.checked)}
      />

      <TextInput
        label={t('email.smtp_username')}
        value={username}
        onChange={(e) => setUsername(e.currentTarget.value)}
        size="sm"
      />

      <PasswordInput
        label={t('email.smtp_password')}
        value={password}
        onChange={(e) => setPassword(e.currentTarget.value)}
        placeholder={existingConfig?.has_password ? '********' : ''}
        autoComplete="off"
        size="sm"
      />

      {provider === 'gmail' && (
        <Text size="xs" c="dimmed">{t('email.smtp_password_hint_gmail')}</Text>
      )}

      <Divider />

      <TextInput
        label={
          <Group gap={4}>
            <span>{t('email.smtp_from_name')}</span>
            <Tooltip label={t('email.smtp_from_name_tooltip')} multiline w={300} withArrow events={{ hover: true, focus: true, touch: true }}>
              <IconInfoCircle size={14} style={{ opacity: 0.5, cursor: 'help' }} />
            </Tooltip>
          </Group>
        }
        value={fromName}
        onChange={(e) => setFromName(e.currentTarget.value)}
        size="sm"
      />

      <TextInput
        label={
          <Group gap={4}>
            <span>{t('email.smtp_from_email')}</span>
            <Tooltip label={t('email.smtp_from_email_tooltip')} multiline w={300} withArrow events={{ hover: true, focus: true, touch: true }}>
              <IconInfoCircle size={14} style={{ opacity: 0.5, cursor: 'help' }} />
            </Tooltip>
          </Group>
        }
        value={fromEmail}
        onChange={(e) => setFromEmail(e.currentTarget.value)}
        size="sm"
      />

      <Group justify="space-between" mt="sm">
        <Group gap="sm">
          <Button
            variant="light"
            size="sm"
            leftSection={<IconPlugConnected size={14} />}
            onClick={() => testMutation.mutate()}
            loading={testMutation.isPending}
          >
            {t('email.smtp_test')}
          </Button>
          {existingConfig && (
            <Button
              variant="light"
              color="red"
              size="sm"
              onClick={() => deleteMutation.mutate()}
              loading={deleteMutation.isPending}
            >
              {t('common.delete')}
            </Button>
          )}
        </Group>
        <Button
          size="sm"
          onClick={() => saveMutation.mutate()}
          loading={saveMutation.isPending}
        >
          {t('email.smtp_save')}
        </Button>
      </Group>
    </Stack>
    </form>
  )
}
