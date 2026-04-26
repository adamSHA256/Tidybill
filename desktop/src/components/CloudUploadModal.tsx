import { useState } from 'react'
import { Modal, Stack, PasswordInput, Button, Group, Text, Anchor } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../api/client'
import { useT } from '../i18n'

interface Props {
  opened: boolean
  transportId: string | null
  onClose: () => void
}

export function CloudUploadModal({ opened, transportId, onClose }: Props) {
  const { t } = useT()
  const qc = useQueryClient()
  const [passphrase, setPassphrase] = useState('')
  const [unencryptedConfirm, setUnencryptedConfirm] = useState(false)

  const generateMnemonic = useMutation({
    mutationFn: () => api.generateMnemonic(),
    onSuccess: (data) => setPassphrase(data.mnemonic),
  })

  const upload = useMutation({
    mutationFn: () =>
      api.cloud.upload(transportId!, unencryptedConfirm ? undefined : passphrase || undefined),
    onSuccess: () => {
      notifications.show({ message: t('cloud.upload.success'), color: 'green' })
      qc.invalidateQueries({ queryKey: ['cloud-transports'] })
      handleClose()
    },
    onError: (e: Error) => {
      notifications.show({ message: `${t('cloud.upload.failed')}: ${e.message}`, color: 'red' })
    },
  })

  function handleClose() {
    setPassphrase('')
    setUnencryptedConfirm(false)
    onClose()
  }

  const canSubmit = unencryptedConfirm || passphrase.length >= 8

  return (
    <Modal
      opened={opened}
      onClose={handleClose}
      title={t('cloud.upload_action')}
      size="sm"
      // While the upload is in flight, lock the modal: clicking outside
      // or pressing Escape would close the modal but the mutation keeps
      // running invisibly — better to force the user to wait for the
      // notification (success or failure) before doing anything else.
      closeOnClickOutside={!upload.isPending}
      closeOnEscape={!upload.isPending}
      withCloseButton={!upload.isPending}
    >
      <Stack gap="md">
        {!unencryptedConfirm ? (
          <>
            <PasswordInput
              label={t('backup.passphrase')}
              placeholder={t('backup.passphrase_placeholder')}
              value={passphrase}
              onChange={(e) => setPassphrase(e.currentTarget.value)}
              autoFocus
            />
            <Group gap="xs">
              <Button
                variant="subtle"
                size="xs"
                leftSection={<span>🔑</span>}
                onClick={() => generateMnemonic.mutate()}
                loading={generateMnemonic.isPending}
              >
                {t('backup.generate_mnemonic')}
              </Button>
            </Group>
            <Text size="xs" c="dimmed">
              {t('cloud.upload.encrypt_hint')}{' '}
              <Anchor
                size="xs"
                onClick={() => setUnencryptedConfirm(true)}
              >
                {t('cloud.upload.unencrypted_link')}
              </Anchor>
            </Text>
          </>
        ) : (
          <Text size="sm" c="orange">
            {t('cloud.upload.unencrypted_warning').replace('{target}', transportId ?? '')}
          </Text>
        )}
        <Group justify="flex-end">
          <Button variant="default" onClick={handleClose}>
            {t('common.cancel')}
          </Button>
          {unencryptedConfirm && (
            <Button variant="default" onClick={() => setUnencryptedConfirm(false)}>
              {t('cloud.upload.back_to_encrypted')}
            </Button>
          )}
          <Button
            onClick={() => upload.mutate()}
            loading={upload.isPending}
            disabled={!canSubmit}
          >
            {t('cloud.upload_action')}
          </Button>
        </Group>
      </Stack>
    </Modal>
  )
}
