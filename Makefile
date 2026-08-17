# Local edit-run loop only. build.sh (and deploy/build-*.sh) remain the
# canonical build path CI and releases actually use — see
# .github/workflows/ci.yml and release.yml — because those need the race
# detector, the deb/rpm packaging, and multi-arch cross-compiles this
# Makefile deliberately doesn't do. This exists to make the everyday "change
# something, see it running" cycle a couple of keystrokes.
SHELL := /usr/bin/env bash
.SHELLFLAGS := -eu -o pipefail -c

# Override with `make run ENV_FILE=deploy/nbdash.env.example` etc. Not
# committed and not defaulted to anything more clever than .env, the same
# file every other doc (README.md, docs/deploying.md) already assumes.
ENV_FILE ?= .env

.PHONY: dev
dev: ui run

# Builds the SPA once into internal/web/static/app/ (see ui/vite.config.ts),
# the same output build.sh embeds — `go run` below picks it up via
# //go:embed the same way a real binary would. For iterating on the
# frontend itself with hot reload, run `npm run dev` inside ui/ directly
# instead (see the migration plan's "Dev workflow" note); this target is
# for "I touched something and want the whole app running," not tight
# frontend-only iteration.
.PHONY: ui
ui:
	cd ui && npm ci && npm run build

# go run, not a built binary — this is the fast loop, not a release
# artifact. $(ENV_FILE) is sourced with allexport on so a plain KEY=VALUE
# file (no `export` needed, matching deploy/nbdash.env.example's format)
# reaches the process as real environment variables.
.PHONY: run
run:
	set -a; . "./$(ENV_FILE)"; set +a; go run ./cmd/nbdash
