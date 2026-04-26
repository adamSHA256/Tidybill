import { DEFAULT_STATE, type TourState, type FlowId } from './types'
const KEY = 'tidybill.tour.v1'

export function readState(): TourState {
  try {
    const raw = localStorage.getItem(KEY)
    if (!raw) return { ...DEFAULT_STATE }
    const parsed = JSON.parse(raw) as Partial<TourState>
    return {
      welcomeSeen: typeof parsed.welcomeSeen === 'boolean' ? parsed.welcomeSeen : DEFAULT_STATE.welcomeSeen,
      completedFlows: Array.isArray(parsed.completedFlows)
        ? parsed.completedFlows.filter((f): f is FlowId =>
            f === 'create-invoice' || f === 'just-show-me' || f === 'advanced')
        : [],
      doNotAutoShow: typeof parsed.doNotAutoShow === 'boolean' ? parsed.doNotAutoShow : DEFAULT_STATE.doNotAutoShow,
    }
  } catch {
    return { ...DEFAULT_STATE }
  }
}

export function writeState(state: TourState): void {
  try { localStorage.setItem(KEY, JSON.stringify(state)) } catch { /* quota exceeded — ignore */ }
}

export function markCompleted(id: FlowId): TourState {
  const s = readState()
  const next: TourState = {
    ...s,
    welcomeSeen: true,
    completedFlows: s.completedFlows.includes(id) ? s.completedFlows : [...s.completedFlows, id],
  }
  writeState(next)
  return next
}

export function markWelcomeSeen(doNotAutoShow: boolean): TourState {
  const s = readState()
  const next: TourState = { ...s, welcomeSeen: true, doNotAutoShow }
  writeState(next)
  return next
}

export function resetState(): void {
  localStorage.removeItem(KEY)
}
