import { useState } from 'react'
import { Modal, Stack, SimpleGrid, Card, Text, Badge } from '@mantine/core'
import { IconBrandGoogleDrive, IconServer, IconCloud, IconBrandDropbox, IconBrandOnedrive } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useT } from '../i18n'
import { api } from '../api/client'
import { GDriveStep } from './connect/GDriveStep'
import { RcloneFormStep } from './connect/RcloneFormStep'

type Step = 'pick' | 'gdrive' | 'sftp' | 'webdav' | 's3' | 'dropbox' | 'onedrive' | 'protondrive' | 'advanced'

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

  type OptionStatus = 'recommended' | 'ready' | 'beta' | 'disabled' | 'coming_soon'
  const options: Array<{
    id: Step
    label: string
    icon: React.ReactNode
    status: OptionStatus
  }> = [
    {
      id: 'protondrive',
      label: t('cloud.rclone.protondrive.label'),
      icon: <IconCloud size={28} />,
      status: 'recommended',
    },
    {
      id: 'sftp',
      label: t('cloud.rclone.sftp.label'),
      icon: <IconServer size={28} />,
      status: 'beta',
    },
    {
      id: 'webdav',
      label: t('cloud.rclone.webdav.label'),
      icon: <IconCloud size={28} />,
      status: 'beta',
    },
    {
      id: 's3',
      label: t('cloud.rclone.s3.label'),
      icon: <IconCloud size={28} />,
      status: 'beta',
    },
    {
      id: 'gdrive',
      label: t('cloud.gdrive.label'),
      icon: <IconBrandGoogleDrive size={28} />,
      status: 'disabled',
    },
    {
      id: 'dropbox',
      label: t('cloud.rclone.dropbox.label'),
      icon: <IconBrandDropbox size={28} />,
      status: 'coming_soon',
    },
    {
      id: 'onedrive',
      label: t('cloud.rclone.onedrive.label'),
      icon: <IconBrandOnedrive size={28} />,
      status: 'coming_soon',
    },
  ]

  const renderBadge = (status: OptionStatus) => {
    if (status === 'recommended') {
      return <Badge size="xs" color="green">{t('cloud.connect.badge_recommended')}</Badge>
    }
    if (status === 'beta') {
      return <Badge size="xs" color="orange">{t('cloud.connect.badge_beta')}</Badge>
    }
    if (status === 'disabled') {
      return <Badge size="xs" color="gray">{t('cloud.connect.badge_disabled')}</Badge>
    }
    if (status === 'coming_soon') {
      return <Badge size="xs" color="gray">{t('cloud.connect.badge_coming_soon')}</Badge>
    }
    return null
  }

  return (
    <Modal
      opened={opened}
      onClose={handleClose}
      title={t('cloud.connect_button')}
      size="md"
    >
      {step === 'pick' && (
        <SimpleGrid cols={2} spacing="sm">
          {options.map((opt) => {
            const isInteractive = opt.status === 'ready' || opt.status === 'beta' || opt.status === 'recommended'
            return (
              <Card
                key={opt.id}
                withBorder
                padding="sm"
                radius="md"
                style={{
                  cursor: isInteractive ? 'pointer' : 'not-allowed',
                  opacity: isInteractive ? 1 : 0.5,
                }}
                onClick={() => isInteractive && setStep(opt.id)}
              >
                <Stack align="center" gap="xs">
                  {opt.icon}
                  <Text size="sm" ta="center">{opt.label}</Text>
                  {renderBadge(opt.status)}
                </Stack>
              </Card>
            )
          })}
        </SimpleGrid>
      )}

      {step === 'gdrive' && (
        <GDriveStep
          transports={transports}
          onClose={handleClose}
          onConnected={handleConnected}
        />
      )}

      {(step === 'sftp' || step === 'webdav' || step === 's3' || step === 'protondrive') && (
        <RcloneFormStep
          backendId={step}
          onClose={handleClose}
          onConnected={handleConnected}
        />
      )}
    </Modal>
  )
}
