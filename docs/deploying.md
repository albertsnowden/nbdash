# Deploying nbdash

This is the operational half of the docs. For what nbdash is, how to
configure it, and how to run it locally, see the [README](../README.md).

## Deploying with systemd

Three ways to get the binary, the unit, and the config template onto a host —
a `.deb` package, an `.rpm` package, or the same three pieces installed by a
plain script. All three end up at the same paths and produce the same
running service; the packages get you native upgrade and uninstall handling
for free instead of asserting it by hand.

### Option A: the `.deb` package

Build it (needs [nfpm](https://nfpm.goreleaser.com) on the build machine —
`go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest` — nothing extra
on the target host):

```sh
./deploy/build-deb.sh                  # dist/nbdash_VERSION_amd64.deb
VERSION=1.0.0 ./deploy/build-deb.sh    # stamp an explicit version
GOARCH=arm64 ./deploy/build-deb.sh     # cross-compile + package for arm64
```

Copy the `.deb` to the target host and:

```sh
sudo dpkg -i nbdash_*_amd64.deb
sudoedit /etc/nbdash/nbdash.env
sudo systemctl enable --now nbdash
journalctl -u nbdash -f
```

The package's `postinst` already runs `daemon-reload` and — if
`nbdash.env` has no `replace-me` placeholders left in it — enables and
starts the service itself, so on a re-install with real config already in
place the two `systemctl`/`sudoedit` lines above are a no-op. `nbdash.env`
is a tracked conffile (`dpkg` calls it that): a first install writes the
template, but a later `dpkg -i` of a newer version never overwrites a
`nbdash.env` you've already edited.

**Uninstalling** is what makes the `.deb` worth it over the script:

```sh
sudo dpkg -r nbdash   # stops + disables the service; removes the
                                  # binary and unit; leaves nbdash.env in
                                  # place, in case you reinstall later
sudo dpkg -P nbdash   # same, plus deletes /etc/nbdash
                                  # (including the client secret) entirely
```

No user to clean up either way — the unit runs under `DynamicUser=yes`, a
transient UID systemd allocates and releases itself. `deploy/deb/{postinst,
prerm,postrm}.sh` are what actually run at each of those steps; `nfpm.yaml`
is the package manifest that wires them in. Verified against a real `dpkg -i`
→ edit `nbdash.env` → `dpkg -i` again (edit survives) → `dpkg -r` (config
survives) → `dpkg -P` (config gone) cycle in a disposable container, not just
read for correctness.

### Option B: the `.rpm` package

Same `nfpm.yaml`, same binary, different packager — but RPM's install/upgrade/
removal model isn't dpkg's, so it gets its own maintainer scriptlets
(`deploy/rpm/{postinstall,preremove,postremove}.sh`, wired in via `nfpm.yaml`'s
`overrides.rpm.scripts`) rather than reusing the Debian ones.

Build it (same nfpm requirement as the `.deb`):

```sh
./deploy/build-rpm.sh                  # dist/nbdash-VERSION-1.x86_64.rpm
VERSION=1.0.0 ./deploy/build-rpm.sh    # stamp an explicit version
GOARCH=arm64 ./deploy/build-rpm.sh     # cross-compile + package for arm64 (aarch64)
```

Copy the `.rpm` to the target host and:

```sh
sudo rpm -i nbdash-*-1.x86_64.rpm
sudoedit /etc/nbdash/nbdash.env
sudo systemctl enable --now nbdash
journalctl -u nbdash -f
```

Same `%post` behavior as the `.deb`'s `postinst`: `daemon-reload` always
runs, and if `nbdash.env` has no `replace-me` placeholders left, it
enables and starts the service itself. `nbdash.env` is packaged
`%config(noreplace)`, RPM's own conffile mechanism — a fresh install writes
the template; `rpm -U` of a newer version leaves an already-edited
`nbdash.env` alone rather than overwriting it, the same guarantee `dpkg`
gives the `.deb`.

**Uninstalling:**

```sh
sudo rpm -e nbdash
```

There's no separate purge step to remember here — unlike `dpkg`, RPM decides
per file at erase time: an untouched `nbdash.env` (still exactly the
packaged template) is deleted outright, while a modified one is renamed to
`nbdash.env.rpmsave` and left in place, so `rpm -e` never silently
discards real config (including the client secret) but also never leaves an
untouched file behind the way `dpkg -r` (without `-P`) does.

No user to clean up either way, same as the `.deb` — `DynamicUser=yes` means
systemd allocates and releases the UID itself. Verified against a real
`rpm -i` → edit `nbdash.env` → `rpm -U` (edit survives, service restarts)
→ `rpm -e` (config saved as `.rpmsave`) cycle, and separately a fresh
install → `rpm -e` with an untouched config (deleted, no `.rpmsave`), in a
disposable container — not just read for correctness.

### Option C: `install.sh`

Copy `dist/nbdash` to the target host, then:

```sh
sudo ./deploy/install.sh dist/nbdash
sudoedit /etc/nbdash/nbdash.env
sudo systemctl enable --now nbdash
journalctl -u nbdash -f
```

`install.sh` places the binary at `/usr/local/bin/nbdash`, the unit
at `/etc/systemd/system/nbdash.service`, and a commented
configuration template at `/etc/nbdash/nbdash.env` (root-only,
mode 0600 — it holds the client secret). Re-running it upgrades the binary and
unit without touching your configuration — the same guarantee the `.deb`'s
conffile handling gives, just asserted by hand instead of by `dpkg`. There is
no scripted uninstall for this path; removing what it installed means
deleting those same three paths yourself and running
`systemctl disable --now nbdash` first.

To do it by hand instead of either, the three pieces are
`deploy/nbdash.service`, `deploy/nbdash.env.example`, and the
binary.

Notes on the unit:

- **`DynamicUser=yes`** — systemd allocates a transient unprivileged UID, so
  there is no account to create. This works because the service is stateless:
  nothing is written to disk.
- **Sessions are lost on restart.** They live in memory, so
  `systemctl restart` logs everyone out. See [Design notes](design.md).
- **No capabilities, `ProtectSystem=strict`, a `@system-service` syscall
  filter.** The process needs only outbound TCP. Binding a port below 1024
  needs the commented `CAP_NET_BIND_SERVICE` lines — prefer terminating TLS in
  a reverse proxy and leaving `LISTEN_ADDR=127.0.0.1:8080`.
- `systemd-analyze verify deploy/nbdash.service` passes clean.

Behind a TLS reverse proxy, set `AUTH_REDIRECT_URI` to the **public** https URL.
That is what gets registered with your identity provider, and its scheme is also
what marks the session cookie `Secure`.

**Sign-out and your provider's own SSO session.** If your provider advertises
`end_session_endpoint` in its discovery document, signing out of the dashboard
also asks the provider to end its session (RP-Initiated Logout), rather than
only clearing the dashboard's cookie — without that, the provider's session
survives, and the very next protected page silently re-authenticates through
it with no prompt, so sign-out looks like it did nothing. The redirect target
the dashboard sends (`post_logout_redirect_uri`) is derived from
`AUTH_REDIRECT_URI`'s own origin (`https://dashboard.example.com/`). Some
providers require that URI to be pre-registered too, the same as the callback
URL above — check your provider's allowlist (e.g. Dex's
`dashboardRedirectURIs`, Keycloak/Auth0's "Allowed Logout URLs") if sign-out
still leaves the SSO session alive.

## Worked example: a memory-constrained host

Debian 13 on a 1 GiB VPS, already running
`netbird-server` (the combined binary) on `127.0.0.1:8443` behind Caddy on
`:443` for `dashboard.example.com`. On a box this small the dashboard must be
capped so it cannot starve the Management server, and it must not be reachable
from the internet at all — only from the NetBird overlay it manages.

### 1. Prerequisites

- `netbird-server` already running and reachable at `127.0.0.1:8443`.
- Caddy already terminating TLS for the domain and proxying
  `/api/*`, `/oauth2/*`, `/relay*`, `/signalexchange*` to it (the `@backend`
  matcher referenced below).
- Root access to install the systemd unit and edit the Caddyfile.

### 2. Configuration

Fill in `/etc/nbdash/nbdash.env` per the
[Configuration](../README.md#configuration) table in the README. One step is easy to miss:

**Register the callback URL with the embedded IdP.** NetBird's combined server
only knows about the redirect URIs it's told about —
`server.auth.dashboardRedirectURIs` in its `config.yaml`, which by default
lists only `/nb-auth` and `/nb-silent-auth` (the old Next.js dashboard's paths).
Add this dashboard's callback:

```yaml
management:
  auth:
    dashboardRedirectURIs:
      - /nb-auth
      - /nb-silent-auth
      - https://dashboard.example.com/auth/callback
```

Then restart `netbird-server` to pick it up. Skipping this step doesn't fail
loudly — the browser reaches the provider's login page fine, and only fails
*after* a successful login, when the provider refuses to redirect back because
the callback URL isn't on its allowlist.

### 3. Install and verify

Build locally, then get the binary and this repo's `deploy/` directory onto the
host — `install.sh` needs its sibling files (`nbdash.service`,
`nbdash.env.example`) alongside it, not just the binary:

```sh
./build.sh
rsync -a dist/nbdash deploy/ root@your-host:/tmp/dashboard-deploy/
```

Then, on the host:

```sh
cd /tmp/dashboard-deploy
sudo ./install.sh nbdash
sudoedit /etc/nbdash/nbdash.env
sudo systemctl enable --now nbdash
systemctl status nbdash
journalctl -u nbdash -n 50 --no-pager
```

A healthy start looks like one line: `msg="dashboard listening" version=...
addr=127.0.0.1:8080 api=... issuer=... token_source=...`. Anything before that
line is OIDC discovery retrying against an unreachable issuer (`config.Load`
or `auth.New` failing); the process exits non-zero rather than serving
half-configured, so `systemctl status` shows `failed` with the reason in the
same journal output.

Until step 4 is done, the dashboard is reachable at `127.0.0.1:8080` on the
host itself but not yet exposed through Caddy — `curl 127.0.0.1:8080/healthz`
from the host should return `ok`.

### 4. Restrict the dashboard to the overlay (Caddy)

The dashboard must be reachable only from NetBird peer addresses
(`100.64.0.0/10`), never from the public internet — unlike `/api/*` and
`/oauth2/*`, which must stay public (peers that aren't enrolled yet need to
reach `/api` to register at all, and a browser doing OIDC login isn't on the
overlay before that login finishes).

> **Do this before you rely on it, not after: a peer's browser does not
> automatically reach this domain over the overlay tunnel just because the
> peer is enrolled.** `dashboard.example.com` has to keep resolving to your
> server's real public IP for everyone — unenrolled peers need `/api/*` to
> register in the first place — so an ordinary browser request from an
> enrolled peer's own machine still goes out that machine's normal internet
> route and arrives at Caddy with its real public IP, not its `100.64.x.x`
> one. The `remote_ip` check below only gates on the source IP once traffic
> already arrives over the tunnel; it does nothing to arrange that on its
> own. Set up NetBird DNS Management first — a nameserver group that
> resolves this exact hostname to the server's own overlay IP, scoped to
> your peers — so peers' browsers genuinely route here over the tunnel.
> Skip that and this step locks *everyone* out, including you: neither
> `@backend` nor `@overlay` matches, this Caddyfile has no fallback
> directive, and Caddy returns an empty 200 — a blank page with no console
> errors. If that happens, `ssh -L 8080:127.0.0.1:8080 you@host` and browse
> `http://localhost:8080` to get back in without touching Caddy.

`deploy/Caddyfile.dashboard-overlay` in this repo is the fragment to merge —
read it for the full reasoning, in particular the ordering requirement
(`@backend` must be evaluated before `@overlay`, or an already-enrolled peer's
own `/api` and `/oauth2` traffic — which arrives *from* an overlay IP — would
be misrouted to the dashboard instead of netbird-server). In short, in the
existing `dashboard.example.com` block:

```caddyfile
# The `not path /api/bff/*` line is required if your existing @backend
# matcher is the plain `path /api/* ...` form: nbdash's own SPA calls its
# BFF endpoints under /api/bff/..., which is a /api/* path too, so without
# this exclusion @backend claims them first and sends them to
# netbird-server, which has no such route.
@backend {
	# /ws-proxy/* is netbird-server's HTTP/1.1-websocket fallback for signal
	# and management traffic (see the netbird repo's own reference config,
	# infrastructure_files/nginx.tmpl.conf) — easy to miss since it isn't
	# gRPC-shaped like the other paths here, but it belongs to netbird-server,
	# not nbdash.
	path /api/* /oauth2/* /relay* /signalexchange* /ws-proxy/*
	not path /api/bff/*
}
handle @backend {
	reverse_proxy h2c://127.0.0.1:8443
}

# insert this immediately after @backend, before any catch-all:
@overlay remote_ip 100.64.0.0/10
handle @overlay {
	reverse_proxy 127.0.0.1:8080
}
```

If the live config currently has a catch-all serving the old Next.js
dashboard's static export, remove it — nbdash replaces it, and leaving
both risks either a duplicate route or the old dashboard staying publicly
reachable regardless of this change.

```sh
caddy validate --config /etc/caddy/Caddyfile
systemctl reload caddy
```

**Verify both directions** — this is the property the whole change exists for.
The gate is on the *source* address of the connection reaching Caddy, not a
different target, so the same URL is used both times; only where the `curl`
runs from differs. One half is checkable now; the other half needs an overlay
peer to run `curl` from, which doesn't exist until step 5 — so run the
non-overlay check now, do step 5, then loop back and run the overlay check.

```sh
# Checkable now — run from anywhere NOT on the NetBird overlay (your own
# laptop, e.g.) — must NOT reach the dashboard:
curl -sI https://dashboard.example.com/peers
# expect: 404 (Caddy's fallthrough — no @backend or @overlay match)

# Same non-overlay run: netbird-server itself must be unaffected by any of
# this —
curl -sI https://dashboard.example.com/api/peers
# expect: 401 (reached and needs auth — not 404, and not routed to :8080)

# The check that catches the @backend-swallows-/api/bff/* failure mode
# directly: nbdash's own BFF calls must NOT be routed to netbird-server
# either. Same non-overlay expectation as /peers above —
curl -sI https://dashboard.example.com/api/bff/permissions
# expect: 404 (falls through past @backend, same as /peers — NOT 401 or any
# response from netbird-server, which would mean the `not path /api/bff/*`
# exclusion above is missing or wrong)

# Needs step 5 done first — SSH into a machine already enrolled as a NetBird
# peer (its own address is 100.64.x.x) and curl from there:
curl -sI https://dashboard.example.com/peers
# expect: 200 (the SPA shell — nbdash serves it to every path unconditionally,
# see internal/web/render.go's AppShell; the app itself checks the session
# client-side and redirects to /login from there, so Caddy never returns a
# 302 for this)

curl -sI https://dashboard.example.com/api/bff/permissions
# expect: 401 (reached nbdash, no session cookie on this bare curl — this is
# the request that silently broke without the @backend fix above: same URL
# shape as /api/peers two checks up, but must resolve on :8080, not :8443)
```

> **Run both checks for real.** The ordering above is reasoned from Caddy's
> matcher semantics, not inferred from a passing test — a Caddyfile that looks
> right and silently leaves the dashboard publicly reachable fails open. Treat
> this step as incomplete until both `curl`s have actually been run against
> your own edge and returned what is expected.

### 5. Bootstrapping the first peer

Chicken-and-egg: the dashboard is now reachable only from the overlay, but
nothing is on the overlay yet, so the UI can't be reached to create the setup
key that would put something on the overlay.

**The simplest correct path** — verified against the client and server source,
not assumed — needs neither a setup key nor a PAT:

```sh
netbird up
```

With no `--setup-key`, this triggers an interactive SSO device-flow login: it
prints a URL and code, you complete the login in any browser, and the CLI
enrolls *that machine* as a peer once it succeeds. This works before the
dashboard is reachable because it only needs `/oauth2/*` and `/api/*`, both
still public per step 4. It also creates the account: NetBird auto-assigns the
**owner** role to the first user ever to log in for the domain
(`management/server/account.go`, "New user + New account ... -> role = owner").
Once this machine has a `100.64.x.x` address, the dashboard is reachable from
it, and every subsequent machine can be enrolled normally through the UI with
an ordinary setup key.

**If you don't want the bootstrapping machine to become a permanent peer** —
e.g. you're doing this from a laptop, not the machine that should actually
hold the first peer identity — the cleanest option is to run `netbird up` from
somewhere disposable instead (a throwaway VM or container), then delete that
peer from the now-reachable dashboard once a real setup key exists.

A setup-key-first bootstrap, entirely by API with no netbird client involved,
is also possible in principle — `POST /api/setup-keys` accepts a plain OIDC
access token as its bearer credential (`security: BearerAuth` in the spec), and
Dex (the embedded IdP) does implement the OAuth2 device authorization grant
server-side. But getting that token requires a raw device-flow exchange against
Dex's `device_authorization` and `token` endpoints, and the netbird CLI itself
never prints its access token to the user — there is no `netbird` command that
hands you a reusable bearer token. **No verified `curl` sequence for this path
is given here, rather than an untested one presented as working** — confirm the
exact endpoints from
`https://your-netbird-server/.well-known/openid-configuration` first
(`device_authorization_endpoint`, `token_endpoint`, `grant_types_supported`)
before scripting this path.

Once you have *any* valid access token from *any* login (the `netbird up` path
above, or a manual device-flow exchange), a setup key needs no further token
minting — call `/api/setup-keys` directly with it. A durable credential for
repeated scripting, if you want one, is a separate step:
`POST /api/users/{userId}/tokens {"name":"...", "expires_in": <days>}` (get
`userId` from `GET /api/users/current`) — that endpoint and request shape are
confirmed against the OpenAPI spec, unlike the token-acquisition step above.

If `server.auth.owner` is set in netbird-server's `config.yaml` (a
pre-provisioned local admin, independent of any external IdP), logging in as
that user instead skips the "first login becomes owner" race entirely.

### 6. Memory

`deploy/nbdash.service` sets `MemoryHigh=180M` / `MemoryMax=220M`, justified
against measured RSS under synthetic load — the full measurement table and its
caveats are in [design.md](design.md#memory). `MemoryHigh` throttles before
`MemoryMax` kills, so a transient spike degrades rather than crash-loops.

Confirm on the real host after the first deploy, and re-check if your account
is materially larger than the synthetic dataset those figures came from:

```sh
systemctl status nbdash | grep Memory
ps -o rss,vsz,cmd -p "$(pgrep -f nbdash)"
```

### 7. Upgrade

`install.sh` doesn't install a copy of itself, so it — and `deploy/` alongside
it — need to make it to the host again on every upgrade, the same as the
first install in step 3:

```sh
./build.sh
rsync -a dist/nbdash deploy/ root@your-host:/tmp/dashboard-deploy/
ssh root@your-host 'cd /tmp/dashboard-deploy && sudo ./install.sh nbdash'
```

`install.sh` is idempotent and never touches `nbdash.env` — it replaces the
binary atomically (`install` to a temp name, then `mv`, so a request in flight
finishes against the old inode rather than hitting "text file busy"), reapplies
the unit, and restarts the service. Sessions are in-memory, so everyone is
logged out on restart — expected, not a bug (see [Design notes](design.md)).

### 8. Rollback

There's no separate rollback command because there's no state to roll back:
config is untouched by an upgrade, and a session store is disposable by
design. Rolling back is redeploying the previous binary:

```sh
# on the host, *before* upgrading, next time — keep a copy of the binary
# currently live so a bad upgrade has something to fall back to:
cp /usr/local/bin/nbdash /root/nbdash.previous

# to actually roll back: install.sh isn't left on the host permanently (see
# step 7), so re-use whatever copy of deploy/ is still at
# /tmp/dashboard-deploy/ from the last upgrade, or re-rsync it from a checkout
# of this repo if that's gone:
cd /tmp/dashboard-deploy   # or wherever deploy/ landed
sudo ./install.sh /root/nbdash.previous
```

If you didn't keep it, rebuild the previous tag with `VERSION=<tag> ./build.sh`
after `git checkout <tag>` and reinstall that.

### 9. Reading the logs

```sh
journalctl -u nbdash -f              # follow
journalctl -u nbdash -n 200           # recent history
journalctl -u nbdash -p err           # errors only
journalctl -u nbdash --since "10 min ago"
```

Logs are structured (`log/slog`, text handler, no timestamp field — journald
already stamps every line, so a second timestamp would just be visual noise).
`msg="request failed"` lines carry `path` and `error`; a burst of
`could not obtain bearer token` at the same moment as a provider outage or
clock skew is the token refresh failing, not a bug in the dashboard itself.

