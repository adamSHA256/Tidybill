# Changelog

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
