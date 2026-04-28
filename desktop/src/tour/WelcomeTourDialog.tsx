import { useState } from 'react'
import { Modal, Stack, Text, UnstyledButton, Checkbox, Paper, SimpleGrid, Group, Button } from '@mantine/core'
import { IconClock, IconFileInvoice, IconMoodSmile, IconBulb } from '@tabler/icons-react'
import { useT } from '../i18n'
import { useTour } from './TourProvider'
import { useIsMobile } from '../hooks/useIsMobile'
import { markWelcomeSeen } from './persistence'
import type { FlowId } from './types'

interface WelcomeTourDialogProps {
  opened: boolean
  onClose: () => void
}

export function WelcomeTourDialog({ opened, onClose }: WelcomeTourDialogProps) {
  const { t } = useT()
  const { startFlow } = useTour()
  const isMobile = useIsMobile()
  const [doNotShow, setDoNotShow] = useState(false)

  const handleOption = (id: FlowId | 'no-thanks') => {
    markWelcomeSeen(doNotShow)
    onClose()
    if (id !== 'no-thanks') startFlow(id)
  }

  const handleClose = () => {
    markWelcomeSeen(doNotShow)
    onClose()
  }

  return (
    <Modal
      opened={opened}
      onClose={handleClose}
      title={t('tour.welcome_title')}
      size="900px"
      centered
      fullScreen={isMobile}
      styles={{ title: { fontSize: isMobile ? '1.125rem' : '1.5rem', fontWeight: 700, lineHeight: 1.3 } }}
    >
      <Stack gap={isMobile ? 'lg' : 'xl'}>
        <Text size="sm" c="dimmed">{t('tour.welcome_intro')}</Text>

        <SimpleGrid cols={{ base: 1, sm: 3 }} spacing="md">
          <UnstyledButton data-tour-option="create-invoice" onClick={() => handleOption('create-invoice')} style={{ width: '100%' }}>
            <Paper p="xl" radius="md" withBorder h="100%" style={{ cursor: 'pointer', textAlign: 'center' }}>
              <Stack align="center" gap="sm">
                <Group gap={6} justify="center">
                  <IconClock size={44} stroke={1.5} />
                  <IconFileInvoice size={44} stroke={1.5} />
                </Group>
                <Text fw={600} size="sm">{t('tour.option_fast')}</Text>
                <Text size="xs" c="dimmed">{t('tour.option_fast_desc')}</Text>
              </Stack>
            </Paper>
          </UnstyledButton>

          <UnstyledButton data-tour-option="just-show-me" onClick={() => handleOption('just-show-me')} style={{ width: '100%' }}>
            <Paper p="xl" radius="md" withBorder h="100%" style={{ cursor: 'pointer', textAlign: 'center' }}>
              <Stack align="center" gap="sm">
                <IconMoodSmile size={44} stroke={1.5} />
                <Text fw={600} size="sm">{t('tour.option_explore')}</Text>
                <Text size="xs" c="dimmed">{t('tour.option_explore_desc')}</Text>
              </Stack>
            </Paper>
          </UnstyledButton>

          <UnstyledButton data-tour-option="advanced" onClick={() => handleOption('advanced')} style={{ width: '100%' }}>
            <Paper p="xl" radius="md" withBorder h="100%" style={{ cursor: 'pointer', textAlign: 'center' }}>
              <Stack align="center" gap="sm">
                <IconBulb size={44} stroke={1.5} />
                <Text fw={600} size="sm">{t('tour.option_advanced')}</Text>
                <Text size="xs" c="dimmed">{t('tour.option_advanced_desc')}</Text>
              </Stack>
            </Paper>
          </UnstyledButton>
        </SimpleGrid>

        <Group
          justify="space-between"
          align={isMobile ? 'stretch' : 'center'}
          wrap={isMobile ? 'wrap' : 'nowrap'}
          gap="md"
        >
          <Checkbox
            label={t('tour.do_not_show_again')}
            checked={doNotShow}
            onChange={(e) => setDoNotShow(e.currentTarget.checked)}
          />
          <Stack gap={4} align={isMobile ? 'stretch' : 'flex-end'} style={isMobile ? { width: '100%' } : undefined}>
            <Button
              data-tour-option="no-thanks"
              variant="default"
              onClick={() => handleOption('no-thanks')}
              fullWidth={isMobile}
            >
              {t('tour.option_no_thanks')}
            </Button>
            <Text size="xs" c="dimmed" ta={isMobile ? 'center' : undefined}>{t('tour.option_no_thanks_desc')}</Text>
          </Stack>
        </Group>
      </Stack>
    </Modal>
  )
}
