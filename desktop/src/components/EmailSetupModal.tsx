import { Modal, Stack, Text, Button } from '@mantine/core'
import { useNavigate } from 'react-router-dom'
import { useT } from '../i18n'
import { useIsMobile } from '../hooks/useIsMobile'

interface Props {
  opened: boolean
  onClose: () => void
  supplierId: string
}

export function EmailSetupModal({ opened, onClose, supplierId: _supplierId }: Props) {
  const { t } = useT()
  const isMobile = useIsMobile()
  const navigate = useNavigate()

  return (
    <Modal opened={opened} onClose={onClose} title={t('email.setup_title')} size="sm" fullScreen={isMobile}>
      <Stack gap="md">
        <Text>{t('email.setup_desc')}</Text>
        <Button fullWidth onClick={() => { onClose(); navigate('/suppliers') }}>
          {t('email.setup_yes')}
        </Button>
        <Button fullWidth variant="light" color="gray" onClick={onClose}>
          {t('email.setup_later')}
        </Button>
        <Button fullWidth variant="subtle" color="gray" disabled>
          {t('email.setup_redirect')}
        </Button>
      </Stack>
    </Modal>
  )
}
