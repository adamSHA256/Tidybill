import {
  Title,
  Text,
  Paper,
  Group,
  Stack,
  TextInput,
  Select,
  NumberInput,
  Table,
  Button,
  Divider,
  ActionIcon,
  SimpleGrid,
  Loader,
  Center,
  Menu,
  Badge,
  Modal,
  Textarea,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { DateInput } from '@mantine/dates'
import { IconTrash, IconPlus, IconPackage } from '@tabler/icons-react'
import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, formatMoney, type BankAccount, type Item, type CustomerItem } from '../api/client'
import { CountrySelect } from '../components/CountrySelect'
import { useT } from '../i18n'

const CREATE_NEW = '__create_new__'
const ADD_CURRENCY = '__add_currency__'
const ADD_VAT_RATE = '__add_vat_rate__'
const ADD_UNIT = '__add_unit__'

interface ItemForm {
  item_id: string
  description: string
  quantity: number
  unit: string
  unit_price: number
  vat_rate: number
}

const emptyItem: ItemForm = { item_id: '', description: '', quantity: 1, unit: 'ks', unit_price: 0, vat_rate: 21 }

export function InvoiceEdit() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { t } = useT()

  const [initialized, setInitialized] = useState(false)
  const [statusWarningOpen, setStatusWarningOpen] = useState(false)
  const [statusConfirmed, setStatusConfirmed] = useState(false)

  const [supplierId, setSupplierId] = useState<string | null>(null)
  const [customerId, setCustomerId] = useState<string | null>(null)
  const [bankAccountId, setBankAccountId] = useState<string | null>(null)
  const [issueDate, setIssueDate] = useState<string | null>(null)
  const [dueDate, setDueDate] = useState<string | null>(null)
  const [currency, setCurrency] = useState('CZK')
  const [invoiceNumber, setInvoiceNumber] = useState('')
  const [notes, setNotes] = useState('')
  const [items, setItems] = useState<ItemForm[]>([{ ...emptyItem }])

  // Modal states for inline creation
  const [customerModalOpen, setCustomerModalOpen] = useState(false)
  const [bankModalOpen, setBankModalOpen] = useState(false)
  const [vatRateModalOpen, setVatRateModalOpen] = useState(false)
  const [newVatRateValue, setNewVatRateValue] = useState('')
  const [vatRateTargetIndex, setVatRateTargetIndex] = useState<number>(-1)
  const [unitModalOpen, setUnitModalOpen] = useState(false)
  const [newUnitValue, setNewUnitValue] = useState('')
  const [unitTargetIndex, setUnitTargetIndex] = useState<number>(-1)
  const [currencyModalOpen, setCurrencyModalOpen] = useState(false)
  const [newCurrencyCode, setNewCurrencyCode] = useState('')
  const [currencyTarget, setCurrencyTarget] = useState<'invoice' | 'bank'>('invoice')

  // Customer form state
  const [cName, setCName] = useState('')
  const [cIco, setCIco] = useState('')
  const [cDic, setCDic] = useState('')
  const [cStreet, setCStreet] = useState('')
  const [cCity, setCCity] = useState('')
  const [cZip, setCZip] = useState('')
  const [cCountry, setCCountry] = useState('CZ')
  const [cEmail, setCEmail] = useState('')
  const [cPhone, setCPhone] = useState('')
  const [cNotes, setCNotes] = useState('')

  // Bank account form state
  const [bName, setBName] = useState('')
  const [bAccountNumber, setBAccountNumber] = useState('')
  const [bIban, setBIban] = useState('')
  const [bSwift, setBSwift] = useState('')
  const [bCurrency, setBCurrency] = useState('CZK')

  const { data: invoice, isLoading: invoiceLoading } = useQuery({
    queryKey: ['invoice', id],
    queryFn: () => api.getInvoice(id!),
    enabled: !!id,
  })

  const { data: suppliers } = useQuery({
    queryKey: ['suppliers'],
    queryFn: api.getSuppliers,
  })

  const { data: customers } = useQuery({
    queryKey: ['customers'],
    queryFn: () => api.getCustomers(),
  })

  const { data: bankAccounts } = useQuery({
    queryKey: ['bank-accounts', supplierId],
    queryFn: () => api.getBankAccounts(supplierId!),
    enabled: !!supplierId,
  })

  const { data: customerItems } = useQuery({
    queryKey: ['customer-items', customerId],
    queryFn: () => api.getCustomerItems(customerId!),
    enabled: !!customerId,
  })

  const { data: mostUsedItems } = useQuery({
    queryKey: ['items-most-used'],
    queryFn: api.getMostUsedItems,
  })

  const { data: vatRates } = useQuery({
    queryKey: ['vat-rates'],
    queryFn: api.getVATRates,
  })
  const vatRateOptions = (vatRates || []).map((r) => String(r.rate))

  const { data: units } = useQuery({
    queryKey: ['units'],
    queryFn: api.getUnits,
  })
  const unitOptions = (units || []).map((u) => u.name)

  const { data: currenciesList } = useQuery({
    queryKey: ['currencies'],
    queryFn: api.getCurrencies,
  })

  const { data: editSettings } = useQuery({
    queryKey: ['settings'],
    queryFn: api.getSettings,
  })
  const defaultVatRate = parseFloat(editSettings?.default_vat_rate || '21') || 21

  // Initialize form from loaded invoice
  useEffect(() => {
    if (invoice && !initialized) {
      setSupplierId(invoice.supplier_id)
      setCustomerId(invoice.customer_id)
      setBankAccountId(invoice.bank_account_id)
      setIssueDate(invoice.issue_date?.slice(0, 10) || null)
      setDueDate(invoice.due_date?.slice(0, 10) || null)
      setCurrency(invoice.currency || 'CZK')
      setInvoiceNumber(invoice.invoice_number || '')
      setNotes(invoice.notes || '')
      if (invoice.items && invoice.items.length > 0) {
        setItems(invoice.items.map((item) => ({
          item_id: item.item_id || '',
          description: item.description,
          quantity: item.quantity,
          unit: item.unit,
          unit_price: item.unit_price,
          vat_rate: item.vat_rate,
        })))
      }
      setInitialized(true)

      // Show status warning for non-draft/created invoices
      if (invoice.status !== 'draft' && invoice.status !== 'created') {
        setStatusWarningOpen(true)
      } else {
        setStatusConfirmed(true)
      }
    }
  }, [invoice, initialized])

  const updateMutation = useMutation({
    mutationFn: (data: Parameters<typeof api.updateInvoice>[1]) =>
      api.updateInvoice(id!, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['invoice', id] })
      queryClient.invalidateQueries({ queryKey: ['invoices'] })
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] })
      queryClient.invalidateQueries({ queryKey: ['items'] })
      notifications.show({ title: t('notify.invoice_updated'), message: t('notify.invoice_updated_msg'), color: 'green' })
      navigate(`/invoices/${id}`)
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const createCustomerMutation = useMutation({
    mutationFn: (data: Partial<import('../api/client').Customer>) => api.createCustomer(data),
    onSuccess: (newCustomer) => {
      queryClient.invalidateQueries({ queryKey: ['customers'] })
      setCustomerId(newCustomer.id)
      setCustomerModalOpen(false)
      notifications.show({ title: t('notify.customer_created'), message: t('notify.customer_saved_msg').replace('{name}', newCustomer.name), color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const createBankMutation = useMutation({
    mutationFn: (data: Partial<BankAccount>) => api.createBankAccount(supplierId!, data),
    onSuccess: (newBank) => {
      queryClient.invalidateQueries({ queryKey: ['bank-accounts', supplierId] })
      setBankAccountId(newBank.id)
      setBankModalOpen(false)
      notifications.show({ title: t('notify.bank_account_created'), message: t('notify.bank_account_created_msg'), color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  const openCustomerModal = () => {
    setCName(''); setCIco(''); setCDic(''); setCStreet(''); setCCity(''); setCZip('')
    setCCountry('CZ'); setCEmail(''); setCPhone(''); setCNotes('')
    setCustomerModalOpen(true)
  }

  const openBankModal = () => {
    setBName(''); setBAccountNumber(''); setBIban(''); setBSwift(''); setBCurrency('CZK')
    setBankModalOpen(true)
  }

  const handleSaveCustomer = () => {
    if (!cName.trim()) {
      notifications.show({ title: t('customer.missing_name_title'), message: t('customer.missing_name_msg'), color: 'orange' })
      return
    }
    createCustomerMutation.mutate({
      name: cName, ico: cIco, dic: cDic, street: cStreet, city: cCity, zip: cZip,
      country: cCountry, email: cEmail, phone: cPhone, notes: cNotes,
    })
  }

  const handleSaveBank = () => {
    if (!bName.trim() && !bAccountNumber.trim()) {
      notifications.show({ title: t('bank_account.missing_fields_title'), message: t('bank_account.missing_fields_msg'), color: 'orange' })
      return
    }
    createBankMutation.mutate({
      name: bName, account_number: bAccountNumber, iban: bIban, swift: bSwift, currency: bCurrency,
    })
  }

  // Build VAT rate select data with "Add new" option
  const vatRateSelectData = [
    ...(vatRateOptions.length > 0 ? vatRateOptions : ['0', '12', '21']).map((r) => ({ value: r, label: `${r}%` })),
    { value: ADD_VAT_RATE, label: `+ ${t('invoice.add_vat_rate')}` },
  ]

  const handleVatRateSelect = (val: string | null, index: number) => {
    if (val === ADD_VAT_RATE) {
      setNewVatRateValue('')
      setVatRateTargetIndex(index)
      setVatRateModalOpen(true)
      return
    }
    updateItem(index, 'vat_rate', Number(val))
  }

  const handleAddVatRate = () => {
    const rate = parseFloat(newVatRateValue.trim())
    if (isNaN(rate) || rate < 0) return
    const currentRates = vatRates || []
    if (!currentRates.some((r) => r.rate === rate)) {
      api.updateVATRates([...currentRates, { rate }]).then(() => {
        queryClient.invalidateQueries({ queryKey: ['vat-rates'] })
      })
    }
    if (vatRateTargetIndex >= 0) updateItem(vatRateTargetIndex, 'vat_rate', rate)
    setVatRateModalOpen(false)
  }

  // Build unit select data with "Add new" option
  const unitSelectData = [
    ...(unitOptions.length > 0 ? unitOptions : ['ks', 'hod', 'den', 'm\u00B2']).map((u) => ({ value: u, label: u })),
    { value: ADD_UNIT, label: `+ ${t('invoice.add_unit')}` },
  ]

  const handleUnitSelect = (val: string | null, index: number) => {
    if (val === ADD_UNIT) {
      setNewUnitValue('')
      setUnitTargetIndex(index)
      setUnitModalOpen(true)
      return
    }
    updateItem(index, 'unit', val || unitOptions[0] || 'ks')
  }

  const handleAddUnit = () => {
    const name = newUnitValue.trim()
    if (!name) return
    const currentUnits = units || []
    if (!currentUnits.some((u) => u.name === name)) {
      api.updateUnits([...currentUnits, { name }]).then(() => {
        queryClient.invalidateQueries({ queryKey: ['units'] })
      })
    }
    if (unitTargetIndex >= 0) updateItem(unitTargetIndex, 'unit', name)
    setUnitModalOpen(false)
  }

  // Build currency select data from API + "Add new"
  const currencyData = [
    ...(currenciesList || []).map((c) => ({ value: c.code, label: c.code })),
    { value: ADD_CURRENCY, label: `+ ${t('invoice.add_currency')}` },
  ]

  const handleCurrencySelect = (v: string | null, target: 'invoice' | 'bank') => {
    if (v === ADD_CURRENCY) {
      setNewCurrencyCode('')
      setCurrencyTarget(target)
      setCurrencyModalOpen(true)
      return
    }
    if (target === 'invoice') setCurrency(v || 'CZK')
    else setBCurrency(v || 'CZK')
  }

  const handleAddCurrency = () => {
    const code = newCurrencyCode.trim().toUpperCase()
    if (!code) return
    const currentCurrencies = currenciesList || []
    if (!currentCurrencies.some((c) => c.code === code)) {
      api.updateCurrencies([...currentCurrencies, { code }]).then(() => {
        queryClient.invalidateQueries({ queryKey: ['currencies'] })
      })
    }
    if (currencyTarget === 'invoice') setCurrency(code)
    else setBCurrency(code)
    setCurrencyModalOpen(false)
  }

  const addItem = () => setItems([...items, { ...emptyItem, vat_rate: defaultVatRate }])
  const removeItem = (index: number) => {
    if (items.length > 1) setItems(items.filter((_, i) => i !== index))
  }
  const updateItem = (index: number, field: keyof ItemForm, value: string | number) => {
    const updated = [...items]
    updated[index] = { ...updated[index], [field]: value }
    setItems(updated)
  }

  const addFromCatalog = (catalogItem: Item, customerItem?: CustomerItem) => {
    const price = customerItem ? customerItem.last_price : catalogItem.default_price
    const qty = customerItem ? customerItem.last_quantity : 1
    const newItem: ItemForm = {
      item_id: catalogItem.id,
      description: catalogItem.description,
      quantity: qty || 1,
      unit: catalogItem.default_unit || 'ks',
      unit_price: price,
      vat_rate: catalogItem.default_vat_rate,
    }
    if (items.length === 1 && !items[0].description) {
      setItems([newItem])
    } else {
      setItems([...items, newItem])
    }
  }

  const subtotal = items.reduce((sum, item) => sum + item.quantity * item.unit_price, 0)
  const vatAmount = items.reduce(
    (sum, item) => sum + (item.quantity * item.unit_price * item.vat_rate) / 100, 0
  )
  const total = subtotal + vatAmount

  const handleSave = () => {
    if (!supplierId || !customerId || !bankAccountId) {
      notifications.show({ title: t('invoice.missing_fields_title'), message: t('invoice.missing_fields_msg'), color: 'orange' })
      return
    }
    if (items.every((i) => !i.description)) {
      notifications.show({ title: t('invoice.missing_items_title'), message: t('invoice.missing_items_msg'), color: 'orange' })
      return
    }
    updateMutation.mutate({
      supplier_id: supplierId,
      customer_id: customerId,
      bank_account_id: bankAccountId,
      invoice_number: invoiceNumber || undefined,
      issue_date: issueDate || undefined,
      due_date: dueDate || undefined,
      currency,
      notes,
      items: items.filter((i) => i.description).map((i) => ({
        item_id: i.item_id || undefined,
        description: i.description,
        quantity: i.quantity,
        unit: i.unit,
        unit_price: i.unit_price,
        vat_rate: i.vat_rate,
      })),
    })
  }

  if (invoiceLoading || !invoice) {
    return <Center h={300}><Loader /></Center>
  }

  const selectedSupplier = suppliers?.find((s) => s.id === supplierId)
  const selectedCustomer = customers?.find((c) => c.id === customerId)

  const customerData = [
    ...(customers || []).map((c) => ({ value: c.id, label: c.name })),
    { value: CREATE_NEW, label: `+ ${t('invoice.create_new_customer')}` },
  ]

  const bankData = [
    ...(bankAccounts || []).map((b) => ({ value: b.id, label: `${b.account_number} (${b.currency})` })),
    { value: CREATE_NEW, label: `+ ${t('invoice.create_new_bank_account')}` },
  ]

  const customerItemIds = new Set((customerItems || []).map((ci: CustomerItem) => ci.item_id))
  const globalSuggestions = (mostUsedItems || []).filter((item: Item) => !customerItemIds.has(item.id))

  const handleCustomerSelect = (v: string | null) => {
    if (v === CREATE_NEW) {
      openCustomerModal()
      return
    }
    setCustomerId(v)
  }

  const handleBankSelect = (v: string | null) => {
    if (v === CREATE_NEW) {
      openBankModal()
      return
    }
    setBankAccountId(v)
  }

  return (
    <Stack gap="lg">
      {/* Status warning modal */}
      <Modal
        opened={statusWarningOpen}
        onClose={() => { setStatusWarningOpen(false); navigate(`/invoices/${id}`) }}
        title={t('invoice.edit_status_warning_title')}
      >
        <Stack gap="md">
          <Text size="sm" dangerouslySetInnerHTML={{
            __html: t('invoice.edit_status_warning').replace('{status}', t(`status.${invoice.status}`))
          }} />
          <Group justify="end">
            <Button variant="default" onClick={() => { setStatusWarningOpen(false); navigate(`/invoices/${id}`) }}>
              {t('common.cancel')}
            </Button>
            <Button color="yellow" onClick={() => { setStatusWarningOpen(false); setStatusConfirmed(true) }}>
              {t('invoice.continue_editing')}
            </Button>
          </Group>
        </Stack>
      </Modal>

      {!statusConfirmed ? (
        <Center h={300}><Loader /></Center>
      ) : (
        <>
          <Group justify="space-between">
            <div>
              <Title order={2}>{t('invoice.edit_title')}</Title>
              <Text c="dimmed" size="sm">{t('invoice.edit_subtitle')}</Text>
            </div>
            <Group>
              <Button variant="default" onClick={() => navigate(`/invoices/${id}`)}>{t('common.cancel')}</Button>
              <Button onClick={handleSave} loading={updateMutation.isPending}>{t('common.save')}</Button>
            </Group>
          </Group>

          <Paper p="md" radius="md" withBorder>
            <Text fw={500} mb="md">{t('invoice.details')}</Text>
            <SimpleGrid cols={{ base: 1, sm: 2, md: 4 }}>
              <TextInput label={t('invoice.invoice_number')} value={invoiceNumber} onChange={(e) => setInvoiceNumber(e.currentTarget.value)} />
              <DateInput label={t('invoice.issue_date')} valueFormat="DD.MM.YYYY" value={issueDate} onChange={setIssueDate} clearable />
              <DateInput label={t('invoice.due_date')} valueFormat="DD.MM.YYYY" value={dueDate} onChange={setDueDate} clearable />
              <Select label={t('invoice.currency')} data={currencyData} value={currency} onChange={(v) => handleCurrencySelect(v, 'invoice')} searchable />
            </SimpleGrid>
          </Paper>

          <SimpleGrid cols={{ base: 1, md: 2 }}>
            <Paper p="md" radius="md" withBorder>
              <Text fw={500} mb="md">{t('invoice.supplier_you')}</Text>
              {selectedSupplier && (
                <Stack gap={4}>
                  <Text size="sm" fw={600}>{selectedSupplier.name}</Text>
                  <Text size="sm" c="dimmed">
                    ICO: {selectedSupplier.ico}{selectedSupplier.dic && ` | DIC: ${selectedSupplier.dic}`}
                  </Text>
                  <Text size="sm" c="dimmed">{selectedSupplier.street}, {selectedSupplier.city}, {selectedSupplier.zip}</Text>
                </Stack>
              )}
              {supplierId && bankAccounts && bankAccounts.length > 0 ? (
                <Select
                  label={t('invoice.bank_account')}
                  mt="sm"
                  data={bankData}
                  value={bankAccountId}
                  onChange={handleBankSelect}
                />
              ) : supplierId && bankAccounts && bankAccounts.length === 0 ? (
                <Button variant="light" mt="sm" leftSection={<IconPlus size={14} />} onClick={openBankModal} fullWidth>
                  {t('invoice.create_first_bank_account')}
                </Button>
              ) : null}
            </Paper>

            <Paper p="md" radius="md" withBorder>
              <Text fw={500} mb="md">{t('invoice.customer')}</Text>
              <Select
                label={t('invoice.select_customer')}
                placeholder={t('invoice.search_customer')}
                data={customerData}
                value={customerId}
                onChange={handleCustomerSelect}
                searchable
              />
              {selectedCustomer && (
                <Stack gap={4} mt="sm">
                  <Text size="sm" c="dimmed">
                    ICO: {selectedCustomer.ico}{selectedCustomer.dic && ` | DIC: ${selectedCustomer.dic}`}
                  </Text>
                  <Text size="sm" c="dimmed">{selectedCustomer.street}, {selectedCustomer.city}, {selectedCustomer.zip}</Text>
                </Stack>
              )}
            </Paper>
          </SimpleGrid>

          <Paper p="md" radius="md" withBorder>
            <Group justify="space-between" mb="md">
              <Text fw={500}>{t('invoice.items')}</Text>
              <Group gap="xs">
                <Menu shadow="md" width={320} position="bottom-end">
                  <Menu.Target>
                    <Button size="xs" variant="light" leftSection={<IconPackage size={14} />}
                      disabled={!customerItems?.length && !globalSuggestions.length}>
                      {t('invoice.from_catalog')}
                    </Button>
                  </Menu.Target>
                  <Menu.Dropdown>
                    {(customerItems || []).length > 0 && (
                      <>
                        <Menu.Label>{t('invoice.customer_items')}</Menu.Label>
                        {(customerItems || []).map((ci: CustomerItem) => {
                          const catalogItem = mostUsedItems?.find((i: Item) => i.id === ci.item_id)
                          return (
                            <Menu.Item key={ci.id}
                              onClick={() => {
                                if (catalogItem) addFromCatalog(catalogItem, ci)
                                else addFromCatalog({
                                  id: ci.item_id,
                                  description: ci.item_description,
                                  default_price: ci.last_price,
                                  default_unit: ci.item_default_unit,
                                  default_vat_rate: ci.item_default_vat,
                                } as Item, ci)
                              }}
                              rightSection={<Badge size="xs" variant="light">{ci.usage_count}x</Badge>}
                            >
                              <Text size="sm">{ci.item_description}</Text>
                              <Text size="xs" c="dimmed">{formatMoney(ci.last_price)} / {ci.item_default_unit}</Text>
                            </Menu.Item>
                          )
                        })}
                      </>
                    )}
                    {globalSuggestions.length > 0 && (
                      <>
                        <Menu.Label>{t('invoice.most_used')}</Menu.Label>
                        {globalSuggestions.slice(0, 10).map((item: Item) => (
                          <Menu.Item key={item.id}
                            onClick={() => addFromCatalog(item)}
                            rightSection={<Badge size="xs" variant="light" color="gray">{item.usage_count}x</Badge>}
                          >
                            <Text size="sm">{item.description}</Text>
                            <Text size="xs" c="dimmed">{formatMoney(item.default_price)} / {item.default_unit}</Text>
                          </Menu.Item>
                        ))}
                      </>
                    )}
                  </Menu.Dropdown>
                </Menu>
                <Button size="xs" leftSection={<IconPlus size={14} />} onClick={addItem}>{t('invoice.add_item')}</Button>
              </Group>
            </Group>

            <Table>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th style={{ width: '35%' }}>{t('invoice.description')}</Table.Th>
                  <Table.Th style={{ width: '10%' }}>{t('invoice.qty')}</Table.Th>
                  <Table.Th style={{ width: '10%' }}>{t('invoice.unit')}</Table.Th>
                  <Table.Th style={{ width: '15%' }}>{t('invoice.unit_price')}</Table.Th>
                  <Table.Th style={{ width: '10%' }}>{t('invoice.vat_pct')}</Table.Th>
                  <Table.Th style={{ width: '15%' }}>{t('invoice.total_col')}</Table.Th>
                  <Table.Th style={{ width: '5%' }}></Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {items.map((item, i) => (
                  <Table.Tr key={i}>
                    <Table.Td>
                      <Group gap={4}>
                        <TextInput size="sm" placeholder={t('invoice.description_placeholder')} value={item.description}
                          onChange={(e) => updateItem(i, 'description', e.currentTarget.value)}
                          style={{ flex: 1 }} />
                        {item.item_id && <Badge size="xs" variant="light" color="blue">{t('invoice.catalog')}</Badge>}
                      </Group>
                    </Table.Td>
                    <Table.Td>
                      <NumberInput size="sm" min={1} value={item.quantity}
                        onChange={(val) => updateItem(i, 'quantity', val || 0)} />
                    </Table.Td>
                    <Table.Td>
                      <Select size="sm" data={unitSelectData} value={item.unit}
                        onChange={(val) => handleUnitSelect(val, i)} />
                    </Table.Td>
                    <Table.Td>
                      <NumberInput size="sm" min={0} value={item.unit_price}
                        onChange={(val) => updateItem(i, 'unit_price', val || 0)} />
                    </Table.Td>
                    <Table.Td>
                      <Select size="sm" data={vatRateSelectData} value={String(item.vat_rate)}
                        onChange={(val) => handleVatRateSelect(val, i)} />
                    </Table.Td>
                    <Table.Td>
                      <Text size="sm" fw={600}>{formatMoney(item.quantity * item.unit_price)}</Text>
                    </Table.Td>
                    <Table.Td>
                      <ActionIcon color="red" variant="light" size="sm" onClick={() => removeItem(i)}
                        disabled={items.length === 1}>
                        <IconTrash size={14} />
                      </ActionIcon>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>

            <Divider my="md" />

            <Stack gap={4} align="end" pr="xl">
              <Group>
                <Text size="sm" c="dimmed" w={180} ta="right">{t('invoice.subtotal')}</Text>
                <Text size="sm" fw={600} w={120} ta="right">{formatMoney(subtotal)}</Text>
              </Group>
              <Group>
                <Text size="sm" c="dimmed" w={180} ta="right">{t('invoice.vat')}</Text>
                <Text size="sm" fw={600} w={120} ta="right">{formatMoney(vatAmount)}</Text>
              </Group>
              <Divider w={300} />
              <Group>
                <Text size="lg" fw={700} w={180} ta="right">{t('invoice.total')}</Text>
                <Text size="lg" fw={700} w={120} ta="right">{formatMoney(total)}</Text>
              </Group>
            </Stack>
          </Paper>

          <TextInput label={t('invoice.notes_label')} placeholder={t('invoice.notes_placeholder')}
            value={notes} onChange={(e) => setNotes(e.currentTarget.value)} />

          <Group justify="end">
            <Button variant="default" onClick={() => navigate(`/invoices/${id}`)}>{t('common.cancel')}</Button>
            <Button onClick={handleSave} loading={updateMutation.isPending}>{t('common.save')}</Button>
          </Group>

          {/* Customer creation modal */}
          <Modal opened={customerModalOpen} onClose={() => setCustomerModalOpen(false)}
            title={t('customer.new_title')} size="lg">
            <Stack gap="md">
              <TextInput label={t('customer.name_label')} value={cName}
                onChange={(e) => setCName(e.currentTarget.value)} required />
              <Group grow>
                <TextInput label={t('customer.ico_label')} value={cIco}
                  onChange={(e) => setCIco(e.currentTarget.value)} />
                <TextInput label={t('customer.dic_label')} value={cDic}
                  onChange={(e) => setCDic(e.currentTarget.value)} />
              </Group>
              <TextInput label={t('customer.street_label')} value={cStreet}
                onChange={(e) => setCStreet(e.currentTarget.value)} />
              <Group grow>
                <TextInput label={t('customer.city_label')} value={cCity}
                  onChange={(e) => setCCity(e.currentTarget.value)} />
                <TextInput label={t('customer.zip_label')} value={cZip}
                  onChange={(e) => setCZip(e.currentTarget.value)} />
              </Group>
              <CountrySelect label={t('customer.country_label')}
                value={cCountry} onChange={(v) => setCCountry(v || 'CZ')} />
              <Group grow>
                <TextInput label={t('customer.email_label')} value={cEmail}
                  onChange={(e) => setCEmail(e.currentTarget.value)} />
                <TextInput label={t('customer.phone_label')} value={cPhone}
                  onChange={(e) => setCPhone(e.currentTarget.value)} />
              </Group>
              <Textarea label={t('customer.notes_label')} value={cNotes}
                onChange={(e) => setCNotes(e.currentTarget.value)} minRows={2} />
              <Group justify="end" mt="md">
                <Button variant="default" onClick={() => setCustomerModalOpen(false)}>{t('common.cancel')}</Button>
                <Button onClick={handleSaveCustomer} loading={createCustomerMutation.isPending}>{t('common.create')}</Button>
              </Group>
            </Stack>
          </Modal>

          {/* Add VAT rate modal */}
          <Modal opened={vatRateModalOpen} onClose={() => setVatRateModalOpen(false)}
            title={t('invoice.new_vat_rate')} size="xs">
            <Stack gap="md">
              <TextInput label={t('invoice.new_vat_rate_label')} placeholder="15"
                value={newVatRateValue} onChange={(e) => setNewVatRateValue(e.currentTarget.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') handleAddVatRate() }}
              />
              <Group justify="end">
                <Button variant="default" onClick={() => setVatRateModalOpen(false)}>{t('common.cancel')}</Button>
                <Button onClick={handleAddVatRate} disabled={!newVatRateValue.trim()}>{t('common.save')}</Button>
              </Group>
            </Stack>
          </Modal>

          {/* Add unit modal */}
          <Modal opened={unitModalOpen} onClose={() => setUnitModalOpen(false)}
            title={t('invoice.new_unit')} size="xs">
            <Stack gap="md">
              <TextInput label={t('invoice.new_unit_label')} placeholder="bal"
                value={newUnitValue} onChange={(e) => setNewUnitValue(e.currentTarget.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') handleAddUnit() }}
              />
              <Group justify="end">
                <Button variant="default" onClick={() => setUnitModalOpen(false)}>{t('common.cancel')}</Button>
                <Button onClick={handleAddUnit} disabled={!newUnitValue.trim()}>{t('common.save')}</Button>
              </Group>
            </Stack>
          </Modal>

          {/* Bank account creation modal */}
          <Modal opened={bankModalOpen} onClose={() => setBankModalOpen(false)}
            title={t('bank_account.new_title')} size="md">
            <Stack gap="md">
              <TextInput label={t('bank_account.name_label')} value={bName}
                onChange={(e) => setBName(e.currentTarget.value)} />
              <TextInput label={t('bank_account.account_number_label')} value={bAccountNumber}
                onChange={(e) => setBAccountNumber(e.currentTarget.value)} required />
              <TextInput label={t('bank_account.iban_label')} value={bIban}
                onChange={(e) => setBIban(e.currentTarget.value)} />
              <Group grow>
                <TextInput label={t('bank_account.swift_label')} value={bSwift}
                  onChange={(e) => setBSwift(e.currentTarget.value)} />
                <Select label={t('bank_account.currency_label')} data={currencyData}
                  value={bCurrency} onChange={(v) => handleCurrencySelect(v, 'bank')} searchable />
              </Group>
              <Group justify="end" mt="md">
                <Button variant="default" onClick={() => setBankModalOpen(false)}>{t('common.cancel')}</Button>
                <Button onClick={handleSaveBank} loading={createBankMutation.isPending}>{t('common.create')}</Button>
              </Group>
            </Stack>
          </Modal>

          {/* Add currency modal */}
          <Modal opened={currencyModalOpen} onClose={() => setCurrencyModalOpen(false)}
            title={t('invoice.add_currency')} size="xs">
            <Stack gap="md">
              <TextInput label={t('invoice.currency_code')} placeholder="BTC"
                value={newCurrencyCode} onChange={(e) => setNewCurrencyCode(e.currentTarget.value.toUpperCase())}
                maxLength={10}
                onKeyDown={(e) => { if (e.key === 'Enter') handleAddCurrency() }}
              />
              <Group justify="end">
                <Button variant="default" onClick={() => setCurrencyModalOpen(false)}>{t('common.cancel')}</Button>
                <Button onClick={handleAddCurrency} disabled={!newCurrencyCode.trim()}>{t('common.save')}</Button>
              </Group>
            </Stack>
          </Modal>
        </>
      )}
    </Stack>
  )
}
