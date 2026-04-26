import type { TourFlow } from '../types'
export const createInvoiceFlow: TourFlow = {
  id: 'create-invoice',
  titleKey: 'tour.create_invoice.title',
  steps: [
    { anchor: 'nav-new-invoice', route: '/', titleKey: 'tour.create_invoice.step1.title', bodyKey: 'tour.create_invoice.step1.body' },
    { anchor: 'invoice-supplier-select', route: '/invoices/new', titleKey: 'tour.create_invoice.step2.title', bodyKey: 'tour.create_invoice.step2.body' },
    { anchor: 'invoice-customer-select', route: '/invoices/new', titleKey: 'tour.create_invoice.step3.title', bodyKey: 'tour.create_invoice.step3.body' },
    { anchor: 'invoice-number-input', route: '/invoices/new', titleKey: 'tour.create_invoice.step4.title', bodyKey: 'tour.create_invoice.step4.body' },
    { anchor: 'invoice-dates', route: '/invoices/new', titleKey: 'tour.create_invoice.step5.title', bodyKey: 'tour.create_invoice.step5.body' },
    { anchor: 'invoice-items-table', route: '/invoices/new', titleKey: 'tour.create_invoice.step6.title', bodyKey: 'tour.create_invoice.step6.body' },
    { anchor: 'invoice-create-submit', route: '/invoices/new', titleKey: 'tour.create_invoice.step7.title', bodyKey: 'tour.create_invoice.step7.body' },
  ],
}
