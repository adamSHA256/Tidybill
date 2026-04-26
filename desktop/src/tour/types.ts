export type FlowId = 'create-invoice' | 'just-show-me' | 'advanced'

export interface TourStep {
  /** data-tour attribute value this step anchors to. */
  anchor: string
  /** i18n key for the tooltip title. */
  titleKey: string
  /** i18n key for the tooltip body. */
  bodyKey: string
  /** Optional: navigate to this route before showing the step. */
  route?: string
  /** Optional: CSS selector that must be visible before showing; waits up to 2000ms then skips. */
  waitFor?: string
}

export interface TourFlow {
  id: FlowId
  titleKey: string
  steps: TourStep[]
}

export interface TourState {
  /** Has the user seen the welcome dialog at least once? */
  welcomeSeen: boolean
  /** Flows the user has finished (reached last step + pressed Done). */
  completedFlows: FlowId[]
  /** True if user checked "don't show again" in welcome dialog. */
  doNotAutoShow: boolean
}

export const DEFAULT_STATE: TourState = {
  welcomeSeen: false,
  completedFlows: [],
  doNotAutoShow: false,
}
