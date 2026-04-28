import { createInvoiceFlow } from './createInvoice'
import { justShowMeFlow } from './justShowMe'
import { advancedFlow } from './advanced'
import { createInvoiceFlowMobile } from './createInvoice.mobile'
import { justShowMeFlowMobile } from './justShowMe.mobile'
import type { FlowId, TourFlow } from '../types'

const DESKTOP: Record<FlowId, TourFlow> = {
  'create-invoice': createInvoiceFlow,
  'just-show-me': justShowMeFlow,
  'advanced': advancedFlow,
}

// Mobile-specific flows. `advanced` is omitted because its anchors live on
// pages shared between desktop and mobile, so the desktop flow works as-is.
const MOBILE: Partial<Record<FlowId, TourFlow>> = {
  'create-invoice': createInvoiceFlowMobile,
  'just-show-me': justShowMeFlowMobile,
}

export function getFlow(id: FlowId, isMobile = false): TourFlow | null {
  if (isMobile && MOBILE[id]) return MOBILE[id]!
  return DESKTOP[id] ?? null
}

export const ALL_FLOWS = DESKTOP
export const ALL_FLOWS_MOBILE: Record<FlowId, TourFlow> = {
  ...DESKTOP,
  ...MOBILE,
}
