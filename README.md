# nbdash

An alternative dashboard for [NetBird](https://github.com/netbirdio/netbird) —
a React SPA embedded in and served by a single static Go binary.

It is an independent reimplementation of the official
[netbirdio/dashboard](https://github.com/netbirdio/dashboard) (Next.js + React),
written from scratch in Go against the same public Management HTTP API, and is a
drop-in replacement for the routes it covers.

> **Not affiliated with NetBird.** nbdash is an unofficial, community project.
> It is not developed, endorsed, sponsored, or supported by NetBird GmbH.
> "NetBird" is used here only to describe what this software interoperates
> with. For the official dashboard, see
> [netbirdio/dashboard](https://github.com/netbirdio/dashboard).

## Status

**Every module in the sidebar except Control Center is implemented
end-to-end.** Control Center is a React Flow graph with no server-rendered
equivalent sketched here — see [Porting the rest](docs/design.md#porting-the-rest).

| | Implemented |
|---|---|
| Peers | list, live search, detail, rename, SSH / login-expiration / inactivity toggles, delete |
| Groups | list, create, rename, edit peer membership, delete — "All" protected to match the server |
| Setup Keys | list, create (with the one-time plaintext reveal and a real group picker), revoke, delete |
| Access Control | list, create (name + first rule), rename/re-describe, delete; rules managed as sub-resources — add, edit, delete, each carrying every other rule forward unchanged |
| Networks | list, create, rename/re-describe, delete (cascades to its resources and routers); resources and routers each independently listed, added, edited, deleted |
| DNS | three tabs mirroring the Next.js dashboard: Nameserver Groups (full CRUD, primary/domain resolving), Settings (the account's single disabled-DNS-management-groups object), Zones (full CRUD, domain locked after creation, cascades to its records) with Records as an independently edited sub-resource |
| Team | two tabs: Users (invite, edit role/groups/blocked, approve/reject pending, resend invite, delete) and Service Users (create, edit, delete) with Personal Access Tokens as an independently created/deleted sub-resource, one-time plaintext reveal |

## Why this differs from the Next.js dashboard

The original is a **static export served by nginx**: `next build` emits `out/`,
the Docker image is nginx plus those files, and `docker/init_react_envs.sh`
rewrites `config.json` at container start so one image works for any deployment.
Auth runs in the browser via `@axa-fr/react-oidc`, and every API call goes
straight from the browser to the Management API with the access token attached.

nbdash is also a React SPA (`ui/`, built with Vite), but instead of nginx
serving its build output as static files from an image, the Go binary embeds
that build directly (`//go:embed`) and serves it itself — and every API call
goes through a same-origin JSON layer (`/api/bff/...`) the same binary also
serves, which holds the OIDC token server-side rather than handing it to the
browser. The deployment model changes accordingly:

| | Next.js dashboard | nbdash |
|---|---|---|
| Runtime | nginx + supervisord + certbot in Docker | one static binary under systemd |
| Artifacts | `out/` (hundreds of files) + image | 1 file, ~7.7 MB |
| Frontend | React, built with Next.js | React, built with Vite, embedded in the Go binary |
| Data fetching | React + SWR, bearer token attached client-side | React + TanStack Query, calling the same-origin `/api/bff/...` JSON layer |
| OIDC flow | in the browser | on the server (PKCE, confidential client) |
| **Access token** | **held in the browser** | **held on the server** |
| Config | `envsubst` into `config.json` at boot | read from the environment at startup |

Moving the token server-side is the substantive security change: the browser
only ever carries an opaque session ID, so a script injected into the page has
nothing to steal.

## Configuration

Variable names match the Next.js dashboard so an existing deployment's
environment carries over.

| Variable | Required | Default | Notes |
|---|---|---|---|
| `NETBIRD_MGMT_API_ENDPOINT` | yes | | Management API base URL; `/api` is appended |
| `AUTH_AUTHORITY` | yes | | OIDC issuer, used for discovery |
| `AUTH_CLIENT_ID` | yes | | |
| `AUTH_CLIENT_SECRET` | no | | Selects the OIDC client mode — see below |
| `AUTH_AUDIENCE` | no | | Auth0 needs it to issue an API JWT; `none` disables it |
| `AUTH_SUPPORTED_SCOPES` | no | `openid profile email offline_access api` | space-separated |
| `AUTH_REDIRECT_URI` | yes | | Must be an absolute `http://`/`https://` URL and be registered with the provider. Its scheme decides the session cookie's `Secure` flag |
| `NETBIRD_TOKEN_SOURCE` | no | `accessToken` | or `idToken` |
| `LISTEN_ADDR` | no | `:8080` | |

`offline_access` matters: without a refresh token, sessions die when the access
token expires instead of renewing.

### OIDC client mode: confidential vs. public

`AUTH_CLIENT_SECRET` set → **confidential client**: the secret authenticates
the dashboard to the token endpoint. This is what external providers (Auth0,
Okta, ...) expect — register the app as a "Regular Web Application", not a
SPA, and they'll issue one.

`AUTH_CLIENT_SECRET` unset → **public client**: no client authentication at
the token endpoint at all. PKCE (S256) is the only thing binding the
authorization code to the browser session that requested it, and it is
mandatory in this mode — not optional, not best-effort. If the provider's
discovery document doesn't advertise `code_challenge_methods_supported`
including `S256`, startup fails immediately with that reason, rather than
completing logins with no such binding.

**This is not a fallback for when you don't have a secret — it's required for
NetBird's own embedded Dex IdP.** Dex registers `nbdash` as a
public client with no secret at all (`idp/dex/provider.go` and
`management/server/idp/embedded.go` in the netbird repo both set
`Public: true` and never assign one). Setting `AUTH_CLIENT_SECRET` against
that server doesn't degrade gracefully — Dex's token endpoint does a
constant-time comparison against its stored (empty) secret and returns a flat
401 on *any* non-empty value presented, so the dashboard could not complete a
single login against the server it's meant to run against. Leave it unset for
NetBird's embedded IdP; set it for Auth0, Okta, or any other provider that
issues one.

Either way, the token never reaches the browser — only the OIDC client
authentication step changes. `journalctl` shows which mode actually got
selected on every startup: `oidc_client_mode=public` or
`oidc_client_mode=confidential`, alongside the rest of the "dashboard
listening" line.

## Building

```sh
./build.sh                  # dist/nbdash
VERSION=v0.1.0 ./build.sh   # stamp a version (also shown by --version)
GOARCH=arm64 ./build.sh     # cross-compile
```

The output is **one file with no runtime dependencies**. The `ui/` SPA is
built with Vite first, then its output plus every other static asset is
compiled in with `//go:embed`, and `CGO_ENABLED=0` removes the libc link, so
the same binary runs on any Linux host of that architecture — glibc, musl, or
a distroless base. The build does need Node — see `build.sh`'s header
comment for why that's a build-time-only dependency, not a runtime one.
Confirm with:

```sh
ldd dist/nbdash      # "not a dynamic executable"
./dist/nbdash --version
```

## Running locally

```sh
export NETBIRD_MGMT_API_ENDPOINT=https://api.netbird.io
export AUTH_AUTHORITY=https://your-tenant.eu.auth0.com
export AUTH_CLIENT_ID=...
export AUTH_CLIENT_SECRET=...
export AUTH_AUDIENCE=https://api.netbird.io

go run ./cmd/nbdash
```

Or, with the same variables in a `.env` file instead of exported by hand:
`make dev` builds the `ui/` SPA once and runs the server; `make run` skips
the frontend build and just runs the server (fast for a Go-only change).
Both need Node now — see build.sh's header comment for why.

Then open http://localhost:8080. Register `http://localhost:8080/auth/callback`
as an allowed callback URL with your provider first, as a **Regular Web
Application** rather than a SPA — the exchange is server-side now.

Against a NetBird server's own embedded Dex IdP instead of Auth0, omit
`AUTH_CLIENT_SECRET` and `AUTH_AUDIENCE` entirely — public client mode, no
audience parameter needed:

```sh
export NETBIRD_MGMT_API_ENDPOINT=https://your-netbird-server
export AUTH_AUTHORITY=https://your-netbird-server
export AUTH_CLIENT_ID=nbdash

go run ./cmd/nbdash
```

Same callback registration step applies, but against NetBird's own
`server.auth.dashboardRedirectURIs` — see
[docs/deploying.md](docs/deploying.md#2-configuration) for the exact
config.yaml snippet.

## Testing

```sh
go test ./...          # backend
cd ui && npm test       # frontend (Vitest + Testing Library)
```

The Go suite stubs both the Management API and the OIDC discovery endpoint,
so it covers the real path from request through session, bearer token, and
JSON API response — no network and no provider account needed. Some tests
drive the full authorization-code flow end to end against a fake OIDC
provider (`internal/api/login_flow_test.go`), including the router-level
security properties (CSRF origin checks, body-size limits, no-directory-
listing) that only manifest once the whole `Server.Routes()` mux is wired up.

`.github/workflows/ci.yml` runs on every push and PR: the frontend build and
test (`npm ci`, `npm run build`, `npm test`) that produces the embedded SPA,
then `gofmt -l`, `go vet`, a `go mod tidy` no-diff check, `go test -race ./...`,
`govulncheck`, a smoke build for both platforms `build.sh` supports (`amd64`
and `arm64`), and the same for `deploy/build-deb.sh` and `deploy/build-rpm.sh`
— so a change that breaks packaging (a bad `nfpm.yaml` field, a maintainer
script typo) fails CI the same way a broken `go build` would, rather than
surfacing on someone's install.
`go-version-file: go.mod` keeps CI on whatever toolchain `go.mod`'s `go`
line names, so bumping that line (e.g. after a stdlib CVE fix release) is
the only step needed to move CI onto it too.

## Layout

```
build.sh              builds the ui/ SPA, then the single static binary
deploy/               systemd unit, env template, install script
ui/                   the React SPA (Vite, TypeScript)
  src/api/              typed fetch wrappers + React Query hooks, one per resource
  src/features/         list/detail/edit components, one directory per resource
  src/components/       shared chrome (sidebar, layout)
  src/styles/            NetBird design tokens as CSS custom properties
cmd/nbdash/           entrypoint, graceful shutdown, OIDC discovery retry
internal/config/      environment configuration
internal/auth/        OIDC code flow, session store
internal/nbapi/       Management API client and types
internal/api/         the JSON backend (/api/bff/...), OIDC routes, security
                      middleware, and the top-level router — nbdash's whole
                      HTTP surface
internal/web/         embeds ui/'s build output plus /static/ assets
  static/app/           ui/'s built JS/CSS/index.html (git-ignored; produced
                        by `npm run build`, see build.sh)
```

`ui/`'s build output is embedded via `//go:embed` rather than loaded from a
CDN, so the binary stays self-contained and the Content-Security-Policy can
forbid every external origin.


## Documentation

| | |
|---|---|
| [docs/deploying.md](docs/deploying.md) | `.deb` / `.rpm` / `install.sh`, the systemd unit, restricting the dashboard to the NetBird overlay with Caddy, memory tuning, upgrade and rollback |
| [docs/design.md](docs/design.md) | Why it is built this way: the Management API behaviours that shaped it, and what is deliberately not ported |
| [SECURITY.md](SECURITY.md) | Reporting a vulnerability, and the security properties this dashboard claims |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Building, testing, and what a good patch looks like |

## Security

The substantive difference from the Next.js dashboard is that the OIDC access
token never reaches the browser. The flow runs server-side with PKCE, tokens are
held in the server's session store, and the browser only ever carries an opaque
session ID in an `HttpOnly`, `Secure`, `SameSite=Lax`, `__Host-`-prefixed
cookie. Script injected into a page has no token to steal.

nbdash performs **no authorization of its own**. Every call to the Management
API carries the signed-in user's own bearer token, so the Management server
decides what that user may see and do — this dashboard cannot grant more access
than the account already has.

See [SECURITY.md](SECURITY.md) for the full list and how to report a problem.

## License

[BSD-3-Clause](LICENSE) © Albert Snowden.

Third-party components and their licenses are listed in
[THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md).

"NetBird" is a trademark of NetBird GmbH. nbdash is an independent project and
is not affiliated with, endorsed by, or supported by NetBird GmbH.
