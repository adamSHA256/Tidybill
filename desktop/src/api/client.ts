let resolvedApiBase: string | null = null

export function isTauri(): boolean {
  return '__TAURI_INTERNALS__' in window
}

export async function initApiBase(): Promise<void> {
  if (!isTauri()) {
    resolvedApiBase = '/api'
    return
  }

  const { invoke } = await import('@tauri-apps/api/core')
  const maxAttempts = 50
  for (let i = 0; i < maxAttempts; i++) {
    const port = await invoke<number>('get_api_port')
    if (port > 0) {
      resolvedApiBase = `http://127.0.0.1:${port}/api`
      return
    }
    await new Promise((r) => setTimeout(r, 200))
  }
  throw new Error('Backend did not start in time')
}

export function getApiBase(): string {
  return resolvedApiBase ?? '/api'
}

export async function checkHealth(): Promise<boolean> {
  try {
    const controller = new AbortController()
    const timeout = setTimeout(() => controller.abort(), 3000)
    const res = await fetch(`${getApiBase()}/health`, { signal: controller.signal })
    clearTimeout(timeout)
    return res.ok
  } catch {
    return false
  }
}

async function request<T>(path: string, options?: RequestInit & { timeout?: number }): Promise<T> {
  const { timeout, ...fetchOptions } = options || {}

  let controller: AbortController | undefined
  let timer: ReturnType<typeof setTimeout> | undefined
  if (timeout) {
    controller = new AbortController()
    timer = setTimeout(() => controller!.abort(), timeout)
  }

  const response = await fetch(`${getApiBase()}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...fetchOptions?.headers,
    },
    ...fetchOptions,
    ...(controller ? { signal: controller.signal } : {}),
  }).catch((err) => {
    if (timer) clearTimeout(timer)
    if (err.name === 'AbortError') throw new Error('TIMEOUT')
    throw err
  })

  if (timer) clearTimeout(timer)

  if (!response.ok) {
    const error = await response.text()
    throw new Error(error || `Request failed: ${response.status}`)
  }

  if (response.status === 204) return undefined as T

  return response.json()
}

export const api = {
  // System
  getFirstRun: () => request<{ first_run: boolean }>('/system/first-run'),
  getLocale: () => request<{ detected_lang: string }>('/system/locale'),
  getAbout: () => request<AboutInfo>('/system/about'),
  getUpdateCheck: () => request<UpdateCheckResult>('/system/update-check'),
  triggerUpdateCheck: () => request<UpdateCheckResult>('/system/update-check', { method: 'POST' }),

  // Dashboard
  getDashboardStats: () => request<DashboardStats>('/dashboard/stats'),

  // Invoices
  getInvoices: (params?: { status?: string; customer_id?: string; supplier_id?: string }) => {
    const query = new URLSearchParams()
    if (params?.status) query.set('status', params.status)
    if (params?.customer_id) query.set('customer_id', params.customer_id)
    if (params?.supplier_id) query.set('supplier_id', params.supplier_id)
    const qs = query.toString()
    return request<Invoice[]>(`/invoices${qs ? '?' + qs : ''}`)
  },
  getInvoice: (id: string) => request<Invoice>(`/invoices/${id}`),
  getNextInvoiceNumber: (supplierId: string) =>
    request<{ invoice_number: string }>(`/invoices/next-number?supplier_id=${encodeURIComponent(supplierId)}`),
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

  // Email
  getEmailPreview: (invoiceId: string) => request<EmailPreview>(`/invoices/${invoiceId}/email-preview`),
  sendInvoiceEmail: (invoiceId: string, data: SendEmailRequest) => request<SendEmailResponse>(`/invoices/${invoiceId}/send-email`, { method: 'POST', body: JSON.stringify(data), timeout: 25000 }),

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
    const response = await fetch(`${getApiBase()}/suppliers/${supplierId}/logo`, {
      method: 'POST',
      body: formData,
    })
    if (!response.ok) {
      const error = await response.text()
      throw new Error(error || `Upload failed: ${response.status}`)
    }
    return response.json()
  },
  getLogoUrl: (supplierId: string) => `${getApiBase()}/suppliers/${supplierId}/logo`,
  deleteLogo: (supplierId: string) =>
    request<void>(`/suppliers/${supplierId}/logo`, { method: 'DELETE' }),

  // SMTP Config
  getSmtpConfig: (supplierId: string) => request<SmtpConfig | { configured: false }>(`/suppliers/${supplierId}/smtp`),
  upsertSmtpConfig: (supplierId: string, data: SmtpConfigInput) => request<SmtpConfig>(`/suppliers/${supplierId}/smtp`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteSmtpConfig: (supplierId: string) => request<void>(`/suppliers/${supplierId}/smtp`, { method: 'DELETE' }),
  testSmtpConnection: (supplierId: string, data: SmtpConfigInput) => request<{ status: string }>(`/suppliers/${supplierId}/smtp/test`, { method: 'POST', body: JSON.stringify(data), timeout: 20000 }),
  copySmtpConfig: (toSupplierId: string, fromSupplierId: string) => request<SmtpConfig>(`/suppliers/${toSupplierId}/smtp/copy/${fromSupplierId}`, { method: 'POST' }),

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
  duplicateTemplate: (id: string, name: string) =>
    request<PDFTemplate>(`/templates/${id}/duplicate`, { method: 'POST', body: JSON.stringify({ name }) }),
  getTemplateSource: (id: string) =>
    request<{ yaml_source: string }>(`/templates/${id}/source`),
  updateTemplateSource: (id: string, yaml_source: string) =>
    request<{ status: string }>(`/templates/${id}/source`, { method: 'PUT', body: JSON.stringify({ yaml_source }) }),
  deleteTemplate: (id: string) =>
    request<void>(`/templates/${id}`, { method: 'DELETE' }),
  getAIPrompt: () =>
    request<{ prompt: string }>('/templates/ai-prompt'),

  // Bank accounts
  updateBankAccount: (id: string, data: Partial<BankAccount>) =>
    request<BankAccount>(`/bank-accounts/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteBankAccount: (id: string) =>
    request<void>(`/bank-accounts/${id}`, { method: 'DELETE' }),

  // Settings
  getSettings: () => request<AppSettings>('/settings'),
  updateSettings: (data: Partial<AppSettings>) =>
    request<AppSettings>('/settings', { method: 'PUT', body: JSON.stringify(data) }),

  // Units
  getUnits: () => request<Unit[]>('/units'),
  updateUnits: (units: Unit[]) =>
    request<Unit[]>('/units', { method: 'PUT', body: JSON.stringify(units) }),

  // Payment Types
  getPaymentTypes: () => request<PaymentType[]>('/payment-types'),
  updatePaymentTypes: (types: PaymentType[]) =>
    request<PaymentType[]>('/payment-types', { method: 'PUT', body: JSON.stringify(types) }),

  // VAT Rates
  getVATRates: () => request<VATRate[]>('/vat-rates'),
  updateVATRates: (rates: VATRate[]) =>
    request<VATRate[]>('/vat-rates', { method: 'PUT', body: JSON.stringify(rates) }),

  // Currencies
  getCurrencies: () => request<CurrencyItem[]>('/currencies'),
  updateCurrencies: (currencies: CurrencyItem[]) =>
    request<CurrencyItem[]>('/currencies', { method: 'PUT', body: JSON.stringify(currencies) }),

  // Due Days Options
  getDueDaysOptions: () => request<DueDaysOption[]>('/due-days'),
  updateDueDaysOptions: (options: DueDaysOption[]) =>
    request<DueDaysOption[]>('/due-days', { method: 'PUT', body: JSON.stringify(options) }),

  // Backup
  exportBackup: async (filters?: ExportFilters): Promise<Blob> => {
    const response = await fetch(`${getApiBase()}/backup/export`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: filters ? JSON.stringify(filters) : '{}',
    })
    if (!response.ok) throw new Error(await response.text())
    return response.blob()
  },

  exportBackupToFile: async (filters?: ExportFilters): Promise<{path: string, filename: string}> => {
    return request<{path: string, filename: string}>('/backup/export-file', {
      method: 'POST',
      body: filters ? JSON.stringify(filters) : '{}',
    })
  },

  importBackup: async (file: File, mode: string = 'merge'): Promise<ImportReport> => {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('mode', mode)
    const response = await fetch(`${getApiBase()}/backup/import`, {
      method: 'POST',
      body: formData,
    })
    if (!response.ok) throw new Error(await response.text())
    return response.json()
  },

  previewImport: async (file: File): Promise<ImportReport> => {
    const formData = new FormData()
    formData.append('file', file)
    const response = await fetch(`${getApiBase()}/backup/import/preview`, {
      method: 'POST',
      body: formData,
    })
    if (!response.ok) throw new Error(await response.text())
    return response.json()
  },
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
  email_sent_at: string | null
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
  ic_dph: string
  email: string
  phone: string
  default_vat_rate: number
  default_due_days: number
  notes: string
  email_custom_template: boolean
  email_subject_template: string
  email_body_template: string
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
  ic_dph: string
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
  is_builtin: boolean
  yaml_source: string
  parent_id: string
}

export interface AboutInfo {
  version: string
  description: string
  github_issues_url: string
  monero_address: string
  bitcoin_address: string
}

export interface CurrencyAmount {
  currency: string
  amount: number
}

export interface DashboardStats {
  total_revenue_month: number
  revenue_by_currency: CurrencyAmount[]
  unpaid_count: number
  unpaid_amount: number
  unpaid_by_currency: CurrencyAmount[]
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

export interface UpdateCheckResult {
  available: boolean
  current_version: string
  latest_version: string
  release_url: string
  release_notes: string
  published_at: string
  checked_at: string
}

export interface AppSettings {
  language: string
  dir_logos?: string
  dir_pdfs?: string
  dir_previews?: string
  default_currency?: string
  default_due_days?: string
  default_vat_rate?: string
  dashboard_widgets?: string
  custom_currencies?: string
  custom_countries?: string
  invoice_default_sort?: string // TODO: also expose in CLI settings menu
  ui_scale?: string
  check_updates?: string
  default_pdf_dir?: string
  default_logo_dir?: string
  default_preview_dir?: string
  'email.default_subject'?: string
  'email.default_body'?: string
  'email.copy_subject'?: string
}

export interface Unit {
  name: string
  is_default?: boolean
}

export interface PaymentType {
  name: string
  code?: string
  is_default?: boolean
  requires_bank_info?: boolean
}

export interface VATRate {
  rate: number
  name?: string
  is_default?: boolean
}

export interface DueDaysOption {
  days: number
  is_default?: boolean
}

export interface CurrencyItem {
  code: string
}

export interface SmtpConfig {
  id: string
  supplier_id: string
  host: string
  port: number
  username: string
  has_password: boolean
  from_name: string
  from_email: string
  use_starttls: boolean
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface SmtpConfigInput {
  host: string
  port: number
  username: string
  password?: string
  from_name: string
  from_email: string
  use_starttls: boolean
  enabled: boolean
}

export interface EmailPreview {
  to: string
  subject: string
  body: string
  pdf_filename: string
  has_smtp: boolean
  already_sent_at: string | null
}

export interface SendEmailRequest {
  to: string
  subject: string
  body: string
  send_copy?: boolean
}

export interface SendEmailResponse {
  ok: boolean
  email_sent_at: string
  status: string
}

export interface ExportFilters {
  supplier_ids?: string[]
  skip_paid_older_than_years?: number
  date_from?: string
  date_to?: string
  exclude_settings?: boolean
}

export interface ImportReport {
  mode: string
  started_at: string
  finished_at: string
  summary: {
    to_insert: number
    to_update: number
    to_skip: number
    conflicts: number
    warnings: number
  }
  details: Record<string, {
    insert: number
    update: number
    skip: number
    conflicts: number
  }>
  conflicts: Array<{
    table: string
    id: string
    type: string
    description: string
    resolution: string
  }>
  warnings: Array<{
    table: string
    id: string
    type: string
    description: string
    resolution: string
  }>
}

export interface CreateInvoiceRequest {
  customer_id: string
  supplier_id: string
  bank_account_id?: string
  invoice_number?: string
  issue_date?: string
  due_date?: string
  taxable_date?: string
  payment_method?: string
  variable_symbol?: string
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

export function isMobileDevice(): boolean {
  return /Android|iPhone|iPad|iPod/i.test(navigator.userAgent)
}

export function getInvoicePdfUrl(id: string): string {
  return `${getApiBase()}/invoices/${id}/pdf-file`
}

export function getPreviewPdfUrl(id: string): string {
  return `${getApiBase()}/templates/${id}/preview-pdf`
}

export async function openInBrowser(url: string): Promise<void> {
  try {
    if (isTauri()) {
      const opener = await import('@tauri-apps/plugin-opener')
      if (isMobileDevice() || url.startsWith('http://') || url.startsWith('https://')) {
        // On mobile, openPath doesn't work (no FileProvider support in opener plugin).
        // Always use openUrl which opens in the system browser.
        await opener.openUrl(url)
      } else {
        await opener.openPath(url)
      }
    } else {
      window.open(url, '_blank')
    }
  } catch (err) {
    console.error('openInBrowser failed:', url, err)
    throw new Error(`Failed to open "${url}": ${err instanceof Error ? err.message : String(err)}`)
  }
}

// Open an invoice PDF - uses HTTP URL on mobile, file path on desktop
export async function openInvoicePdf(invoiceId: string, filePath: string): Promise<void> {
  if (isMobileDevice()) {
    return openInBrowser(getInvoicePdfUrl(invoiceId))
  }
  return openInBrowser(filePath)
}

// Open a template preview PDF - uses HTTP URL on mobile, file path on desktop
export async function openTemplatePreview(templateId: string, filePath: string): Promise<void> {
  if (isMobileDevice()) {
    return openInBrowser(getPreviewPdfUrl(templateId))
  }
  return openInBrowser(filePath)
}

export async function openFolder(filePath: string): Promise<void> {
  if (isMobileDevice()) return
  try {
    if (isTauri()) {
      const { revealItemInDir } = await import('@tauri-apps/plugin-opener')
      await revealItemInDir(filePath)
    }
  } catch (err) {
    console.error('openFolder failed:', filePath, err)
    throw new Error(`Failed to open folder for "${filePath}": ${err instanceof Error ? err.message : String(err)}`)
  }
}

export async function sharePdf(filePath: string, filename: string): Promise<void> {
  if (isTauri() && isMobileDevice()) {
    const { invoke } = await import('@tauri-apps/api/core')
    await invoke('plugin:sharesheet|share_file', {
      filePath,
      mimeType: 'application/pdf',
      title: filename,
    })
    return
  }
  // Desktop/web fallback - not applicable
}

export async function shareFile(filePath: string, filename: string, mimeType = 'application/octet-stream'): Promise<void> {
  if (isTauri() && isMobileDevice()) {
    const { invoke } = await import('@tauri-apps/api/core')
    await invoke('plugin:sharesheet|share_file', {
      filePath,
      mimeType,
      title: filename,
    })
    return
  }
}

export function formatMoney(amount: number, currency = ''): string {
  const formatted = amount.toLocaleString('cs-CZ', { minimumFractionDigits: 0, maximumFractionDigits: 2 })
  return currency ? formatted + ' ' + currency : formatted
}

// Helper to format date from ISO to Czech format
export function formatDate(iso: string): string {
  if (!iso || iso === '0001-01-01T00:00:00Z') return '—'
  const d = new Date(iso)
  return d.toLocaleDateString('cs-CZ')
}
