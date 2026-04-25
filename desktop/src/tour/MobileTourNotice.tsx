import { Modal, Text, Button, Stack } from '@mantine/core'
import { useT } from '../i18n'

interface MobileTourNoticeProps {
  opened: boolean
  onClose: () => void
}

export function MobileTourNotice({ opened, onClose }: MobileTourNoticeProps) {
  const { t } = useT()
  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={t('tour.mobile_title')}
      size="sm"
      centered
    >
      <Stack gap="md">
        <Text size="sm">{t('tour.mobile_body')}</Text>
        <Button onClick={onClose}>{t('common.ok')}</Button>
      </Stack>
    </Modal>
  )
}
