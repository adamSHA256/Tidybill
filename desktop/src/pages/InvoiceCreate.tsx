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
  Alert,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { DateInput } from '@mantine/dates'
import { IconTrash, IconPlus, IconPackage, IconAlertTriangle } from '@tabler/icons-react'
import { useState, useEffect, useRef } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, formatMoney, type Supplier, type BankAccount, type Item, type CustomerItem } from '../api/client'
import { CountrySelect } from '../components/CountrySelect'
import { useT } from '../i18n'

const CREATE_NEW = '__create_new__'
const ADD_CURRENCY = '__add_currency__'
const ADD_VAT_RATE = '__add_vat_rate__'
const ADD_UNIT = '__add_unit__'
const ADD_PAYMENT_TYPE = '__add_payment_type__'

interface ItemForm {
  item_id: string
  description: string
  quantity: number
  unit: string
  unit_price: number
  vat_rate: number
}

const emptyItem: ItemForm = { item_id: '', description: '', quantity: 1, unit: 'ks', unit_price: 0, vat_rate: 21 }

function generateVariableSymbol(invoiceNumber: string): string {
  return invoiceNumber.replace(/-/g, '').replace(/^[A-Z]+/, '')
}

interface DuplicateFromState {
  invoice_number: string
  supplier_id: string
  customer_id: string
  bank_account_id: string
  currency: string
  payment_method: string
  notes: string
  items: ItemForm[]
}

export function InvoiceCreate() {
  const navigate = useNavigate()
  const location = useLocation()
  const queryClient = useQueryClient()
  const { t } = useT()

  const duplicateFrom = (location.state as { duplicateFrom?: DuplicateFromState } | null)?.duplicateFrom || null

  const { data: settings } = useQuery({
    queryKey: ['settings'],
    queryFn: api.getSettings,
  })

  const { data: dueDaysOptions } = useQuery({
    queryKey: ['due-days'],
    queryFn: api.getDueDaysOptions,
  })
  const globalDueDays = (dueDaysOptions || []).find((d) => d.is_default)?.days || dueDaysOptions?.[0]?.days || 14
  const globalCurrency = settings?.default_currency || 'CZK'

  const { data: paymentTypes } = useQuery({
    queryKey: ['payment-types'],
    queryFn: api.getPaymentTypes,
  })

  const { data: currenciesList } = useQuery({
    queryKey: ['currencies'],
    queryFn: api.getCurrencies,
  })

  const [supplierId, setSupplierId] = useState<string | null>(duplicateFrom?.supplier_id || null)
  const [customerId, setCustomerId] = useState<string | null>(duplicateFrom?.customer_id || null)
  const [bankAccountId, setBankAccountId] = useState<string | null>(duplicateFrom?.bank_account_id || null)
  const [issueDate, setIssueDate] = useState<string | null>(new Date().toISOString().slice(0, 10))
  const [taxableDate, setTaxableDate] = useState<string | null>(null)
  const [dueDate, setDueDate] = useState<string | null>(null)
  const [dueDateInitialized, setDueDateInitialized] = useState(false)
  const [currency, setCurrency] = useState(duplicateFrom?.currency || '')
  const [currencyInitialized, setCurrencyInitialized] = useState(!!duplicateFrom)
  const [invoiceNumber, setInvoiceNumber] = useState('')
  const [paymentMethod, setPaymentMethod] = useState<string | null>(duplicateFrom?.payment_method || null)
  const [notes, setNotes] = useState(duplicateFrom?.notes || '')
  const [items, setItems] = useState<ItemForm[]>(
    duplicateFrom?.items?.length ? duplicateFrom.items : [{ ...emptyItem }]
  )
  const [dueDateChangedByCustomer, setDueDateChangedByCustomer] = useState(false)
  const dueDateTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const [variableSymbol, setVariableSymbol] = useState('')
  const [vsChangedByInvoiceNumber, setVsChangedByInvoiceNumber] = useState(false)
  const vsTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Modal states
  const [supplierModalOpen, setSupplierModalOpen] = useState(false)
  const [customerModalOpen, setCustomerModalOpen] = useState(false)
  const [bankModalOpen, setBankModalOpen] = useState(false)
  const [currencyModalOpen, setCurrencyModalOpen] = useState(false)
  const [newCurrencyCode, setNewCurrencyCode] = useState('')
  const [currencyTarget, setCurrencyTarget] = useState<'invoice' | 'bank'>('invoice')
  const [vatRateModalOpen, setVatRateModalOpen] = useState(false)
  const [newVatRateValue, setNewVatRateValue] = useState('')
  const [vatRateTargetIndex, setVatRateTargetIndex] = useState<number>(-1)
  const [unitModalOpen, setUnitModalOpen] = useState(false)
  const [newUnitValue, setNewUnitValue] = useState('')
  const [unitTargetIndex, setUnitTargetIndex] = useState<number>(-1)
  const [paymentTypeModalOpen, setPaymentTypeModalOpen] = useState(false)
  const [newPaymentTypeName, setNewPaymentTypeName] = useState('')

  // Supplier form state
  const [sName, setSName] = useState('')
  const [sIco, setSIco] = useState('')
  const [sDic, setSDic] = useState('')
  const [sIcDph, setSIcDph] = useState('')
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
  const [cIcDph, setCIcDph] = useState('')
  const [cStreet, setCStreet] = useState('')
  const [cCity, setCCity] = useState('')
  const [cZip, setCZip] = useState('')
  const [cCountry, setCCountry] = useState('CZ')
  const [cEmail, setCEmail] = useState('')
  const [cPhone, setCPhone] = useState('')
  const [cDueDays, setCDueDays] = useState<number>(0)
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

  const { data: units } = useQuery({
    queryKey: ['units'],
    queryFn: api.getUnits,
  })
  const unitOptions = (units || []).map((u) => u.name)
  const defaultUnit = (units || []).find((u) => u.is_default)?.name || unitOptions[0] || 'ks'

  const { data: vatRates } = useQuery({
    queryKey: ['vat-rates'],
    queryFn: api.getVATRates,
  })
  const vatRateOptions = (vatRates || []).map((r) => String(r.rate))
  const defaultVatRate = (vatRates || []).find((r) => r.is_default)?.rate ?? 21

  // Auto-select default supplier
  const defaultSupplier = suppliers?.find((s: Supplier) => s.is_default)
  const selectedSupplierId = supplierId || defaultSupplier?.id || null

  const { data: bankAccounts } = useQuery({
    queryKey: ['bank-accounts', selectedSupplierId],
    queryFn: () => api.getBankAccounts(selectedSupplierId!),
    enabled: !!selectedSupplierId,
  })

  // Auto-fetch next invoice number when supplier is selected
  const { data: nextNumberData } = useQuery({
    queryKey: ['next-invoice-number', selectedSupplierId],
    queryFn: () => api.getNextInvoiceNumber(selectedSupplierId!),
    enabled: !!selectedSupplierId,
  })

  // Pre-fill invoice number when supplier changes
  useEffect(() => {
    if (nextNumberData?.invoice_number) {
      setInvoiceNumber(nextNumberData.invoice_number)
    }
  }, [nextNumberData])

  // Auto-generate variable symbol when invoice number changes
  useEffect(() => {
    if (invoiceNumber) {
      const newVs = generateVariableSymbol(invoiceNumber)
      setVariableSymbol((prev) => {
        if (prev && prev !== newVs) {
          setVsChangedByInvoiceNumber(true)
          if (vsTimerRef.current) clearTimeout(vsTimerRef.current)
          vsTimerRef.current = setTimeout(() => setVsChangedByInvoiceNumber(false), 10000)
        }
        return newVs
      })
    }
  }, [invoiceNumber])

  // Initialize default VAT rate from VAT rates array for the initial empty item
  useEffect(() => {
    if (vatRates && vatRates.length > 0) {
      setItems((prev) => prev.length === 1 && !prev[0].description ? [{ ...prev[0], vat_rate: defaultVatRate }] : prev)
    }
  }, [vatRates]) // eslint-disable-line react-hooks/exhaustive-deps

  // Initialize default unit from settings for the initial empty item
  useEffect(() => {
    if (units && units.length > 0) {
      setItems((prev) => prev.length === 1 && !prev[0].description ? [{ ...prev[0], unit: defaultUnit }] : prev)
    }
  }, [units]) // eslint-disable-line react-hooks/exhaustive-deps

  // Initialize currency from settings (will be overridden by bank account below)
  useEffect(() => {
    if (!currencyInitialized && settings) {
      setCurrency(globalCurrency)
      setCurrencyInitialized(true)
    }
  }, [settings, currencyInitialized, globalCurrency])

  // Initialize due date from due days options (or customer default)
  useEffect(() => {
    if (!dueDateInitialized && dueDaysOptions) {
      // When duplicating, wait for customers to load so we can use customer's due days
      if (duplicateFrom?.customer_id && !customers) return
      const cust = duplicateFrom?.customer_id ? customers?.find((c) => c.id === duplicateFrom.customer_id) : null
      const days = (cust && cust.default_due_days > 0) ? cust.default_due_days : globalDueDays
      setDueDate(new Date(Date.now() + days * 86400000).toISOString().slice(0, 10))
      setDueDateInitialized(true)
    }
  }, [dueDaysOptions, dueDateInitialized, globalDueDays, customers]) // eslint-disable-line react-hooks/exhaustive-deps

  // Initialize payment method from default payment type
  useEffect(() => {
    if (paymentTypes && !paymentMethod) {
      const def = paymentTypes.find((pt) => pt.is_default)
      if (def) setPaymentMethod(def.name)
      else if (paymentTypes.length > 0) setPaymentMethod(paymentTypes[0].name)
    }
  }, [paymentTypes]) // eslint-disable-line react-hooks/exhaustive-deps

  // Cleanup timers on unmount
  useEffect(() => {
    return () => {
      if (dueDateTimerRef.current) clearTimeout(dueDateTimerRef.current)
      if (vsTimerRef.current) clearTimeout(vsTimerRef.current)
    }
  }, [])

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

  // Compute whether selected payment method requires bank info
  const selectedPaymentType = paymentTypes?.find((pt) => pt.name === paymentMethod)
  const requiresBankInfo = !selectedPaymentType || selectedPaymentType.requires_bank_info !== false

  // Auto-set currency from supplier's default bank account (regardless of payment method)
  useEffect(() => {
    if (defaultBank && !bankAccountId) {
      setCurrency(defaultBank.currency)
    }
  }, [defaultBank?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  // When payment method changes to non-bank, clear bank selection but keep currency from supplier
  useEffect(() => {
    if (!requiresBankInfo) {
      setBankAccountId(null)
      // Currency stays from supplier's default bank; fall back to global only if no bank exists
      if (!defaultBank) {
        setCurrency(globalCurrency)
      }
    }
  }, [requiresBankInfo]) // eslint-disable-line react-hooks/exhaustive-deps

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
    ...(unitOptions.length > 0 ? unitOptions : ['ks', 'hod', 'den', 'm\u00B2']).map((u) => {
      const key = 'unit.' + u; const translated = t(key); return { value: u, label: translated !== key ? translated : u }
    }),
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
    if (target === 'invoice') setCurrency(v || globalCurrency)
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

  // Build payment type select data with "Add new" option
  const paymentTypeSelectData = [
    ...(paymentTypes || []).map((pt) => ({ value: pt.name, label: pt.name })),
    { value: ADD_PAYMENT_TYPE, label: `+ ${t('settings.payment_type_placeholder')}` },
  ]

  const handlePaymentTypeSelect = (val: string | null) => {
    if (val === ADD_PAYMENT_TYPE) {
      setNewPaymentTypeName('')
      setPaymentTypeModalOpen(true)
      return
    }
    setPaymentMethod(val)
  }

  const handleAddPaymentType = () => {
    const name = newPaymentTypeName.trim()
    if (!name) return
    const current = paymentTypes || []
    if (!current.some((pt) => pt.name === name)) {
      api.updatePaymentTypes([...current, { name }]).then(() => {
        queryClient.invalidateQueries({ queryKey: ['payment-types'] })
      })
    }
    setPaymentMethod(name)
    setPaymentTypeModalOpen(false)
  }

  // Modal helpers
  const openSupplierModal = () => {
    setSName(''); setSIco(''); setSDic(''); setSIcDph(''); setSStreet(''); setSCity(''); setSZip('')
    setSCountry('CZ'); setSEmail(''); setSPhone(''); setSWebsite(''); setSInvoicePrefix('')
    setSIsVatPayer(false); setSNotes('')
    setSupplierModalOpen(true)
  }

  const openCustomerModal = () => {
    setCName(''); setCIco(''); setCDic(''); setCIcDph(''); setCStreet(''); setCCity(''); setCZip('')
    setCCountry('CZ'); setCEmail(''); setCPhone(''); setCDueDays(0); setCNotes('')
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
      name: sName, ico: sIco, dic: sDic, ic_dph: sIcDph, street: sStreet, city: sCity, zip: sZip,
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
      name: cName, ico: cIco, dic: cDic, ic_dph: cIcDph, street: cStreet, city: cCity, zip: cZip,
      country: cCountry, email: cEmail, phone: cPhone, default_due_days: cDueDays, notes: cNotes,
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

  const addItem = () => setItems([...items, { ...emptyItem, unit: defaultUnit, vat_rate: defaultVatRate }])
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
      unit: catalogItem.default_unit || defaultUnit,
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
    if (!selectedSupplierId || !customerId || (requiresBankInfo && !selectedBankId)) {
      notifications.show({ title: t('invoice.missing_fields_title'), message: requiresBankInfo ? t('invoice.missing_fields_msg') : t('invoice.missing_fields_msg_no_bank'), color: 'orange' })
      return
    }
    if (items.every((i) => !i.description)) {
      notifications.show({ title: t('invoice.missing_items_title'), message: t('invoice.missing_items_msg'), color: 'orange' })
      return
    }
    createMutation.mutate({
      supplier_id: selectedSupplierId,
      customer_id: customerId,
      bank_account_id: requiresBankInfo ? selectedBankId! : undefined,
      invoice_number: invoiceNumber || undefined,
      issue_date: issueDate || undefined,
      taxable_date: taxableDate ?? issueDate ?? undefined,
      due_date: dueDate || undefined,
      payment_method: paymentMethod || undefined,
      currency,
      notes,
      variable_symbol: requiresBankInfo ? variableSymbol : undefined,
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
    // Recalculate due date from customer's default_due_days or global setting
    const cust = customers?.find((c) => c.id === v)
    const days = (cust && cust.default_due_days > 0) ? cust.default_due_days : globalDueDays
    setDueDate(new Date(Date.now() + days * 86400000).toISOString().slice(0, 10))

    // Show indicator if customer has custom due days different from global
    if (cust && cust.default_due_days > 0 && cust.default_due_days !== globalDueDays) {
      setDueDateChangedByCustomer(true)
      if (dueDateTimerRef.current) clearTimeout(dueDateTimerRef.current)
      dueDateTimerRef.current = setTimeout(() => setDueDateChangedByCustomer(false), 10000)
    } else {
      setDueDateChangedByCustomer(false)
    }
  }

  const handleBankSelect = (v: string | null) => {
    if (v === CREATE_NEW) {
      openBankModal()
      return
    }
    setBankAccountId(v)
    // Auto-set currency from selected bank account
    const acc = bankAccounts?.find((b) => b.id === v)
    if (acc) {
      setCurrency(acc.currency)
    }
  }

  // Currency mismatch detection
  const selectedBank = bankAccounts?.find((b) => b.id === selectedBankId)
  const currencyMismatch = selectedBank && currency && selectedBank.currency !== currency

  return (
    <Stack gap="lg">
      <Group justify="space-between">
        <div>
          <Title order={2}>{t('invoice.create_title')}</Title>
          <Text c="dimmed" size="sm">
            {duplicateFrom
              ? t('invoice.duplicate_subtitle').replace('{number}', duplicateFrom.invoice_number)
              : t('invoice.create_subtitle')}
          </Text>
        </div>
        <Group>
          <Button variant="default" onClick={() => navigate('/invoices')}>{t('common.cancel')}</Button>
          <Button onClick={handleCreate} loading={createMutation.isPending}>{t('common.create')}</Button>
        </Group>
      </Group>

      <Paper p="md" radius="md" withBorder>
        <SimpleGrid cols={{ base: 1, sm: 2, md: 3 }}>
          <TextInput label={t('invoice.invoice_number')} placeholder={t('invoice.invoice_number_placeholder')} value={invoiceNumber} onChange={(e) => setInvoiceNumber(e.currentTarget.value)} />
          <DateInput label={t('invoice.issue_date')} valueFormat="DD.MM.YYYY" value={issueDate} onChange={setIssueDate} clearable />
          <DateInput label={t('invoice.taxable_date')} valueFormat="DD.MM.YYYY" value={taxableDate ?? issueDate} onChange={setTaxableDate} clearable />
          <Select label={t('invoice.payment_method')} data={paymentTypeSelectData} value={paymentMethod} onChange={handlePaymentTypeSelect} searchable />
          <Select label={t('invoice.currency')} data={currencyData} value={currency} onChange={(v) => handleCurrencySelect(v, 'invoice')} searchable />
          <div>
            <DateInput
              label={t('invoice.due_date')}
              valueFormat="DD.MM.YYYY"
              value={dueDate}
              onChange={setDueDate}
              clearable
              styles={dueDateChangedByCustomer ? { input: { borderColor: 'var(--mantine-primary-color-6)', borderWidth: 2 } } : undefined}
            />
            {dueDateChangedByCustomer && (
              <Text size="xs" c="var(--mantine-primary-color-7)" mt={4}>{t('invoice.due_date_changed_by_customer')}</Text>
            )}
          </div>
          {requiresBankInfo && (
            <div>
              <TextInput
                label={t('invoice.variable_symbol_label')}
                value={variableSymbol}
                onChange={(e) => setVariableSymbol(e.currentTarget.value)}
                styles={vsChangedByInvoiceNumber ? { input: { borderColor: 'var(--mantine-primary-color-6)', borderWidth: 2 } } : undefined}
              />
              {vsChangedByInvoiceNumber && (
                <Text size="xs" c="var(--mantine-primary-color-7)" mt={4}>{t('invoice.vs_changed_by_invoice_number')}</Text>
              )}
            </div>
          )}
        </SimpleGrid>
        {currencyMismatch && (
          <Alert variant="light" color="orange" mt="sm" icon={<IconAlertTriangle size={16} />}>
            {t('invoice.currency_mismatch').replace('{bank}', selectedBank!.currency).replace('{invoice}', currency)}
          </Alert>
        )}
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
                ICO: {selectedSupplier.ico}{selectedSupplier.dic && ` | DIC: ${selectedSupplier.dic}`}{selectedSupplier.ic_dph && ` | IC DPH: ${selectedSupplier.ic_dph}`}
              </Text>
              <Text size="sm" c="dimmed">{selectedSupplier.street}, {selectedSupplier.city}, {selectedSupplier.zip}</Text>
            </Stack>
          )}
          {requiresBankInfo && (
            <>
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
            </>
          )}
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
                ICO: {selectedCustomer.ico}{selectedCustomer.dic && ` | DIC: ${selectedCustomer.dic}`}{selectedCustomer.ic_dph && ` | IC DPH: ${selectedCustomer.ic_dph}`}
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
                  <Text size="sm" fw={600}>{formatMoney(item.quantity * item.unit_price, currency)}</Text>
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
            <Text size="sm" fw={600} w={120} ta="right">{formatMoney(subtotal, currency)}</Text>
          </Group>
          <Group>
            <Text size="sm" c="dimmed" w={180} ta="right">{t('invoice.vat')}</Text>
            <Text size="sm" fw={600} w={120} ta="right">{formatMoney(vatAmount, currency)}</Text>
          </Group>
          <Divider w={300} />
          <Group>
            <Text size="lg" fw={700} w={180} ta="right">{t('invoice.total')}</Text>
            <Text size="lg" fw={700} w={120} ta="right">{formatMoney(total, currency)}</Text>
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
          {sCountry.toUpperCase() === 'SK' && (
            <TextInput label={t('supplier.ic_dph_label')} value={sIcDph}
              onChange={(e) => setSIcDph(e.currentTarget.value)} />
          )}
          <TextInput label={t('supplier.street_label')} value={sStreet}
            onChange={(e) => setSStreet(e.currentTarget.value)} />
          <Group grow>
            <TextInput label={t('supplier.city_label')} value={sCity}
              onChange={(e) => setSCity(e.currentTarget.value)} />
            <TextInput label={t('supplier.zip_label')} value={sZip}
              onChange={(e) => setSZip(e.currentTarget.value)} />
          </Group>
          <CountrySelect label={t('supplier.country_label')}
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
          {cCountry.toUpperCase() === 'SK' && (
            <TextInput label={t('customer.ic_dph_label')} value={cIcDph}
              onChange={(e) => setCIcDph(e.currentTarget.value)} />
          )}
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
          <NumberInput label={t('customer.default_due_days_label')}
            description={t('customer.default_due_days_desc')}
            value={cDueDays} onChange={(v) => setCDueDays(Number(v) || 0)}
            min={0} max={365} w={200} />
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

      {/* Add payment type modal */}
      <Modal opened={paymentTypeModalOpen} onClose={() => setPaymentTypeModalOpen(false)}
        title={t('settings.payment_type_placeholder')} size="xs">
        <Stack gap="md">
          <TextInput label={t('invoice.payment_method')} placeholder={t('settings.payment_type_placeholder')}
            value={newPaymentTypeName} onChange={(e) => setNewPaymentTypeName(e.currentTarget.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') handleAddPaymentType() }}
          />
          <Group justify="end">
            <Button variant="default" onClick={() => setPaymentTypeModalOpen(false)}>{t('common.cancel')}</Button>
            <Button onClick={handleAddPaymentType} disabled={!newPaymentTypeName.trim()}>{t('common.save')}</Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  )
}
