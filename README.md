# 🧾 TidyBill

**Clean invoices, zero clutter.** A local-first invoice manager for Czech and Slovak freelancers.

Single binary. No cloud. No subscription. Just your invoices, tidy and organized.

---

## ✨ Features

- **Full CLI interface** — create invoices, manage customers & suppliers from terminal
- **PDF generation** — professional invoices with QR payment codes (SPAYD format)
- **SQLite database** — single-file storage, fast and portable
- **Multi-supplier** — manage multiple companies from one installation
- **Multi-currency** — CZK, EUR, and others with per-supplier bank accounts
- **Smart numbering** — automatic invoice numbers (VF26-00001 format)
- **Cross-platform** — runs on Linux and Windows

## 🛠 Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go |
| Database | SQLite (pure Go, `modernc.org/sqlite`) |
| PDF | Maroto v2 (pure Go, built-in QR codes) |
| Distribution | Single binary, no dependencies |

## 🚀 Quick Start

```bash
# Build
make build

# Run
./tidybill
```

On first run, TidyBill walks you through setting up your company profile and bank account.

### Build for other platforms

```bash
make build-linux     # Linux amd64
make build-windows   # Windows amd64
make build-all       # Both
```

## 📋 Usage

TidyBill uses an interactive terminal menu:

```
╔════════════════════════════════════════════════════════════╗
║                      TIDYBILL v0.1                         ║
║  Firma: Your Company s.r.o.                                ║
╠════════════════════════════════════════════════════════════╣
║                                                            ║
  1) Create new invoice
  2) Create invoice from existing
  3) List invoices
  4) Unpaid invoices                     [3 unpaid, 1 overdue]
  5) Customers
  6) Items catalog
  7) Suppliers (your companies)
  ...
```

### Invoice workflow

1. Select customer (or create new)
2. Add line items with quantity, price, VAT
3. Review summary
4. Save and generate PDF
5. PDF includes QR code for bank payment

### Data location

| OS | Path |
|----|------|
| Linux | `~/.config/tidybill/` |
| Windows | `%APPDATA%\TidyBill\` |
| macOS | `~/Library/Application Support/TidyBill/` |

## 🗺 Roadmap

- [x] **Phase 1** — CLI core (suppliers, customers, invoices, database)
- [x] **Phase 2** — PDF generation with Maroto + QR codes
- [ ] **Phase 3** — Full CLI features (items catalog, search, bulk operations)
- [ ] **Phase 4** — Internationalization (CS/SK/EN)
- [ ] **Phase 5** — Encrypted export/import for device sync
- [ ] **Phase 6** — React web frontend (embedded in binary, `tidybill --gui`)
- [ ] **Phase 7** — Polish & release (installers, templates)

## 📄 License

MIT
