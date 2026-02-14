import { Routes, Route } from 'react-router-dom'
import { AppShell } from './components/AppShell'
import { ApiHealthGuard } from './components/ApiHealthGuard'
import { Dashboard } from './pages/Dashboard'
import { InvoiceList } from './pages/InvoiceList'
import { InvoiceCreate } from './pages/InvoiceCreate'
import { InvoiceDetail } from './pages/InvoiceDetail'
import { CustomerList } from './pages/CustomerList'
import { SupplierList } from './pages/SupplierList'
import { ItemCatalog } from './pages/ItemCatalog'
import { Settings } from './pages/Settings'
import { Templates } from './pages/Templates'

export default function App() {
  return (
    <ApiHealthGuard>
      <AppShell>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/invoices" element={<InvoiceList />} />
          <Route path="/invoices/new" element={<InvoiceCreate />} />
          <Route path="/invoices/:id" element={<InvoiceDetail />} />
          <Route path="/customers" element={<CustomerList />} />
          <Route path="/suppliers" element={<SupplierList />} />
          <Route path="/items" element={<ItemCatalog />} />
          <Route path="/templates" element={<Templates />} />
          <Route path="/settings" element={<Settings />} />
        </Routes>
      </AppShell>
    </ApiHealthGuard>
  )
}
