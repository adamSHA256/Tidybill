import { describe, it, expect } from 'vitest'
import { getFlow } from '../flows'

describe('getFlow device dispatch', () => {
  it('returns desktop flow when isMobile is false', () => {
    const f = getFlow('create-invoice', false)
    expect(f?.steps[0].anchor).toBe('nav-new-invoice')
  })

  it('returns mobile flow when isMobile is true and mobile variant exists', () => {
    const f = getFlow('create-invoice', true)
    expect(f?.steps[0].anchor).toBe('m-tab-new-invoice')
  })

  it('returns mobile just-show-me variant when isMobile is true', () => {
    const f = getFlow('just-show-me', true)
    expect(f?.steps[1].anchor).toBe('m-tab-invoices')
    const last = f!.steps[f!.steps.length - 1]
    expect(last.anchor).toBe('m-more-settings')
    expect(last.route).toBe('/more')
  })

  it('falls back to desktop flow for `advanced` on mobile (no mobile variant)', () => {
    const onMobile = getFlow('advanced', true)
    const onDesktop = getFlow('advanced', false)
    expect(onMobile).toBe(onDesktop)
    expect(onMobile?.steps[0].anchor).toBe('page-templates')
  })

  it('defaults to desktop when isMobile is omitted', () => {
    const f = getFlow('create-invoice')
    expect(f?.steps[0].anchor).toBe('nav-new-invoice')
  })
})
