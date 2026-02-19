import {
  Title,
  Text,
  Paper,
  Stack,
  Select,
  Switch,
  Group,
  SimpleGrid,
  Badge,
  Loader,
  Center,
  TextInput,
  Button,
  Pill,
  SegmentedControl,
  Anchor,
  ActionIcon,
  Code,
  CopyButton,
  Tooltip,
} from '@mantine/core'
import { IconCopy, IconCheck } from '@tabler/icons-react'
import { notifications } from '@mantine/notifications'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { api, openInBrowser, type Unit, type PDFTemplate, type VATRate, type CurrencyItem, type PaymentType, type DueDaysOption } from '../api/client'
import { applyZoom } from '../utils/zoom'
import { useT } from '../i18n'

const langOptions = [
  { value: 'cs', label: 'Čeština' },
  { value: 'sk', label: 'Slovenčina' },
  { value: 'en', label: 'English' },
]


interface DashboardWidgets {
  revenue: boolean
  unpaid: boolean
  customers: boolean
  invoices_month: boolean
  overdue: boolean
  recent: boolean
  quick_actions: boolean
}

const defaultWidgets: DashboardWidgets = {
  revenue: true,
  unpaid: true,
  customers: true,
  invoices_month: true,
  overdue: true,
  recent: true,
  quick_actions: true,
}

function parseWidgets(raw?: string): DashboardWidgets {
  if (!raw) return { ...defaultWidgets }
  try {
    return { ...defaultWidgets, ...JSON.parse(raw) }
  } catch {
    return { ...defaultWidgets }
  }
}

export function Settings() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const { t, setLang } = useT()

  const { data: settings, isLoading } = useQuery({
    queryKey: ['settings'],
    queryFn: api.getSettings,
  })

  const { data: templates } = useQuery({
    queryKey: ['templates'],
    queryFn: api.getTemplates,
  })

  const { data: aboutInfo } = useQuery({
    queryKey: ['about'],
    queryFn: api.getAbout,
  })

  const [dirLogos, setDirLogos] = useState('')
  const [dirPdfs, setDirPdfs] = useState('')
  const [dirPreviews, setDirPreviews] = useState('')
  const [newUnitName, setNewUnitName] = useState('')

  const { data: units } = useQuery({
    queryKey: ['units'],
    queryFn: api.getUnits,
  })

  const [localUnits, setLocalUnits] = useState<Unit[]>([])
  const [dashWidgets, setDashWidgets] = useState<DashboardWidgets>(defaultWidgets)

  const { data: currencies } = useQuery({
    queryKey: ['currencies'],
    queryFn: api.getCurrencies,
  })
  const [localCurrencies, setLocalCurrencies] = useState<CurrencyItem[]>([])
  const [newCurrencyCode, setNewCurrencyCode] = useState('')

  const { data: vatRates } = useQuery({
    queryKey: ['vat-rates'],
    queryFn: api.getVATRates,
  })
  const [localVATRates, setLocalVATRates] = useState<VATRate[]>([])
  const [newVATRate, setNewVATRate] = useState('')

  const { data: paymentTypes } = useQuery({
    queryKey: ['payment-types'],
    queryFn: api.getPaymentTypes,
  })
  const [localPaymentTypes, setLocalPaymentTypes] = useState<PaymentType[]>([])
  const [newPaymentTypeName, setNewPaymentTypeName] = useState('')
  const [newPaymentTypeRequiresBank, setNewPaymentTypeRequiresBank] = useState(true)

  const { data: dueDaysOptions } = useQuery({
    queryKey: ['due-days'],
    queryFn: api.getDueDaysOptions,
  })
  const [localDueDays, setLocalDueDays] = useState<DueDaysOption[]>([])
  const [newDueDaysValue, setNewDueDaysValue] = useState('')

  useEffect(() => {
    if (settings) {
      setDirLogos(settings.dir_logos || '')
      setDirPdfs(settings.dir_pdfs || '')
      setDirPreviews(settings.dir_previews || '')
      setDashWidgets(parseWidgets(settings.dashboard_widgets))
    }
  }, [settings])

  useEffect(() => {
    if (currencies) {
      setLocalCurrencies(currencies)
    }
  }, [currencies])

  useEffect(() => {
    if (units) {
      setLocalUnits(units)
    }
  }, [units])

  useEffect(() => {
    if (vatRates) {
      setLocalVATRates(vatRates)
    }
  }, [vatRates])

  useEffect(() => {
    if (paymentTypes) {
      setLocalPaymentTypes(paymentTypes)
    }
  }, [paymentTypes])

  useEffect(() => {
    if (dueDaysOptions) {
      setLocalDueDays(dueDaysOptions)
    }
  }, [dueDaysOptions])

  const unitsMutation = useMutation({
    mutationFn: api.updateUnits,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['units'] })
      notifications.show({ title: t('notify.units_saved'), message: t('notify.units_saved_msg'), color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const currenciesMutation = useMutation({
    mutationFn: api.updateCurrencies,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['currencies'] })
      notifications.show({ title: t('notify.currencies_saved'), message: t('notify.currencies_saved_msg'), color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const vatRatesMutation = useMutation({
    mutationFn: api.updateVATRates,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['vat-rates'] })
      notifications.show({ title: t('notify.vat_rates_saved'), message: t('notify.vat_rates_saved_msg'), color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const paymentTypesMutation = useMutation({
    mutationFn: api.updatePaymentTypes,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['payment-types'] })
      notifications.show({ title: t('notify.payment_types_saved'), message: t('notify.payment_types_saved_msg'), color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const dueDaysMutation = useMutation({
    mutationFn: api.updateDueDaysOptions,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['due-days'] })
      notifications.show({ title: t('notify.due_days_saved'), message: t('notify.due_days_saved_msg'), color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const updateMutation = useMutation({
    mutationFn: api.updateSettings,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings'] })
      notifications.show({ title: t('notify.settings_saved'), message: t('notify.settings_saved_msg'), color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  if (isLoading) {
    return <Center h={300}><Loader /></Center>
  }

  const comingSoonBadge = <Badge size="xs" variant="light" color="gray">{t('settings.coming_soon')}</Badge>

  const defaultTemplate = templates?.find((tmpl: PDFTemplate) => tmpl.is_default)

  const handleWidgetToggle = (key: keyof DashboardWidgets, checked: boolean) => {
    const next = { ...dashWidgets, [key]: checked }
    setDashWidgets(next)
    updateMutation.mutate({ dashboard_widgets: JSON.stringify(next) })
  }

  return (
    <Stack gap="lg">
      <div>
        <Title order={2}>{t('settings.title')}</Title>
        <Text c="dimmed" size="sm">{t('settings.subtitle')}</Text>
      </div>

      {/* Row 1: General + Directories */}
      <SimpleGrid cols={{ base: 1, md: 2 }}>
        <Paper p="md" radius="md" withBorder>
          <Text fw={500} mb="md">{t('settings.general')}</Text>
          <Stack gap="md">
            <Select
              label={t('settings.language')}
              data={langOptions}
              value={settings?.language || 'cs'}
              onChange={(v) => { if (v) setLang(v as 'cs' | 'sk' | 'en') }}
              w={300}
            />
            <div>
              <Text size="sm" fw={500} mb={4}>{t('settings.ui_scale')}</Text>
              <Text size="xs" c="dimmed" mb="xs">{t('settings.ui_scale_desc')}</Text>
              <SegmentedControl
                value={String(Math.round((parseFloat(settings?.ui_scale || '1') || 1) * 100))}
                onChange={(val) => {
                  const factor = Number(val) / 100
                  applyZoom(factor)
                  api.updateSettings({ ui_scale: String(factor) }).then(() => {
                    queryClient.invalidateQueries({ queryKey: ['settings'] })
                  })
                }}
                data={[
                  { value: '100', label: '100%' },
                  { value: '125', label: '125%' },
                  { value: '150', label: '150%' },
                  { value: '175', label: '175%' },
                  { value: '200', label: '200%' },
                ]}
              />
            </div>
            <Group gap="xs">
              <Select
                label={t('settings.date_format')}
                data={['DD.MM.YYYY', 'YYYY-MM-DD', 'MM/DD/YYYY']}
                defaultValue="DD.MM.YYYY"
                w={300}
                disabled
              />
              {comingSoonBadge}
            </Group>
          </Stack>
        </Paper>

        <Paper p="md" radius="md" withBorder>
          <Text fw={500} mb="md">{t('settings.directories')}</Text>
          <Stack gap="md">
            <TextInput
              label={t('settings.dir_logos')}
              placeholder={t('settings.dir_placeholder')}
              value={dirLogos}
              onChange={(e) => setDirLogos(e.currentTarget.value)}
            />
            <TextInput
              label={t('settings.dir_pdfs')}
              placeholder={t('settings.dir_placeholder')}
              value={dirPdfs}
              onChange={(e) => setDirPdfs(e.currentTarget.value)}
            />
            <TextInput
              label={t('settings.dir_previews')}
              placeholder={t('settings.dir_placeholder')}
              value={dirPreviews}
              onChange={(e) => setDirPreviews(e.currentTarget.value)}
            />
            <Button
              w={200}
              onClick={() => updateMutation.mutate({
                dir_logos: dirLogos,
                dir_pdfs: dirPdfs,
                dir_previews: dirPreviews,
              })}
              loading={updateMutation.isPending}
            >
              {t('settings.save_directories')}
            </Button>
          </Stack>
        </Paper>
      </SimpleGrid>

      {/* Row 2: Dashboard — switches in 2-column grid */}
      <Paper p="md" radius="md" withBorder>
        <Text fw={500} mb="md">{t('settings.dashboard')}</Text>
        <Text c="dimmed" size="sm" mb="md">{t('settings.dashboard_desc')}</Text>
        <SimpleGrid cols={{ base: 1, xs: 2 }} spacing="sm">
          <Switch
            label={t('dashboard.total_revenue')}
            checked={dashWidgets.revenue}
            onChange={(e) => handleWidgetToggle('revenue', e.currentTarget.checked)}
          />
          <Switch
            label={t('dashboard.unpaid_invoices')}
            checked={dashWidgets.unpaid}
            onChange={(e) => handleWidgetToggle('unpaid', e.currentTarget.checked)}
          />
          <Switch
            label={t('dashboard.active_customers')}
            checked={dashWidgets.customers}
            onChange={(e) => handleWidgetToggle('customers', e.currentTarget.checked)}
          />
          <Switch
            label={t('dashboard.invoices_month')}
            checked={dashWidgets.invoices_month}
            onChange={(e) => handleWidgetToggle('invoices_month', e.currentTarget.checked)}
          />
          <Switch
            label={t('dashboard.overdue_title')}
            checked={dashWidgets.overdue}
            onChange={(e) => handleWidgetToggle('overdue', e.currentTarget.checked)}
          />
          <Switch
            label={t('dashboard.recent_invoices')}
            checked={dashWidgets.recent}
            onChange={(e) => handleWidgetToggle('recent', e.currentTarget.checked)}
          />
          <Switch
            label={t('dashboard.quick_actions')}
            checked={dashWidgets.quick_actions}
            onChange={(e) => handleWidgetToggle('quick_actions', e.currentTarget.checked)}
          />
        </SimpleGrid>
      </Paper>

      {/* Row 3: Invoices + Currencies */}
      <SimpleGrid cols={{ base: 1, md: 2 }}>
        <Paper p="md" radius="md" withBorder>
          <Text fw={500} mb="md">{t('settings.invoices')}</Text>
          <Stack gap="md">
            {/* TODO: also expose this setting in CLI settings menu */}
            <Select
              label={t('settings.invoice_default_sort')}
              description={t('settings.invoice_default_sort_desc')}
              data={[
                { value: 'created_at', label: t('invoice.created_at') },
                { value: 'issue_date', label: t('invoice.issue_date') },
                { value: 'due_date', label: t('invoice.due_date') },
                { value: 'invoice_number', label: t('invoice.number') },
                { value: 'total', label: t('invoice.amount') },
              ]}
              value={settings?.invoice_default_sort || 'created_at'}
              onChange={(v) => { if (v) updateMutation.mutate({ invoice_default_sort: v }) }}
              w={300}
            />
            <Group>
              <Switch label={t('settings.auto_number')} defaultChecked disabled />
              {comingSoonBadge}
            </Group>
          </Stack>
        </Paper>

        <Paper p="md" radius="md" withBorder>
          <Text fw={500} mb="xs">{t('settings.currencies')}</Text>
          <Text c="dimmed" size="sm" mb="md">{t('settings.currencies_desc')}</Text>
          <Stack gap="md">
            <Group gap="xs" wrap="wrap">
              {localCurrencies.map((c, i) => (
                <Pill
                  key={c.code}
                  size="lg"
                  withRemoveButton={localCurrencies.length > 1}
                  onRemove={() => {
                    setLocalCurrencies(localCurrencies.filter((_, idx) => idx !== i))
                  }}
                >
                  {c.code}
                </Pill>
              ))}
            </Group>
            <Group>
              <TextInput
                placeholder={t('settings.currency_placeholder')}
                value={newCurrencyCode}
                onChange={(e) => setNewCurrencyCode(e.currentTarget.value.toUpperCase())}
                w={250}
                maxLength={10}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && newCurrencyCode.trim()) {
                    const code = newCurrencyCode.trim().toUpperCase()
                    if (!localCurrencies.some((c) => c.code === code)) {
                      setLocalCurrencies([...localCurrencies, { code }])
                      setNewCurrencyCode('')
                    }
                  }
                }}
              />
              <Button
                variant="light"
                size="sm"
                disabled={!newCurrencyCode.trim()}
                onClick={() => {
                  const code = newCurrencyCode.trim().toUpperCase()
                  if (code && !localCurrencies.some((c) => c.code === code)) {
                    setLocalCurrencies([...localCurrencies, { code }])
                    setNewCurrencyCode('')
                  }
                }}
              >
                {t('settings.add_currency')}
              </Button>
            </Group>
            <Button
              w={200}
              onClick={() => currenciesMutation.mutate(localCurrencies)}
              loading={currenciesMutation.isPending}
            >
              {t('settings.save_currencies')}
            </Button>
          </Stack>
        </Paper>
      </SimpleGrid>

      <Text c="dimmed" size="sm" fs="italic">{t('settings.defaults_hint')}</Text>

      {/* Row 4: Units + Due Days */}
      <SimpleGrid cols={{ base: 1, md: 2 }}>
        <Paper p="md" radius="md" withBorder>
          <Text fw={500} mb="xs">{t('settings.units')}</Text>
          <Text c="dimmed" size="sm" mb="md">{t('settings.units_desc')}</Text>
          <Stack gap="md">
            <Group gap="xs" wrap="wrap">
              {localUnits.map((u, i) => (
                <Pill
                  key={u.name}
                  size="lg"
                  withRemoveButton={localUnits.length > 1}
                  onRemove={() => {
                    const next = localUnits.filter((_, idx) => idx !== i)
                    if (u.is_default && next.length > 0) next[0].is_default = true
                    setLocalUnits(next)
                  }}
                  styles={{ root: { cursor: 'pointer', border: u.is_default ? '2px solid var(--mantine-color-blue-5)' : undefined } }}
                  onClick={() => {
                    setLocalUnits(localUnits.map((unit, idx) => ({ ...unit, is_default: idx === i })))
                  }}
                >
                  {u.name}
                </Pill>
              ))}
            </Group>
            <Group>
              <TextInput
                placeholder={t('settings.unit_placeholder')}
                value={newUnitName}
                onChange={(e) => setNewUnitName(e.currentTarget.value)}
                w={250}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && newUnitName.trim()) {
                    setLocalUnits([...localUnits, { name: newUnitName.trim() }])
                    setNewUnitName('')
                  }
                }}
              />
              <Button
                variant="light"
                size="sm"
                disabled={!newUnitName.trim()}
                onClick={() => {
                  setLocalUnits([...localUnits, { name: newUnitName.trim() }])
                  setNewUnitName('')
                }}
              >
                {t('settings.add_unit')}
              </Button>
            </Group>
            <Button
              w={200}
              onClick={() => unitsMutation.mutate(localUnits)}
              loading={unitsMutation.isPending}
            >
              {t('settings.save_units')}
            </Button>
          </Stack>
        </Paper>

        <Paper p="md" radius="md" withBorder>
          <Text fw={500} mb="xs">{t('settings.due_days')}</Text>
          <Text c="dimmed" size="sm" mb="md">{t('settings.due_days_desc')}</Text>
          <Stack gap="md">
            <Group gap="xs" wrap="wrap">
              {localDueDays.map((d, i) => (
                <Pill
                  key={d.days}
                  size="lg"
                  withRemoveButton={localDueDays.length > 1}
                  onRemove={() => {
                    const next = localDueDays.filter((_, idx) => idx !== i)
                    if (d.is_default && next.length > 0) next[0].is_default = true
                    setLocalDueDays(next)
                  }}
                  styles={{ root: { cursor: 'pointer', border: d.is_default ? '2px solid var(--mantine-color-blue-5)' : undefined } }}
                  onClick={() => {
                    setLocalDueDays(localDueDays.map((opt, idx) => ({ ...opt, is_default: idx === i })))
                  }}
                >
                  {d.days}
                </Pill>
              ))}
            </Group>
            <Text c="dimmed" size="xs" fs="italic">{t('settings.due_days_customer_hint')}</Text>
            <Group>
              <TextInput
                placeholder={t('settings.due_days_placeholder')}
                value={newDueDaysValue}
                onChange={(e) => setNewDueDaysValue(e.currentTarget.value)}
                w={250}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && newDueDaysValue.trim()) {
                    const days = parseInt(newDueDaysValue.trim(), 10)
                    if (!isNaN(days) && days > 0 && !localDueDays.some((d) => d.days === days)) {
                      setLocalDueDays([...localDueDays, { days }])
                      setNewDueDaysValue('')
                    }
                  }
                }}
              />
              <Button
                variant="light"
                size="sm"
                disabled={!newDueDaysValue.trim()}
                onClick={() => {
                  const days = parseInt(newDueDaysValue.trim(), 10)
                  if (!isNaN(days) && days > 0 && !localDueDays.some((d) => d.days === days)) {
                    setLocalDueDays([...localDueDays, { days }])
                    setNewDueDaysValue('')
                  }
                }}
              >
                {t('settings.add_due_days')}
              </Button>
            </Group>
            <Button
              w={200}
              onClick={() => dueDaysMutation.mutate(localDueDays)}
              loading={dueDaysMutation.isPending}
            >
              {t('settings.save_due_days')}
            </Button>
          </Stack>
        </Paper>
      </SimpleGrid>

      {/* Row 5: VAT Rates + Payment Types */}
      <SimpleGrid cols={{ base: 1, md: 2 }}>
        <Paper p="md" radius="md" withBorder>
          <Text fw={500} mb="xs">{t('settings.vat_rates')}</Text>
          <Text c="dimmed" size="sm" mb="md">{t('settings.vat_rates_desc')}</Text>
          <Stack gap="md">
            <Group gap="xs" wrap="wrap">
              {localVATRates.map((r, i) => (
                <Pill
                  key={`${r.rate}-${i}`}
                  size="lg"
                  withRemoveButton={localVATRates.length > 1}
                  onRemove={() => {
                    const next = localVATRates.filter((_, idx) => idx !== i)
                    if (r.is_default && next.length > 0) next[0].is_default = true
                    setLocalVATRates(next)
                  }}
                  styles={{ root: { cursor: 'pointer', border: r.is_default ? '2px solid var(--mantine-color-blue-5)' : undefined } }}
                  onClick={() => {
                    setLocalVATRates(localVATRates.map((rate, idx) => ({ ...rate, is_default: idx === i })))
                  }}
                >
                  {r.rate}%{r.name ? ` (${r.name})` : ''}
                </Pill>
              ))}
            </Group>
            <Group>
              <TextInput
                placeholder={t('settings.vat_rate_placeholder')}
                value={newVATRate}
                onChange={(e) => setNewVATRate(e.currentTarget.value)}
                w={250}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && newVATRate.trim()) {
                    const rate = parseFloat(newVATRate.trim())
                    if (!isNaN(rate) && rate >= 0 && !localVATRates.some((r) => r.rate === rate)) {
                      setLocalVATRates([...localVATRates, { rate }])
                      setNewVATRate('')
                    }
                  }
                }}
              />
              <Button
                variant="light"
                size="sm"
                disabled={!newVATRate.trim()}
                onClick={() => {
                  const rate = parseFloat(newVATRate.trim())
                  if (!isNaN(rate) && rate >= 0 && !localVATRates.some((r) => r.rate === rate)) {
                    setLocalVATRates([...localVATRates, { rate }])
                    setNewVATRate('')
                  }
                }}
              >
                {t('settings.add_vat_rate')}
              </Button>
            </Group>
            <Button
              w={200}
              onClick={() => vatRatesMutation.mutate(localVATRates)}
              loading={vatRatesMutation.isPending}
            >
              {t('settings.save_vat_rates')}
            </Button>
          </Stack>
        </Paper>

        <Paper p="md" radius="md" withBorder>
          <Text fw={500} mb="xs">{t('settings.payment_types')}</Text>
          <Text c="dimmed" size="sm" mb="md">{t('settings.payment_types_desc')}</Text>
          <Stack gap="md">
            <Group gap="xs" wrap="wrap">
              {localPaymentTypes.map((pt, i) => (
                <Pill
                  key={pt.name}
                  size="lg"
                  withRemoveButton={!pt.code && localPaymentTypes.length > 1}
                  onRemove={() => {
                    const next = localPaymentTypes.filter((_, idx) => idx !== i)
                    if (pt.is_default && next.length > 0) next[0].is_default = true
                    setLocalPaymentTypes(next)
                  }}
                  styles={{ root: { cursor: 'pointer', border: pt.is_default ? '2px solid var(--mantine-color-blue-5)' : undefined } }}
                  onClick={() => {
                    setLocalPaymentTypes(localPaymentTypes.map((p, idx) => ({ ...p, is_default: idx === i })))
                  }}
                >
                  {pt.name}
                </Pill>
              ))}
            </Group>
            <Text c="dimmed" size="sm">{t('settings.payment_types_bank_desc')}</Text>
            {localPaymentTypes.filter((pt) => !pt.code).length === 0 ? (
              <Text c="dimmed" size="sm" fs="italic">{t('settings.no_custom_payment_types')}</Text>
            ) : (
              <Stack gap="xs">
                {localPaymentTypes.filter((pt) => !pt.code).map((pt) => {
                  const idx = localPaymentTypes.indexOf(pt)
                  return (
                    <Switch
                      key={pt.name}
                      label={pt.name}
                      checked={pt.requires_bank_info !== false}
                      onChange={(e) => {
                        setLocalPaymentTypes(localPaymentTypes.map((p, j) =>
                          j === idx ? { ...p, requires_bank_info: e.currentTarget.checked } : p
                        ))
                      }}
                      size="sm"
                    />
                  )
                })}
              </Stack>
            )}
            <Group>
              <TextInput
                placeholder={t('settings.payment_type_placeholder')}
                value={newPaymentTypeName}
                onChange={(e) => setNewPaymentTypeName(e.currentTarget.value)}
                w={250}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && newPaymentTypeName.trim()) {
                    const name = newPaymentTypeName.trim()
                    if (!localPaymentTypes.some((pt) => pt.name === name)) {
                      setLocalPaymentTypes([...localPaymentTypes, { name, requires_bank_info: newPaymentTypeRequiresBank }])
                      setNewPaymentTypeName('')
                      setNewPaymentTypeRequiresBank(true)
                    }
                  }
                }}
              />
              <Button
                variant="light"
                size="sm"
                disabled={!newPaymentTypeName.trim()}
                onClick={() => {
                  const name = newPaymentTypeName.trim()
                  if (name && !localPaymentTypes.some((pt) => pt.name === name)) {
                    setLocalPaymentTypes([...localPaymentTypes, { name, requires_bank_info: newPaymentTypeRequiresBank }])
                    setNewPaymentTypeName('')
                    setNewPaymentTypeRequiresBank(true)
                  }
                }}
              >
                {t('settings.add_payment_type')}
              </Button>
            </Group>
            <Switch
              label={t('settings.requires_bank_info')}
              description={t('settings.requires_bank_info_desc')}
              checked={newPaymentTypeRequiresBank}
              onChange={(e) => setNewPaymentTypeRequiresBank(e.currentTarget.checked)}
              size="sm"
            />
            <Button
              w={200}
              onClick={() => paymentTypesMutation.mutate(localPaymentTypes)}
              loading={paymentTypesMutation.isPending}
            >
              {t('settings.save_payment_types')}
            </Button>
          </Stack>
        </Paper>
      </SimpleGrid>

      {/* Row 6: PDF Output + About */}
      <SimpleGrid cols={{ base: 1, md: 2 }}>
        <Paper p="md" radius="md" withBorder>
          <Text fw={500} mb="md">{t('settings.pdf_output')}</Text>
          <Stack gap="md">
            <Group gap="md">
              <Text size="sm">{t('settings.default_template')}:</Text>
              <Badge variant="light" color="blue" size="lg">
                {defaultTemplate?.name || '—'}
              </Badge>
            </Group>
            <Text c="dimmed" size="sm">{t('settings.pdf_output_hint')}</Text>
            <Button
              w={250}
              variant="light"
              onClick={() => navigate('/templates')}
            >
              {t('settings.go_to_templates')}
            </Button>
          </Stack>
        </Paper>

        {aboutInfo && (
          <Paper p="md" radius="md" withBorder>
            <Group mb="md" gap="sm">
              <Title order={3}>TidyBill</Title>
              <Badge variant="light" color="blue" size="lg">v{aboutInfo.version}</Badge>
            </Group>
            <Text size="sm" mb="xs">{t('about.description')}</Text>
            <Text size="sm" c="dimmed" mb="lg">{t('about.opensource')}</Text>

            <Text fw={500} size="sm" mb={4}>{t('about.issues_title')}</Text>
            <Anchor
              size="sm"
              mb="lg"
              onClick={(e) => { e.preventDefault(); openInBrowser(aboutInfo.github_issues_url) }}
              style={{ cursor: 'pointer' }}
            >
              {t('about.issues_link')}
            </Anchor>

            <Text fw={500} size="sm" mt="md" mb={4}>{t('about.support_title')}</Text>
            <Text size="sm" c="dimmed" mb="sm">{t('about.support_desc')}</Text>
            <Stack gap="xs">
              <Group gap="xs">
                <Text size="sm" fw={500} w={100}>Monero (XMR)</Text>
                <Code>{aboutInfo.monero_address}</Code>
                <CopyButton value={aboutInfo.monero_address}>
                  {({ copied, copy }) => (
                    <Tooltip label={copied ? t('about.copied') : t('common.copy')}>
                      <ActionIcon variant="subtle" color={copied ? 'teal' : 'gray'} onClick={copy}>
                        {copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
                      </ActionIcon>
                    </Tooltip>
                  )}
                </CopyButton>
              </Group>
              <Group gap="xs">
                <Text size="sm" fw={500} w={100}>Bitcoin (BTC)</Text>
                <Code>{aboutInfo.bitcoin_address}</Code>
                <CopyButton value={aboutInfo.bitcoin_address}>
                  {({ copied, copy }) => (
                    <Tooltip label={copied ? t('about.copied') : t('common.copy')}>
                      <ActionIcon variant="subtle" color={copied ? 'teal' : 'gray'} onClick={copy}>
                        {copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
                      </ActionIcon>
                    </Tooltip>
                  )}
                </CopyButton>
              </Group>
            </Stack>
          </Paper>
        )}
      </SimpleGrid>

    </Stack>
  )
}
