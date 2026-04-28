#!/usr/bin/env sh
# openrouter-cli installer
#
#   curl -fsSL https://raw.githubusercontent.com/vstratful/openrouter-cli/main/install.sh | sh
#
# Environment overrides:
#   OPENROUTER_CLI_VERSION       Pin a version (e.g. "v0.1.1"). Default: latest release.
#   OPENROUTER_CLI_INSTALL_DIR   Where to put the binary. Default: first writable dir
#                                from $PATH in this priority list:
#                                  ~/.local/bin, ~/bin, /usr/local/bin
#                                Falls back to ~/.local/bin (and warns if not on $PATH).
#   OPENROUTER_CLI_NO_VERIFY     If "1", skip SHA-256 checksum verification.
#
# Exits non-zero on any failure. Writes the binary atomically (tempdir + mv).

set -eu

REPO="vstratful/openrouter-cli"
BIN_NAME="openrouter"

log()  { printf 'openrouter-cli installer: %s\n' "$*" >&2; }
fail() { log "error: $*"; exit 1; }

require() {
  if ! command -v "$1" >/dev/null 2>&1; then
    fail "missing required command: $1"
  fi
}

# --- prerequisites ----------------------------------------------------------
require uname
require tar
require mkdir
require mv
require chmod

if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL "$1"; }
  fetch_to() { curl -fsSL -o "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO- "$1"; }
  fetch_to() { wget -qO "$2" "$1"; }
else
  fail "need curl or wget"
fi

# --- detect platform -------------------------------------------------------
os_raw=$(uname -s)
case "$os_raw" in
  Linux)  os=linux ;;
  Darwin) os=darwin ;;
  *)      fail "unsupported OS: $os_raw (this script supports linux and darwin; on Windows download the .zip from Releases manually)" ;;
esac

arch_raw=$(uname -m)
case "$arch_raw" in
  x86_64|amd64)   arch=amd64 ;;
  aarch64|arm64)  arch=arm64 ;;
  *)              fail "unsupported architecture: $arch_raw" ;;
esac

# --- resolve version -------------------------------------------------------
version=${OPENROUTER_CLI_VERSION:-}
if [ -z "$version" ]; then
  log "resolving latest release tag..."
  version=$(fetch "https://api.github.com/repos/${REPO}/releases/latest" \
    | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
    | head -n1)
  [ -n "$version" ] || fail "could not determine latest release tag from GitHub API"
fi
case "$version" in v*) ;; *) version="v$version" ;; esac
version_no_v=${version#v}
log "version: $version"

# --- resolve install dir ---------------------------------------------------
on_path() {
  case ":$PATH:" in *":$1:"*) return 0 ;; *) return 1 ;; esac
}

writable_or_root() {
  [ -w "$1" ] || [ "$(id -u)" = "0" ]
}

install_dir=${OPENROUTER_CLI_INSTALL_DIR:-}
if [ -z "$install_dir" ]; then
  for d in "$HOME/.local/bin" "$HOME/bin" /usr/local/bin; do
    if on_path "$d" && [ -d "$d" ] && writable_or_root "$d"; then
      install_dir=$d
      break
    fi
  done
fi
if [ -z "$install_dir" ]; then
  install_dir="$HOME/.local/bin"
  log "no writable \$PATH directory found; falling back to $install_dir"
fi
mkdir -p "$install_dir" || fail "could not create $install_dir"
writable_or_root "$install_dir" || fail "$install_dir is not writable"

if ! on_path "$install_dir"; then
  log "warning: $install_dir is not on \$PATH"
  log "         add this to your shell profile to fix:"
  log "             export PATH=\"$install_dir:\$PATH\""
fi
log "install dir: $install_dir"

# --- download + verify -----------------------------------------------------
asset="openrouter-cli_${version_no_v}_${os}_${arch}.tar.gz"
asset_url="https://github.com/${REPO}/releases/download/${version}/${asset}"
checksums_url="https://github.com/${REPO}/releases/download/${version}/checksums.txt"

tmp=$(mktemp -d 2>/dev/null || mktemp -d -t openrouter-cli) || fail "mktemp failed"
trap 'rm -rf "$tmp"' EXIT

log "downloading $asset"
fetch_to "$asset_url" "$tmp/$asset" || fail "download failed: $asset_url"

if [ "${OPENROUTER_CLI_NO_VERIFY:-}" = "1" ]; then
  log "skipping checksum verification (OPENROUTER_CLI_NO_VERIFY=1)"
else
  log "downloading checksums.txt"
  fetch_to "$checksums_url" "$tmp/checksums.txt" || fail "download failed: $checksums_url"
  expected=$(awk -v a="$asset" '$2 == a { print $1 }' "$tmp/checksums.txt")
  [ -n "$expected" ] || fail "asset $asset not found in checksums.txt"

  if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "$tmp/$asset" | awk '{print $1}')
  elif command -v shasum >/dev/null 2>&1; then
    actual=$(shasum -a 256 "$tmp/$asset" | awk '{print $1}')
  else
    fail "neither sha256sum nor shasum available; rerun with OPENROUTER_CLI_NO_VERIFY=1 to skip (not recommended)"
  fi

  if [ "$actual" != "$expected" ]; then
    fail "checksum mismatch for $asset (expected $expected, got $actual)"
  fi
  log "checksum ok"
fi

# --- extract + install -----------------------------------------------------
tar -xzf "$tmp/$asset" -C "$tmp" "$BIN_NAME" || fail "tar extract failed"
chmod +x "$tmp/$BIN_NAME"
mv "$tmp/$BIN_NAME" "$install_dir/$BIN_NAME"
log "installed: $install_dir/$BIN_NAME"

# --- smoke test ------------------------------------------------------------
if on_path "$install_dir"; then
  if "$BIN_NAME" --version >/dev/null 2>&1; then
    log "verified: $("$BIN_NAME" --version)"
  fi
else
  log "binary in place; reopen your shell or update \$PATH to use 'openrouter'"
fi

log "next: run '$BIN_NAME agent-setup' for the agent guide, or '$BIN_NAME --help'"
