import type { TourFlow } from '../types'
export const justShowMeFlow: TourFlow = {
  id: 'just-show-me',
  titleKey: 'tour.just_show_me.title',
  steps: [
    { anchor: 'dashboard-stats', route: '/', titleKey: 'tour.just_show_me.step1.title', bodyKey: 'tour.just_show_me.step1.body' },
    { anchor: 'nav-invoices', route: '/', titleKey: 'tour.just_show_me.step2.title', bodyKey: 'tour.just_show_me.step2.body' },
    { anchor: 'page-customers', route: '/customers', titleKey: 'tour.just_show_me.step3.title', bodyKey: 'tour.just_show_me.step3.body' },
    { anchor: 'page-suppliers', route: '/suppliers', titleKey: 'tour.just_show_me.step4.title', bodyKey: 'tour.just_show_me.step4.body' },
    { anchor: 'page-items', route: '/items', titleKey: 'tour.just_show_me.step5.title', bodyKey: 'tour.just_show_me.step5.body' },
    { anchor: 'page-templates', route: '/templates', titleKey: 'tour.just_show_me.step6.title', bodyKey: 'tour.just_show_me.step6.body' },
    { anchor: 'nav-settings', route: '/templates', titleKey: 'tour.just_show_me.step7.title', bodyKey: 'tour.just_show_me.step7.body' },
  ],
}
