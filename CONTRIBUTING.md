# Contributing to TidyBill

Thanks for your interest in contributing! TidyBill is a small project and every contribution matters — whether it's a bug report, a new PDF template, or a code fix. No experience required.

## PDF Templates

TidyBill uses YAML-based invoice templates. If you've designed a template that looks better or more professional than the built-in ones, we'd love to include it!

Templates live in `internal/service/template_yaml_builtins.go` and use a declarative layout system with sections, rows, columns, and conditional blocks.

To contribute a template:

1. Duplicate one of the built-in templates (classic, modern, minimal) from the app's **PDF templates** screen
2. Customize it — layout, colors, spacing, fonts
3. Export the YAML and open a pull request

Even small improvements to existing templates (better alignment, nicer spacing, cleaner look) are welcome.

## Reporting Issues

Found a bug? Something doesn't work as expected? Please open an issue. You don't need to be technical — just describe what happened:

- **What were you doing?** (e.g. "I was creating an invoice for a customer with a long name")
- **What did you expect?** (e.g. "The name should fit on one line")
- **What happened instead?** (e.g. "The name overflowed into the next column")
- **Where in the app?** (CLI or Desktop, which screen/menu)
- **Screenshot** if possible — especially for PDF layout or UI issues

That's it. Don't worry about formatting or labels — a clear description is enough.

## Setting Up the Development Environment

### Prerequisites

- **Go** 1.24+
- **Node.js** 20+ and **pnpm**
- **Rust** toolchain (for the desktop app only)
- **Tauri CLI** (`cargo install tauri-cli`)

### Getting started

```bash
git clone https://github.com/adamSHA256/Tidybill.git
cd Tidybill

# Build and run the CLI
make build
./tidybill
```

On first run, TidyBill creates a SQLite database and walks you through an initial setup wizard (company name, bank account, etc.).

### Desktop app development

```bash
make desktop-dev     # Starts Tauri dev mode with hot-reload
```

This builds the Go sidecar binary and launches the React frontend with Vite. Changes to the frontend are reflected instantly; changes to Go code require restarting.

### Seed data

Instead of manually creating test customers and invoices, you can populate the database with sample data:

```bash
make seed            # Seeds with Czech data (default)
make seed L=en       # Seeds with English data
make seed L=sk       # Seeds with Slovak data
```

This gives you suppliers, customers, catalog items, and invoices to work with immediately. The database must be initialized first (run `./tidybill` once).

**Note:** Seeding clears existing data in the database. Don't run it on a database with real invoices.

### Project structure

```
cmd/tidybill/        CLI + API server entry point
cmd/seed/            Test data seeder
internal/
  api/               REST API handlers (used by desktop app)
  cli/               Terminal UI (interactive menus)
  database/          SQLite setup + migrations
    repository/      Data access layer
  i18n/              Translations (cs.go, sk.go, en.go)
  model/             Data structures
  service/           Business logic + PDF generation
  config/            App configuration
desktop/
  src/               React frontend (TypeScript, Mantine)
    locales/         Frontend translations (cs.json, sk.json, en.json)
    pages/           Route-level components
    components/      Reusable UI components
  src-tauri/         Tauri/Rust shell
```

## Pull Requests

1. Fork the repo and create a branch from `main`
2. Make your changes
3. Test both CLI and desktop if your change affects shared code (anything in `internal/`)
4. Open a pull request with a short description of what you changed and why

Small, focused PRs are easier to review. If you're planning a bigger change, open an issue first to discuss the approach.

## Translations

TidyBill supports Czech, Slovak, and English. Translations live in two places:

- **Backend** (Go): `internal/i18n/cs.go`, `sk.go`, `en.go`
- **Frontend** (React): `desktop/src/locales/cs.json`, `sk.json`, `en.json`

If you spot a translation mistake or want to improve wording, those are easy fixes.

Adding a new language requires adding a new file in both places and registering it in `internal/i18n/i18n.go`. Open an issue if you're interested — we can guide you through it.
