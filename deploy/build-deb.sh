#!/usr/bin/env bash
#
# Build the .deb package: compiles the binary via build.sh, then packages it
# with nfpm (https://nfpm.goreleaser.com) using nfpm.yaml at the repo root.
#
# Usage:
#   ./deploy/build-deb.sh                  # dist/nbdash_VERSION_amd64.deb
#   VERSION=1.0.0 ./deploy/build-deb.sh    # stamp an explicit version
#   GOARCH=arm64 ./deploy/build-deb.sh     # cross-compile + package for arm64
#
# Requires nfpm on PATH:
#   go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
source deploy/pkg-common.sh  # builds the binary, sets GOARCH/RAW_VERSION/PKG_VERSION

DEB_OUT="dist/nbdash_${PKG_VERSION}_${GOARCH}.deb"
echo "==> Packaging ${DEB_OUT}"
ARCH="$GOARCH" VERSION="$PKG_VERSION" \
  nfpm package --config nfpm.yaml --packager deb --target "$DEB_OUT"
rm -f dist/nbdash-packaging-input

echo "==> Done: ${DEB_OUT} ($(du -h "$DEB_OUT" | cut -f1))"
