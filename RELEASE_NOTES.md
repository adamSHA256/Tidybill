# Download for Windows

### [>>> Click here to download TidyBill for Windows <<<](https://github.com/adamSHA256/Tidybill/releases/download/v0.2.0/TidyBill_Windows_Desktop_x64_setup.exe)

Double-click the downloaded file to install. Windows may show a security warning — this is normal for apps that aren't code-signed yet:

1. Click **"More info"**
2. Click **"Run anyway"**

<img src="https://raw.githubusercontent.com/adamSHA256/Tidybill/main/.github/smart-screen.png" alt="SmartScreen warning" width="400">

---

# Download for Linux, macOS & CLI version
<details> <summary>Open here</summary>

### Desktop App (GUI)

| Platform | File | Notes |
|----------|------|-------|
| **Linux (Debian/Ubuntu)** | [TidyBill_Linux_Desktop_amd64.deb](https://github.com/adamSHA256/Tidybill/releases/download/v0.2.0/TidyBill_Linux_Desktop_amd64.deb) | `sudo dpkg -i TidyBill_Linux_Desktop_amd64.deb` |
| **Linux (Fedora/RHEL)** | [TidyBill_Linux_Desktop_x86_64.rpm](https://github.com/adamSHA256/Tidybill/releases/download/v0.2.0/TidyBill_Linux_Desktop_x86_64.rpm) | `sudo rpm -i TidyBill_Linux_Desktop_x86_64.rpm` |
| **macOS (Apple Silicon)** | [TidyBill_Mac_Desktop_aarch64.dmg](https://github.com/adamSHA256/Tidybill/releases/download/v0.2.0/TidyBill_Mac_Desktop_aarch64.dmg) | For M1/M2/M3/M4 Macs. Gatekeeper may warn — open System Settings → Privacy & Security → click "Open Anyway" |
| **macOS (Intel)** | [TidyBill_Mac_Desktop_x64.dmg](https://github.com/adamSHA256/Tidybill/releases/download/v0.2.0/TidyBill_Mac_Desktop_x64.dmg) | For older Intel Macs. Gatekeeper may warn — open System Settings → Privacy & Security → click "Open Anyway" |

### CLI (terminal only, no GUI)

| Platform | File |
|----------|------|
| **Linux** | [tidybill-linux-terminal-amd64](https://github.com/adamSHA256/Tidybill/releases/download/v0.2.0/tidybill-linux-terminal-amd64) |
| **Windows** | [tidybill-windows-terminal-amd64.exe](https://github.com/adamSHA256/Tidybill/releases/download/v0.2.0/tidybill-windows-terminal-amd64.exe) |
| **macOS (Apple Silicon)** | [tidybill-mac-terminal-arm64](https://github.com/adamSHA256/Tidybill/releases/download/v0.2.0/tidybill-mac-terminal-arm64) |
| **macOS (Intel)** | [tidybill-mac-terminal-amd64](https://github.com/adamSHA256/Tidybill/releases/download/v0.2.0/tidybill-mac-terminal-amd64) |

</details>

# Download for Android
<details> <summary>Open here</summary>

### APK

| Platform | File | Notes |
|----------|------|-------|
| **Android (arm64)** | [TidyBill_Android_arm64.apk](https://github.com/adamSHA256/Tidybill/releases/download/v0.2.0/TidyBill_Android_arm64.apk) | For most modern Android phones. Requires Android 7.0+ |

To install: download the APK, open it, and allow installation from unknown sources when prompted. Your phone may show a security warning — tap **"Install anyway"**.

</details>

---

## What's new in v0.2.0

- **Android app** — TidyBill now runs natively on Android with full invoice management, PDF generation, and native share sheet
- **Overdue fix** — overdue is now a computed flag, no longer overwrites invoice status (e.g. partially paid invoices stay partially paid)
- **UI improvements** — dynamic PDF button labels, touch-friendly tooltips, mobile-optimized dashboard defaults
- **Bug fixes** — PDF opening/sharing on Android, hidden desktop-only UI on mobile

See [README](https://github.com/adamSHA256/Tidybill#readme) for features and screenshots.

**Full Changelog**: https://github.com/adamSHA256/Tidybill/compare/v0.1.6...v0.2.0
