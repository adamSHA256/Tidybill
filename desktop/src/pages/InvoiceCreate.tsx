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
  Switch,
  Textarea,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { IconTrash, IconPlus, IconPackage } from '@tabler/icons-react'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, formatMoney, type Supplier, type BankAccount, type Item, type CustomerItem } from '../api/client'
import { useT } from '../i18n'

const CREATE_NEW = '__create_new__'

interface ItemForm {
  item_id: string
  description: string
  quantity: number
  unit: string
  unit_price: number
  vat_rate: number
}

const emptyItem: ItemForm = { item_id: '', description: '', quantity: 1, unit: 'ks', unit_price: 0, vat_rate: 21 }

export function InvoiceCreate() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { t } = useT()

  const [supplierId, setSupplierId] = useState<string | null>(null)
  const [customerId, setCustomerId] = useState<string | null>(null)
  const [bankAccountId, setBankAccountId] = useState<string | null>(null)
  const [issueDate, setIssueDate] = useState(new Date().toISOString().slice(0, 10))
  const [dueDate, setDueDate] = useState(
    new Date(Date.now() + 14 * 86400000).toISOString().slice(0, 10)
  )
  const [currency, setCurrency] = useState('CZK')
  const [invoiceNumber, setInvoiceNumber] = useState('')
  const [notes, setNotes] = useState('')
  const [items, setItems] = useState<ItemForm[]>([{ ...emptyItem, unit: 'hod' }])

  // Modal states
  const [supplierModalOpen, setSupplierModalOpen] = useState(false)
  const [customerModalOpen, setCustomerModalOpen] = useState(false)
  const [bankModalOpen, setBankModalOpen] = useState(false)

  // Supplier form state
  const [sName, setSName] = useState('')
  const [sIco, setSIco] = useState('')
  const [sDic, setSDic] = useState('')
  const [sStreet, setSStreet] = useState('')
  const [sCity, setSCity] = useState('')
  const [sZip, setSZip] = useState('')
  const [sCountry, setSCountry] = useState('CZ')
  const [sEmail, setSEmail] = useState('')
  const [sPhone, setSPhone] = useState('')
  const [sWebsite, setSWebsite] = useState('')
  const [sInvoicePrefix, setSInvoicePrefix] = useState('')
  const [sIsVatPayer, setSIsVatPayer] = useState(false)
  const [sNotes, setSNotes] = useState('')

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

  const { data: suppliers, isLoading: suppliersLoading } = useQuery({
    queryKey: ['suppliers'],
    queryFn: api.getSuppliers,
  })

  const { data: customers, isLoading: customersLoading } = useQuery({
    queryKey: ['customers'],
    queryFn: () => api.getCustomers(),
  })

  // Auto-select default supplier
  const defaultSupplier = suppliers?.find((s: Supplier) => s.is_default)
  const selectedSupplierId = supplierId || defaultSupplier?.id || null

  const { data: bankAccounts } = useQuery({
    queryKey: ['bank-accounts', selectedSupplierId],
    queryFn: () => api.getBankAccounts(selectedSupplierId!),
    enabled: !!selectedSupplierId,
  })

  // Catalog queries
  const { data: customerItems } = useQuery({
    queryKey: ['customer-items', customerId],
    queryFn: () => api.getCustomerItems(customerId!),
    enabled: !!customerId,
  })

  const { data: mostUsedItems } = useQuery({
    queryKey: ['items-most-used'],
    queryFn: api.getMostUsedItems,
  })

  // Auto-select default bank account
  const defaultBank = bankAccounts?.find((b: BankAccount) => b.is_default)
  const selectedBankId = bankAccountId || defaultBank?.id || null

  const selectedSupplier = suppliers?.find((s) => s.id === selectedSupplierId)
  const selectedCustomer = customers?.find((c) => c.id === customerId)

  const createMutation = useMutation({
    mutationFn: api.createInvoice,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['invoices'] })
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] })
      queryClient.invalidateQueries({ queryKey: ['items'] })
      notifications.show({ title: t('notify.invoice_created'), message: t('notify.invoice_created_msg'), color: 'green' })
      navigate('/invoices')
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  // Inline creation mutations
  const createSupplierMutation = useMutation({
    mutationFn: (data: Partial<Supplier>) => api.createSupplier(data),
    onSuccess: (newSupplier) => {
      queryClient.invalidateQueries({ queryKey: ['suppliers'] })
      setSupplierId(newSupplier.id)
      setBankAccountId(null)
      setSupplierModalOpen(false)
      notifications.show({ title: t('notify.supplier_created'), message: t('notify.supplier_saved_msg').replace('{name}', newSupplier.name), color: 'green' })
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
    mutationFn: (data: Partial<BankAccount>) => api.createBankAccount(selectedSupplierId!, data),
    onSuccess: (newBank) => {
      queryClient.invalidateQueries({ queryKey: ['bank-accounts', selectedSupplierId] })
      setBankAccountId(newBank.id)
      setBankModalOpen(false)
      notifications.show({ title: t('notify.bank_account_created'), message: t('notify.bank_account_created_msg'), color: 'green' })
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

  // Modal helpers
  const openSupplierModal = () => {
    setSName(''); setSIco(''); setSDic(''); setSStreet(''); setSCity(''); setSZip('')
    setSCountry('CZ'); setSEmail(''); setSPhone(''); setSWebsite(''); setSInvoicePrefix('')
    setSIsVatPayer(false); setSNotes('')
    setSupplierModalOpen(true)
  }

  const openCustomerModal = () => {
    setCName(''); setCIco(''); setCDic(''); setCStreet(''); setCCity(''); setCZip('')
    setCCountry('CZ'); setCEmail(''); setCPhone(''); setCNotes('')
    setCustomerModalOpen(true)
  }

  const openBankModal = () => {
    setBName(''); setBAccountNumber(''); setBIban(''); setBSwift(''); setBCurrency('CZK')
    setBankModalOpen(true)
  }

  const handleSaveSupplier = () => {
    if (!sName.trim()) {
      notifications.show({ title: t('supplier.missing_name_title'), message: t('supplier.missing_name_msg'), color: 'orange' })
      return
    }
    createSupplierMutation.mutate({
      name: sName, ico: sIco, dic: sDic, street: sStreet, city: sCity, zip: sZip,
      country: sCountry, email: sEmail, phone: sPhone, website: sWebsite,
      invoice_prefix: sInvoicePrefix, is_vat_payer: sIsVatPayer, notes: sNotes,
    })
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

  const addItem = () => setItems([...items, { ...emptyItem }])
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
    // Replace empty first row or append
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

  const handleCreate = () => {
    if (!selectedSupplierId || !customerId || !selectedBankId) {
      notifications.show({ title: t('invoice.missing_fields_title'), message: t('invoice.missing_fields_msg'), color: 'orange' })
      return
    }
    if (items.every((i) => !i.description)) {
      notifications.show({ title: t('invoice.missing_items_title'), message: t('invoice.missing_items_msg'), color: 'orange' })
      return
    }
    createMutation.mutate({
      supplier_id: selectedSupplierId,
      customer_id: customerId,
      bank_account_id: selectedBankId,
      invoice_number: invoiceNumber || undefined,
      issue_date: issueDate,
      due_date: dueDate,
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

  if (suppliersLoading || customersLoading) {
    return <Center h={300}><Loader /></Center>
  }

  // Build select data with "Create new" sentinel
  const supplierData = [
    ...(suppliers || []).map((s) => ({ value: s.id, label: s.name })),
    { value: CREATE_NEW, label: `+ ${t('invoice.create_new_supplier')}` },
  ]

  const customerData = [
    ...(customers || []).map((c) => ({ value: c.id, label: c.name })),
    { value: CREATE_NEW, label: `+ ${t('invoice.create_new_customer')}` },
  ]

  const bankData = [
    ...(bankAccounts || []).map((b) => ({ value: b.id, label: `${b.account_number} (${b.currency})` })),
    { value: CREATE_NEW, label: `+ ${t('invoice.create_new_bank_account')}` },
  ]

  // Build catalog suggestions: customer items first, then global most-used (deduplicated)
  const customerItemIds = new Set((customerItems || []).map((ci: CustomerItem) => ci.item_id))
  const globalSuggestions = (mostUsedItems || []).filter((item: Item) => !customerItemIds.has(item.id))

  const handleSupplierSelect = (v: string | null) => {
    if (v === CREATE_NEW) {
      openSupplierModal()
      return
    }
    setSupplierId(v)
    setBankAccountId(null)
  }

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
      <Group justify="space-between">
        <div>
          <Title order={2}>{t('invoice.create_title')}</Title>
          <Text c="dimmed" size="sm">{t('invoice.create_subtitle')}</Text>
        </div>
        <Group>
          <Button variant="default" onClick={() => navigate('/invoices')}>{t('common.cancel')}</Button>
          <Button onClick={handleCreate} loading={createMutation.isPending}>{t('common.create')}</Button>
        </Group>
      </Group>

      <Paper p="md" radius="md" withBorder>
        <Text fw={500} mb="md">{t('invoice.details')}</Text>
        <SimpleGrid cols={{ base: 1, sm: 2, md: 4 }}>
          <TextInput label={t('invoice.invoice_number')} placeholder={t('invoice.invoice_number_placeholder')} value={invoiceNumber} onChange={(e) => setInvoiceNumber(e.currentTarget.value)} description={t('invoice.invoice_number_desc')} />
          <TextInput label={t('invoice.issue_date')} type="date" value={issueDate} onChange={(e) => setIssueDate(e.currentTarget.value)} />
          <TextInput label={t('invoice.due_date')} type="date" value={dueDate} onChange={(e) => setDueDate(e.currentTarget.value)} />
          <Select label={t('invoice.currency')} data={['CZK', 'EUR', 'USD']} value={currency} onChange={(v) => setCurrency(v || 'CZK')} />
        </SimpleGrid>
      </Paper>

      <SimpleGrid cols={{ base: 1, md: 2 }}>
        <Paper p="md" radius="md" withBorder>
          <Text fw={500} mb="md">{t('invoice.supplier_you')}</Text>
          {suppliers && suppliers.length === 0 ? (
            <Paper
              p="xl"
              radius="md"
              withBorder
              style={{ borderStyle: 'dashed', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: 100 }}
              onClick={openSupplierModal}
            >
              <Stack align="center" gap="xs">
                <IconPlus size={24} color="gray" />
                <Text c="dimmed" size="sm">{t('invoice.no_suppliers_create')}</Text>
              </Stack>
            </Paper>
          ) : (
            <Select
              label={t('invoice.select_supplier')}
              placeholder={t('invoice.select_supplier_placeholder')}
              data={supplierData}
              value={selectedSupplierId}
              onChange={handleSupplierSelect}
            />
          )}
          {selectedSupplier && (
            <Stack gap={4} mt="sm">
              <Text size="sm" c="dimmed">
                ICO: {selectedSupplier.ico}{selectedSupplier.dic && ` | DIC: ${selectedSupplier.dic}`}
              </Text>
              <Text size="sm" c="dimmed">{selectedSupplier.street}, {selectedSupplier.city}, {selectedSupplier.zip}</Text>
            </Stack>
          )}
          {selectedSupplierId && bankAccounts && bankAccounts.length > 0 ? (
            <Select
              label={t('invoice.bank_account')}
              mt="sm"
              data={bankData}
              value={selectedBankId}
              onChange={handleBankSelect}
            />
          ) : selectedSupplierId && bankAccounts && bankAccounts.length === 0 ? (
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
                  <Select size="sm" data={['ks', 'hod', 'den', 'm\u00B2']} value={item.unit}
                    onChange={(val) => updateItem(i, 'unit', val || 'ks')} />
                </Table.Td>
                <Table.Td>
                  <NumberInput size="sm" min={0} value={item.unit_price}
                    onChange={(val) => updateItem(i, 'unit_price', val || 0)} />
                </Table.Td>
                <Table.Td>
                  <Select size="sm" data={['0', '12', '21']} value={String(item.vat_rate)}
                    onChange={(val) => updateItem(i, 'vat_rate', Number(val))} />
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
        <Button variant="default" onClick={() => navigate('/invoices')}>{t('common.cancel')}</Button>
        <Button onClick={handleCreate} loading={createMutation.isPending}>{t('invoice.create_button')}</Button>
      </Group>

      {/* Supplier creation modal */}
      <Modal opened={supplierModalOpen} onClose={() => setSupplierModalOpen(false)}
        title={t('supplier.new_title')} size="lg">
        <Stack gap="md">
          <TextInput label={t('supplier.name_label')} value={sName}
            onChange={(e) => setSName(e.currentTarget.value)} required />
          <Group grow>
            <TextInput label={t('supplier.ico_label')} value={sIco}
              onChange={(e) => setSIco(e.currentTarget.value)} />
            <TextInput label={t('supplier.dic_label')} value={sDic}
              onChange={(e) => setSDic(e.currentTarget.value)} />
          </Group>
          <TextInput label={t('supplier.street_label')} value={sStreet}
            onChange={(e) => setSStreet(e.currentTarget.value)} />
          <Group grow>
            <TextInput label={t('supplier.city_label')} value={sCity}
              onChange={(e) => setSCity(e.currentTarget.value)} />
            <TextInput label={t('supplier.zip_label')} value={sZip}
              onChange={(e) => setSZip(e.currentTarget.value)} />
          </Group>
          <Select label={t('supplier.country_label')} data={['CZ', 'SK', 'DE', 'AT', 'PL', 'HU']}
            value={sCountry} onChange={(v) => setSCountry(v || 'CZ')} />
          <Group grow>
            <TextInput label={t('supplier.email_label')} value={sEmail}
              onChange={(e) => setSEmail(e.currentTarget.value)} />
            <TextInput label={t('supplier.phone_label')} value={sPhone}
              onChange={(e) => setSPhone(e.currentTarget.value)} />
          </Group>
          <Group grow>
            <TextInput label={t('supplier.website_label')} value={sWebsite}
              onChange={(e) => setSWebsite(e.currentTarget.value)} />
            <TextInput label={t('supplier.invoice_prefix_label')} value={sInvoicePrefix}
              onChange={(e) => setSInvoicePrefix(e.currentTarget.value)} />
          </Group>
          <Switch label={t('supplier.is_vat_payer_label')} checked={sIsVatPayer}
            onChange={(e) => setSIsVatPayer(e.currentTarget.checked)} />
          <Textarea label={t('supplier.notes_label')} value={sNotes}
            onChange={(e) => setSNotes(e.currentTarget.value)} minRows={2} />
          <Group justify="end" mt="md">
            <Button variant="default" onClick={() => setSupplierModalOpen(false)}>{t('common.cancel')}</Button>
            <Button onClick={handleSaveSupplier} loading={createSupplierMutation.isPending}>{t('common.create')}</Button>
          </Group>
        </Stack>
      </Modal>

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
          <Select label={t('customer.country_label')} data={['CZ', 'SK', 'DE', 'AT', 'PL', 'HU']}
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
            <Select label={t('bank_account.currency_label')} data={['CZK', 'EUR', 'USD']}
              value={bCurrency} onChange={(v) => setBCurrency(v || 'CZK')} />
          </Group>
          <Group justify="end" mt="md">
            <Button variant="default" onClick={() => setBankModalOpen(false)}>{t('common.cancel')}</Button>
            <Button onClick={handleSaveBank} loading={createBankMutation.isPending}>{t('common.create')}</Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  )
}
