import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MantineProvider } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { DatesProvider } from '@mantine/dates'
import 'dayjs/locale/cs'

const supplierId = 's1'
const customerId = 'c1'
const bankId = 'b1'

const createInvoice = vi.fn((..._args: unknown[]) => Promise.resolve({
  id: 'new-invoice-id',
  invoice_number: 'TST-001',
  supplier_id: supplierId,
  customer_id: customerId,
  bank_account_id: bankId,
  status: 'created',
  issue_date: '2026-02-03T00:00:00Z',
  due_date: '2026-02-17T00:00:00Z',
  taxable_date: '2026-02-03T00:00:00Z',
  paid_date: null, payment_method: 'Převodem', variable_symbol: 'TST001',
  currency: 'CZK', exchange_rate: 1, subtotal: 100, vat_total: 21, total: 121,
  notes: '', internal_notes: '', language: 'cs', pdf_path: '',
  template_id: 'table', email_sent_at: null, created_at: '', updated_at: '',
}))

vi.mock('../../api/client', async () => {
  const formatMoney = (n: number) => String(n)
  return {
    formatMoney,
    api: {
      getSettings: () => Promise.resolve({ default_currency: 'CZK', default_vat_rate: '21' }),
      getDueDaysOptions: () => Promise.resolve([{ days: 14, is_default: true }]),
      getPaymentTypes: () => Promise.resolve([{ name: 'Převodem', is_default: true, requires_bank_info: true }]),
      getCurrencies: () => Promise.resolve([{ code: 'CZK' }, { code: 'EUR' }]),
      getSuppliers: () => Promise.resolve([
        { id: supplierId, name: 'Acme', is_default: true, ico: '123', dic: '', ic_dph: '', street: 'St', city: 'Praha', zip: '110 00', country: 'CZ' },
      ]),
      getCustomers: () => Promise.resolve([
        { id: customerId, name: 'Cust', ico: '999', dic: '', ic_dph: '', street: 'St2', city: 'Brno', zip: '602 00', country: 'CZ', default_due_days: 0 },
      ]),
      getBankAccounts: () => Promise.resolve([
        { id: bankId, supplier_id: supplierId, name: 'B', account_number: '123/4500', iban: '', swift: '', currency: 'CZK', is_default: true, qr_type: 'spayd' },
      ]),
      getNextInvoiceNumber: () => Promise.resolve({ invoice_number: 'TST-001' }),
      getUnits: () => Promise.resolve([{ name: 'ks', is_default: true }]),
      getVATRates: () => Promise.resolve([{ rate: 21, is_default: true }]),
      getCustomerItems: () => Promise.resolve([]),
      getMostUsedItems: () => Promise.resolve([]),
      createInvoice: (data: unknown) => createInvoice(data),
    },
  }
})

import { I18nProvider } from '../../i18n'
import { InvoiceCreate } from '../InvoiceCreate'

function Landing() { return <div>landing-detail-page</div> }

function renderApp() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
  const router = createMemoryRouter(
    [
      { path: '/invoices/new', element: <I18nProvider><InvoiceCreate /></I18nProvider> },
      { path: '/invoices/:id', element: <Landing /> },
    ],
    { initialEntries: ['/invoices/new'] },
  )
  render(
    <QueryClientProvider client={queryClient}>
      <MantineProvider>
        <DatesProvider settings={{ locale: 'cs' }}>
          <Notifications />
          <RouterProvider router={router} />
        </DatesProvider>
      </MantineProvider>
    </QueryClientProvider>,
  )
}

describe('InvoiceCreate — post-save navigation', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: () => ({
        matches: false, media: '', onchange: null,
        addListener: () => {}, removeListener: () => {},
        addEventListener: () => {}, removeEventListener: () => {}, dispatchEvent: () => false,
      }),
    })
  })

  it('does NOT show the unsaved-changes modal after the invoice is created', async () => {
    const user = userEvent.setup()
    renderApp()

    // Wait for form to load (item description input is the most reliable anchor)
    const descInput = await screen.findByPlaceholderText('Popis položky', {}, { timeout: 10000 })

    // Pick the customer so handleCreate validation passes
    const customerSelect = screen.getByLabelText('Vyberte odběratele', { selector: 'input' }) as HTMLInputElement
    await user.click(customerSelect)
    const custOption = await screen.findByText('Cust')
    await user.click(custOption)

    // Type item description so updateItem fires and sets formTouched=true
    await user.type(descInput, 'Test item')

    // Submit
    const createBtn = await screen.findByRole('button', { name: 'Vytvořit fakturu' })
    await user.click(createBtn)

    // Wait for createInvoice to be called
    await waitFor(() => expect(createInvoice).toHaveBeenCalled(), { timeout: 10000 })

    // Wait for navigation to land on detail page
    await screen.findByText('landing-detail-page', {}, { timeout: 10000 })

    // The leave-confirm modal must NOT be displayed
    expect(screen.queryByText(/Opustit tvorbu faktury/)).not.toBeInTheDocument()
    expect(screen.queryByText(/neuložené změny/i)).not.toBeInTheDocument()
  }, 20000)
})
