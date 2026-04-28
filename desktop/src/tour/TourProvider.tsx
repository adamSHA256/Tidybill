import { createContext, useContext, useRef, useState, useCallback, useEffect, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { driver } from 'driver.js'
import type { Driver } from 'driver.js'
import 'driver.js/dist/driver.css'
import './driver.css'
import { useT } from '../i18n'
import { useIsMobile } from '../hooks/useIsMobile'
import { getFlow } from './flows'
import type { FlowId, TourStep } from './types'
import { markCompleted } from './persistence'

interface TourCtx {
  startFlow: (id: FlowId) => void
  stop: () => void
  isRunning: boolean
  currentFlowId: FlowId | null
}
const Ctx = createContext<TourCtx>({
  startFlow: () => {}, stop: () => {}, isRunning: false, currentFlowId: null,
})

async function waitForAnchor(anchor: string, timeoutMs: number): Promise<HTMLElement | null> {
  const selector = `[data-tour="${anchor}"]`
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    const el = document.querySelector<HTMLElement>(selector)
    if (el) return el
    await new Promise((r) => setTimeout(r, 60))
  }
  return null
}

function safeDestroy(d: Driver | null) {
  try { d?.destroy() } catch { /* noop */ }
}

export function TourProvider({ children }: { children: ReactNode }) {
  const { t } = useT()
  const navigate = useNavigate()
  const isMobile = useIsMobile()
  const driverRef = useRef<Driver | null>(null)
  const cancelledRef = useRef(false)
  const [currentFlowId, setCurrentFlowId] = useState<FlowId | null>(null)
  const [isRunning, setIsRunning] = useState(false)

  const stop = useCallback(() => {
    safeDestroy(driverRef.current)
  }, [])

  const startFlow = useCallback((id: FlowId) => {
    const flow = getFlow(id, isMobile)
    if (!flow) return
    safeDestroy(driverRef.current)
    cancelledRef.current = false

    const prefersReduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    let idx = 0

    const runStep = async () => {
      if (cancelledRef.current) return
      if (idx >= flow.steps.length) {
        safeDestroy(driverRef.current)
        return
      }
      const step: TourStep = flow.steps[idx]
      if (step.route && window.location.pathname !== step.route) {
        navigate(step.route)
      }
      const waitSelector = step.waitFor ?? step.anchor
      const el = await waitForAnchor(waitSelector, 2000)
      if (cancelledRef.current) return
      if (!el) {
        console.warn('tour: anchor missing', step.anchor)
        idx += 1
        return runStep()
      }
      driverRef.current?.highlight({
        element: `[data-tour="${step.anchor}"]`,
        popover: {
          title: t(step.titleKey),
          description: t(step.bodyKey),
          showButtons: ['next', 'close'],
          nextBtnText: t('tour.next'),
          doneBtnText: t('tour.done'),
          prevBtnText: t('tour.prev'),
          progressText: `${idx + 1} / ${flow.steps.length}`,
          onNextClick: () => { idx += 1; runStep() },
          onCloseClick: () => safeDestroy(driverRef.current),
        },
      })
    }

    driverRef.current = driver({
      animate: !prefersReduced,
      allowClose: true,
      overlayOpacity: 0.5,
      stagePadding: isMobile ? 4 : 6,
      smoothScroll: true,
      popoverClass: isMobile ? 'tb-tour-mobile' : undefined,
      onDestroyed: () => {
        cancelledRef.current = true
        setIsRunning(false)
        // idx is incremented past the last step only via Next/Done on the final
        // popover. Closing with ✕ on the last step leaves idx at flow.steps.length - 1,
        // which must NOT count as completion.
        const completed = idx >= flow.steps.length
        if (completed) markCompleted(id)
        setCurrentFlowId(null)
      },
    })
    setCurrentFlowId(id)
    setIsRunning(true)
    runStep()
  }, [navigate, t, isMobile])

  useEffect(() => () => { safeDestroy(driverRef.current) }, [])

  return <Ctx.Provider value={{ startFlow, stop, isRunning, currentFlowId }}>{children}</Ctx.Provider>
}

// eslint-disable-next-line react-refresh/only-export-components
export function useTour() { return useContext(Ctx) }
