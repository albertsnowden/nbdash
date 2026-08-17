#!/usr/bin/env bash
#
# Build the nbdash binary.
#
# The result is one self-contained file: the ui/ SPA's built assets and every
# other static file are compiled in via //go:embed, and CGO is off so there is
# no libc link. Copy it to any Linux host of the same architecture and run
# it — nothing else needs to be installed on the *target*.
#
# The *build* itself needs Node — ui/'s dist lands in
# internal/web/static/app/ (see ui/vite.config.ts), which the Go build then
# embeds like any other static asset. This is a real, deliberate departure
# from this project's earlier "no Node anywhere in the pipeline" precedent,
# from back when the frontend was server-rendered with a small vendored
# htmx.min.js — a first-party bundle that changes on every commit is a worse
# fit for "commit the built artifact" than a third-party file that rarely
# changes. SKIP_FRONTEND=1 skips it for a rebuild that only needs to re-link
# Go (e.g. this script's own second, cross-compiled invocation in CI, which
# reruns with SKIP_TESTS=1 for the same reason).
#
# Usage:
#   ./build.sh                     # dist/nbdash for this platform
#   VERSION=v0.1.0 ./build.sh      # stamp an explicit version
#   GOARCH=arm64 ./build.sh        # cross-compile
#   OUT=/tmp/nbd ./build.sh        # write somewhere else
#   SKIP_FRONTEND=1 ./build.sh     # reuse whatever's already in
#                                   internal/web/static/app/
#
set -euo pipefail

cd "$(dirname "$0")"

# Prefer the git tag so a deployed binary can be traced back to a commit.
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-amd64}"
OUT="${OUT:-dist/nbdash}"

mkdir -p "$(dirname "$OUT")"

if [[ "${SKIP_FRONTEND:-0}" != "1" ]]; then
  echo "==> Building frontend (npm ci && npm run build)"
  (cd ui && npm ci && npm run build)
fi

# Gate the build on the race detector. The dashboard shares one *Session pointer
# across concurrent requests — the SPA fires several /api/bff/... calls in
# parallel on most pages (React Query issuing more than one query on mount) —
# so unsynchronised access is a live risk here, not a theoretical one, and plain
# `go test` will not see it.
#
# This runs for the host: -race needs cgo, which the CGO_ENABLED=0 cross-compile
# below deliberately does not have. The env assignment there is per-command, so
# it does not affect this step.
if [[ "${SKIP_TESTS:-0}" != "1" ]]; then
  echo "==> Testing (go test -race ./...)"
  go test -race ./...
fi

echo "==> Building ${GOOS}/${GOARCH} version=${VERSION}"

# -trimpath keeps build machine paths out of the binary.
# -s -w drop the symbol and DWARF tables, roughly a third of the size.
CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build \
  -trimpath \
  -ldflags "-s -w -X main.version=${VERSION}" \
  -o "$OUT" \
  ./cmd/nbdash

echo "==> Done: ${OUT} ($(du -h "$OUT" | cut -f1))"
