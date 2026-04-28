import type { TourFlow } from '../types'
export const createInvoiceFlowMobile: TourFlow = {
  id: 'create-invoice',
  titleKey: 'tour.create_invoice.title',
  steps: [
    { anchor: 'm-tab-new-invoice',     route: '/',             titleKey: 'tour.create_invoice.step1.title', bodyKey: 'tour.create_invoice.mobile.step1.body' },
    { anchor: 'm-invoice-supplier',    route: '/invoices/new', titleKey: 'tour.create_invoice.step2.title', bodyKey: 'tour.create_invoice.step2.body' },
    { anchor: 'm-invoice-customer',    route: '/invoices/new', titleKey: 'tour.create_invoice.step3.title', bodyKey: 'tour.create_invoice.step3.body' },
    { anchor: 'm-invoice-number',      route: '/invoices/new', titleKey: 'tour.create_invoice.step4.title', bodyKey: 'tour.create_invoice.step4.body' },
    { anchor: 'm-invoice-dates',       route: '/invoices/new', titleKey: 'tour.create_invoice.step5.title', bodyKey: 'tour.create_invoice.step5.body' },
    { anchor: 'm-invoice-items',       route: '/invoices/new', titleKey: 'tour.create_invoice.step6.title', bodyKey: 'tour.create_invoice.step6.body' },
    { anchor: 'm-invoice-create-submit', route: '/invoices/new', titleKey: 'tour.create_invoice.step7.title', bodyKey: 'tour.create_invoice.step7.body' },
  ],
}
