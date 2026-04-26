import { useState } from 'react'
import {
  Stack, Title, Text, UnstyledButton, Paper, SimpleGrid,
  Group, Button, Accordion, ThemeIcon,
} from '@mantine/core'
import {
  IconHelpCircle, IconCompass,
  IconFileInvoice, IconUsers, IconBuilding, IconRefresh, IconBug,
} from '@tabler/icons-react'
import { useT } from '../i18n'
import { useIsMobile } from '../hooks/useIsMobile'
import { WelcomeTourDialog } from '../tour/WelcomeTourDialog'
import { MobileTourNotice } from '../tour/MobileTourNotice'

type View = 'hub' | 'help'

export function HelpPage() {
  const { t } = useT()
  const isMobile = useIsMobile()
  const [view, setView] = useState<View>('hub')
  const [guideOpen, setGuideOpen] = useState(false)
  const [mobileNoticeOpen, setMobileNoticeOpen] = useState(false)

  const openGuide = () => {
    if (isMobile) setMobileNoticeOpen(true)
    else setGuideOpen(true)
  }

  if (view === 'help') {
    return <HelpContent onBack={() => setView('hub')} />
  }

  return (
    <Stack gap="xl">
      <div>
        <Title order={2}>{t('help.page_title')}</Title>
        <Text c="dimmed" size="sm">{t('help.subtitle')}</Text>
      </div>

      <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
        <UnstyledButton onClick={() => setView('help')} style={{ width: '100%' }}>
          <Paper p="xl" radius="md" withBorder h="100%" style={{ cursor: 'pointer', textAlign: 'center' }}>
            <Stack align="center" gap="sm">
              <ThemeIcon size={52} radius="md" variant="light">
                <IconHelpCircle size={30} stroke={1.5} />
              </ThemeIcon>
              <Text fw={600}>{t('help.card_help')}</Text>
              <Text size="xs" c="dimmed">{t('help.card_help_desc')}</Text>
            </Stack>
          </Paper>
        </UnstyledButton>

        <UnstyledButton onClick={openGuide} style={{ width: '100%' }}>
          <Paper p="xl" radius="md" withBorder h="100%" style={{ cursor: 'pointer', textAlign: 'center' }}>
            <Stack align="center" gap="sm">
              <ThemeIcon size={52} radius="md" variant="light" color="teal">
                <IconCompass size={30} stroke={1.5} />
              </ThemeIcon>
              <Text fw={600}>{t('help.card_guide')}</Text>
              <Text size="xs" c="dimmed">{t('help.card_guide_desc')}</Text>
            </Stack>
          </Paper>
        </UnstyledButton>
      </SimpleGrid>

      <WelcomeTourDialog opened={guideOpen && !isMobile} onClose={() => setGuideOpen(false)} />
      <MobileTourNotice opened={mobileNoticeOpen} onClose={() => setMobileNoticeOpen(false)} />
    </Stack>
  )
}

function HelpContent({ onBack }: { onBack: () => void }) {
  const { t } = useT()

  const sections = [
    { value: 'invoice',   icon: IconFileInvoice, title: t('help.section_invoice'),   body: t('help.section_invoice_body') },
    { value: 'customers', icon: IconUsers,        title: t('help.section_customers'), body: t('help.section_customers_body') },
    { value: 'suppliers', icon: IconBuilding,     title: t('help.section_suppliers'), body: t('help.section_suppliers_body') },
    { value: 'sync',      icon: IconRefresh,      title: t('help.section_sync'),      body: t('help.section_sync_body') },
  ]

  return (
    <Stack gap="xl">
      <Group justify="space-between" align="center">
        <Title order={2}>{t('help.sections_title')}</Title>
        <Button variant="subtle" onClick={onBack}>{t('help.back')}</Button>
      </Group>

      <Accordion variant="separated" radius="md">
        {sections.map(({ value, icon: Icon, title, body }) => (
          <Accordion.Item key={value} value={value}>
            <Accordion.Control icon={<Icon size={18} stroke={1.5} />}>
              <Text fw={500}>{title}</Text>
            </Accordion.Control>
            <Accordion.Panel>
              <Text size="sm" c="dimmed">{body}</Text>
            </Accordion.Panel>
          </Accordion.Item>
        ))}

        <Accordion.Item value="faq">
          <Accordion.Control icon={<IconBug size={18} stroke={1.5} />}>
            <Text fw={500}>{t('help.section_faq')}</Text>
          </Accordion.Control>
          <Accordion.Panel>
            <Accordion variant="contained" radius="md">
              <Accordion.Item value="qr">
                <Accordion.Control>
                  <Text size="sm" fw={500}>{t('help.faq_qr_q')}</Text>
                </Accordion.Control>
                <Accordion.Panel>
                  <Text size="sm" c="dimmed">{t('help.faq_qr_a')}</Text>
                </Accordion.Panel>
              </Accordion.Item>
            </Accordion>
          </Accordion.Panel>
        </Accordion.Item>
      </Accordion>
    </Stack>
  )
}
