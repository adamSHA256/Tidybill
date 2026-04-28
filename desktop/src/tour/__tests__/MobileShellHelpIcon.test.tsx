import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MantineProvider } from '@mantine/core'

vi.mock('../../api/client', () => ({
  api: {
    getSettings: () => Promise.resolve({}),
    updateSettings: () => Promise.resolve({}),
  },
}))

import { I18nProvider } from '../../i18n'
import { MobileShell } from '../../components/MobileShell'

function renderAt(route: string) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
  const router = createMemoryRouter(
    [{
      path: '*',
      element: (
        <I18nProvider>
          <MobileShell><div data-testid="page" /></MobileShell>
        </I18nProvider>
      ),
    }],
    { initialEntries: [route] }
  )
  render(
    <QueryClientProvider client={queryClient}>
      <MantineProvider>
        <RouterProvider router={router} />
      </MantineProvider>
    </QueryClientProvider>
  )
}

// aria-label resolves via t('tour.help_aria'); the default locale is 'cs'
const HELP_LABEL = 'Nápověda'

describe('MobileShell help icon', () => {
  it('renders on the dashboard route ("/")', () => {
    renderAt('/')
    expect(screen.queryByLabelText(HELP_LABEL)).not.toBeNull()
  })

  it('is hidden on /invoices', () => {
    renderAt('/invoices')
    expect(screen.queryByLabelText(HELP_LABEL)).toBeNull()
  })

  it('is hidden on /invoices/new', () => {
    renderAt('/invoices/new')
    expect(screen.queryByLabelText(HELP_LABEL)).toBeNull()
  })

  it('is hidden on /more', () => {
    renderAt('/more')
    expect(screen.queryByLabelText(HELP_LABEL)).toBeNull()
  })

  it('is hidden on /settings (deep route)', () => {
    renderAt('/settings')
    expect(screen.queryByLabelText(HELP_LABEL)).toBeNull()
  })
})
