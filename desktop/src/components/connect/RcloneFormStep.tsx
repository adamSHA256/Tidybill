import { useState } from 'react'
import { Stack, Button, TextInput, PasswordInput, NumberInput, Select, Checkbox, Alert, Loader, Center, Collapse, Anchor } from '@mantine/core'
import { IconAlertCircle } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../../api/client'
import { useT } from '../../i18n'
import { notifications } from '@mantine/notifications'

interface RcloneFormStepProps {
  backendId: string
  onClose: () => void
  onConnected: () => void
}

export function RcloneFormStep({ backendId, onClose, onConnected }: RcloneFormStepProps) {
  const { t } = useT()
  const [values, setValues] = useState<Record<string, string>>({})
  const [connecting, setConnecting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [advancedOpen, setAdvancedOpen] = useState(false)

  const { data: backendsData, isLoading } = useQuery({
    queryKey: ['rclone-backends'],
    queryFn: () => api.cloud.rcloneBackends(),
  })

  if (isLoading) {
    return <Center><Loader size="sm" /></Center>
  }

  const backend = backendsData?.backends.find((b) => b.id === backendId)
  if (!backend) {
    return <Alert color="red">{t('cloud.connect.unknown_backend').replace('{backend}', backendId)}</Alert>
  }

  const handleSubmit = async () => {
    setError(null)
    setConnecting(true)
    try {
      const result = await api.cloud.rcloneConnect(backendId, values)
      notifications.show({
        title: t('cloud.connect.connected_title'),
        message: t('cloud.connect.connected_message').replace('{target}', result.account_label),
        color: 'green',
      })
      onConnected()
      onClose()
    } catch (err: unknown) {
      const rawMsg = err instanceof Error ? err.message : String(err)
      // Proton Drive returns {error_code, error_raw} — look up a translated message.
      if (backendId === 'protondrive') {
        try {
          const parsed = JSON.parse(rawMsg) as { error_code?: string; error_raw?: string }
          if (parsed.error_code) {
            const key = `cloud.rclone.protondrive.error.${parsed.error_code}`
            const translated = t(key)
            setError(translated || parsed.error_raw || rawMsg)
            return
          }
        } catch { /* not JSON — fall through */ }
      }
      setError(rawMsg)
    } finally {
      setConnecting(false)
    }
  }

  const setValue = (name: string, val: string) => {
    setValues((prev) => ({ ...prev, [name]: val }))
  }

  const isProton = backendId === 'protondrive'

  // Treat "t returned the key itself" as "translation missing", so we can
  // fall through to a sensible default instead of showing a raw dotted key.
  const tOpt = (key: string): string | undefined => {
    const v = t(key)
    return v === key ? undefined : v
  }

  const renderField = (field: typeof backend.fields[number]) => {
    if (field.generated) return null

    const label = tOpt(`cloud.rclone.${backendId}.${field.name}.label`) ?? field.name
    const description = tOpt(`cloud.rclone.${backendId}.${field.name}.help`)

    if (field.kind === 'password') {
      return (
        <PasswordInput
          key={field.name}
          label={label}
          description={description}
          required={field.required}
          value={values[field.name] || ''}
          onChange={(e) => setValue(field.name, e.currentTarget.value)}
        />
      )
    }
    if (field.kind === 'number') {
      return (
        <NumberInput
          key={field.name}
          label={label}
          description={description}
          required={field.required}
          value={values[field.name] ? parseInt(values[field.name]) : (field.default ? parseInt(field.default) : '')}
          onChange={(val) => setValue(field.name, String(val))}
        />
      )
    }
    if (field.kind === 'select' && field.options) {
      return (
        <Select
          key={field.name}
          label={label}
          description={description}
          required={field.required}
          data={field.options}
          value={values[field.name] || field.default || null}
          onChange={(val) => setValue(field.name, val || '')}
        />
      )
    }
    if (field.kind === 'checkbox') {
      return (
        <Checkbox
          key={field.name}
          label={label}
          checked={values[field.name] === 'true'}
          onChange={(e) => setValue(field.name, e.currentTarget.checked ? 'true' : 'false')}
        />
      )
    }
    return (
      <TextInput
        key={field.name}
        label={label}
        description={description}
        required={field.required}
        placeholder={field.default}
        value={values[field.name] || ''}
        onChange={(e) => setValue(field.name, e.currentTarget.value)}
      />
    )
  }

  // For Proton Drive, split fields into main (required user-input) and advanced (optional).
  const advancedFieldNames = isProton ? ['mailbox_password'] : []
  const mainFields = backend.fields.filter((f) => !f.generated && !advancedFieldNames.includes(f.name))
  const advancedFields = backend.fields.filter((f) => !f.generated && advancedFieldNames.includes(f.name))

  return (
    <Stack gap="md">
      {error && (
        <Alert color="red" icon={<IconAlertCircle size={16} />}>
          {error}
        </Alert>
      )}
      {mainFields.map(renderField)}
      {/* S3 bucket is a special field not from the backend schema */}
      {backendId === 's3' && (
        <TextInput
          label={tOpt('cloud.rclone.s3.bucket.label') ?? 'Bucket'}
          description={tOpt('cloud.rclone.s3.bucket.help')}
          required
          value={values['bucket'] || ''}
          onChange={(e) => setValue('bucket', e.currentTarget.value)}
        />
      )}
      {advancedFields.length > 0 && (
        <>
          <Anchor size="sm" onClick={() => setAdvancedOpen((o) => !o)} style={{ cursor: 'pointer' }}>
            {t('cloud.rclone.protondrive.advanced_toggle')}
          </Anchor>
          <Collapse in={advancedOpen}>
            <Stack gap="md">
              {advancedFields.map(renderField)}
            </Stack>
          </Collapse>
        </>
      )}
      <Button onClick={handleSubmit} loading={connecting}>
        {isProton && connecting ? t('cloud.rclone.protondrive.connecting') : t('cloud.connect.connect_btn')}
      </Button>
      <Button variant="subtle" onClick={onClose}>{t('cloud.connect.cancel_btn')}</Button>
    </Stack>
  )
}
