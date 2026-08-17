# Third-party notices

nbdash itself is licensed under BSD-3-Clause (see [LICENSE](LICENSE)). It
bundles or builds upon the third-party work listed below.

## Vendored into the binary

The dashboard's own React SPA (`ui/`) is built at compile time and its
output — bundled JS/CSS under `internal/web/static/app/` — is embedded via
`//go:embed` and served from `/static/`, so it ships inside the single
binary rather than being fetched from a CDN — required by the
`script-src 'self'` Content-Security-Policy. `ui/`'s own npm dependencies
(React, react-router, TanStack Query — see `ui/package.json`) are resolved
at build time and compiled into that bundle; none are vendored into this
repository. Run `cd ui && npm ls --all` for the authoritative, resolved
frontend set and their licenses.

## Go module dependencies

Resolved at build time from `go.mod`; none are vendored into this repository.

| Module | License |
|---|---|
| [github.com/coreos/go-oidc/v3](https://github.com/coreos/go-oidc) | Apache-2.0 |
| [golang.org/x/oauth2](https://pkg.go.dev/golang.org/x/oauth2) | BSD-3-Clause |
| [github.com/go-jose/go-jose/v4](https://github.com/go-jose/go-jose) (indirect) | Apache-2.0 |

Run `go list -m all` for the authoritative, resolved set.

## Design references

The colour tokens in `ui/src/styles/tokens.css` match NetBird's own dashboard
palette (brand orange plus a neutral gray scale), reproduced as CSS custom
properties so nbdash's UI looks at home next to the official one. No NetBird
code is included.

## Trademarks

"NetBird" is a trademark of NetBird GmbH. nbdash is an independent, unofficial
project and is not affiliated with, endorsed by, or supported by NetBird GmbH.
The mark is used here only nominatively, to identify the software this project
interoperates with.
