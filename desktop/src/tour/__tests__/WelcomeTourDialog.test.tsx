import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MantineProvider } from '@mantine/core'
import { ModalsProvider } from '@mantine/modals'
import React from 'react'

const mockStartFlow = vi.fn()

vi.mock('../TourProvider', () => ({
  useTour: () => ({ startFlow: mockStartFlow, stop: vi.fn(), isRunning: false, currentFlowId: null }),
  TourProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

vi.mock('../../api/client', () => ({
  api: {
    getSettings: () => Promise.resolve({}),
    updateSettings: () => Promise.resolve({}),
  },
}))

import { WelcomeTourDialog } from '../WelcomeTourDialog'
import { readState, resetState } from '../persistence'
import { I18nProvider } from '../../i18n'

function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
}

function renderDialog(onClose = vi.fn()) {
  render(
    <QueryClientProvider client={makeQueryClient()}>
      <MantineProvider>
        <ModalsProvider>
          <MemoryRouter>
            <I18nProvider>
              <WelcomeTourDialog opened={true} onClose={onClose} />
            </I18nProvider>
          </MemoryRouter>
        </ModalsProvider>
      </MantineProvider>
    </QueryClientProvider>
  )
  return onClose
}

describe('WelcomeTourDialog', () => {
  beforeEach(() => {
    resetState()
    vi.clearAllMocks()
  })

  it('renders all four option cards', () => {
    renderDialog()
    expect(document.querySelector('[data-tour-option="create-invoice"]')).not.toBeNull()
    expect(document.querySelector('[data-tour-option="just-show-me"]')).not.toBeNull()
    expect(document.querySelector('[data-tour-option="advanced"]')).not.toBeNull()
    expect(document.querySelector('[data-tour-option="no-thanks"]')).not.toBeNull()
  })

  it('renders do-not-show checkbox checked by default', () => {
    renderDialog()
    expect(screen.getByRole('checkbox')).toBeChecked()
  })

  it('clicking fast option calls startFlow and closes, welcomeSeen=true doNotAutoShow=true', async () => {
    const user = userEvent.setup()
    const onClose = renderDialog()
    await user.click(document.querySelector('[data-tour-option="create-invoice"]')!)
    expect(mockStartFlow).toHaveBeenCalledWith('create-invoice')
    expect(onClose).toHaveBeenCalled()
    const state = readState()
    expect(state.welcomeSeen).toBe(true)
    expect(state.doNotAutoShow).toBe(true)
  })

  it('clicking explore option calls startFlow with just-show-me', async () => {
    const user = userEvent.setup()
    renderDialog()
    await user.click(document.querySelector('[data-tour-option="just-show-me"]')!)
    expect(mockStartFlow).toHaveBeenCalledWith('just-show-me')
  })

  it('clicking advanced option calls startFlow with advanced', async () => {
    const user = userEvent.setup()
    renderDialog()
    await user.click(document.querySelector('[data-tour-option="advanced"]')!)
    expect(mockStartFlow).toHaveBeenCalledWith('advanced')
  })

  it('clicking no-thanks does not call startFlow, closes, persists welcomeSeen=true', async () => {
    const user = userEvent.setup()
    const onClose = renderDialog()
    await user.click(document.querySelector('[data-tour-option="no-thanks"]')!)
    expect(mockStartFlow).not.toHaveBeenCalled()
    expect(onClose).toHaveBeenCalled()
    expect(readState().welcomeSeen).toBe(true)
  })

  it('unchecking do-not-show then picking option persists doNotAutoShow=false', async () => {
    const user = userEvent.setup()
    renderDialog()
    await user.click(screen.getByRole('checkbox'))
    await user.click(document.querySelector('[data-tour-option="create-invoice"]')!)
    expect(readState().doNotAutoShow).toBe(false)
  })

  it('unchecking do-not-show then closing with ESC persists doNotAutoShow=false', async () => {
    const user = userEvent.setup()
    renderDialog()
    await user.click(screen.getByRole('checkbox'))
    await user.keyboard('{Escape}')
    expect(readState().doNotAutoShow).toBe(false)
  })

  it('closing with ESC (no path chosen) persists welcomeSeen=true doNotAutoShow=true', async () => {
    const user = userEvent.setup()
    const onClose = renderDialog()
    await user.keyboard('{Escape}')
    const state = readState()
    expect(state.welcomeSeen).toBe(true)
    expect(state.doNotAutoShow).toBe(true)
    expect(onClose).toHaveBeenCalled()
  })
})
