import {
  Title,
  Text,
  Paper,
  Stack,
  Select,
  Switch,
  Group,
  Badge,
  Loader,
  Center,
  TextInput,
  Button,
  Pill,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { api, type Unit, type PDFTemplate, type VATRate, type CurrencyItem } from '../api/client'
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
        <Text fw={500} mb="md">{t('settings.invoices')}</Text>
        <Stack gap="md">
          <Select
            label={t('settings.default_vat')}
            data={(localVATRates.length > 0 ? localVATRates : [{ rate: 0 }, { rate: 12 }, { rate: 21 }])
              .map((r) => ({ value: String(r.rate), label: `${r.rate}%` }))}
            value={settings?.default_vat_rate || '21'}
            onChange={(v) => { if (v) updateMutation.mutate({ default_vat_rate: v }) }}
            w={300}
          />
          <Select
            label={t('settings.default_due')}
            data={['7', '14', '30', '60']}
            value={settings?.default_due_days || '14'}
            onChange={(v) => { if (v) updateMutation.mutate({ default_due_days: v }) }}
            w={300}
          />
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
        <Text fw={500} mb="md">{t('settings.directories')}</Text>
        <Stack gap="md">
          <TextInput
            label={t('settings.dir_logos')}
            placeholder={t('settings.dir_placeholder')}
            value={dirLogos}
            onChange={(e) => setDirLogos(e.currentTarget.value)}
            w={500}
          />
          <TextInput
            label={t('settings.dir_pdfs')}
            placeholder={t('settings.dir_placeholder')}
            value={dirPdfs}
            onChange={(e) => setDirPdfs(e.currentTarget.value)}
            w={500}
          />
          <TextInput
            label={t('settings.dir_previews')}
            placeholder={t('settings.dir_placeholder')}
            value={dirPreviews}
            onChange={(e) => setDirPreviews(e.currentTarget.value)}
            w={500}
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
                {u.is_default && (
                  <Badge size="xs" variant="light" color="blue" ml={4}>{t('settings.default_unit_label')}</Badge>
                )}
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
                  setLocalVATRates(localVATRates.filter((_, idx) => idx !== i))
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
        <Text fw={500} mb="md">{t('settings.dashboard')}</Text>
        <Text c="dimmed" size="sm" mb="md">{t('settings.dashboard_desc')}</Text>
        <Stack gap="sm">
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
        </Stack>
      </Paper>

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

    </Stack>
  )
}
