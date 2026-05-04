#!/usr/bin/env bash
# scripts/install-tools.sh
#
# Install every tool `task check` depends on.
# Supports: Debian/Ubuntu (apt), macOS (brew), generic (go install).
# Idempotent — safe to re-run.
#
# Usage:
#   bash scripts/install-tools.sh          # install everything
#   bash scripts/install-tools.sh go       # install only Go tools
#   bash scripts/install-tools.sh frontend # install only frontend tools
#
# After this script:
#   task hooks     # install git hooks
#   task check     # verify everything works

set -euo pipefail

# ---- utility ----

log() { printf '▶ %s\n' "$*"; }
done_() { printf '✓ %s\n' "$*"; }
skip() { printf '○ %s (already installed)\n' "$*"; }
fail() {
  printf '✗ %s\n' "$*" >&2
  exit 1
}

have() { command -v "$1" >/dev/null 2>&1; }

OS="$(uname -s)"
SCOPE="${1:-all}"

# Detect architecture once (used for binary-download tools).
UNAME_M="$(uname -m)"
case "$UNAME_M" in
  x86_64 | amd64) ARCH_GO="amd64" ;;
  aarch64 | arm64) ARCH_GO="arm64" ;;
  *) fail "Unsupported architecture: $UNAME_M" ;;
esac

# Detect OS kernel token for URL templates.
case "$OS" in
  Linux) KERNEL="linux" ;;
  Darwin) KERNEL="darwin" ;;
esac

# ---- prerequisites ----

case "$OS" in
  Linux)
    have apt-get || fail "apt-get not found; this script supports Debian/Ubuntu or macOS (brew)."
    ;;
  Darwin)
    have brew || fail "Homebrew not found. Install: https://brew.sh"
    ;;
  *)
    fail "Unsupported OS: $OS. Install tools manually per Taskfile.yaml."
    ;;
esac

have go || fail "Go not installed. Install Go 1.26+ first: https://go.dev/dl/"

go_version="$(go env GOVERSION | sed 's/^go//')"
log "Go version: $go_version"

# ---- install block: Go tools that install cleanly via `go install` ----

install_go_tools() {
  log "Installing Go tools..."

  # Tools installable via `go install`. Order matters — task first so later
  # invocations of `task` work if this script gets interrupted.
  GOTOOLS=(
    "github.com/go-task/task/v3/cmd/task@latest"
    "github.com/evilmartians/lefthook@latest"
    "github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8"
    "golang.org/x/vuln/cmd/govulncheck@latest"
    "mvdan.cc/gofumpt@latest"
    "golang.org/x/tools/cmd/goimports@latest"
    "mvdan.cc/sh/v3/cmd/shfmt@latest"
    "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest"
    "github.com/pressly/goose/v3/cmd/goose@latest"
  )

  # Note: gitleaks and sqlc are NOT in this list.
  #
  # gitleaks — the repo moved from zricethezav/gitleaks to gitleaks/gitleaks
  # but the go.mod still declares the old module path, so `go install` fails
  # with a module-path-conflict error (GitHub issue #1839, unfixed as of 2026).
  #
  # sqlc — v1.28+ go.mod contains `replace` directives, which make `go install`
  # refuse to build the target (Go language restriction).
  #
  # Both install cleanly from their prebuilt binary releases. See below.

  for pkg in "${GOTOOLS[@]}"; do
    name="$(basename "${pkg%%@*}")"
    if have "$name"; then
      skip "$name"
    else
      log "go install $pkg"
      go install "$pkg"
      done_ "$name"
    fi
  done

  # ---- vacuum — binary release ----
  if have vacuum; then
    skip "vacuum"
  else
    log "Installing vacuum..."
    mkdir -p "$HOME/.local/bin"
    curl -fsSL https://quobix.com/scripts/install_vacuum.sh | INSTALL_DIR="$HOME/.local/bin" sh
    done_ "vacuum"
  fi

  # ---- gitleaks — binary release (works around unfixed module-path bug) ----
  install_gitleaks_binary

  # ---- sqlc — binary release (works around replace-directive restriction) ----
  install_sqlc_binary
}

# Download and install gitleaks from the GitHub release page.
# Upstream: https://github.com/gitleaks/gitleaks/releases
install_gitleaks_binary() {
  if have gitleaks; then
    skip "gitleaks"
    return 0
  fi

  log "Installing gitleaks (from binary release)..."

  # Resolve latest release tag via GitHub API (no auth; public repo).
  local tag
  tag="$(curl -fsSL https://api.github.com/repos/gitleaks/gitleaks/releases/latest \
    | grep -oE '"tag_name":\s*"v[^"]+"' | head -1 | cut -d'"' -f4)"
  if [ -z "$tag" ]; then
    fail "Could not determine latest gitleaks release tag"
  fi
  local version="${tag#v}"

  # Filename convention (as of v8.x):
  # gitleaks_8.30.1_linux_x64.tar.gz
  # gitleaks_8.30.1_darwin_arm64.tar.gz
  local arch_token
  case "$ARCH_GO" in
    amd64) arch_token="x64" ;;
    arm64) arch_token="arm64" ;;
  esac

  local tarball="gitleaks_${version}_${KERNEL}_${arch_token}.tar.gz"
  local url="https://github.com/gitleaks/gitleaks/releases/download/${tag}/${tarball}"

  mkdir -p "$HOME/.local/bin"
  local tmp
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' RETURN

  log "  download: $url"
  curl -fsSL "$url" -o "$tmp/gitleaks.tar.gz"
  tar -xzf "$tmp/gitleaks.tar.gz" -C "$tmp" gitleaks
  mv "$tmp/gitleaks" "$HOME/.local/bin/gitleaks"
  chmod +x "$HOME/.local/bin/gitleaks"

  done_ "gitleaks ${version} → ~/.local/bin/gitleaks"
  trap - RETURN
}

# Download and install sqlc from the GitHub release page.
# Upstream: https://github.com/sqlc-dev/sqlc/releases
install_sqlc_binary() {
  if have sqlc; then
    skip "sqlc"
    return 0
  fi

  log "Installing sqlc (from binary release)..."

  local tag
  tag="$(curl -fsSL https://api.github.com/repos/sqlc-dev/sqlc/releases/latest \
    | grep -oE '"tag_name":\s*"v[^"]+"' | head -1 | cut -d'"' -f4)"
  if [ -z "$tag" ]; then
    fail "Could not determine latest sqlc release tag"
  fi
  local version="${tag#v}"

  # Filename convention:
  # sqlc_1.31.0_linux_amd64.tar.gz
  # sqlc_1.31.0_darwin_arm64.zip   (note: darwin ships as zip)
  local filename ext
  if [ "$KERNEL" = "darwin" ]; then
    filename="sqlc_${version}_${KERNEL}_${ARCH_GO}.zip"
    ext="zip"
  else
    filename="sqlc_${version}_${KERNEL}_${ARCH_GO}.tar.gz"
    ext="tar.gz"
  fi
  local url="https://github.com/sqlc-dev/sqlc/releases/download/${tag}/${filename}"

  mkdir -p "$HOME/.local/bin"
  local tmp
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' RETURN

  log "  download: $url"
  curl -fsSL "$url" -o "$tmp/sqlc.${ext}"

  if [ "$ext" = "zip" ]; then
    have unzip || fail "unzip required for sqlc on macOS. Install: brew install unzip"
    unzip -q "$tmp/sqlc.zip" -d "$tmp"
  else
    tar -xzf "$tmp/sqlc.tar.gz" -C "$tmp"
  fi

  mv "$tmp/sqlc" "$HOME/.local/bin/sqlc"
  chmod +x "$HOME/.local/bin/sqlc"

  done_ "sqlc ${version} → ~/.local/bin/sqlc"
  trap - RETURN
}

# ---- install block: system linters (OS-specific) ----

install_system_tools() {
  log "Installing system linters..."

  case "$OS" in
    Linux)
      local pkgs=(
        shellcheck
        yamllint
      )
      local missing=()
      for p in "${pkgs[@]}"; do
        dpkg -l "$p" >/dev/null 2>&1 || missing+=("$p")
      done
      if [ "${#missing[@]}" -gt 0 ]; then
        log "sudo apt-get install -y ${missing[*]}"
        sudo apt-get update -qq
        sudo apt-get install -y "${missing[@]}"
      else
        skip "shellcheck + yamllint (apt)"
      fi
      ;;
    Darwin)
      for p in shellcheck yamllint; do
        if brew list --formula "$p" >/dev/null 2>&1; then
          skip "$p"
        else
          log "brew install $p"
          brew install "$p"
        fi
      done
      ;;
  esac
}

# ---- install block: Rust-based tools (typos, lychee) ----

install_rust_tools() {
  log "Installing Rust-based tools..."

  # typos
  if have typos; then
    skip "typos"
  else
    case "$OS" in
      Linux)
        local ver="1.26.0"
        local tarball
        tarball="typos-v${ver}-$(uname -m)-unknown-linux-musl.tar.gz"
        local url="https://github.com/crate-ci/typos/releases/download/v${ver}/${tarball}"
        log "Downloading typos v${ver}..."
        curl -fsSL "$url" -o /tmp/typos.tar.gz
        mkdir -p "$HOME/.local/bin"
        tar -xzf /tmp/typos.tar.gz -C /tmp ./typos
        mv /tmp/typos "$HOME/.local/bin/typos"
        chmod +x "$HOME/.local/bin/typos"
        rm -f /tmp/typos.tar.gz
        done_ "typos (to ~/.local/bin/typos — add to PATH if not already)"
        ;;
      Darwin)
        brew install typos-cli
        done_ "typos"
        ;;
    esac
  fi

  # lychee
  if have lychee; then
    skip "lychee"
  else
    case "$OS" in
      Linux)
        log "Installing lychee via cargo-binstall or cargo..."
        if have cargo-binstall; then
          cargo binstall -y lychee
        elif have cargo; then
          cargo install lychee
        else
          log "Skipping lychee (no cargo). Install via: cargo install lychee  (or system pkg)"
        fi
        ;;
      Darwin)
        brew install lychee
        ;;
    esac
  fi
}

# ---- install block: Python tools (sqlfluff) ----

install_python_tools() {
  log "Installing Python tools..."

  if have sqlfluff; then
    skip "sqlfluff"
  else
    have pipx || fail "pipx not installed. Install: sudo apt install pipx   (or brew install pipx)"
    pipx install sqlfluff
    done_ "sqlfluff"
  fi
}

# ---- install block: Node-ecosystem tools ----

install_node_tools() {
  log "Installing Node-ecosystem tools (markdownlint, prettier)..."

  have npm || fail "npm not installed. Install Node.js 20+ first."

  if have markdownlint-cli2; then
    skip "markdownlint-cli2"
  else
    npm install -g markdownlint-cli2@latest
    done_ "markdownlint-cli2"
  fi

  if have prettier; then
    skip "prettier"
  else
    npm install -g prettier@latest
    done_ "prettier"
  fi
}

# ---- install block: frontend (bun + workspace deps) ----

install_frontend_tools() {
  log "Installing frontend tooling..."

  if have bun; then
    skip "bun"
  else
    log "Installing bun..."
    curl -fsSL https://bun.sh/install | bash
    done_ "bun (restart shell or source ~/.bashrc to use)"
  fi

  if [ -d web/ ] && [ -f web/package.json ]; then
    log "Installing web/ dependencies..."
    (cd web && bun install)
    done_ "web/ deps"
  fi
}

# ---- install block: container scanners (Grype) ----

install_security_tools() {
  log "Installing security tooling..."

  if have grype; then
    skip "grype"
  else
    log "Installing grype..."
    curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh \
      | sh -s -- -b "$HOME/.local/bin"
    done_ "grype"
  fi
}

# ---- dispatch ----

case "$SCOPE" in
  go) install_go_tools ;;
  system) install_system_tools ;;
  rust) install_rust_tools ;;
  python) install_python_tools ;;
  node) install_node_tools ;;
  frontend) install_frontend_tools ;;
  security) install_security_tools ;;
  all)
    install_go_tools
    install_system_tools
    install_rust_tools
    install_python_tools
    install_node_tools
    install_frontend_tools
    install_security_tools
    ;;
  *)
    fail "Unknown scope: $SCOPE (valid: all go system rust python node frontend security)"
    ;;
esac

echo ""
done_ "Tooling install complete."
echo ""
echo "Next steps:"
echo "  task hooks       # install git hooks (lefthook)"
echo "  task check       # verify everything works"
echo ""
echo "Ensure these are on your PATH:"
echo "  \$HOME/go/bin"
echo "  \$HOME/.local/bin"
echo "  \$HOME/.bun/bin"
