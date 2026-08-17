# Contributing

Thanks for taking a look. This is a small project with a deliberate shape, so
this file is mostly about what that shape is — following it makes review fast.

Security problems go to [SECURITY.md](SECURITY.md), not to a public issue.

## Getting set up

```sh
git clone https://github.com/albertsnowden/nbdash
cd nbdash
go test ./...
```

No network, no NetBird account and no identity provider are needed to run the
suite: it stubs both the Management API and the OIDC discovery endpoint.

To run it for real, see [Running locally](README.md#running-locally).

## Before you open a PR

```sh
gofmt -l .          # must print nothing
go vet ./...
go test -race ./...
```

`./build.sh` runs the race suite for you and refuses to build if it fails.
CI runs all of the above plus `govulncheck`, a `go mod tidy` no-diff check, and
smoke builds of the binary, the `.deb` and the `.rpm` for both architectures.

## What a good patch looks like

**Comments explain why, not what.** This codebase is unusually heavily
commented, and the comments carry the reasoning that is not recoverable from
the code: why `SameSite` is `Lax` and not `Strict`, why the client secret is
optional, why a policy's rules are full-replaced but a network's resources are
not. If you change behaviour that a comment explains, update the comment in the
same commit. If you find yourself writing a comment that restates the code,
delete it.

**Security properties get a test, not a promise.** The list in
[SECURITY.md](SECURITY.md) is pinned by tests — a capturing logger proves the
setup-key plaintext is never logged; a request with `Sec-Fetch-Site:
cross-site` proves the origin check fires. New properties of that kind need the
same treatment, because the next person to touch the code will not have read
this file.

**Templates are executed in tests.** `html/template` resolves field names at
runtime, so a typo in a template is invisible to the compiler. Any new page or
partial needs a test that actually renders it.

**Never introduce `template.HTML` or an equivalent escape hatch.** There are
none in the tree today and the CSP is written on the assumption that there
never will be. If you think you need one, open an issue first.

## Adding a Management API resource

Every module follows the same four pieces — Peers is the smallest complete
example to read first:

1. Types in `internal/nbapi/`, mirrored by hand from `openapi.yml`.
2. Endpoint methods in a file next to `peers.go`. Path segments that come from
   user input go through `url.PathEscape`, without exception.
3. Handlers and routes in `internal/handlers/`.
4. A page template in `internal/web/templates/pages/`, plus partials for
   whatever htmx swaps.

[docs/design.md](docs/design.md) covers the Management API behaviours that
shaped this — in particular which resources are full-replaced on write and
which are addressable sub-resources, which is the single most common source of
silent data loss in this domain.

## Scope

nbdash covers the NetBird Management API surface the sidebar exposes. Things
that are deliberately out of scope, and why, are listed under "Porting the
rest" in [docs/design.md](docs/design.md) — read that before starting work on
Control Center or the self-service invite flow, which have both been considered
and declined for stated reasons.

## Licensing

Contributions are accepted under [BSD-3-Clause](LICENSE), the project's own
license. There is no CLA.

Do not copy code, markup, or styling from
[netbirdio/dashboard](https://github.com/netbirdio/dashboard); it is AGPL-3.0
and incompatible with this project's license. nbdash is an independent
reimplementation against the public HTTP API and must stay one.
