import { describe, it, expect, beforeEach, vi } from 'vitest'
import { readState, writeState, markCompleted, markWelcomeSeen, resetState } from '../persistence'
import { DEFAULT_STATE } from '../types'

const KEY = 'tidybill.tour.v1'

beforeEach(() => {
  localStorage.clear()
})

describe('readState', () => {
  it('returns DEFAULT_STATE when localStorage is empty', () => {
    expect(readState()).toEqual(DEFAULT_STATE)
  })

  it('returns DEFAULT_STATE when localStorage contains malformed JSON', () => {
    localStorage.setItem(KEY, 'not-valid-json{')
    expect(readState()).toEqual(DEFAULT_STATE)
  })

  it('fills missing fields with defaults when some keys are absent', () => {
    localStorage.setItem(KEY, JSON.stringify({ welcomeSeen: true }))
    const state = readState()
    expect(state.welcomeSeen).toBe(true)
    expect(state.completedFlows).toEqual([])
    expect(state.doNotAutoShow).toBe(false)
  })

  it('filters out unknown flow ids from completedFlows', () => {
    localStorage.setItem(KEY, JSON.stringify({ completedFlows: ['create-invoice', 'unknown-flow', 'advanced'] }))
    const state = readState()
    expect(state.completedFlows).toEqual(['create-invoice', 'advanced'])
  })

  it('coerces completedFlows to [] when it is not an array', () => {
    localStorage.setItem(KEY, JSON.stringify({ completedFlows: 'create-invoice' }))
    const state = readState()
    expect(state.completedFlows).toEqual([])
  })

  it('round-trips state via writeState then readState', () => {
    const original = { welcomeSeen: true, completedFlows: ['just-show-me' as const], doNotAutoShow: true }
    writeState(original)
    expect(readState()).toEqual(original)
  })
})

describe('writeState', () => {
  it('does not throw when localStorage.setItem throws (quota exceeded)', () => {
    const original = Storage.prototype.setItem
    Storage.prototype.setItem = vi.fn().mockImplementation(() => { throw new Error('QuotaExceededError') })
    expect(() => writeState(DEFAULT_STATE)).not.toThrow()
    Storage.prototype.setItem = original
  })
})

describe('markCompleted', () => {
  it('sets welcomeSeen to true and adds the flow id', () => {
    const state = markCompleted('create-invoice')
    expect(state.welcomeSeen).toBe(true)
    expect(state.completedFlows).toContain('create-invoice')
  })

  it('is idempotent — duplicate calls do not duplicate the id', () => {
    markCompleted('create-invoice')
    const state = markCompleted('create-invoice')
    expect(state.completedFlows.filter((f) => f === 'create-invoice')).toHaveLength(1)
  })

  it('preserves other flows already completed', () => {
    markCompleted('just-show-me')
    const state = markCompleted('advanced')
    expect(state.completedFlows).toContain('just-show-me')
    expect(state.completedFlows).toContain('advanced')
  })
})

describe('markWelcomeSeen', () => {
  it('sets welcomeSeen=true and doNotAutoShow=true when called with true', () => {
    const state = markWelcomeSeen(true)
    expect(state.welcomeSeen).toBe(true)
    expect(state.doNotAutoShow).toBe(true)
  })

  it('sets welcomeSeen=true and doNotAutoShow=false when called with false', () => {
    const state = markWelcomeSeen(false)
    expect(state.welcomeSeen).toBe(true)
    expect(state.doNotAutoShow).toBe(false)
  })
})

describe('resetState', () => {
  it('removes the key; subsequent readState returns DEFAULT_STATE', () => {
    writeState({ welcomeSeen: true, completedFlows: ['advanced'], doNotAutoShow: true })
    resetState()
    expect(readState()).toEqual(DEFAULT_STATE)
  })
})
