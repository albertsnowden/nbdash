#!/usr/bin/env bash
#
# Shared by build-deb.sh and build-rpm.sh: builds the binary and computes a
# package version string valid for both formats. Not meant to run on its own
# — source it, don't execute it.
#
# Both Debian and RPM require a package Version field to start with a digit
# and forbid a bare "-" inside it (Debian reads the tail after the last "-"
# as the debian_revision; RPM uses "-" only as the Version/Release
# separator and disallows it inside either component). build.sh's own
# VERSION — a git tag ("v1.0.0-3-gabc1234") or, with no tags, an abbreviated
# commit SHA, optionally "-dirty" — satisfies neither guarantee on its own,
# so this produces a second, sanitised value (PKG_VERSION) rather than
# handing either packager the raw one.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if ! command -v nfpm >/dev/null 2>&1; then
  echo "ERROR: nfpm not found on PATH." >&2
  echo "       go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest" >&2
  exit 1
fi

GOARCH="${GOARCH:-amd64}"
case "$GOARCH" in
  amd64|arm64) ;;
  *) echo "ERROR: unsupported GOARCH for packaging: $GOARCH (amd64 or arm64)" >&2; exit 1 ;;
esac

RAW_VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"

PKG_VERSION="${RAW_VERSION#v}"           # v1.0.0 -> 1.0.0 (tag-based describe)
case "$PKG_VERSION" in
  [0-9]*) : ;;                            # already starts with a digit
  *) PKG_VERSION="0~${PKG_VERSION}" ;;    # no-tags fallback may start with a letter
esac
# "~dirty" is a valid pre-release marker in both formats; a bare trailing
# "-dirty" is not, in either. Any other "-" left over (from a tag describe's
# "-N-gSHA" suffix) becomes a "." for the same reason.
PKG_VERSION="$(echo "$PKG_VERSION" | sed -E 's/-dirty$/~dirty/; s/-/./g')"

BIN_OUT="dist/nbdash-${GOARCH}"
echo "==> Building binary (${GOARCH}, version ${RAW_VERSION})"
OUT="$BIN_OUT" GOARCH="$GOARCH" VERSION="$RAW_VERSION" ./build.sh

# nfpm.yaml's contents[].src only supports environment expansion for a few
# top-level scalar fields, not per-arch paths — stage the binary at the one
# fixed name it references instead of fighting that.
cp "$BIN_OUT" dist/nbdash-packaging-input
