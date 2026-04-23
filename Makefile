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

.PHONY: build run clean build-linux build-windows build-all desktop desktop-sidecar desktop-dev seed check

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

desktop: desktop-sidecar
	cd desktop && pnpm install && pnpm tauri build

desktop-sidecar:
	@if [ -z "$(GDRIVE_CLIENT_ID)" ]; then \
		echo "warning: GDrive credentials not injected — set GOOGLE_OAUTH_JSON or GDRIVE_CLIENT_ID/SECRET. Connect to Google Drive will fail at runtime."; \
	fi
	CGO_ENABLED=0 go build -ldflags "$(GDRIVE_LDFLAGS)" -o desktop/src-tauri/binaries/$(APP_NAME)-$(TRIPLE) $(SRC)

desktop-dev: desktop-sidecar
	cd desktop && pnpm tauri dev

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
