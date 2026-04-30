# Changelog

## v0.5.2

### Fixed
- Creating a new invoice (or "Nová z této") no longer pops the "Opustit tvorbu faktury? Máte neuložené změny" warning after a successful save. The form is correctly considered saved before the post-save navigation runs, so the leave-confirm blocker stays out of the way.
- Typing an issue/taxable/due date in Czech format (e.g. `03.02.2026`) is now parsed correctly as Feb 3 instead of being silently swapped to March 2 by dayjs's fallback parser. Affects every date input in the app (invoice forms, sync filters). Picking a date from the calendar was already correct; only the typed-input path was wrong.

## v0.5.1

### Fixed
- Backups containing time fields (e.g. `email_sent_at`) made on one machine could fail to load when imported on another with a different local timezone. After import, the affected invoices/customers/suppliers became unreadable and the corresponding pages showed a SQL scan error. The DB now writes timestamps in an unambiguous SQLite-native format that round-trips reliably across platforms and zones. **No action required for most users** — existing data continues to work. If you hit the symptom (preview shows N invoices, then list is empty after import), open Sync → Restore from cloud and the next import will fix it.

## v0.5.0

### New
- **Cloud Sync** — sync your data to your own Google Drive, OneDrive, Dropbox, S3, SFTP, WebDAV, or Proton Drive via rclone, end-to-end encrypted with your master key. Configure once in Settings, then upload/restore from the unified Cloud panel.
- **Auto-backup** — scheduled backups to your cloud with grandfather-father-son retention (keeps every backup from the last week, then daily for a month, weekly for half a year, monthly forever). Pruning runs after each successful backup; the most recent backup is always kept.
- **Auto-sync** — on startup, TidyBill checks your cloud for a more recent backup and prompts before applying when local and cloud diverge.
- **Master key & 12-word recovery phrase** — a single BIP-39 mnemonic protects all your encrypted data and works across every device. Replaces per-backup passphrases for both export/import and cloud sync.
- **Guided tour** — interactive walkthrough on first run with three paths (create-invoice, just-show-me, advanced), reachable from a Help button in the sidebar. Separate flows for desktop and mobile.
- **In-app Help hub** — accordion-style help page with section guides and an FAQ (e.g., "QR code isn't showing on my invoice"), accessible from any page.
- **Welcome dialog** — first-run picker for which guided flow to start with; the "don't show again" checkbox is unchecked by default.
- **Default backup directory** — configurable in Settings.
- Plain-language intros and tooltips on Cloud Sync, auto-backup, and auto-sync sections for non-technical users.
- rclone form i18n — labels, help text, and button strings translated across all cloud backends.
- Refresh button on cloud file list in the Import panel.

### Fixed
- rclone `operations/stat` unmarshal: Size and ModTime came back as zero values silently because the response is wrapped in `{"item": {…}}`. Upload now returns the correct metadata.
- Cancel in-flight cloud upload on shutdown; expose in-progress state to the UI.
- Recovery phrase copy/cancel/change-phrase warning flows now match what actually happens.
- Cloud import passphrase: dropped the field where unused, kept where needed (gated on `blob.encrypted`), with actionable error alerts instead of confusing dialogs.
- Proton Drive anti-abuse system: identify as TidyBill (not rclone) in `app_version` so uploads aren't blocked; rate-limit error (Code 2028) maps to an actionable message.
- HTTP timeout for rcd: dropped short `ResponseHeaderTimeout`, raised total to 10 minutes — legitimate operations like large copyfile genuinely take that long.
- copyfile with local files now uses split `srcFs`/`srcRemote` so absolute paths don't get joined to rcd's CWD.
- ISO-8601 second-precision timestamp in upload filenames (no more colon characters that some backends reject).
- Cloud transports cached in localStorage + "Connecting" badge + translated status strings.
- Cloud upload modal locks while upload is in flight.
- Mobile help icon restricted to dashboard; More-page tools grouped for clarity.
- Tour no longer marks a flow completed when user closes the last step with X.
- Keychain: phrase normalization (case + whitespace), backend errors surfaced, fail-fast on missing dataDir.

### Build
- CI: per-platform rclone sidecar fetch + GDrive OAuth credential injection in release pipeline; docs for required GitHub Actions secrets.
- Makefile: auto-fetch rclone sidecar binary if missing; rclone binaries are gitignored.
- Android: Make targets + dependency tracking + docs for AAR rebuilds via gomobile.
- Stopped tracking auto-generated `gen/android/app/src/main/assets/tauri.conf.json` and `tauri.properties` — CI regenerates them.

## v0.4.3

### New
- Leave confirmation when navigating away from unsaved invoice creation
- Customer edit modal on invoice detail with disclaimer about future invoices
- Customer email prompt when sending invoice to customer without email
- PDF regeneration hint modal after editing invoice/customer that has existing PDF
- Dynamic invoice validation messages (shows exactly which fields are missing)
- Error translation mapping for common backend errors (SMTP, validation, etc.)
- Read-only invoice number in edit mode with workflow explanation
- Update check section in Settings page
- Extended About page text with Hitchhiker's Guide reference
- Better SMTP setup tooltip with step-by-step guidance

### Fixed
- Windows NSIS installer: data no longer wiped during upgrade (uninstall-before-reinstall flow)
- Windows NSIS installer: text now recommends "Just install" instead of "Uninstall first"
- Folder reveal (Open folder) now works on non-standard drive letters (G:, Z:, network shares)
- macOS Intel build: fixed asset rename from .dmg to .app.tar.gz in CI

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
