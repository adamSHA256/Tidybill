import { useState } from 'react'
import { Stack, Button, TextInput, PasswordInput, NumberInput, Select, Checkbox, Alert, Loader, Center } from '@mantine/core'
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

  const { data: backendsData, isLoading } = useQuery({
    queryKey: ['rclone-backends'],
    queryFn: () => api.cloud.rcloneBackends(),
  })

  if (isLoading) {
    return <Center><Loader size="sm" /></Center>
  }

  const backend = backendsData?.backends.find((b) => b.id === backendId)
  if (!backend) {
    return <Alert color="red">Unknown backend: {backendId}</Alert>
  }

  const handleSubmit = async () => {
    setError(null)
    setConnecting(true)
    try {
      const result = await api.cloud.rcloneConnect(backendId, values)
      notifications.show({
        title: 'Connected',
        message: `Connected to ${result.account_label}`,
        color: 'green',
      })
      onConnected()
      onClose()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setConnecting(false)
    }
  }

  const setValue = (name: string, val: string) => {
    setValues((prev) => ({ ...prev, [name]: val }))
  }

  return (
    <Stack gap="md">
      {error && (
        <Alert color="red" icon={<IconAlertCircle size={16} />}>
          {error}
        </Alert>
      )}
      {backend.fields.map((field) => {
        const label = t(`cloud.rclone.${backendId}.${field.name}`) || field.name
        if (field.kind === 'password') {
          return (
            <PasswordInput
              key={field.name}
              label={label}
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
            required={field.required}
            placeholder={field.default}
            value={values[field.name] || ''}
            onChange={(e) => setValue(field.name, e.currentTarget.value)}
          />
        )
      })}
      {/* S3 bucket is a special field not from the backend schema */}
      {backendId === 's3' && (
        <TextInput
          label={t('cloud.rclone.s3.bucket')}
          required
          value={values['bucket'] || ''}
          onChange={(e) => setValue('bucket', e.currentTarget.value)}
        />
      )}
      <Button onClick={handleSubmit} loading={connecting}>
        Connect
      </Button>
      <Button variant="subtle" onClick={onClose}>Cancel</Button>
    </Stack>
  )
}
