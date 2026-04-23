import { useState } from 'react'
import { Modal, Stack, SimpleGrid, Card, Text, Badge } from '@mantine/core'
import { IconBrandGoogleDrive, IconServer, IconCloud, IconBrandDropbox, IconBrandOnedrive } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useT } from '../i18n'
import { api } from '../api/client'
import { GDriveStep } from './connect/GDriveStep'
import { RcloneFormStep } from './connect/RcloneFormStep'

type Step = 'pick' | 'gdrive' | 'sftp' | 'webdav' | 's3' | 'dropbox' | 'onedrive' | 'advanced'

interface ConnectCloudModalProps {
  opened: boolean
  onClose: () => void
  onConnected: () => void
}

export function ConnectCloudModal({ opened, onClose, onConnected }: ConnectCloudModalProps) {
  const { t } = useT()
  const [step, setStep] = useState<Step>('pick')

  const { data: transportsData } = useQuery({
    queryKey: ['cloud-transports'],
    queryFn: () => api.cloud.transports().then((r) => r.transports),
    refetchInterval: 5_000,
  })
  const transports = transportsData || []

  const handleClose = () => {
    setStep('pick')
    onClose()
  }

  const handleConnected = () => {
    onConnected()
    handleClose()
  }

  const options = [
    {
      id: 'gdrive' as Step,
      label: t('cloud.gdrive.label'),
      icon: <IconBrandGoogleDrive size={28} />,
      disabled: false,
    },
    {
      id: 'sftp' as Step,
      label: t('cloud.rclone.sftp.label'),
      icon: <IconServer size={28} />,
      disabled: false,
    },
    {
      id: 'webdav' as Step,
      label: t('cloud.rclone.webdav.label'),
      icon: <IconCloud size={28} />,
      disabled: false,
    },
    {
      id: 's3' as Step,
      label: t('cloud.rclone.s3.label'),
      icon: <IconCloud size={28} />,
      disabled: false,
    },
    {
      id: 'dropbox' as Step,
      label: t('cloud.rclone.dropbox.label'),
      icon: <IconBrandDropbox size={28} />,
      disabled: true,
    },
    {
      id: 'onedrive' as Step,
      label: t('cloud.rclone.onedrive.label'),
      icon: <IconBrandOnedrive size={28} />,
      disabled: true,
    },
  ]

  return (
    <Modal
      opened={opened}
      onClose={handleClose}
      title={t('cloud.connect_button')}
      size="md"
    >
      {step === 'pick' && (
        <SimpleGrid cols={2} spacing="sm">
          {options.map((opt) => (
            <Card
              key={opt.id}
              withBorder
              padding="sm"
              radius="md"
              style={{ cursor: opt.disabled ? 'not-allowed' : 'pointer', opacity: opt.disabled ? 0.5 : 1 }}
              onClick={() => !opt.disabled && setStep(opt.id)}
            >
              <Stack align="center" gap="xs">
                {opt.icon}
                <Text size="sm" ta="center">{opt.label}</Text>
                {opt.disabled && <Badge size="xs" color="gray">Coming soon</Badge>}
              </Stack>
            </Card>
          ))}
        </SimpleGrid>
      )}

      {step === 'gdrive' && (
        <GDriveStep
          transports={transports}
          onClose={handleClose}
          onConnected={handleConnected}
        />
      )}

      {(step === 'sftp' || step === 'webdav' || step === 's3') && (
        <RcloneFormStep
          backendId={step}
          onClose={handleClose}
          onConnected={handleConnected}
        />
      )}
    </Modal>
  )
}
