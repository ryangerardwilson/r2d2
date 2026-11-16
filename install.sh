#!/usr/bin/env bash
set -euo pipefail

# r2d2 is installed from this checkout, never from a GitHub release.

APP=r2d2
INSTALL_DIR="${R2D2_INSTALL_DIR:-$HOME/.local/bin}"
MODULE_PATH="github.com/ryangerardwilson/r2d2"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

usage() {
  cat <<EOF
${APP} installer

Usage:
  install.sh                 Install from this checkout (git-stamped)
  install.sh from <path>     Install from a source checkout (git-stamped)
  install.sh help            Show this help

Installs to \$R2D2_INSTALL_DIR or ~/.local/bin.
EOF
}

die() {
  printf 'Error: %s\n' "$*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "'$1' is required"
}

source_version_meta() {
  local source_path="$1"
  local version commit source_hash
  commit="$(git -C "$source_path" rev-parse --short HEAD 2>/dev/null || echo unknown)"
  if [[ -n "$(git -C "$source_path" status --porcelain --untracked-files=normal -- . 2>/dev/null)" ]]; then
    commit="${commit}-dirty"
  fi
  version="$(sed -n 's/^[[:space:]]*Version[[:space:]]*= *"\([^"]*\)".*/\1/p' "$source_path/internal/version/version.go" | head -1)"
  if [[ -z "$version" ]]; then
    version="0.0.0-dev"
  fi
  source_hash="$(
    find "$source_path" -type f -name '*.go' -not -path '*/.git/*' -print0 |
      sort -z |
      xargs -0 sha256sum |
      sha256sum |
      awk '{print substr($1, 1, 16)}'
  )"
  printf '%s %s %s\n' "$version" "$commit" "$source_hash"
}

install_from_source() {
  local source_path=$1
  [[ -d "$source_path" ]] || die "source path does not exist: $source_path"
  require_command go
  mkdir -p "$INSTALL_DIR"
  local version commit source_hash build_time ldflags pkg
  read -r version commit source_hash < <(source_version_meta "$source_path")
  build_time="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  pkg="${MODULE_PATH}/internal/version"
  ldflags="-s -w -X ${pkg}.Version=${version} -X ${pkg}.Commit=${commit} -X ${pkg}.SourceHash=${source_hash} -X ${pkg}.BuildTime=${build_time}"
  (cd "$source_path" && GOWORK=off CGO_ENABLED=0 go build -ldflags "$ldflags" -o "$INSTALL_DIR/$APP" ./cmd/r2d2)
  chmod 755 "$INSTALL_DIR/$APP"
  "$INSTALL_DIR/$APP" version
}

case "${1:-install}" in
  install)
    install_from_source "$SCRIPT_DIR"
    ;;
  from)
    [[ -n "${2:-}" ]] || die "from requires a path"
    install_from_source "$2"
    ;;
  help|-h|--help)
    usage
    ;;
  *)
    die "unknown argument: $1"
    ;;
esac
