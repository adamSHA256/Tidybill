import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, waitFor, fireEvent, act } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MantineProvider } from '@mantine/core'

interface TestDriverOpts { animate?: boolean; onDestroyed?: () => void }

let capturedDriverOpts: TestDriverOpts | null = null
const mockHighlight = vi.fn()

vi.mock('driver.js', () => ({
  driver: vi.fn((opts: TestDriverOpts) => {
    capturedDriverOpts = opts
    return {
      highlight: mockHighlight,
      destroy: () => { capturedDriverOpts?.onDestroyed?.() },
    }
  }),
}))

vi.mock('../../api/client', () => ({
  api: {
    getSettings: () => Promise.resolve({}),
    updateSettings: () => Promise.resolve({}),
  },
}))

import { TourProvider, useTour } from '../TourProvider'
import { I18nProvider } from '../../i18n'
import { resetState } from '../persistence'

function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
}

// Renders anchors for a given set and exposes tour controls.
function TourHarness({ anchors }: { anchors: string[] }) {
  const { startFlow } = useTour()
  return (
    <>
      {anchors.map((a) => <div key={a} data-tour={a} />)}
      <button data-testid="start-create" onClick={() => startFlow('create-invoice')}>start</button>
    </>
  )
}

function renderWithRouter(anchors: string[], initialPath = '/') {
  const queryClient = makeQueryClient()
  const router = createMemoryRouter(
    [
      {
        path: '/',
        element: (
          <I18nProvider>
            <TourProvider>
              <TourHarness anchors={anchors} />
            </TourProvider>
          </I18nProvider>
        ),
      },
      // Route for invoices/new — same anchors stay in DOM because TourHarness is at '/'
      // but navigate() changes the router location, which we can assert.
    ],
    { initialEntries: [initialPath] }
  )

  render(
    <QueryClientProvider client={queryClient}>
      <MantineProvider>
        <RouterProvider router={router} />
      </MantineProvider>
    </QueryClientProvider>
  )

  return router
}

describe('route-handling integration', () => {
  beforeEach(() => {
    resetState()
    vi.clearAllMocks()
    capturedDriverOpts = null
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('flow navigates to step route and highlights anchor after it mounts', async () => {
    // create-invoice: step 1 at '/', step 2 at '/invoices/new'
    // All anchors are in the DOM so waitForAnchor succeeds on any route.
    const router = renderWithRouter([
      'nav-new-invoice',
      'invoice-supplier-select',
    ])

    fireEvent.click(document.querySelector('[data-testid="start-create"]')!)

    // Step 1 (nav-new-invoice at '/') highlighted first
    await waitFor(() => expect(mockHighlight).toHaveBeenCalledTimes(1))
    expect(router.state.location.pathname).toBe('/')

    // Advance to step 2 — TourProvider calls navigate('/invoices/new')
    mockHighlight.mock.calls[0][0].popover.onNextClick()

    await waitFor(() => expect(mockHighlight).toHaveBeenCalledTimes(2))
    // Navigation happened
    expect(router.state.location.pathname).toBe('/invoices/new')
    // Step 2 anchor highlighted (proves waitForAnchor found it after navigation)
    expect(mockHighlight.mock.calls[1][0].element).toBe('[data-tour="invoice-supplier-select"]')
  })

  it('anchor missing on next-step route (>2s) → step skipped, flow continues (fake timers)', async () => {
    vi.useFakeTimers()
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    // Only provide anchor for step 1; step 2+ have no anchors → will timeout
    renderWithRouter(['nav-new-invoice'])

    fireEvent.click(document.querySelector('[data-testid="start-create"]')!)

    // Step 1: anchor found, highlighted immediately
    await act(async () => { await vi.advanceTimersByTimeAsync(100) })
    expect(mockHighlight).toHaveBeenCalledTimes(1)

    // Advance to step 2 by calling onNextClick on step 1
    mockHighlight.mock.calls[0][0].popover.onNextClick()

    // Step 2 anchor missing — waitForAnchor times out after 2000ms
    await act(async () => { await vi.advanceTimersByTimeAsync(2100) })

    // console.warn fired for the missing anchor on '/invoices/new'
    expect(warnSpy).toHaveBeenCalledWith('tour: anchor missing', 'invoice-supplier-select')

    warnSpy.mockRestore()
  })
})
