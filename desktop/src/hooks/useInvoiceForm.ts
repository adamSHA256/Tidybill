import { useState, useEffect, useRef } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { notifications } from '@mantine/notifications'
import { api, formatMoney, type Supplier, type BankAccount, type Item, type CustomerItem } from '../api/client'
import { useT } from '../i18n'

const CREATE_NEW = '__create_new__'
const ADD_CURRENCY = '__add_currency__'
const ADD_VAT_RATE = '__add_vat_rate__'
const ADD_UNIT = '__add_unit__'
const ADD_PAYMENT_TYPE = '__add_payment_type__'

export { CREATE_NEW, ADD_CURRENCY, ADD_VAT_RATE, ADD_UNIT, ADD_PAYMENT_TYPE }

export interface ItemForm {
  item_id: string
  description: string
  quantity: number
  unit: string
  unit_price: number
  vat_rate: number
}

export const emptyItem: ItemForm = { item_id: '', description: '', quantity: 1, unit: 'ks', unit_price: 0, vat_rate: 21 }

function generateVariableSymbol(invoiceNumber: string): string {
  return invoiceNumber.replace(/-/g, '').replace(/^[A-Z]+/, '')
}

export interface DuplicateFromState {
  invoice_number: string
  supplier_id: string
  customer_id: string
  bank_account_id: string
  currency: string
  payment_method: string
  notes: string
  items: ItemForm[]
}

export function useInvoiceForm() {
  const navigate = useNavigate()
  const location = useLocation()
  const queryClient = useQueryClient()
  const { t } = useT()

  const duplicateFrom = (location.state as { duplicateFrom?: DuplicateFromState } | null)?.duplicateFrom || null

  // ── Queries ──────────────────────────────────────────────
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

  // ── Core state ───────────────────────────────────────────
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

  // ── Modal states ─────────────────────────────────────────
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

  // ── Supplier form state ──────────────────────────────────
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

  // ── Customer form state ──────────────────────────────────
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

  // ── Bank account form state ──────────────────────────────
  const [bName, setBName] = useState('')
  const [bAccountNumber, setBAccountNumber] = useState('')
  const [bIban, setBIban] = useState('')
  const [bSwift, setBSwift] = useState('')
  const [bCurrency, setBCurrency] = useState('CZK')
  const [bIsDefault, setBIsDefault] = useState(false)
  const [bQrType, setBQrType] = useState('spayd')

  // ── Derived supplier/bank ────────────────────────────────
  const defaultSupplier = suppliers?.find((s: Supplier) => s.is_default)
  const selectedSupplierId = supplierId || defaultSupplier?.id || null

  const { data: bankAccounts } = useQuery({
    queryKey: ['bank-accounts', selectedSupplierId],
    queryFn: () => api.getBankAccounts(selectedSupplierId!),
    enabled: !!selectedSupplierId,
  })

  const { data: nextNumberData } = useQuery({
    queryKey: ['next-invoice-number', selectedSupplierId],
    queryFn: () => api.getNextInvoiceNumber(selectedSupplierId!),
    enabled: !!selectedSupplierId,
  })

  const defaultBank = bankAccounts?.find((b: BankAccount) => b.is_default)
  const selectedBankId = bankAccountId || defaultBank?.id || null

  const selectedPaymentType = paymentTypes?.find((pt) => pt.name === paymentMethod)
  const requiresBankInfo = !selectedPaymentType || selectedPaymentType.requires_bank_info !== false

  const selectedSupplier = suppliers?.find((s) => s.id === selectedSupplierId)
  const selectedCustomer = customers?.find((c) => c.id === customerId)
  const selectedBank = bankAccounts?.find((b) => b.id === selectedBankId)
  const currencyMismatch = !!(selectedBank && currency && selectedBank.currency !== currency)

  // ── Catalog queries ──────────────────────────────────────
  const { data: customerItems } = useQuery({
    queryKey: ['customer-items', customerId],
    queryFn: () => api.getCustomerItems(customerId!),
    enabled: !!customerId,
  })

  const { data: mostUsedItems } = useQuery({
    queryKey: ['items-most-used'],
    queryFn: api.getMostUsedItems,
  })

  const customerItemIds = new Set((customerItems || []).map((ci: CustomerItem) => ci.item_id))
  const globalSuggestions = (mostUsedItems || []).filter((item: Item) => !customerItemIds.has(item.id))

  // ── Effects ──────────────────────────────────────────────
  useEffect(() => {
    if (nextNumberData?.invoice_number) {
      setInvoiceNumber(nextNumberData.invoice_number)
    }
  }, [nextNumberData])

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

  const vatRatesInitialized = useRef(false)
  useEffect(() => {
    if (vatRates && vatRates.length > 0 && !vatRatesInitialized.current) {
      vatRatesInitialized.current = true
      setItems((prev) => prev.length === 1 && !prev[0].description ? [{ ...prev[0], vat_rate: defaultVatRate }] : prev)
    }
  }, [vatRates]) // eslint-disable-line react-hooks/exhaustive-deps

  const unitsInitialized = useRef(false)
  useEffect(() => {
    if (units && units.length > 0 && !unitsInitialized.current) {
      unitsInitialized.current = true
      setItems((prev) => prev.length === 1 && !prev[0].description ? [{ ...prev[0], unit: defaultUnit }] : prev)
    }
  }, [units]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (!currencyInitialized && settings) {
      setCurrency(globalCurrency)
      setCurrencyInitialized(true)
    }
  }, [settings, currencyInitialized, globalCurrency])

  useEffect(() => {
    if (!dueDateInitialized && dueDaysOptions) {
      if (duplicateFrom?.customer_id && !customers) return
      const cust = duplicateFrom?.customer_id ? customers?.find((c) => c.id === duplicateFrom.customer_id) : null
      const days = (cust && cust.default_due_days > 0) ? cust.default_due_days : globalDueDays
      setDueDate(new Date(Date.now() + days * 86400000).toISOString().slice(0, 10))
      setDueDateInitialized(true)
    }
  }, [dueDaysOptions, dueDateInitialized, globalDueDays, customers]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (paymentTypes && !paymentMethod) {
      const def = paymentTypes.find((pt) => pt.is_default)
      if (def) setPaymentMethod(def.name)
      else if (paymentTypes.length > 0) setPaymentMethod(paymentTypes[0].name)
    }
  }, [paymentTypes]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    return () => {
      if (dueDateTimerRef.current) clearTimeout(dueDateTimerRef.current)
      if (vsTimerRef.current) clearTimeout(vsTimerRef.current)
    }
  }, [])

  useEffect(() => {
    if (defaultBank && !bankAccountId) {
      setCurrency(defaultBank.currency)
    }
  }, [defaultBank?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (!requiresBankInfo) {
      setBankAccountId(null)
      if (!defaultBank) {
        setCurrency(globalCurrency)
      }
    }
  }, [requiresBankInfo]) // eslint-disable-line react-hooks/exhaustive-deps

  // ── Mutations ────────────────────────────────────────────
  const createMutation = useMutation({
    mutationFn: api.createInvoice,
    onSuccess: (newInvoice) => {
      queryClient.invalidateQueries({ queryKey: ['invoices'] })
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] })
      queryClient.invalidateQueries({ queryKey: ['items'] })
      notifications.show({ title: t('notify.invoice_created'), message: t('notify.invoice_created_msg'), color: 'green' })
      navigate(`/invoices/${newInvoice.id}`)
    },
    onError: (err: Error) => {
      notifications.show({ title: t('common.error'), message: err.message, color: 'red' })
    },
  })

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

  // ── Select data builders ─────────────────────────────────
  const baseVatRates = vatRateOptions.length > 0 ? vatRateOptions : ['0', '12', '21']
  const itemVatRates = items.map((item) => String(item.vat_rate)).filter((r) => r && !baseVatRates.includes(r))
  const allVatRates = [...baseVatRates, ...itemVatRates.filter((r, i) => itemVatRates.indexOf(r) === i)]
  const vatRateSelectData = [
    ...allVatRates.map((r) => ({ value: r, label: `${r}%` })),
    { value: ADD_VAT_RATE, label: `+ ${t('invoice.add_vat_rate')}` },
  ]

  const baseUnits = unitOptions.length > 0 ? unitOptions : ['ks', 'hod', 'den', 'm\u00B2']
  const itemUnits = items.map((item) => item.unit).filter((u) => u && !baseUnits.includes(u))
  const allUnits = [...baseUnits, ...itemUnits.filter((u, i) => itemUnits.indexOf(u) === i)]
  const unitSelectData = [
    ...allUnits.map((u) => {
      const key = 'unit.' + u; const translated = t(key); return { value: u, label: translated !== key ? translated : u }
    }),
    { value: ADD_UNIT, label: `+ ${t('invoice.add_unit')}` },
  ]

  const currencyData = [
    ...(currenciesList || []).map((c) => ({ value: c.code, label: c.code })),
    { value: ADD_CURRENCY, label: `+ ${t('invoice.add_currency')}` },
  ]

  const paymentTypeSelectData = [
    ...(paymentTypes || []).map((pt) => ({ value: pt.name, label: pt.name })),
    { value: ADD_PAYMENT_TYPE, label: `+ ${t('settings.payment_type_placeholder')}` },
  ]

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

  // ── Handlers ─────────────────────────────────────────────
  const handleVatRateSelect = (val: string | null, index: number) => {
    if (!val) return
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
    const idx = vatRateTargetIndex
    const currentRates = vatRates || []
    if (!currentRates.some((r) => r.rate === rate)) {
      api.updateVATRates([...currentRates, { rate }]).then(() => {
        queryClient.invalidateQueries({ queryKey: ['vat-rates'] })
      })
    }
    if (idx >= 0) {
      setItems(prev => prev.map((item, i) =>
        i === idx ? { ...item, vat_rate: rate } : item
      ))
    }
    setVatRateModalOpen(false)
  }

  const handleUnitSelect = (val: string | null, index: number) => {
    if (!val) return
    if (val === ADD_UNIT) {
      setNewUnitValue('')
      setUnitTargetIndex(index)
      setUnitModalOpen(true)
      return
    }
    updateItem(index, 'unit', val)
  }

  const handleAddUnit = async () => {
    const name = newUnitValue.trim()
    if (!name) return
    const idx = unitTargetIndex
    const currentUnits = units || []
    if (!currentUnits.some((u) => u.name === name)) {
      await api.updateUnits([...currentUnits, { name }])
      await queryClient.invalidateQueries({ queryKey: ['units'] })
    }
    if (idx >= 0) {
      setItems(prev => prev.map((item, i) =>
        i === idx ? { ...item, unit: name } : item
      ))
    }
    setUnitModalOpen(false)
  }

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
    setBIsDefault(!bankAccounts || bankAccounts.length === 0)
    setBQrType('spayd')
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
      is_default: bIsDefault, qr_type: bQrType,
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
    const cust = customers?.find((c) => c.id === v)
    const days = (cust && cust.default_due_days > 0) ? cust.default_due_days : globalDueDays
    const newDueDate = new Date(Date.now() + days * 86400000).toISOString().slice(0, 10)
    if (dueDate && newDueDate !== dueDate) {
      setDueDateChangedByCustomer(true)
      if (dueDateTimerRef.current) clearTimeout(dueDateTimerRef.current)
      dueDateTimerRef.current = setTimeout(() => setDueDateChangedByCustomer(false), 10000)
    } else {
      setDueDateChangedByCustomer(false)
    }
    setDueDate(newDueDate)
  }

  const handleBankSelect = (v: string | null) => {
    if (v === CREATE_NEW) {
      openBankModal()
      return
    }
    setBankAccountId(v)
    const acc = bankAccounts?.find((b) => b.id === v)
    if (acc) {
      setCurrency(acc.currency)
    }
  }

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

  return {
    // Loading
    isLoading: suppliersLoading || customersLoading,

    // Navigation
    navigate,

    // Translation
    t,

    // Duplicate
    duplicateFrom,

    // Core state
    supplierId: selectedSupplierId,
    customerId,
    bankAccountId: selectedBankId,
    issueDate,
    setIssueDate,
    taxableDate,
    setTaxableDate,
    dueDate,
    setDueDate,
    currency,
    invoiceNumber,
    setInvoiceNumber,
    paymentMethod,
    notes,
    setNotes,
    variableSymbol,
    setVariableSymbol,
    items,

    // UI indicators
    dueDateChangedByCustomer,
    vsChangedByInvoiceNumber,
    currencyMismatch,

    // Entities
    suppliers,
    selectedSupplier,
    selectedCustomer,
    selectedBank,
    bankAccounts,
    requiresBankInfo,

    // Computed
    subtotal,
    vatAmount,
    total,

    // Select data
    supplierData,
    customerData,
    bankData,
    currencyData,
    vatRateSelectData,
    unitSelectData,
    paymentTypeSelectData,

    // Catalog
    customerItems,
    globalSuggestions,
    mostUsedItems,

    // Create
    handleCreate,
    createPending: createMutation.isPending,

    // Entity select handlers
    handleSupplierSelect,
    handleCustomerSelect,
    handleBankSelect,
    handleCurrencySelect,
    handleVatRateSelect,
    handleUnitSelect,
    handlePaymentTypeSelect,

    // Item handlers
    addItem,
    removeItem,
    updateItem,
    addFromCatalog,
    formatMoney,

    // Modal open handlers
    openSupplierModal,
    openCustomerModal,
    openBankModal,

    // Modal states and form fields (grouped for modals component)
    modals: {
      // Supplier modal
      supplierModalOpen, setSupplierModalOpen,
      sName, setSName, sIco, setSIco, sDic, setSDic, sIcDph, setSIcDph,
      sStreet, setSStreet, sCity, setSCity, sZip, setSZip, sCountry, setSCountry,
      sEmail, setSEmail, sPhone, setSPhone, sWebsite, setSWebsite,
      sInvoicePrefix, setSInvoicePrefix, sIsVatPayer, setSIsVatPayer, sNotes, setSNotes,
      handleSaveSupplier, createSupplierPending: createSupplierMutation.isPending,

      // Customer modal
      customerModalOpen, setCustomerModalOpen,
      cName, setCName, cIco, setCIco, cDic, setCDic, cIcDph, setCIcDph,
      cStreet, setCStreet, cCity, setCCity, cZip, setCZip, cCountry, setCCountry,
      cEmail, setCEmail, cPhone, setCPhone, cDueDays, setCDueDays, cNotes, setCNotes,
      handleSaveCustomer, createCustomerPending: createCustomerMutation.isPending,

      // Bank modal
      bankModalOpen, setBankModalOpen,
      bName, setBName, bAccountNumber, setBAccountNumber, bIban, setBIban,
      bSwift, setBSwift, bCurrency, setBCurrency, bIsDefault, setBIsDefault,
      bQrType, setBQrType,
      handleSaveBank, createBankPending: createBankMutation.isPending,

      // Currency modal
      currencyModalOpen, setCurrencyModalOpen,
      newCurrencyCode, setNewCurrencyCode,
      handleAddCurrency,

      // VAT rate modal
      vatRateModalOpen, setVatRateModalOpen,
      newVatRateValue, setNewVatRateValue,
      handleAddVatRate,

      // Unit modal
      unitModalOpen, setUnitModalOpen,
      newUnitValue, setNewUnitValue,
      handleAddUnit,

      // Payment type modal
      paymentTypeModalOpen, setPaymentTypeModalOpen,
      newPaymentTypeName, setNewPaymentTypeName,
      handleAddPaymentType,

      // Shared data for modals
      currencyData,
      handleCurrencySelect,
    },
  }
}

export type InvoiceFormReturn = ReturnType<typeof useInvoiceForm>
