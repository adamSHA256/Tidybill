import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, waitFor } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MantineProvider } from '@mantine/core'
import { ModalsProvider } from '@mantine/modals'
import { DatesProvider } from '@mantine/dates'
import React from 'react'

vi.mock('../../api/client', () => ({
  api: {
    getFirstRun: () => Promise.resolve({ first_run: false }),
    getSettings: () => Promise.resolve({}),
    getSuppliers: () => Promise.resolve([]),
    getCustomers: () => Promise.resolve([]),
    getInvoices: () => Promise.resolve([]),
    getDashboardStats: () => Promise.resolve({
      revenue_by_currency: [],
      unpaid_count: 0,
      unpaid_by_currency: [],
      active_customers: 0,
      invoices_this_month: 0,
    }),
    getItems: () => Promise.resolve([]),
    getTemplates: () => Promise.resolve([]),
    getVATRates: () => Promise.resolve([{ rate: 21, is_default: true }]),
    getUnits: () => Promise.resolve([{ name: 'ks', is_default: true }]),
    getPaymentTypes: () => Promise.resolve([]),
    getCurrencies: () => Promise.resolve([]),
    getDueDaysOptions: () => Promise.resolve([{ days: 14, is_default: true }]),
    getBankAccounts: () => Promise.resolve([]),
    getMostUsedItems: () => Promise.resolve([]),
    getCustomerItems: () => Promise.resolve([]),
    getNextInvoiceNumber: () => Promise.resolve('VF2024001'),
    getItemCategories: () => Promise.resolve([]),
    getUpdateCheck: () => Promise.resolve(null),
    updateSettings: () => Promise.resolve({}),
    getLocale: () => Promise.resolve({ detected_lang: 'cs' }),
  },
  formatMoney: (amount: number) => String(amount),
  isMobileDevice: () => true,
  isTauri: () => false,
  getApiBase: () => '/api',
  shareFile: () => Promise.resolve(),
  openTemplatePreview: () => Promise.resolve(),
  openInBrowser: () => Promise.resolve(),
}))

import { I18nProvider } from '../../i18n'
import { TourProvider } from '../TourProvider'
import { MobileShell } from '../../components/MobileShell'
import { Dashboard } from '../../pages/Dashboard'
import { MobileInvoiceCreate } from '../../pages/mobile/InvoiceCreate'
import { CustomerList } from '../../pages/CustomerList'
import { SupplierList } from '../../pages/SupplierList'
import { ItemCatalog } from '../../pages/ItemCatalog'
import { Templates } from '../../pages/Templates'
import { Automatizace } from '../../pages/Automatizace'
import { SyncPage } from '../../pages/SyncPage'
import { Settings } from '../../pages/Settings'
import { MorePage } from '../../pages/MorePage'
import { ALL_FLOWS_MOBILE } from '../flows'

const routeToPage: Record<string, React.ComponentType> = {
  '/': Dashboard,
  '/invoices/new': MobileInvoiceCreate,
  '/customers': CustomerList,
  '/suppliers': SupplierList,
  '/items': ItemCatalog,
  '/templates': Templates,
  '/automatizace': Automatizace,
  '/sync': SyncPage,
  '/settings': Settings,
  '/more': MorePage,
}

function makeQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
    },
  })
}

function renderAtRoute(route: string) {
  const Page = routeToPage[route] ?? Dashboard
  const queryClient = makeQueryClient()

  const router = createMemoryRouter(
    [
      {
        path: route,
        element: (
          <I18nProvider>
            <TourProvider>
              <MobileShell>
                <Page />
              </MobileShell>
            </TourProvider>
          </I18nProvider>
        ),
      },
    ],
    { initialEntries: [route] }
  )

  render(
    <QueryClientProvider client={queryClient}>
      <MantineProvider>
        <ModalsProvider>
          <DatesProvider settings={{ locale: 'cs' }}>
            <RouterProvider router={router} />
          </DatesProvider>
        </ModalsProvider>
      </MantineProvider>
    </QueryClientProvider>
  )
}

describe('mobile flows anchors', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  for (const flow of Object.values(ALL_FLOWS_MOBILE)) {
    for (const step of flow.steps) {
      it(`mobile flow "${flow.id}" anchor "${step.anchor}" exists at route "${step.route ?? '/'}"`, async () => {
        renderAtRoute(step.route ?? '/')

        await waitFor(
          () => {
            const el = document.querySelector(`[data-tour="${step.anchor}"]`)
            expect(el, `data-tour="${step.anchor}" not found in DOM`).not.toBeNull()
          },
          { timeout: 5000 }
        )
      })
    }
  }
})
