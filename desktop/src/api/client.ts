const API_BASE = '/api'

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
    ...options,
  })

  if (!response.ok) {
    const error = await response.text()
    throw new Error(error || `Request failed: ${response.status}`)
  }

  if (response.status === 204) return undefined as T

  return response.json()
}

export const api = {
  // Dashboard
  getDashboardStats: () => request<DashboardStats>('/dashboard/stats'),

  // Invoices
  getInvoices: (params?: { status?: string; customer_id?: string }) => {
    const query = new URLSearchParams()
    if (params?.status) query.set('status', params.status)
    if (params?.customer_id) query.set('customer_id', params.customer_id)
    const qs = query.toString()
    return request<Invoice[]>(`/invoices${qs ? '?' + qs : ''}`)
  },
  getInvoice: (id: string) => request<Invoice>(`/invoices/${id}`),
  createInvoice: (data: CreateInvoiceRequest) =>
    request<Invoice>('/invoices', { method: 'POST', body: JSON.stringify(data) }),
  updateInvoice: (id: string, data: Partial<CreateInvoiceRequest> & { internal_notes?: string }) =>
    request<Invoice>(`/invoices/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteInvoice: (id: string) =>
    request<void>(`/invoices/${id}`, { method: 'DELETE' }),
  updateInvoiceStatus: (id: string, status: string) =>
    request<Invoice>(`/invoices/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) }),
  updateInvoiceNotes: (id: string, internal_notes: string) =>
    request<Invoice>(`/invoices/${id}/notes`, { method: 'PUT', body: JSON.stringify({ internal_notes }) }),
  generatePDF: (id: string) =>
    request<{ path: string }>(`/invoices/${id}/pdf`, { method: 'POST' }),

  // Customers
  getCustomers: (q?: string) => {
    const qs = q ? `?q=${encodeURIComponent(q)}` : ''
    return request<Customer[]>(`/customers${qs}`)
  },
  getCustomer: (id: string) => request<Customer>(`/customers/${id}`),
  createCustomer: (data: Partial<Customer>) =>
    request<Customer>('/customers', { method: 'POST', body: JSON.stringify(data) }),
  updateCustomer: (id: string, data: Partial<Customer>) =>
    request<Customer>(`/customers/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteCustomer: (id: string) =>
    request<void>(`/customers/${id}`, { method: 'DELETE' }),

  // Suppliers
  getSuppliers: () => request<Supplier[]>('/suppliers'),
  getSupplier: (id: string) => request<Supplier>(`/suppliers/${id}`),
  createSupplier: (data: Partial<Supplier>) =>
    request<Supplier>('/suppliers', { method: 'POST', body: JSON.stringify(data) }),
  updateSupplier: (id: string, data: Partial<Supplier>) =>
    request<Supplier>(`/suppliers/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteSupplier: (id: string) =>
    request<void>(`/suppliers/${id}`, { method: 'DELETE' }),
  uploadLogo: async (supplierId: string, file: File): Promise<Supplier> => {
    const formData = new FormData()
    formData.append('logo', file)
    const response = await fetch(`${API_BASE}/suppliers/${supplierId}/logo`, {
      method: 'POST',
      body: formData,
    })
    if (!response.ok) {
      const error = await response.text()
      throw new Error(error || `Upload failed: ${response.status}`)
    }
    return response.json()
  },
  getLogoUrl: (supplierId: string) => `${API_BASE}/suppliers/${supplierId}/logo`,
  deleteLogo: (supplierId: string) =>
    request<void>(`/suppliers/${supplierId}/logo`, { method: 'DELETE' }),

  // Bank accounts
  getBankAccounts: (supplierId: string) =>
    request<BankAccount[]>(`/suppliers/${supplierId}/bank-accounts`),
  createBankAccount: (supplierId: string, data: Partial<BankAccount>) =>
    request<BankAccount>(`/suppliers/${supplierId}/bank-accounts`, {
      method: 'POST', body: JSON.stringify(data),
    }),

  // Items catalog
  getItems: (q?: string) => {
    const qs = q ? `?q=${encodeURIComponent(q)}` : ''
    return request<Item[]>(`/items${qs}`)
  },
  getItem: (id: string) => request<Item>(`/items/${id}`),
  createItem: (data: Partial<Item>) =>
    request<Item>('/items', { method: 'POST', body: JSON.stringify(data) }),
  updateItem: (id: string, data: Partial<Item>) =>
    request<Item>(`/items/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteItem: (id: string) =>
    request<void>(`/items/${id}`, { method: 'DELETE' }),
  getMostUsedItems: () => request<Item[]>('/items/most-used'),
  getItemCategories: () => request<string[]>('/items/categories'),
  getCustomerItems: (customerId: string) =>
    request<CustomerItem[]>(`/customers/${customerId}/items`),

  // Templates
  getTemplates: () => request<PDFTemplate[]>('/templates'),
  getTemplate: (id: string) => request<PDFTemplate>(`/templates/${id}`),
  updateTemplate: (id: string, data: Partial<PDFTemplate>) =>
    request<PDFTemplate>(`/templates/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  setDefaultTemplate: (id: string) =>
    request<void>(`/templates/${id}/default`, { method: 'PUT' }),
  generatePreview: (id: string) =>
    request<{ path: string }>(`/templates/${id}/preview`, { method: 'POST' }),
  generateAllPreviews: () =>
    request<Record<string, string>>('/templates/preview-all', { method: 'POST' }),

  // Bank accounts
  updateBankAccount: (id: string, data: Partial<BankAccount>) =>
    request<BankAccount>(`/bank-accounts/${id}`, { method: 'PUT', body: JSON.stringify(data) }),

  // Settings
  getSettings: () => request<AppSettings>('/settings'),
  updateSettings: (data: Partial<AppSettings>) =>
    request<AppSettings>('/settings', { method: 'PUT', body: JSON.stringify(data) }),
}

// Types matching Go models exactly
export interface Invoice {
  id: string
  invoice_number: string
  supplier_id: string
  customer_id: string
  bank_account_id: string
  status: InvoiceStatus
  issue_date: string
  due_date: string
  paid_date: string | null
  taxable_date: string
  payment_method: string
  variable_symbol: string
  currency: string
  exchange_rate: number
  subtotal: number
  vat_total: number
  total: number
  notes: string
  internal_notes: string
  language: string
  pdf_path: string
  template_id: string
  created_at: string
  updated_at: string
  items?: InvoiceItem[]
  customer?: Customer
  supplier?: Supplier
}

export type InvoiceStatus = 'draft' | 'created' | 'sent' | 'paid' | 'overdue' | 'partially_paid' | 'cancelled'

export interface InvoiceItem {
  id: string
  invoice_id: string
  item_id: string
  description: string
  quantity: number
  unit: string
  unit_price: number
  vat_rate: number
  subtotal: number
  vat_amount: number
  total: number
  position: number
}

export interface Customer {
  id: string
  name: string
  street: string
  city: string
  zip: string
  region: string
  country: string
  ico: string
  dic: string
  email: string
  phone: string
  default_vat_rate: number
  default_due_days: number
  notes: string
  created_at: string
  updated_at: string
}

export interface Supplier {
  id: string
  name: string
  street: string
  city: string
  zip: string
  country: string
  ico: string
  dic: string
  phone: string
  email: string
  logo_path: string
  is_vat_payer: boolean
  is_default: boolean
  invoice_prefix: string
  website: string
  notes: string
  language: string
  created_at: string
  updated_at: string
}

export interface BankAccount {
  id: string
  supplier_id: string
  name: string
  account_number: string
  iban: string
  swift: string
  currency: string
  is_default: boolean
  qr_type: string
  created_at: string
}

export interface PDFTemplate {
  id: string
  name: string
  template_code: string
  config_json: string
  is_default: boolean
  supplier_id: string
  description: string
  show_logo: boolean
  show_qr: boolean
  show_notes: boolean
  preview_path: string
  sort_order: number
}

export interface DashboardStats {
  total_revenue_month: number
  unpaid_count: number
  unpaid_amount: number
  overdue_count: number
  active_customers: number
  invoices_this_month: number
}

export interface Item {
  id: string
  description: string
  default_price: number
  default_unit: string
  default_vat_rate: number
  category: string
  last_used_price: number
  last_customer_id: string
  usage_count: number
  created_at: string
  updated_at: string
}

export interface CustomerItem {
  id: string
  customer_id: string
  item_id: string
  last_price: number
  last_quantity: number
  usage_count: number
  last_used_at: string
  item_description: string
  item_category: string
  item_default_unit: string
  item_default_vat: number
}

export interface AppSettings {
  language: string
  dir_logos?: string
  dir_pdfs?: string
  dir_previews?: string
}

export interface CreateInvoiceRequest {
  customer_id: string
  supplier_id: string
  bank_account_id: string
  invoice_number?: string
  issue_date?: string
  due_date?: string
  currency?: string
  notes?: string
  items: {
    item_id?: string
    description: string
    quantity: number
    unit: string
    unit_price: number
    vat_rate: number
  }[]
}

// Helper to format Czech money
export function formatMoney(amount: number, currency = 'Kc'): string {
  return amount.toLocaleString('cs-CZ', { minimumFractionDigits: 0, maximumFractionDigits: 2 }) + ' ' + currency
}

// Helper to format date from ISO to Czech format
export function formatDate(iso: string): string {
  if (!iso || iso === '0001-01-01T00:00:00Z') return '—'
  const d = new Date(iso)
  return d.toLocaleDateString('cs-CZ')
}
