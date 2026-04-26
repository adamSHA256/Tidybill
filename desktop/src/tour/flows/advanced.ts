import type { TourFlow } from '../types'
export const advancedFlow: TourFlow = {
  id: 'advanced',
  titleKey: 'tour.advanced.title',
  steps: [
    { anchor: 'page-templates',    route: '/templates',   titleKey: 'tour.advanced.step1.title', bodyKey: 'tour.advanced.step1.body' },
    { anchor: 'page-automatizace', route: '/automatizace', titleKey: 'tour.advanced.step2.title', bodyKey: 'tour.advanced.step2.body' },
    { anchor: 'customer-add-btn',  route: '/customers',   titleKey: 'tour.advanced.step3.title', bodyKey: 'tour.advanced.step3.body' },
    { anchor: 'page-suppliers',    route: '/suppliers',   titleKey: 'tour.advanced.step4.title', bodyKey: 'tour.advanced.step4.body' },
    { anchor: 'page-sync',         route: '/sync',        titleKey: 'tour.advanced.step5.title', bodyKey: 'tour.advanced.step5.body' },
    { anchor: 'page-settings',     route: '/settings',    titleKey: 'tour.advanced.step6.title', bodyKey: 'tour.advanced.step6.body' },
  ],
}
