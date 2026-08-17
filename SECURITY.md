# Security policy

## Reporting a vulnerability

**Do not open a public issue for a security problem.**

Report it privately through GitHub's
[private vulnerability reporting](https://github.com/albertsnowden/nbdash/security/advisories/new)
on this repository. That opens a draft advisory only you and the maintainers
can see.

Please include:

- what an attacker can do, and what they need to start (a session? a link an
  admin clicks? network position?)
- the steps to reproduce, ideally against a local `go run ./cmd/nbdash`
- the version or commit you tested

You should get an acknowledgement within a week. This is a small volunteer
project with no paid on-call and no bug bounty — fixes are best-effort, but
reports are genuinely welcome and will be credited in the release notes unless
you would rather they weren't.

## Supported versions

Only the latest release and `main` receive fixes. There are no long-term
support branches.

## What nbdash claims

These are the properties worth attacking. If you can break one, it is a bug.

**The access token never reaches the browser.** The OIDC authorization-code
flow runs server-side with PKCE. Tokens live in the server's session store; the
browser only ever holds an opaque session ID.

**The session cookie is hard to steal or fixate.** `HttpOnly`, `Secure` (on any
https deployment), `SameSite=Lax`, `Path=/`, no `Domain`, and the `__Host-`
prefix on https, so a sibling subdomain cannot overwrite it.

**Sessions expire on the server.** A 12-hour absolute lifetime from creation
that is never extended, plus a 2-hour idle timeout. The cookie's own `MaxAge` is
only a hint to a cooperating browser; the store is what enforces this.

**Every state-changing request is checked for origin.** `Sec-Fetch-Site` where
the browser sends it, `Origin` as a fallback. This is a server-side check and
does not rely on the browser honouring `SameSite`.

**Output is escaped by construction.** All HTML goes through `html/template`
with no `template.HTML` escape hatches anywhere in the tree, under a CSP of
`script-src 'self'` with no `'unsafe-inline'`.

**No secrets are logged.** A created setup key's plaintext and a created
personal access token's plaintext are each shown exactly once, in the single
response that mints them, and never written to the log. Two tests pin this with
a capturing logger rather than trusting review.

## What nbdash does not claim

**It performs no authorization of its own.** Every Management API call carries
the signed-in user's own bearer token, so the Management server decides what
that user may see and do. nbdash cannot grant access the account does not
already have — and equally, it is not a second line of defence if the
Management server's own checks are wrong. "A non-admin can see admin data"
is a Management API issue, not an nbdash one, unless nbdash sent the wrong
token.

**There is no rate limiting.** Nothing throttles `/login` or any other route.
Each login attempt allocates a pending-authorization entry, reaped after 15
minutes, so sustained request floods are a real denial-of-service avenue.
Deployments are expected to sit behind a reverse proxy that handles this — and
ideally, as in [docs/deploying.md](docs/deploying.md), to not be reachable from
the public internet at all.

**Sessions are in memory.** They do not survive a restart and do not work
across replicas. This is a single-binary design, not a clustered one.

**It does not perform RP-initiated logout.** Signing out of nbdash drops the
local session; the user stays signed in at the identity provider.
