import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MantineProvider } from '@mantine/core'
import { ModalsProvider } from '@mantine/modals'
vi.mock('../../api/client', () => ({
  api: {
    getSettings: () => Promise.resolve({}),
    updateSettings: () => Promise.resolve({}),
  },
}))

import { MobileTourNotice } from '../MobileTourNotice'
import { I18nProvider } from '../../i18n'

function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
}

function renderNotice(onClose = vi.fn()) {
  render(
    <QueryClientProvider client={makeQueryClient()}>
      <MantineProvider>
        <ModalsProvider>
          <MemoryRouter>
            <I18nProvider>
              <MobileTourNotice opened={true} onClose={onClose} />
            </I18nProvider>
          </MemoryRouter>
        </ModalsProvider>
      </MantineProvider>
    </QueryClientProvider>
  )
  return onClose
}

describe('MobileTourNotice', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('renders title from i18n', () => {
    renderNotice()
    // Czech: "Jen na počítači"
    expect(screen.getByText('Jen na počítači')).toBeInTheDocument()
  })

  it('renders body text from i18n', () => {
    renderNotice()
    expect(screen.getByText(/desktopové verzi/i)).toBeInTheDocument()
  })

  it('clicking OK calls onClose', async () => {
    const user = userEvent.setup()
    const onClose = renderNotice()
    await user.click(screen.getByRole('button', { name: /^ok$/i }))
    expect(onClose).toHaveBeenCalledOnce()
  })
})
