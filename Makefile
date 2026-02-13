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

.PHONY: build run clean build-linux build-windows build-all desktop desktop-sidecar desktop-dev

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
desktop: desktop-sidecar
	cd desktop && pnpm install && pnpm tauri build

desktop-sidecar:
	CGO_ENABLED=0 go build -o desktop/src-tauri/binaries/$(APP_NAME)-$(TRIPLE) $(SRC)

desktop-dev: desktop-sidecar
	cd desktop && pnpm tauri dev
