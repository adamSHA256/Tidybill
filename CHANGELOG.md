# Changelog

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
