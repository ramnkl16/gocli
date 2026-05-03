#!/usr/bin/env bash
# Install latest gocli from GitHub Releases into ~/.local/bin and ensure PATH.
# Usage:
#   curl -sSfL https://raw.githubusercontent.com/YOUR_ORG/gocli/main/scripts/install.sh | bash
# Override repo (owner/name):
#   GOCLI_GITHUB_REPO=myorg/ezcli curl -sSfL ... | bash

set -euo pipefail

REPO="${GOCLI_GITHUB_REPO:-yourorg/gocli}"
BASE="https://github.com/${REPO}"

case "$(uname -s)" in
Darwin) os_title=Darwin ;;
Linux) os_title=Linux ;;
*)
	echo "install.sh supports macOS and Linux only. On Windows use scripts/install.ps1" >&2
	exit 1
	;;
esac

arch="$(uname -m)"
case "$arch" in
x86_64 | amd64) goarch=amd64 ;;
arm64 | aarch64) goarch=arm64 ;;
*)
	echo "unsupported CPU architecture: $arch" >&2
	exit 1
	;;
esac

asset="gocli_${os_title}_${goarch}.tar.gz"
url="${BASE}/releases/latest/download/${asset}"

tmpdir="$(mktemp -d)"
cleanup() { rm -rf "$tmpdir"; }
trap cleanup EXIT

echo "Downloading ${url}"
curl -sSfL "$url" -o "${tmpdir}/archive.tar.gz"
tar -xzf "${tmpdir}/archive.tar.gz" -C "$tmpdir"

install_dir="${HOME}/.local/bin"
mkdir -p "$install_dir"
binary="$(find "$tmpdir" -maxdepth 1 -type f -name gocli -print -quit)"
if [[ -z "$binary" ]]; then
	echo "could not find gocli binary in archive" >&2
	exit 1
fi
install -m 0755 "$binary" "${install_dir}/gocli"

shell_rc=""
case "${SHELL:-}" in
*zsh) shell_rc="${HOME}/.zprofile" ;;
*bash) shell_rc="${HOME}/.bash_profile" ;;
esac
if [[ -z "$shell_rc" ]]; then
	shell_rc="${HOME}/.zprofile"
fi
line='export PATH="$HOME/.local/bin:$PATH"'
if [[ ! ":${PATH}:" == *":${HOME}/.local/bin:"* ]]; then
	if [[ ! -f "$shell_rc" ]] || ! grep -qF '.local/bin' "$shell_rc" 2>/dev/null; then
		printf '\n# gocli installer\n%s\n' "$line" >>"$shell_rc"
		echo "Appended PATH hint to ${shell_rc} (open a new terminal or: source \"$shell_rc\")"
	fi
else
	echo "~/.local/bin is already on PATH"
fi

echo "Installed gocli to ${install_dir}/gocli"
"${install_dir}/gocli" version || true
