import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MantineProvider } from '@mantine/core'
import { ModalsProvider } from '@mantine/modals'
import { DatesProvider } from '@mantine/dates'
import React from 'react'

vi.mock('driver.js', () => ({
  driver: vi.fn(() => ({
    highlight: vi.fn(),
    destroy: vi.fn(),
  })),
}))

vi.mock('../../api/client', () => ({
  api: {
    getFirstRun: vi.fn(() => Promise.resolve({ first_run: false })),
    getSettings: vi.fn(() => Promise.resolve({ language: 'cs' })),
    updateSettings: vi.fn(() => Promise.resolve({ language: 'cs' })),
  },
  checkHealth: vi.fn(() => Promise.resolve(true)),
}))

vi.mock('../../pages/SetupWizard', () => ({
  SetupWizard: ({ onComplete }: { onComplete: () => void }) => (
    <button data-testid="wizard-complete" onClick={onComplete}>complete wizard</button>
  ),
}))

vi.mock('../../components/AppShell', () => ({
  AppShell: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
}))

vi.mock('../../components/MobileShell', () => ({
  MobileShell: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
}))

vi.mock('../../components/ApiHealthGuard', () => ({
  ApiHealthGuard: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

import { api } from '../../api/client'
import AppLayout from '../../App'
import { I18nProvider } from '../../i18n'
import { writeState, resetState } from '../persistence'

function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
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

function renderApp() {
  const queryClient = makeQueryClient()
  const router = createMemoryRouter(
    [
      {
        path: '/',
        element: (
          <I18nProvider>
            <AppLayout />
          </I18nProvider>
        ),
        children: [
          { index: true, element: <div data-testid="home">home</div> },
        ],
      },
    ],
    { initialEntries: ['/'] }
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

describe('mobile-gate', () => {
  beforeEach(() => {
    resetState()
    vi.clearAllMocks()
    setMatchMedia(false)
    vi.mocked(api.getFirstRun).mockResolvedValue({ first_run: false })
    vi.mocked(api.getSettings).mockResolvedValue({ language: 'cs' })
    vi.mocked(api.updateSettings).mockResolvedValue({ language: 'cs' })
  })

  afterEach(() => {
    setMatchMedia(false)
  })

  it('first run + desktop + welcomeSeen=false → dialog visible after wizard completes', async () => {
    vi.mocked(api.getFirstRun).mockResolvedValue({ first_run: true })
    renderApp()
    const user = userEvent.setup()
    await waitFor(() => expect(screen.getByTestId('wizard-complete')).toBeInTheDocument())
    await user.click(screen.getByTestId('wizard-complete'))
    await waitFor(() => expect(screen.getByText('Průvodce TidyBillem')).toBeInTheDocument())
  })

  it('first run + mobile + welcomeSeen=false → dialog NOT visible', async () => {
    setMatchMedia(true)
    vi.mocked(api.getFirstRun).mockResolvedValue({ first_run: true })
    renderApp()
    const user = userEvent.setup()
    await waitFor(() => expect(screen.getByTestId('wizard-complete')).toBeInTheDocument())
    await user.click(screen.getByTestId('wizard-complete'))
    await waitFor(() => expect(screen.getByTestId('home')).toBeInTheDocument())
    expect(screen.queryByText('Průvodce TidyBillem')).toBeNull()
  })

  it('non-first-run + desktop + welcomeSeen=false → dialog visible automatically', async () => {
    renderApp()
    await waitFor(() => expect(screen.getByText('Průvodce TidyBillem')).toBeInTheDocument())
  })

  it('non-first-run + desktop + welcomeSeen=true → dialog NOT auto-opened', async () => {
    writeState({ welcomeSeen: true, completedFlows: [], doNotAutoShow: false })
    renderApp()
    await waitFor(() => expect(screen.getByTestId('home')).toBeInTheDocument())
    expect(screen.queryByText('Průvodce TidyBillem')).toBeNull()
  })

  it('non-first-run + desktop + doNotAutoShow=true → dialog NOT auto-opened', async () => {
    writeState({ welcomeSeen: true, completedFlows: [], doNotAutoShow: true })
    renderApp()
    await waitFor(() => expect(screen.getByTestId('home')).toBeInTheDocument())
    expect(screen.queryByText('Průvodce TidyBillem')).toBeNull()
  })

})
