import { createBrowserRouter } from 'react-router-dom'
import AppLayout from './App'
import { ResponsivePage } from './App'
import { Dashboard } from './pages/Dashboard'
import { InvoiceList } from './pages/InvoiceList'
import { MobileInvoiceList } from './pages/mobile/InvoiceList'
import { InvoiceCreate } from './pages/InvoiceCreate'
import { MobileInvoiceCreate } from './pages/mobile/InvoiceCreate'
import { InvoiceEdit } from './pages/InvoiceEdit'
import { MobileInvoiceEdit } from './pages/mobile/InvoiceEdit'
import { InvoiceDetail } from './pages/InvoiceDetail'
import { MobileInvoiceDetail } from './pages/mobile/InvoiceDetail'
import { CustomerList } from './pages/CustomerList'
import { SupplierList } from './pages/SupplierList'
import { ItemCatalog } from './pages/ItemCatalog'
import { Settings } from './pages/Settings'
import { Automatizace } from './pages/Automatizace'
import { Templates } from './pages/Templates'
import { TemplateEditor } from './pages/TemplateEditor'
import { SyncPage } from './pages/SyncPage'
import { MorePage } from './pages/MorePage'
import { About } from './pages/About'
import { MobileAbout } from './pages/mobile/About'
import { HelpPage } from './pages/HelpPage'

export const router = createBrowserRouter([
  {
    element: <AppLayout />,
    children: [
      { path: '/', element: <Dashboard /> },
      { path: '/invoices', element: <ResponsivePage Desktop={InvoiceList} Mobile={MobileInvoiceList} /> },
      { path: '/invoices/new', element: <ResponsivePage Desktop={InvoiceCreate} Mobile={MobileInvoiceCreate} /> },
      { path: '/invoices/:id', element: <ResponsivePage Desktop={InvoiceDetail} Mobile={MobileInvoiceDetail} /> },
      { path: '/invoices/:id/edit', element: <ResponsivePage Desktop={InvoiceEdit} Mobile={MobileInvoiceEdit} /> },
      { path: '/customers', element: <CustomerList /> },
      { path: '/suppliers', element: <SupplierList /> },
      { path: '/items', element: <ItemCatalog /> },
      { path: '/templates', element: <Templates /> },
      { path: '/template-editor/:id', element: <TemplateEditor /> },
      { path: '/settings', element: <Settings /> },
      { path: '/automatizace', element: <Automatizace /> },
      { path: '/sync', element: <SyncPage /> },
      { path: '/help', element: <HelpPage /> },
      { path: '/more', element: <MorePage /> },
      { path: '/about', element: <ResponsivePage Desktop={About} Mobile={MobileAbout} /> },
    ],
  },
])
