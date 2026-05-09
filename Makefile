APP_NAME := tidybill
SRC := ./cmd/tidybill/

# === Platform triple detection (Tauri sidecar naming) ===
UNAME := $(shell uname -s)
ARCH := $(shell uname -m)
ifeq ($(UNAME),Linux)
  ifeq ($(ARCH),aarch64)
    TRIPLE := aarch64-unknown-linux-gnu
  else
    TRIPLE := x86_64-unknown-linux-gnu
  endif
else ifeq ($(UNAME),Darwin)
  ifeq ($(ARCH),arm64)
    TRIPLE := aarch64-apple-darwin
  else
    TRIPLE := x86_64-apple-darwin
  endif
else
  TRIPLE := x86_64-pc-windows-msvc
endif

.PHONY: build run clean build-linux build-windows build-all desktop desktop-sidecar desktop-dev desktop-fetch-rclone desktop-fetch-rclone-linux desktop-fetch-rclone-windows desktop-fetch-rclone-osx seed check android-aar android-apk android-build android-check-aar version version-minor version-major version-dry

# === CLI targets (unchanged) ===
build:
	go build -o $(APP_NAME) $(SRC)

run: build
	./$(APP_NAME)

clean:
	rm -f $(APP_NAME) $(APP_NAME).exe $(APP_NAME)-linux
	rm -f desktop/src-tauri/binaries/$(APP_NAME)-*

build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(APP_NAME)-linux $(SRC)

build-windows:
	GOOS=windows GOARCH=amd64 go build -o $(APP_NAME).exe $(SRC)

build-all: build-linux build-windows

# === Desktop app (Go + React + Tauri) ===
export APPIMAGE_EXTRACT_AND_RUN := 1
export NO_STRIP := 1
export LINUXDEPLOY_DISABLE_STRIP := 1

# Google Drive OAuth credentials are injected at link time into the
# gdrive package's ClientID/ClientSecret vars (see internal/cloud/gdrive/client_id.go).
# The JSON file MUST live OUTSIDE this repo — only its path is referenced here.
# Override per invocation: make desktop-dev GOOGLE_OAUTH_JSON=/path/to/oauth.json
# In CI, skip the JSON and pass the values directly:
#   make desktop-sidecar GDRIVE_CLIENT_ID=$$ID GDRIVE_CLIENT_SECRET=$$SECRET
GOOGLE_OAUTH_JSON ?= ../google-oauth.json
GDRIVE_CLIENT_ID ?= $(shell [ -f $(GOOGLE_OAUTH_JSON) ] && jq -r '.installed.client_id // ""' $(GOOGLE_OAUTH_JSON) 2>/dev/null)
GDRIVE_CLIENT_SECRET ?= $(shell [ -f $(GOOGLE_OAUTH_JSON) ] && jq -r '.installed.client_secret // ""' $(GOOGLE_OAUTH_JSON) 2>/dev/null)
GDRIVE_LDFLAGS := -X github.com/adamSHA256/tidybill/internal/cloud/gdrive.ClientID=$(GDRIVE_CLIENT_ID) -X github.com/adamSHA256/tidybill/internal/cloud/gdrive.ClientSecret=$(GDRIVE_CLIENT_SECRET)

# Detect which "fetch-rclone --only" group corresponds to current TRIPLE
FETCH_OS := $(if $(findstring linux,$(TRIPLE)),linux,$(if $(findstring darwin,$(TRIPLE)),osx,$(if $(findstring windows,$(TRIPLE)),windows,)))

# rclone binary is .exe-suffixed on Windows, plain elsewhere.
RCLONE_EXT := $(if $(findstring windows,$(TRIPLE)),.exe,)
RCLONE_BIN := desktop/src-tauri/binaries/rclone-$(TRIPLE)$(RCLONE_EXT)

$(RCLONE_BIN):
	./scripts/fetch-rclone.sh --only $(FETCH_OS)

desktop-fetch-rclone:
	./scripts/fetch-rclone.sh

desktop-fetch-rclone-linux:
	./scripts/fetch-rclone.sh --only linux

desktop-fetch-rclone-windows:
	./scripts/fetch-rclone.sh --only windows

desktop-fetch-rclone-osx:
	./scripts/fetch-rclone.sh --only osx

desktop: desktop-sidecar
	cd desktop && pnpm install && pnpm tauri build

desktop-sidecar: $(RCLONE_BIN)
	@if [ -z "$(GDRIVE_CLIENT_ID)" ]; then \
		echo "warning: GDrive credentials not injected — set GOOGLE_OAUTH_JSON or GDRIVE_CLIENT_ID/SECRET. Connect to Google Drive will fail at runtime."; \
	fi
	CGO_ENABLED=0 go build -ldflags "$(GDRIVE_LDFLAGS)" -o desktop/src-tauri/binaries/$(APP_NAME)-$(TRIPLE) $(SRC)

desktop-dev: desktop-sidecar
	cd desktop && pnpm tauri dev

# === Release / version bump ===
# `make version`         → patch bump (e.g. 0.5.2 → 0.5.3) + changelog editor + commit + tag + push
# `make version-minor`   → minor bump
# `make version-major`   → major bump
# `make version-dry`     → show what would change, no writes/git
# Pass-through extra args via ARGS, e.g.:
#   make version ARGS="--version 0.6.0"
#   make version ARGS="--no-push"
#   make version ARGS="--skip-check"
ARGS ?=
version:
	./scripts/release.sh $(ARGS)

version-minor:
	./scripts/release.sh --bump minor $(ARGS)

version-major:
	./scripts/release.sh --bump major $(ARGS)

version-dry:
	./scripts/release.sh --dry-run $(ARGS)

# === Check (same as CI) ===
check:
	go build ./...
	go vet ./...
	go test ./...
	cd desktop && pnpm tsc -b && pnpm lint

# === Seed test data ===
# Usage: make seed L=cs  (or sk, en)
L ?= cs
seed:
	go run ./cmd/seed --lang $(L)

# === Android (gomobile + Tauri) ===
# Critical: when Go code under internal/ or pkg/mobile/ changes, the AAR
# MUST be rebuilt before the APK, or the APK ships a stale Go binary
# missing the new routes/handlers. android-build chains both correctly.
ANDROID_NDK_HOME ?= $(HOME)/Android/Sdk/ndk/27.1.12297006
ANDROID_AAR := desktop/src-tauri/gen/android/app/libs/tidybill.aar

# Rebuild the Go AAR if any tracked Go source under internal/ or pkg/mobile/
# is newer. The dep list is exhaustive on purpose — gomobile bind is slow
# (~30-60s) and we don't want spurious rebuilds, but we DO want a stale Go
# source under internal/ to trigger one.
ANDROID_GO_SOURCES := $(shell find internal pkg/mobile -name '*.go' 2>/dev/null) go.mod go.sum

$(ANDROID_AAR): $(ANDROID_GO_SOURCES)
	@if [ ! -d "$(ANDROID_NDK_HOME)" ]; then \
		echo "ERROR: ANDROID_NDK_HOME=$(ANDROID_NDK_HOME) does not exist."; \
		echo "Set ANDROID_NDK_HOME to the path of your installed NDK or install the NDK via Android Studio SDK Manager."; \
		exit 1; \
	fi
	@command -v gomobile >/dev/null 2>&1 || { echo "ERROR: gomobile not on PATH. Install: go install golang.org/x/mobile/cmd/gomobile@latest && gomobile init"; exit 1; }
	ANDROID_NDK_HOME=$(ANDROID_NDK_HOME) gomobile bind -v \
	  -o $(ANDROID_AAR) \
	  -target=android/arm64 -androidapi 24 \
	  ./pkg/mobile

android-aar: $(ANDROID_AAR)

# android-check-aar verifies the on-disk AAR is newer than every Go source
# under internal/ and pkg/mobile/. Fails loudly with a fix command if
# anything is newer — catches the "stale AAR shipped to phone" trap.
android-check-aar:
	@if [ ! -f $(ANDROID_AAR) ]; then \
		echo "ERROR: $(ANDROID_AAR) not found. Run: make android-aar"; exit 1; \
	fi
	@stale=$$(find internal pkg/mobile go.mod go.sum -newer $(ANDROID_AAR) 2>/dev/null | head -5); \
	if [ -n "$$stale" ]; then \
		echo "ERROR: AAR is older than these Go sources:"; \
		echo "$$stale"; \
		echo "Fix: make android-aar"; \
		exit 1; \
	fi
	@echo "AAR is up to date."

# android-apk builds the universal APK (release, unsigned). Depends on a
# fresh AAR — Make handles the rebuild order automatically.
android-apk: $(ANDROID_AAR)
	cd desktop && npx tauri android build --apk --target aarch64

# android-build is the convenience target users invoke after editing Go
# code: rebuilds AAR if needed, then builds the APK. Sign + install steps
# remain manual (signing key choice depends on dev vs release).
android-build: android-apk
	@echo
	@echo "Built unsigned APK at:"
	@find desktop/src-tauri/gen/android/app/build/outputs/apk -name 'app-universal-release-unsigned.apk' -print 2>/dev/null
	@echo
	@echo "To sign with the local debug key + install:"
	@echo "  apksigner sign --ks ~/.android/debug.keystore --ks-pass pass:android --key-pass pass:android --ks-key-alias androiddebugkey <apk>"
	@echo "  adb install -r <apk>"
