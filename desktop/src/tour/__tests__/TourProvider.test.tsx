import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, fireEvent, act } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MantineProvider } from '@mantine/core'

// Minimal subset of Config we need to inspect in tests.
interface TestDriverOpts { animate?: boolean; onDestroyed?: () => void }

let capturedDriverOpts: TestDriverOpts | null = null
const mockHighlight = vi.fn()
const mockDestroy = vi.fn()

vi.mock('driver.js', () => ({
  driver: vi.fn((opts: TestDriverOpts) => {
    capturedDriverOpts = opts
    return {
      highlight: mockHighlight,
      destroy: () => {
        mockDestroy()
        capturedDriverOpts?.onDestroyed?.()
      },
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
import * as persistence from '../persistence'

function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
}

function TestConsumer({ anchors = [] }: { anchors?: string[] }) {
  const { startFlow, stop, isRunning, currentFlowId } = useTour()
  return (
    <>
      <div data-testid="is-running">{String(isRunning)}</div>
      <div data-testid="flow-id">{currentFlowId ?? 'none'}</div>
      {anchors.map((a) => <div key={a} data-tour={a} />)}
      <button data-testid="start-create" onClick={() => startFlow('create-invoice')}>start</button>
      <button data-testid="start-unknown" onClick={() => (startFlow as (id: string) => void)('no-such-flow')}>unknown</button>
      <button data-testid="stop" onClick={() => stop()}>stop</button>
    </>
  )
}

function renderProvider(anchors: string[] = []) {
  render(
    <QueryClientProvider client={makeQueryClient()}>
      <MantineProvider>
        <MemoryRouter>
          <I18nProvider>
            <TourProvider>
              <TestConsumer anchors={anchors} />
            </TourProvider>
          </I18nProvider>
        </MemoryRouter>
      </MantineProvider>
    </QueryClientProvider>
  )
}

const ALL_CREATE_ANCHORS = [
  'nav-new-invoice',
  'invoice-supplier-select',
  'invoice-customer-select',
  'invoice-number-input',
  'invoice-dates',
  'invoice-items-table',
  'invoice-create-submit',
]

describe('TourProvider', () => {
  beforeEach(() => {
    persistence.resetState()
    vi.clearAllMocks()
    capturedDriverOpts = null
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('startFlow sets isRunning=true and currentFlowId', async () => {
    renderProvider(['nav-new-invoice'])
    fireEvent.click(screen.getByTestId('start-create'))
    await waitFor(() => {
      expect(screen.getByTestId('is-running')).toHaveTextContent('true')
      expect(screen.getByTestId('flow-id')).toHaveTextContent('create-invoice')
    })
  })

  it('flow completes through all steps → markCompleted called', async () => {
    const markSpy = vi.spyOn(persistence, 'markCompleted')
    renderProvider(ALL_CREATE_ANCHORS)
    fireEvent.click(screen.getByTestId('start-create'))

    for (let i = 0; i < 7; i++) {
      await waitFor(() => expect(mockHighlight).toHaveBeenCalledTimes(i + 1))
      mockHighlight.mock.calls[i][0].popover.onNextClick()
    }

    await waitFor(() => expect(markSpy).toHaveBeenCalledWith('create-invoice'))
    expect(persistence.readState().completedFlows).toContain('create-invoice')
  })

  it('closing with X on the last step does NOT mark the flow as completed', async () => {
    const markSpy = vi.spyOn(persistence, 'markCompleted')
    renderProvider(ALL_CREATE_ANCHORS)
    fireEvent.click(screen.getByTestId('start-create'))

    // Advance through Next 6 times so the LAST popover (idx=6) is showing.
    for (let i = 0; i < 6; i++) {
      await waitFor(() => expect(mockHighlight).toHaveBeenCalledTimes(i + 1))
      mockHighlight.mock.calls[i][0].popover.onNextClick()
    }
    await waitFor(() => expect(mockHighlight).toHaveBeenCalledTimes(7))

    // User presses X on the last step instead of Done.
    fireEvent.click(screen.getByTestId('stop'))

    await waitFor(() => expect(screen.getByTestId('is-running')).toHaveTextContent('false'))
    expect(markSpy).not.toHaveBeenCalled()
    expect(persistence.readState().completedFlows).not.toContain('create-invoice')
  })

  it('startFlow with unknown id returns silently without error', () => {
    renderProvider()
    expect(() => fireEvent.click(screen.getByTestId('start-unknown'))).not.toThrow()
    expect(screen.getByTestId('is-running')).toHaveTextContent('false')
  })

  it('anchor missing on step → console.warn called (fake timers)', async () => {
    vi.useFakeTimers()
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
    renderProvider([]) // no anchors in DOM

    fireEvent.click(screen.getByTestId('start-create'))

    // Advance past waitForAnchor's 2000ms timeout for the first step
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2100)
    })

    expect(warnSpy).toHaveBeenCalledWith('tour: anchor missing', 'nav-new-invoice')
    warnSpy.mockRestore()
  })

  it('stop() destroys driver and resets isRunning to false', async () => {
    renderProvider(['nav-new-invoice'])
    fireEvent.click(screen.getByTestId('start-create'))

    await waitFor(() => expect(screen.getByTestId('is-running')).toHaveTextContent('true'))

    fireEvent.click(screen.getByTestId('stop'))

    await waitFor(() => {
      expect(screen.getByTestId('is-running')).toHaveTextContent('false')
      expect(screen.getByTestId('flow-id')).toHaveTextContent('none')
    })
    expect(mockDestroy).toHaveBeenCalled()
  })

  it('prefers-reduced-motion: reduce → driver created with animate=false', async () => {
    const original = window.matchMedia
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: (query: string) => ({
        matches: query.includes('reduce'),
        media: query, onchange: null,
        addListener: () => {}, removeListener: () => {},
        addEventListener: () => {}, removeEventListener: () => {},
        dispatchEvent: () => false,
      }),
    })

    renderProvider(['nav-new-invoice'])
    fireEvent.click(screen.getByTestId('start-create'))

    await waitFor(() => expect(capturedDriverOpts).not.toBeNull())
    expect(capturedDriverOpts!.animate).toBe(false)

    Object.defineProperty(window, 'matchMedia', { writable: true, value: original })
  })
})
