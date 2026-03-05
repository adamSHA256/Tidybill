# Mobile (Android) Build

## Architecture

The desktop app spawns Go as a **separate process** (sidecar). Android can't do that, so Go is compiled into a **shared library** (`.so`) loaded directly into the app process via gomobile.

```
Desktop:   Tauri → spawns Go binary → HTTP server on random port
Android:   Tauri → Kotlin loads Go .so → HTTP server on port 18080
```

The React frontend is identical on both platforms. It calls `http://127.0.0.1:<port>/api/...`.

## Key Files

| File | Purpose |
|------|---------|
| `pkg/mobile/mobile.go` | Go entry point for Android. Exports `StartServer(dataDir)` and `StopServer()` |
| `internal/config/config.go` | `NewWithDataDir()` constructor for Android (skips OS-specific path detection) |
| `tools.go` | Keeps `golang.org/x/mobile/bind` in go.mod (required by gomobile) |
| `desktop/src-tauri/src/lib.rs` | `#[cfg(desktop)]` = sidecar, `#[cfg(mobile)]` = fixed port 18080 |
| `desktop/src-tauri/tauri.conf.json` | Base config (includes `externalBin` for desktop sidecar) |
| `desktop/src-tauri/tauri.android.conf.json` | Android override: sets `externalBin: []` (no sidecar) |
| `desktop/src-tauri/capabilities/default.json` | Desktop-only: shell:spawn, shell:execute (platforms: linux/windows/macOS) |
| `desktop/src-tauri/capabilities/mobile.json` | Android-only: core + dialog (no shell permissions) |
| `gen/android/app/libs/tidybill.aar` | Compiled Go backend as Android library (built by gomobile) |
| `gen/android/app/.../MainActivity.kt` | Calls `Mobile.startServer()` in onCreate, `Mobile.stopServer()` in onDestroy |
| `gen/android/app/build.gradle.kts` | `usesCleartextTraffic=true` (Go uses plain HTTP on localhost) |
| `gen/android/app/proguard-rules.pro` | Keep rules for `mobile.**` and `go.**` (prevent R8 stripping) |

## Build Commands

### Prerequisites (one-time)

```bash
# Rust Android targets
rustup target add aarch64-linux-android

# gomobile
go install golang.org/x/mobile/cmd/gomobile@latest
go install golang.org/x/mobile/cmd/gobind@latest

# Environment (add to ~/.bashrc)
export ANDROID_HOME="$HOME/Android/Sdk"
export NDK_HOME="$ANDROID_HOME/ndk/27.1.12297006"
```

### Build APK

```bash
# 1. Build Go shared library (from project root)
export ANDROID_NDK_HOME="$HOME/Android/Sdk/ndk/27.1.12297006"
gomobile bind -v \
  -o desktop/src-tauri/gen/android/app/libs/tidybill.aar \
  -target=android/arm64 -androidapi 24 \
  ./pkg/mobile

# 2. Build APK (from desktop/ dir, arm64 only to limit RAM usage)
cd desktop
npx tauri android build --apk --target aarch64

# 3. Sign with debug key
apksigner sign \
  --ks ~/.android/debug.keystore \
  --ks-pass pass:android --key-pass pass:android \
  --ks-key-alias androiddebugkey \
  src-tauri/gen/android/app/build/outputs/apk/universal/release/app-universal-release-unsigned.apk

# 4. Install
adb install -r <path-to-signed.apk>
```

### Important: rebuild AAR when Go code changes

If you change any Go code (`internal/`, `pkg/mobile/`), you MUST rebuild the AAR (step 1) before rebuilding the APK (step 2). The APK bundles whatever AAR is in `libs/`.

### RAM warning

Building for all architectures uses ~30-40GB RAM. Always use `--target aarch64` during development (builds for arm64 only, ~8GB RAM).

## Current Limitations (PoC)

- Port is hardcoded to 18080 (should be dynamic with Kotlin→Rust bridge)
- No mobile-specific UI (sidebar disappears on small screens)
- No PDF sharing (desktop uses "open folder", Android needs share intent)
- No Android file picker adaptation
- Default Android icon (no custom app icon)
- Debug-signed APK only (needs proper keystore for Play Store)

## How the Conditional Compilation Works

In `lib.rs`, Tauri sets `#[cfg(desktop)]` or `#[cfg(mobile)]` based on the build target:

```rust
#[cfg(desktop)]   // Only compiled for Linux/Windows/macOS
{
    // Shell plugin, sidecar spawning, exit handler
}

#[cfg(mobile)]    // Only compiled for Android/iOS
{
    // Set port to 18080 (Go server started by Kotlin)
}
```

Both paths use the same `get_api_port` Tauri command — the frontend doesn't know which platform it's on.
