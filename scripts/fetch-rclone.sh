#!/usr/bin/env bash
# Fetch the bundled rclone binaries used by TidyBill's cloud transport.
#
# This script downloads each supported OS/arch archive from
# downloads.rclone.org, verifies against the SHA256SUMS published for
# that release, extracts the rclone binary, and places it at the
# target-triple path Tauri expects under desktop/src-tauri/binaries/.
#
# Run this before `pnpm tauri build` (or any bundle/package command).
# Idempotent: skips download if the extracted binary already matches
# the expected hash.
#
# Usage:
#   scripts/fetch-rclone.sh                 # fetch all 5 targets
#   scripts/fetch-rclone.sh --only linux    # fetch only Linux targets
#   scripts/fetch-rclone.sh --version 1.73.5  # pin a specific version

set -euo pipefail

RCLONE_VERSION="${RCLONE_VERSION:-1.73.5}"
ONLY=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) RCLONE_VERSION="$2"; shift 2 ;;
    --only)    ONLY="$2"; shift 2 ;;
    -h|--help)
      sed -n '2,20p' "$0"
      exit 0
      ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

# Resolve paths relative to the repo root (script lives in app_web_dev/scripts/).
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BIN_DIR="$REPO_ROOT/desktop/src-tauri/binaries"
CACHE_DIR="$REPO_ROOT/.rclone-cache"

mkdir -p "$BIN_DIR" "$CACHE_DIR"

# Target-triple -> rclone asset suffix. Keep in sync with Phase 0.3 of
# IMPLEMENTATION_PLAN_CLOUD_TRANSPORT1.md.
#
# Format: <target-triple>|<rclone-asset-name-without-version>
TARGETS=(
  "x86_64-unknown-linux-gnu|linux-amd64"
  "aarch64-unknown-linux-gnu|linux-arm64"
  "x86_64-pc-windows-msvc|windows-amd64"
  "x86_64-apple-darwin|osx-amd64"
  "aarch64-apple-darwin|osx-arm64"
)

BASE_URL="https://downloads.rclone.org/v${RCLONE_VERSION}"
CHECKSUMS_URL="${BASE_URL}/SHA256SUMS"
CHECKSUMS_FILE="$CACHE_DIR/SHA256SUMS-${RCLONE_VERSION}"

# Download the release manifest once.
if [[ ! -s "$CHECKSUMS_FILE" ]]; then
  echo ">>> fetching ${CHECKSUMS_URL}"
  curl --fail --silent --show-error --location "$CHECKSUMS_URL" -o "$CHECKSUMS_FILE"
fi

# sha256 on Linux vs macOS.
if command -v sha256sum >/dev/null 2>&1; then
  SHA_CMD="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
  SHA_CMD="shasum -a 256"
else
  echo "error: no sha256sum or shasum on PATH" >&2
  exit 1
fi

expected_hash() {
  # args: asset filename. Returns hex hash from manifest, or empty string.
  awk -v n="$1" '$2 == n { print $1 }' "$CHECKSUMS_FILE"
}

compute_hash() {
  # args: path. Returns hex hash.
  $SHA_CMD "$1" | awk '{ print $1 }'
}

fetch_one() {
  local target="$1"
  local suffix="$2"
  local asset="rclone-v${RCLONE_VERSION}-${suffix}.zip"
  local url="${BASE_URL}/${asset}"
  local zip_path="$CACHE_DIR/${asset}"

  local want_zip_hash
  want_zip_hash="$(expected_hash "$asset")"
  if [[ -z "$want_zip_hash" ]]; then
    echo "error: no SHA256 for $asset in $CHECKSUMS_FILE" >&2
    return 1
  fi

  # Skip download if the cached zip is correct.
  if [[ -f "$zip_path" ]] && [[ "$(compute_hash "$zip_path")" == "$want_zip_hash" ]]; then
    echo ">>> cached: $asset"
  else
    echo ">>> downloading: $url"
    curl --fail --silent --show-error --location "$url" -o "$zip_path"
    got="$(compute_hash "$zip_path")"
    if [[ "$got" != "$want_zip_hash" ]]; then
      echo "error: $asset sha256 mismatch" >&2
      echo "  expected: $want_zip_hash" >&2
      echo "  got:      $got" >&2
      rm -f "$zip_path"
      return 1
    fi
  fi

  # Extract just the rclone binary, strip the versioned top-level dir.
  local ext="" bin_name="rclone"
  case "$suffix" in
    windows-*) ext=".exe"; bin_name="rclone.exe" ;;
  esac

  local dest="$BIN_DIR/rclone-${target}${ext}"
  local tmp_extract="$CACHE_DIR/extract-${target}"
  rm -rf "$tmp_extract"
  mkdir -p "$tmp_extract"

  if ! command -v unzip >/dev/null 2>&1; then
    echo "error: unzip not installed" >&2
    return 1
  fi
  unzip -q -o "$zip_path" -d "$tmp_extract"

  # The archive contains one top-level dir named like "rclone-v1.73.5-linux-amd64/".
  local inner
  inner="$(find "$tmp_extract" -maxdepth 1 -mindepth 1 -type d | head -n 1)"
  if [[ -z "$inner" ]] || [[ ! -f "$inner/$bin_name" ]]; then
    echo "error: $bin_name not found inside $asset" >&2
    return 1
  fi

  install -m 0755 "$inner/$bin_name" "$dest"
  echo ">>> installed: $dest"

  rm -rf "$tmp_extract"
}

for entry in "${TARGETS[@]}"; do
  target="${entry%%|*}"
  suffix="${entry##*|}"

  if [[ -n "$ONLY" ]] && [[ "$suffix" != *"$ONLY"* ]] && [[ "$target" != *"$ONLY"* ]]; then
    continue
  fi

  fetch_one "$target" "$suffix"
done

echo
echo "done. Binaries in: $BIN_DIR"
ls -la "$BIN_DIR" | grep -E "rclone-" || true
