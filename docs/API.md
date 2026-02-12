# TidyBill REST API

## Running

```bash
# Build
go build -o tidybill ./cmd/tidybill/

# Start API server
./tidybill --gui
./tidybill --gui --port 9090   # custom port

# CLI mode (unchanged)
./tidybill
```

API runs on `http://localhost:8080` by default.
During development, React (Vite) runs on `:5173` and proxies `/api/*` to `:8080`.

---

## Endpoints

### Dashboard

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/dashboard/stats` | Get dashboard statistics |

**GET /api/dashboard/stats**

Response:
```json
{
  "total_revenue_month": 42350.00,
  "unpaid_count": 3,
  "unpaid_amount": 18500.00,
  "overdue_count": 1,
  "active_customers": 7,
  "invoices_this_month": 12
}
```

---

### Invoices

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/invoices` | List all invoices |
| POST | `/api/invoices` | Create new invoice |
| GET | `/api/invoices/{id}` | Get invoice by ID |
| PUT | `/api/invoices/{id}` | Update invoice |
| DELETE | `/api/invoices/{id}` | Delete invoice |
| PUT | `/api/invoices/{id}/status` | Update invoice status |
| POST | `/api/invoices/{id}/pdf` | Generate PDF for invoice |

**GET /api/invoices**

Query params:
- `status` — filter by status (`draft`, `created`, `sent`, `paid`, `overdue`, `partially_paid`, `cancelled`)
- `customer_id` — filter by customer

**POST /api/invoices**

```json
{
  "customer_id": "uuid",
  "supplier_id": "uuid",
  "bank_account_id": "uuid",
  "issue_date": "2025-02-11",
  "due_date": "2025-02-25",
  "currency": "CZK",
  "notes": "optional note",
  "items": [
    {
      "description": "Web development - January 2025",
      "quantity": 40,
      "unit": "hod",
      "unit_price": 800,
      "vat_rate": 21
    }
  ]
}
```

Invoice number and variable symbol are auto-generated.

**PUT /api/invoices/{id}/status**

```json
{
  "status": "sent"
}
```

**POST /api/invoices/{id}/pdf**

Generates PDF and returns path:
```json
{
  "path": "/home/user/.config/tidybill/pdfs/2025/VF25-00001.pdf"
}
```

---

### Customers

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/customers` | List all customers |
| POST | `/api/customers` | Create customer |
| GET | `/api/customers/{id}` | Get customer by ID |
| PUT | `/api/customers/{id}` | Update customer |
| DELETE | `/api/customers/{id}` | Delete customer |

**GET /api/customers**

Query params:
- `q` — search by name or IČO

**POST /api/customers**

```json
{
  "name": "Apertia s.r.o.",
  "ico": "87654321",
  "dic": "CZ87654321",
  "street": "Jiná ulice 456",
  "city": "Brno",
  "zip": "602 00",
  "country": "CZ",
  "email": "info@apertia.cz",
  "phone": "+420 123 456 789"
}
```

---

### Suppliers

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/suppliers` | List all suppliers |
| POST | `/api/suppliers` | Create supplier |
| GET | `/api/suppliers/{id}` | Get supplier by ID |
| PUT | `/api/suppliers/{id}` | Update supplier |
| DELETE | `/api/suppliers/{id}` | Delete supplier |

**POST /api/suppliers**

```json
{
  "name": "My Company s.r.o.",
  "ico": "12345678",
  "dic": "CZ12345678",
  "street": "Ulice 123",
  "city": "Praha 1",
  "zip": "110 00",
  "country": "CZ",
  "is_vat_payer": true,
  "invoice_prefix": "VF"
}
```

---

### Bank Accounts

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/suppliers/{id}/bank-accounts` | List bank accounts for supplier |
| POST | `/api/suppliers/{id}/bank-accounts` | Create bank account for supplier |

**POST /api/suppliers/{id}/bank-accounts**

```json
{
  "name": "Main CZK account",
  "account_number": "1234567890/0100",
  "iban": "CZ6508000000001234567890",
  "swift": "GIBACZPX",
  "currency": "CZK",
  "is_default": true
}
```

---

## Error Responses

All errors return JSON:
```json
{
  "error": "description of what went wrong"
}
```

HTTP status codes:
- `400` — Bad request (missing required fields, invalid JSON)
- `404` — Resource not found
- `500` — Internal server error
