import { useState } from 'react'
import { Routes, Route } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Center, Loader } from '@mantine/core'
import { AppShell } from './components/AppShell'
import { MobileShell } from './components/MobileShell'
import { useIsMobile } from './hooks/useIsMobile'
import { MorePage } from './pages/MorePage'
import { ApiHealthGuard } from './components/ApiHealthGuard'
import { SetupWizard } from './pages/SetupWizard'
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
import { Templates } from './pages/Templates'
import { TemplateEditor } from './pages/TemplateEditor'
import { About } from './pages/About'
import { MobileAbout } from './pages/mobile/About'
import { api } from './api/client'

export default function App() {
  const [wizardDone, setWizardDone] = useState(false)

  const { data: firstRunData, isLoading, isError } = useQuery({
    queryKey: ['first-run'],
    queryFn: api.getFirstRun,
    retry: 3,
    retryDelay: 1000,
  })

  const showWizard = !wizardDone && firstRunData?.first_run === true
  const isMobile = useIsMobile()
  const Shell = isMobile ? MobileShell : AppShell
  const InvoiceListPage = isMobile ? MobileInvoiceList : InvoiceList
  const InvoiceCreatePage = isMobile ? MobileInvoiceCreate : InvoiceCreate
  const InvoiceDetailPage = isMobile ? MobileInvoiceDetail : InvoiceDetail
  const InvoiceEditPage = isMobile ? MobileInvoiceEdit : InvoiceEdit
  const AboutPage = isMobile ? MobileAbout : About

  return (
    <ApiHealthGuard>
      {(isLoading || (isError && !firstRunData)) ? (
        <Center h="100vh">
          <Loader />
        </Center>
      ) : showWizard ? (
        <SetupWizard onComplete={() => setWizardDone(true)} />
      ) : (
        <Shell>
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/invoices" element={<InvoiceListPage />} />
            <Route path="/invoices/new" element={<InvoiceCreatePage />} />
            <Route path="/invoices/:id" element={<InvoiceDetailPage />} />
            <Route path="/invoices/:id/edit" element={<InvoiceEditPage />} />
            <Route path="/customers" element={<CustomerList />} />
            <Route path="/suppliers" element={<SupplierList />} />
            <Route path="/items" element={<ItemCatalog />} />
            <Route path="/templates" element={<Templates />} />
            <Route path="/template-editor/:id" element={<TemplateEditor />} />
            <Route path="/settings" element={<Settings />} />
            <Route path="/more" element={<MorePage />} />
            <Route path="/about" element={<AboutPage />} />
          </Routes>
        </Shell>
      )}
    </ApiHealthGuard>
  )
}
