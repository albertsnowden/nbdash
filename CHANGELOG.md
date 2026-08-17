# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project aims
to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html) from its
first tagged release.

## [Unreleased]

First public release preparation. The project was renamed from
`netbird-dashboard` / `go-dashboard` to **nbdash** in this window; anyone
running a pre-release build should read the migration note below.

### Added

- BSD-3-Clause `LICENSE`, `THIRD-PARTY-NOTICES.md`, `SECURITY.md`,
  `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md` and a Dependabot config — the project
  had no license file at all before this, which left it legally unusable by
  anyone else.
- Vendored htmx now carries its 0BSD license text, upstream URL, version
  (2.0.4) and a checksum, in `internal/web/static/htmx.min.js.LICENSE`.
- The `Authenticator` takes a logger and records failed logins: provider-side
  callback errors, failed code exchanges, ID-token verification failures, and
  nonce mismatches. None of this was previously observable.
- Tests pinning the security fixes below: logout method and cross-site
  rejection, absence of static directory listings, non-reflection of provider
  error parameters, and the `__Host-` cookie prefix.

### Changed

- **Renamed to `nbdash`.** Module path is now
  `github.com/albertsnowden/nbdash`. The binary, the `.deb`/`.rpm` package, and
  the systemd unit are all `nbdash`; configuration moved from
  `/etc/netbird-dashboard/dashboard.env` to `/etc/nbdash/nbdash.env`. The UI no
  longer presents itself as "NetBird" — nbdash is an unofficial project and now
  says so.
- Colour tokens in `app.css` are now derived from Tailwind's published `slate`
  and `indigo` ramps (MIT) with semantic names, replacing values previously
  copied from the AGPL-licensed official dashboard.
- `go.mod` requires Go 1.26.6.
- The README was split into `README.md`, `docs/deploying.md` and
  `docs/design.md`. Deployment docs no longer reference a specific private
  host.
- CI pins `govulncheck` and `nfpm` instead of installing `@latest` on every
  push.

### Fixed

- **8 Go standard-library vulnerabilities**, by requiring Go 1.26.6. The most
  significant is `GO-2026-6091`, a JavaScript-context tracking bug in
  `html/template` — the library every page here is rendered through.
- **Unbounded OIDC network calls.** Discovery, code exchange, token refresh and
  JWKS fetches all fell back to `http.DefaultClient`, which has no timeout. A
  refresh additionally holds the session mutex, so one unresponsive identity
  provider could block every request for that session indefinitely. All
  provider calls now run on a client with a 15-second timeout.
- **Logout was a state-changing `GET`**, and therefore exempt from the
  same-site origin check — any page could sign an admin out with an `<img>`
  tag. It is now a `POST`.
- **The OIDC callback reflected provider-supplied `error` and
  `error_description` parameters** into its response. Not executable script
  (`text/plain` plus `nosniff`), but it let a crafted link render arbitrary
  text on the dashboard's own origin. They are now logged, not displayed.
- **The session and login-state cookies now use the `__Host-` prefix** on https
  deployments, so a sibling subdomain cannot overwrite them — a gap `SameSite`
  does not close.
- **`/static/` served a directory listing**, handing an unauthenticated visitor
  a free inventory of the origin's assets.
- Two `onclick="this.select()"` attributes never ran: the CSP is
  `script-src 'self'` with no `'unsafe-inline'`, so the browser blocked them
  and logged a violation on every render. Moved into `copy.js`.
- `.tag-dead` referenced an undefined `--nb-gray-600` custom property and
  rendered with an inherited colour.

### Migration

Upgrading from a pre-rename build is not automatic — the package name changed,
so the new package will not replace the old one:

```sh
sudo cp /etc/netbird-dashboard/dashboard.env /root/nbdash.env.bak
sudo dpkg -P netbird-dashboard        # or: sudo rpm -e netbird-dashboard
sudo dpkg -i nbdash_*_amd64.deb       # or: sudo rpm -i nbdash-*.rpm
sudo cp /root/nbdash.env.bak /etc/nbdash/nbdash.env
sudo chmod 0600 /etc/nbdash/nbdash.env
sudo systemctl enable --now nbdash
```

The environment file's contents are unchanged — only its path moved.

[Unreleased]: https://github.com/albertsnowden/nbdash/commits/main
