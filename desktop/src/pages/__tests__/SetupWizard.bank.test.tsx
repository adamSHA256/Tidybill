import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MantineProvider } from '@mantine/core'
import { Notifications } from '@mantine/notifications'

const createBankAccount = vi.fn((..._args: unknown[]) => Promise.resolve({
  id: 'b1', supplier_id: 's1', name: '', account_number: '', iban: '', swift: '',
  currency: 'CZK', is_default: true, qr_type: 'spayd', created_at: '',
}))

vi.mock('../../api/client', () => ({
  api: {
    getLocale: () => Promise.resolve({ detected_lang: 'cs' }),
    getSettings: () => Promise.resolve({ default_pdf_dir: '/tmp' }),
    updateSettings: () => Promise.resolve({}),
    createSupplier: () => Promise.resolve({ id: 's1', name: 'Acme', country: 'CZ' }),
    createBankAccount: (supId: string, data: unknown) => createBankAccount(supId, data),
  },
}))

import { I18nProvider } from '../../i18n'
import { SetupWizard } from '../SetupWizard'

function renderWizard() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
  const router = createMemoryRouter(
    [{
      path: '/',
      element: (
        <I18nProvider>
          <SetupWizard onComplete={() => {}} />
        </I18nProvider>
      ),
    }],
    { initialEntries: ['/'] }
  )
  render(
    <QueryClientProvider client={queryClient}>
      <MantineProvider>
        <Notifications />
        <RouterProvider router={router} />
      </MantineProvider>
    </QueryClientProvider>
  )
}

function setMatchMedia(mobile: boolean) {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: (query: string) => ({
      matches: mobile && query.includes('48em'),
      media: query, onchange: null,
      addListener: () => {}, removeListener: () => {},
      addEventListener: () => {}, removeEventListener: () => {},
      dispatchEvent: () => false,
    }),
  })
}

describe('SetupWizard bank step — IBAN-only', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setMatchMedia(false)
  })

  it('accepts IBAN-only and calls createBankAccount without an account-number error', async () => {
    renderWizard()
    const user = userEvent.setup()

    // Step 0: Language → click Next
    await user.click(screen.getByRole('button', { name: /Další/i }))

    // Step 1: Supplier → fill name, click Next
    const nameInput = await screen.findByLabelText(/Název/i)
    await user.type(nameInput, 'Acme')
    // Click the primary Next button at the bottom of the supplier step
    const buttons = screen.getAllByRole('button', { name: /Další/i })
    await user.click(buttons[buttons.length - 1])

    // Wait until bank step is rendered (IBAN field appears)
    const ibanInput = await screen.findByLabelText(/IBAN/i, {}, { timeout: 5000 })

    // Fill ONLY the IBAN — leave account number blank
    await user.type(ibanInput, 'CZ6508000000192000145399')

    // Click Next on the bank step
    const nextBtns = screen.getAllByRole('button', { name: /Další/i })
    await user.click(nextBtns[nextBtns.length - 1])

    // The mutation should fire with iban set and account_number empty
    await waitFor(() => {
      expect(createBankAccount).toHaveBeenCalled()
    }, { timeout: 5000 })
    const [, payload] = createBankAccount.mock.calls[0] as unknown as [string, { account_number: string; iban: string }]
    expect(payload.iban).toContain('CZ65')
    expect(payload.account_number.trim()).toBe('')

    // No "account number is required" notification should be visible
    expect(screen.queryByText(/Číslo účtu nebo IBAN je povinné/i)).toBeNull()
  })

  it('accepts IBAN-only on mobile (matchMedia mobile)', async () => {
    setMatchMedia(true)
    renderWizard()
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: /Další/i }))

    const nameInput = await screen.findByLabelText(/Název/i)
    await user.type(nameInput, 'Acme')
    let buttons = screen.getAllByRole('button', { name: /Další/i })
    await user.click(buttons[buttons.length - 1])

    const ibanInput = await screen.findByLabelText(/IBAN/i, {}, { timeout: 5000 })
    await user.type(ibanInput, 'CZ6508000000192000145399')

    buttons = screen.getAllByRole('button', { name: /Další/i })
    await user.click(buttons[buttons.length - 1])

    await waitFor(() => {
      expect(createBankAccount).toHaveBeenCalled()
    }, { timeout: 5000 })
    const [, payload] = createBankAccount.mock.calls[0] as unknown as [string, { account_number: string; iban: string }]
    expect(payload.iban).toContain('CZ65')
    expect(payload.account_number.trim()).toBe('')
    expect(screen.queryByText(/Číslo účtu nebo IBAN je povinné/i)).toBeNull()
  })
})
