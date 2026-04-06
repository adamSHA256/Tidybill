# Changelog

## v0.4.1

### Fixed
- Bank account validation: require account number OR IBAN (not both), with info tooltip on both fields
- Auto-update check: GET endpoint now refreshes stale cache when 24-hour cooldown elapsed
- Update check UI: "up to date" text left-aligned and double-clickable to re-check
- URL opening: switched from broken opener plugin to shell plugin for HTTP/HTTPS URLs
- File opening: custom Rust command bypasses Tauri opener scope, fixing non-standard drive letters (G:, Z:, network shares)
- PDF templates: account number, IBAN, and SWIFT shown only when non-empty (all 4 Go templates + 4 YAML templates)
- Windows NSIS installer: "Just install" is now the default option instead of "Uninstall first"
- Windows NSIS uninstall: now prompts before deleting user data instead of silently wiping it

### New
- SWIFT field added to setup wizard (alongside currency)
- About page: description shown before update check section
- Settings: PDF output section moved next to General on desktop

## v0.4.0

### New
- Encrypted backup & restore — export your full database as a single `.tidybill` file with optional XChaCha20-Poly1305 encryption (Argon2id key derivation, BIP-39 recovery mnemonics)
- Four import modes — smart merge (keeps newer records), full replace, force overwrite, and preview (dry-run)
- Sync page in desktop app with export filters (by supplier, date range, skip old paid invoices), encryption toggle, and detailed import preview
- CLI sync menu — interactive export/import with encryption support
- Android share sheet integration for exporting backup files
- BIP-39 mnemonic generator for creating strong, recoverable passphrases
- Email template defaults now configurable via API (`email.default_subject`, `email.default_body`, `email.copy_subject`)

### Fixed
- Save dialog now shows the actual saved path instead of the default filename
- Android sharesheet plugin deserialization error
- Import preview now simulates the correct mode (force/replace/merge) for accurate results
- SMTP password not-configured error now explains the post-import reconfiguration step
- Backend passphrase validation enforces minimum 8 characters (previously only checked in UI)
- Import mode parameter is now strictly validated — invalid values return 400 instead of silently defaulting
- Invoice number collision suffix is now unique (appends counter to avoid creating new conflicts)

### Changed
- Email template defaults moved from frontend to Go backend (persisted on startup)
- Export uses read-only transaction with single-connection isolation for consistent snapshots

## v0.3.0

### New
- Send invoices by email directly from the app — SMTP integration with per-supplier configuration, customizable email templates per customer, placeholder variables, send-copy option, and connection testing
- Automation page for managing default email templates across all customers
- Update notifications — opt-in check for new versions on startup (at most once per 24 hours), with manual check available anytime
- Privacy-friendly update check — the app never connects to the internet without explicit user permission, configurable in the setup wizard and About page
- Desktop "About" page with version info, update status, and donation addresses

### Changed
- About section moved from Settings to its own dedicated page (accessible from sidebar on desktop, Více tab on mobile)

## v0.2.0

### New
- Android app (APK) — TidyBill now runs natively on Android via Tauri 2 mobile + Go backend (gomobile)
- Native Android share sheet for sharing invoice PDFs
- Mobile-optimized About page
- Dynamic PDF button label — shows "Regenerate PDF" when PDF already exists
- Tooltips now work on mobile (tap to show)
- Mobile-specific dashboard defaults (hide less useful widgets)

### Fixed
- Overdue status is now a computed flag instead of stored status — no longer overwrites the real invoice status (e.g. partially paid invoices stay partially paid even when overdue)
- PDF opening on Android via HTTP URL (tauri-plugin-opener replaces broken shell:open)
- Hidden desktop-only UI on mobile (folder open buttons, directories settings, generate all previews)

### Changed
- "Open folder" button replaced with "Share" on mobile invoice detail
- Overdue removed from user-selectable statuses (automatic only)

## v0.1.6

Initial public release with CLI + Desktop GUI for Linux, Windows, and macOS.
