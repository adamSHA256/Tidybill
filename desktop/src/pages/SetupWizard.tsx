import { useState, useEffect } from 'react'
import {
  Stepper,
  Button,
  Group,
  Stack,
  TextInput,
  Select,
  Switch,
  Title,
  Text,
  Paper,
  Container,
  ScrollArea,
  SegmentedControl,
  Divider,
  Center,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useQuery } from '@tanstack/react-query'
import { api, type Supplier, type BankAccount } from '../api/client'
import { useT } from '../i18n'
import { IconFolderOpen } from '@tabler/icons-react'

const langLabels: Record<string, string> = {
  cs: 'Čeština',
  sk: 'Slovenčina',
  en: 'English',
}

const currencyForLang: Record<string, string> = {
  cs: 'CZK',
  sk: 'EUR',
  en: 'EUR',
}

interface Props {
  onComplete: () => void
}

export function SetupWizard({ onComplete }: Props) {
  const { t, setLang } = useT()
  const [active, setActive] = useState(0)
  const [saving, setSaving] = useState(false)

  // Step 1: Language — switches live as user picks
  const [selectedLang, setSelectedLang] = useState<string>('cs')

  // Step 2: Supplier
  const [supplierSkipped, setSupplierSkipped] = useState(false)
  const [supplierName, setSupplierName] = useState('')
  const [supplierStreet, setSupplierStreet] = useState('')
  const [supplierCity, setSupplierCity] = useState('')
  const [supplierZip, setSupplierZip] = useState('')
  const [supplierCountry, setSupplierCountry] = useState('CZ')
  const [supplierIco, setSupplierIco] = useState('')
  const [supplierDic, setSupplierDic] = useState('')
  const [supplierIcDph, setSupplierIcDph] = useState('')
  const [supplierPhone, setSupplierPhone] = useState('')
  const [supplierEmail, setSupplierEmail] = useState('')
  const [supplierVat, setSupplierVat] = useState(false)
  const [supplierPrefix, setSupplierPrefix] = useState('VF')
  const [createdSupplierId, setCreatedSupplierId] = useState<string | null>(null)

  // Step 3: Bank Account
  const [bankSkipped, setBankSkipped] = useState(false)
  const [bankName, setBankName] = useState('')
  const [bankAccountNumber, setBankAccountNumber] = useState('')
  const [bankIban, setBankIban] = useState('')
  const [bankCurrency, setBankCurrency] = useState('CZK')

  // Step 4: PDF Directory
  const [pdfDir, setPdfDir] = useState('')

  // Step 5: Defaults (due days only)
  const [dueDays, setDueDays] = useState('14')

  // Load OS locale to pre-select language radio (but don't change i18n)
  const { data: localeData } = useQuery({
    queryKey: ['system-locale'],
    queryFn: api.getLocale,
  })

  // Load current settings for PDF dir default
  const { data: settings } = useQuery({
    queryKey: ['settings'],
    queryFn: api.getSettings,
  })

  // Pre-select language from OS and switch i18n immediately
  useEffect(() => {
    if (localeData?.detected_lang) {
      const detected = localeData.detected_lang as 'cs' | 'sk' | 'en'
      setSelectedLang(detected)
      setLang(detected)
    }
  }, [localeData]) // eslint-disable-line react-hooks/exhaustive-deps

  // Pre-fill PDF dir: use the config default (always set by Go), fall back to saved override
  useEffect(() => {
    if (settings?.default_pdf_dir) {
      setPdfDir(settings.dir_pdfs || settings.default_pdf_dir)
    }
  }, [settings])

  const showBankStep = !supplierSkipped

  const goNext = () => setActive((c) => c + 1)
  const goBack = () => setActive((c) => c - 1)

  // Switch language live when user clicks a different option
  const handleLangChange = (value: string) => {
    setSelectedLang(value)
    setLang(value as 'cs' | 'sk' | 'en')
    setBankCurrency(currencyForLang[value] || 'EUR')
  }

  const handleLangNext = () => {
    goNext()
  }

  // Update bank account default name when language actually changes (after step 1 confirm)
  useEffect(() => {
    setBankName(t('wizard.default_account_name'))
  }, [t])

  const handleSupplierNext = async () => {
    if (!supplierName.trim()) {
      notifications.show({
        title: t('supplier.missing_name_title'),
        message: t('supplier.missing_name_msg'),
        color: 'red',
      })
      return
    }
    setSaving(true)
    try {
      const supplier = await api.createSupplier({
        name: supplierName,
        street: supplierStreet,
        city: supplierCity,
        zip: supplierZip,
        country: supplierCountry,
        ico: supplierIco,
        dic: supplierDic,
        ic_dph: supplierIcDph,
        phone: supplierPhone,
        email: supplierEmail,
        is_vat_payer: supplierVat,
        invoice_prefix: supplierPrefix,
      } as Partial<Supplier>)
      setCreatedSupplierId(supplier.id)
      setSupplierSkipped(false)
      goNext()
    } catch (err) {
      notifications.show({
        title: t('common.error'),
        message: err instanceof Error ? err.message : String(err),
        color: 'red',
      })
    } finally {
      setSaving(false)
    }
  }

  const handleSupplierSkip = () => {
    setSupplierSkipped(true)
    setBankSkipped(true)
    // Jump past bank step (step 2) directly to PDF dir (step 3)
    setActive(3)
  }

  const handleBankNext = async () => {
    if (!createdSupplierId) return
    setSaving(true)
    try {
      await api.createBankAccount(createdSupplierId, {
        name: bankName,
        account_number: bankAccountNumber,
        iban: bankIban,
        currency: bankCurrency,
        is_default: true,
      } as Partial<BankAccount>)
      setBankSkipped(false)
      goNext()
    } catch (err) {
      notifications.show({
        title: t('common.error'),
        message: err instanceof Error ? err.message : String(err),
        color: 'red',
      })
    } finally {
      setSaving(false)
    }
  }

  const handleBankSkip = () => {
    setBankSkipped(true)
    goNext()
  }

  const handleFinish = async () => {
    setSaving(true)
    try {
      await api.updateSettings({
        default_due_days: dueDays,
        ...(pdfDir ? { dir_pdfs: pdfDir } : {}),
      })
      onComplete()
    } catch (err) {
      notifications.show({
        title: t('common.error'),
        message: err instanceof Error ? err.message : String(err),
        color: 'red',
      })
    } finally {
      setSaving(false)
    }
  }

  const handleChooseFolder = async () => {
    try {
      const { open } = await import('@tauri-apps/plugin-dialog')
      const selected = await open({ directory: true, multiple: false, defaultPath: pdfDir || undefined })
      if (selected) {
        setPdfDir(selected as string)
      }
    } catch {
      // Not running in Tauri (e.g. dev browser) — ignore silently
    }
  }

  // When going back from PDF dir step and supplier was skipped, go to supplier step
  const handlePdfDirBack = () => {
    if (supplierSkipped) {
      setActive(1)
    } else {
      goBack()
    }
  }

  return (
    <ScrollArea h="100vh" type="auto">
      <Container size="sm" py="xl">
        <Stack gap="xl">
        <div style={{ textAlign: 'center' }}>
          <Title order={1}>{t('wizard.title')}</Title>
          <Text c="dimmed" size="lg" mt="xs">
            {t('wizard.subtitle')}
          </Text>
        </div>

        <Stepper
          active={active}
          onStepClick={setActive}
          allowNextStepsSelect={false}
          size="sm"
        >
          {/* Step 0: Language */}
          <Stepper.Step label={t('wizard.step_language')}>
            <Paper p="xl" radius="md" withBorder mt="md">
              <Stack gap="lg">
                <Text fw={500} size="lg">
                  Choose language / Vyberte jazyk / Zvoľte jazyk
                </Text>
                <SegmentedControl
                  value={selectedLang}
                  onChange={handleLangChange}
                  data={[
                    { label: 'Čeština', value: 'cs' },
                    { label: 'Slovenčina', value: 'sk' },
                    { label: 'English', value: 'en' },
                  ]}
                  fullWidth
                  size="lg"
                />
              </Stack>
            </Paper>
            <Group justify="flex-end" mt="xl">
              <Button onClick={handleLangNext}>{t('wizard.next')}</Button>
            </Group>
          </Stepper.Step>

          {/* Step 1: Supplier */}
          <Stepper.Step label={t('wizard.step_supplier')}>
            <Paper p="xl" radius="md" withBorder mt="md">
              <Stack gap="md">
                <TextInput
                  label={t('supplier.name_label')}
                  value={supplierName}
                  onChange={(e) => setSupplierName(e.currentTarget.value)}
                  required
                />
                <Group grow>
                  <TextInput
                    label={t('supplier.street_label')}
                    value={supplierStreet}
                    onChange={(e) => setSupplierStreet(e.currentTarget.value)}
                  />
                </Group>
                <Group grow>
                  <TextInput
                    label={t('supplier.city_label')}
                    value={supplierCity}
                    onChange={(e) => setSupplierCity(e.currentTarget.value)}
                  />
                  <TextInput
                    label={t('supplier.zip_label')}
                    value={supplierZip}
                    onChange={(e) => setSupplierZip(e.currentTarget.value)}
                    w={120}
                  />
                  <TextInput
                    label={t('supplier.country_label')}
                    value={supplierCountry}
                    onChange={(e) => setSupplierCountry(e.currentTarget.value)}
                    w={80}
                  />
                </Group>
                <Group grow>
                  <TextInput
                    label={t('supplier.ico_label')}
                    value={supplierIco}
                    onChange={(e) => setSupplierIco(e.currentTarget.value)}
                  />
                  <TextInput
                    label={t('supplier.dic_label')}
                    value={supplierDic}
                    onChange={(e) => setSupplierDic(e.currentTarget.value)}
                  />
                </Group>
                {supplierCountry.toUpperCase() === 'SK' && (
                  <TextInput
                    label={t('supplier.ic_dph_label')}
                    value={supplierIcDph}
                    onChange={(e) => setSupplierIcDph(e.currentTarget.value)}
                  />
                )}
                {supplierDic && (
                  <Switch
                    label={t('supplier.is_vat_payer_label')}
                    checked={supplierVat}
                    onChange={(e) => setSupplierVat(e.currentTarget.checked)}
                  />
                )}
                <Group grow>
                  <TextInput
                    label={t('supplier.phone_label')}
                    value={supplierPhone}
                    onChange={(e) => setSupplierPhone(e.currentTarget.value)}
                  />
                  <TextInput
                    label={t('supplier.email_label')}
                    value={supplierEmail}
                    onChange={(e) => setSupplierEmail(e.currentTarget.value)}
                  />
                </Group>
                <TextInput
                  label={t('supplier.invoice_prefix_label')}
                  value={supplierPrefix}
                  onChange={(e) => setSupplierPrefix(e.currentTarget.value)}
                  w={120}
                />
              </Stack>
            </Paper>
            <Group justify="space-between" mt="xl">
              <Button variant="default" onClick={goBack}>
                {t('wizard.back')}
              </Button>
              <Group>
                <Button
                  variant="subtle"
                  color="gray"
                  onClick={handleSupplierSkip}
                >
                  {t('wizard.skip_for_now')}
                </Button>
                <Button onClick={handleSupplierNext} loading={saving}>
                  {t('wizard.next')}
                </Button>
              </Group>
            </Group>
          </Stepper.Step>

          {/* Step 2: Bank Account (only if supplier was not skipped) */}
          <Stepper.Step label={t('wizard.step_bank')}>
            {showBankStep ? (
              <Paper p="xl" radius="md" withBorder mt="md">
                <Stack gap="md">
                  <TextInput
                    label={t('bank_account.name_label')}
                    value={bankName}
                    onChange={(e) => setBankName(e.currentTarget.value)}
                  />
                  <TextInput
                    label={t('bank_account.account_number_label')}
                    value={bankAccountNumber}
                    onChange={(e) =>
                      setBankAccountNumber(e.currentTarget.value)
                    }
                  />
                  <TextInput
                    label={t('bank_account.iban_label')}
                    value={bankIban}
                    onChange={(e) => setBankIban(e.currentTarget.value)}
                  />
                  <Select
                    label={t('bank_account.currency_label')}
                    data={['CZK', 'EUR', 'USD']}
                    value={bankCurrency}
                    onChange={(v) => {
                      if (v) setBankCurrency(v)
                    }}
                    w={200}
                  />
                </Stack>
              </Paper>
            ) : (
              <Paper p="xl" radius="md" withBorder mt="md">
                <Center>
                  <Text c="dimmed">{t('wizard.skip_supplier_note')}</Text>
                </Center>
              </Paper>
            )}
            <Group justify="space-between" mt="xl">
              <Button variant="default" onClick={goBack}>
                {t('wizard.back')}
              </Button>
              <Group>
                {showBankStep && (
                  <Button
                    variant="subtle"
                    color="gray"
                    onClick={handleBankSkip}
                  >
                    {t('wizard.skip_for_now')}
                  </Button>
                )}
                <Button
                  onClick={showBankStep ? handleBankNext : goNext}
                  loading={saving}
                >
                  {t('wizard.next')}
                </Button>
              </Group>
            </Group>
          </Stepper.Step>

          {/* Step 3: PDF Directory */}
          <Stepper.Step label={t('wizard.step_pdf_dir')}>
            <Paper p="xl" radius="md" withBorder mt="md">
              <Stack gap="md">
                <Text c="dimmed" size="sm">
                  {t('wizard.pdf_dir_desc')}
                </Text>
                <Group align="flex-end">
                  <TextInput
                    label={t('settings.dir_pdfs')}
                    value={pdfDir}
                    onChange={(e) => setPdfDir(e.currentTarget.value)}
                    placeholder={t('settings.dir_placeholder')}
                    style={{ flex: 1 }}
                  />
                  <Button
                    variant="light"
                    leftSection={<IconFolderOpen size={16} />}
                    onClick={handleChooseFolder}
                  >
                    {t('wizard.choose_folder')}
                  </Button>
                </Group>
              </Stack>
            </Paper>
            <Group justify="space-between" mt="xl">
              <Button variant="default" onClick={handlePdfDirBack}>
                {t('wizard.back')}
              </Button>
              <Button onClick={goNext}>{t('wizard.next')}</Button>
            </Group>
          </Stepper.Step>

          {/* Step 4: Defaults (due days only) */}
          <Stepper.Step label={t('wizard.step_defaults')}>
            <Paper p="xl" radius="md" withBorder mt="md">
              <Stack gap="md">
                <Text fw={500}>{t('wizard.due_days_label')}</Text>
                <SegmentedControl
                  value={dueDays}
                  onChange={setDueDays}
                  data={[
                    { label: '7', value: '7' },
                    { label: '14', value: '14' },
                    { label: '30', value: '30' },
                    { label: '60', value: '60' },
                  ]}
                />
              </Stack>
            </Paper>
            <Group justify="space-between" mt="xl">
              <Button variant="default" onClick={goBack}>
                {t('wizard.back')}
              </Button>
              <Button onClick={goNext}>{t('wizard.next')}</Button>
            </Group>
          </Stepper.Step>

          {/* Step 5: Summary */}
          <Stepper.Step label={t('wizard.step_summary')}>
            <Paper p="xl" radius="md" withBorder mt="md">
              <Stack gap="sm">
                <SummaryRow
                  label={t('wizard.summary_language')}
                  value={langLabels[selectedLang] || selectedLang}
                />
                <Divider />
                <SummaryRow
                  label={t('wizard.summary_supplier')}
                  value={
                    supplierSkipped
                      ? `— (${t('wizard.summary_skipped')})`
                      : supplierName
                  }
                />
                <Divider />
                <SummaryRow
                  label={t('wizard.summary_bank')}
                  value={
                    supplierSkipped || bankSkipped
                      ? `— (${t('wizard.summary_skipped')})`
                      : bankIban || bankAccountNumber || bankName
                  }
                />
                <Divider />
                <SummaryRow
                  label={t('wizard.summary_pdf_dir')}
                  value={pdfDir || '—'}
                />
                <Divider />
                <SummaryRow
                  label={t('wizard.summary_due_days')}
                  value={dueDays}
                />
              </Stack>
            </Paper>
            <Text c="dimmed" size="sm" ta="center" mt="md">
              {t('wizard.settings_hint')}
            </Text>
            <Title order={3} ta="center" mt="lg">
              {t('wizard.happy_invoicing')}
            </Title>
            <Group justify="space-between" mt="xl">
              <Button variant="default" onClick={goBack}>
                {t('wizard.back')}
              </Button>
              <Button onClick={handleFinish} loading={saving} size="md">
                {t('wizard.finish')}
              </Button>
            </Group>
          </Stepper.Step>
        </Stepper>
        </Stack>
      </Container>
    </ScrollArea>
  )
}

function SummaryRow({ label, value }: { label: string; value: string }) {
  return (
    <Group justify="space-between">
      <Text fw={500} size="sm">
        {label}
      </Text>
      <Text size="sm" c="dimmed">
        {value}
      </Text>
    </Group>
  )
}
