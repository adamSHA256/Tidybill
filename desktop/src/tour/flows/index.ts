import { createInvoiceFlow } from './createInvoice'
import { justShowMeFlow } from './justShowMe'
import { advancedFlow } from './advanced'
import type { FlowId, TourFlow } from '../types'
const ALL: Record<FlowId, TourFlow> = {
  'create-invoice': createInvoiceFlow,
  'just-show-me': justShowMeFlow,
  'advanced': advancedFlow,
}
export function getFlow(id: FlowId): TourFlow | null { return ALL[id] ?? null }
export const ALL_FLOWS = ALL
