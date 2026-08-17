#!/usr/bin/env bash
#
# Build the .rpm package: compiles the binary via build.sh, then packages it
# with nfpm (https://nfpm.goreleaser.com) using nfpm.yaml at the repo root.
# See deploy/build-deb.sh for the .deb equivalent — same binary, same
# nfpm.yaml, different packager and RPM's own maintainer-scriptlet model
# (deploy/rpm/*.sh, wired in via nfpm.yaml's rpm override).
#
# Usage:
#   ./deploy/build-rpm.sh                  # dist/nbdash-VERSION-1.x86_64.rpm
#   VERSION=1.0.0 ./deploy/build-rpm.sh    # stamp an explicit version
#   GOARCH=arm64 ./deploy/build-rpm.sh     # cross-compile + package for arm64
#
# Requires nfpm on PATH:
#   go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
source deploy/pkg-common.sh  # builds the binary, sets GOARCH/RAW_VERSION/PKG_VERSION

# RPM's Arch tag uses its own vocabulary, not Go's GOARCH — nfpm.yaml's
# arch field takes whatever ARCH is passed and maps it per packager, but
# the target filename below is ours to name, so map it here too rather
# than let the two disagree.
case "$GOARCH" in
  amd64) RPM_ARCH=x86_64 ;;
  arm64) RPM_ARCH=aarch64 ;;
esac

RPM_OUT="dist/nbdash-${PKG_VERSION}-1.${RPM_ARCH}.rpm"
echo "==> Packaging ${RPM_OUT}"
ARCH="$GOARCH" VERSION="$PKG_VERSION" \
  nfpm package --config nfpm.yaml --packager rpm --target "$RPM_OUT"
rm -f dist/nbdash-packaging-input

echo "==> Done: ${RPM_OUT} ($(du -h "$RPM_OUT" | cut -f1))"
