# Design notes

Why nbdash is built the way it is: the decisions that are not obvious from
reading the code, and the Management API behaviours that forced them.

**Types are hand-written, not generated.** `internal/nbapi/types.go` mirrors
`shared/management/http/api/openapi.yml` from the netbird repo. That spec already
generates Go types (`types.gen.go`), but importing them pulls the entire netbird
module — wireguard-go, gRPC, pion — in for a handful of structs. Once the ported
surface grows past a few resources, run `oapi-codegen` against the same
`openapi.yml` into this package; the names here already match what it emits.

**Peers are fetched in full and filtered in-process.** The API's `?name=` and
`?ip=` filters cannot express a search that spans name, hostname, address, OS and
groups at once. The Next.js dashboard also loads the list in full and filters
client-side. Revisit if accounts reach thousands of peers.

**Sessions are in-memory.** They do not survive a restart or span replicas.
`internal/auth.Store` is the only thing to replace for Redis or signed cookies.

**`PeerRequest` replaces the whole editable set.** The Management API treats a
`PUT` as a full replacement, so a partial body silently clears omitted fields.
The settings form always submits every toggle it owns.

**A setup key's plaintext is shown exactly once, by construction, not by
convention.** `CreateSetupKey`'s response is the only place the plaintext ever
exists in this codebase; every other endpoint returns it masked
(`"A6160****"`). `pageData.CreatedSetupKey` is populated only on that one
response and is never fetched by a GET, so there is no "hide it now" logic to
get wrong — reloading the page is what makes the reveal disappear, because
nothing server-side is still holding the value to show. Nothing in this path
logs the created key; `TestCreatedSetupKeyPlaintextNeverReachesLogger` pins
that with a capturing logger rather than trusting a code review to catch a
future regression.

**`SetupKeyRequest.AutoGroups` must never be a nil slice.** The update
endpoint's handler rejects `nil` outright (`"AutoGroups field is invalid"`) —
a nil Go slice marshals to JSON `null`, and that's the exact value it rejects.
Revoking a key means fetching its current groups first and resending them
(the endpoint replaces the whole editable set, same as `PeerRequest`), and
`revokeSetupKey` normalises a nil result from that fetch to `[]string{}`
before resending. `CreateSetupKeyRequest.AutoGroups` is not held to this: the
create endpoint's handler normalises `nil` to an empty slice itself, so the
create form sends `r.Form["auto_groups"]` straight through with no guard —
only the update path needs one, because only its handler is strict about it.

**A group edit form is safe to full-replace precisely because it always
starts from the true current state.** `GroupRequest.Peers` replaces a group's
whole membership on every write — `UpdateGroup` diffs the submitted list
against what the group actually has and removes anything missing, so an edit
form that didn't show (and resend) current members would silently clear them.
This dashboard avoids that by construction rather than by convention:
`groupDetail` always renders the form from a fresh `GetGroup` call, with every
peer checkbox's checked state precomputed from that response
(`pageData.SelectedPeerIDs`), so whatever is checked at submit time already
*is* the complete desired membership — there is no separate "merge with
current" step, because the current state was already on the page.

**A policy's rules are edited as sub-resources, not through one dynamic
multi-rule form.** `PolicyRequest.Rules` replaces a policy's *entire* rule set
on every write — `savePolicy` server-side rebuilds `policy.Rules` from the
request unconditionally, so a write that omits an existing rule deletes it.
Rather than build a client-side form that can add/remove/reorder an arbitrary
number of rule blocks, every rule mutation (`createRule`, `updateRule`,
`deleteRule`) follows the same "fetch, mutate one thing, resend everything"
shape `revokeSetupKey` established for `AutoGroups` and `updateGroup`
established for membership: fetch the policy fresh with `GetPolicy`, change
exactly the one rule being added/edited/removed, convert every other rule
back to its request shape unchanged (`policyRuleToRequest`), and `PUT` the
whole set back. `deleteRule` additionally refuses client-side to remove a
policy's last rule — the server requires at least one, and "policy rules
cannot be empty" is a confusing error for an action that looks like "delete a
rule," not "empty out a whole policy."

**A network's resources and routers are independent REST sub-resources, not
one full-replaced array.** Unlike a policy's rules, `/networks/{id}/resources`
and `/networks/{id}/routers` each have their own ID and their own
GET/PUT/DELETE endpoints server-side — so, unlike `createRule`/`updateRule`,
`createResource`/`updateResource`/`createRouter`/`updateRouter` never fetch
the parent network and resend anything else; each addresses and replaces
exactly the one sub-resource it targets. The only full-replace-on-write
discipline that survives from Groups/Policies is one level further in:
`NetworkResourceRequest.Groups` replaces one resource's *own* whole group
membership, so `resourceDetail` still has to render its edit form from a
fresh `GetNetworkResource` the same way `groupDetail` does for group members.

**A network router's create endpoint silently ignores the `enabled` field it
was sent.** `createRouter` in the real server's
`networks/routers_handler.go` builds the router from the request and then
sets `router.Enabled = true` unconditionally, regardless of what the request
body carried — confirmed by reading that handler, not assumed. This
dashboard's create form doesn't offer the toggle at all, rather than show a
control the server would silently overrule; the toggle only appears on the
edit form, where the override does not apply and the field is honoured
normally. `NetworkRouterRequest`'s doc comment in `internal/nbapi/types.go`
carries the same note next to the type it affects.

**A network router's `Peer`/`PeerGroups` aren't resolved to names by the
server, unlike a `NetworkResource`'s `Groups`.** The read response just
echoes back the same raw IDs the write side uses — confirmed with a pinned
test (`TestGetNetworkRouterReturnsRawPeerGroupIDs`) precisely so a future
server change that starts resolving them shows up as a test failure here
instead of a silent UI regression. This dashboard resolves both itself for
display: `pageData.PeerNames` (new) and the same `pageData.GroupNames` map
Setup Keys' `AutoGroups` already established, both built once per request
that needs them rather than repeatedly in the template.

**DNS bundles three independent sub-features behind one sidebar item, the
same tab layout the Next.js dashboard uses.** Nameserver Groups and Zones are
both normal List/Create/Edit/Delete resources; DNS Settings is a singleton —
`GET`/`PUT` only, no list, no ID, no create or delete, because there is
exactly one per account. All three share one `dns_tabs.html` partial and a
second-level `pageData.DNSTab` distinct from the sidebar's own
`Active="dns"`.

**A nameserver group's `Primary`, `Domains` and `SearchDomainsEnabled` form a
tri-state constraint, not three independent booleans.** `Primary` and a
non-empty `Domains` are mutually exclusive and one is required
(`validateDomainInput` in `management/server/nameserver.go`), and
`SearchDomainsEnabled` is rejected outright when `Primary` is true — the same
either-or shape `NetworkRouterRequest`'s `Peer`/`PeerGroups` uses, just named
differently. The server's own validation error on nameserver count is
misleading: it says "the list of nameservers should be 1 or 3," but the
actual check (`validateNSList`) only rejects 0 or more than 3, so 1, 2 or 3
are all valid — this dashboard's nameserver-count validation matches the
code, not the message. Because the count is capped at 3, the create/edit form
renders three fixed, independently-optional IP+port slots rather than a
dynamic add/remove list — no client-side JS needed to stay within the only
range the server will ever accept.

**A DNS zone's `Domain` is immutable after creation.** `UpdateZone` in
`management/internals/modules/zones/manager` rejects a write that changes it
("zone domain cannot be updated"), so the edit form renders it `readonly`
rather than offer a control that would silently fail to do what it looks
like it does — `readonly`, not `disabled`, because `ZoneRequest.Domain` is
required on every write and a `disabled` input submits nothing at all.
Domain must also be unique account-wide, checked against every other zone's
domain and the account's own peer DNS domain alike.

**A DNS record's name validation is looser than the OpenAPI spec's
description text implies.** The spec says a record's `name` "must be a
subdomain within or match the zone's domain," but `Record.Validate` in
`management/internals/modules/zones/records/record.go` only checks it is a
syntactically valid domain — there is no check relating it to the owning
zone's domain at all. This dashboard does not invent a stricter client-side
rule the server itself does not enforce; the whole point of reading the
handler and manager code for every feature in this project, not just the
spec, is catching gaps exactly like this one.

**The Management API has no single-user GET.** `/api/users` supports GET
(the whole list), POST; `/api/users/{userId}` supports PUT and DELETE — there
is no `GET /api/users/{userId}`. Every user "detail" page in this dashboard
fetches the full, correctly-filtered list (`?service_user=true` or `false`)
and finds the one it wants by ID (`findUser`), rather than a `GetUser` call
that does not exist. Regular users and service users share one `nbapi.User`
Go type — the API itself does, distinguishing only by `is_service_user` — but
every fetch, create and render path keeps the two explicitly separated so
the Users and Service Users tabs' tables never mix.

**Setting a user's role to "owner" doesn't just change a permission level —
it transfers account ownership**, and only the current owner may do it
(`handleOwnerRoleTransfer` in `management/server/user.go`). This dashboard's
role picker never offers "owner" as a selectable target for exactly that
reason, on both the invite/create forms (where the server would reject it
outright — `validateUserInvite`, `createServiceUser`) and the edit form
(where sending it would succeed for an owner and silently transfer
ownership for anyone else). The option only ever appears already-selected,
and only for a user who already holds the role — see
`user_edit_fields.html`'s doc comment and `UserRequest`'s in
`internal/nbapi/types.go`.

**An admin editing their own account can't change their role or block
themselves at all** (`validateUserUpdate` rejects both server-side). Rather
than submit a change the server would just reject, the edit form swaps the
role `<select>` for a static display plus a hidden field carrying the
current value forward when `.User.IsCurrent`, and omits the blocked toggle
entirely — correct as well as safe, since a blocked user has no session to
reach this form with in the first place. The same reasoning that led Zone's
domain field to `readonly` instead of `disabled` applies here too: a
`disabled` `<select>` submits nothing, so the hidden field is what actually
carries the value through.

**A Personal Access Token's plaintext is shown exactly once, by
construction, matching Setup Keys' `CreatedSetupKey` exactly.**
`pageData.CreatedPAT` is populated only by a successful token create and
never by a GET, so — as with setup keys — there is no "hide it now" logic
to get wrong; reloading the page is what makes the reveal disappear, because
nothing server-side is still holding the plaintext to show.
`TestCreatedPATPlaintextNeverReachesLogger` pins that with a capturing
logger, the same test shape `TestCreatedSetupKeyPlaintextNeverReachesLogger`
already established.

## Porting the rest

Every module in the sidebar's own list is ported, each the same four pieces
the Peers slice first demonstrated:

1. Types in `internal/nbapi/` mirrored from `openapi.yml`.
2. Endpoint methods in a new file next to `peers.go`.
3. Handlers + routes in `internal/handlers/`.
4. A page template, plus partials for whatever htmx swaps.

Peers, Groups, Setup Keys, Access Control, Networks, DNS and Team together are
the reference implementation for that shape. Control Center is the one
deliberate omission: it is a React Flow graph, and no server-rendered
equivalent is sketched here — porting it would mean designing a new UI
paradigm for this codebase, not applying the existing one to an eighth
resource.

Team's own scope was narrowed once, on purpose: the Next.js dashboard also
offers a link-based self-service invite flow (`POST /api/users/invites`,
regenerate, and a public, unauthenticated `/accept` endpoint where the
invitee sets their own password before ever reaching an OIDC login). That
flow doesn't fit this dashboard's architecture — every route except
`/login`, `/auth/callback` and `/logout` sits behind the OIDC session
middleware, and an invitee accepting a link has no session yet by
definition. Building it would mean adding a second, parallel authentication
path, not another CRUD module — a large enough change to warrant its own
design pass rather than folding it into Team's four-piece template. This
dashboard's Users tab uses the classic `POST /api/users` invite instead,
which sends the invite through the configured IdP and needs nothing public.
Self-service password change (`PUT /api/users/{userId}/password`, "users can
only change their own password") was left out for the same shape of reason:
it's a "my account" feature, and no "my account" module exists here to hang
it off of.

## Memory

The systemd unit caps this process at `MemoryHigh=180M` / `MemoryMax=220M`.
Those numbers are measured, not guessed, and this is where the measurement
lives so the unit file does not carry a dated table that will rot.

**Methodology.** A fake OIDC issuer and a fake Management API, `GOMAXPROCS=1`
to model a single-vCPU host, exercising every module rather than just `/peers`:

| Scenario | RSS |
|---|---|
| idle after startup | 24.5 MiB |
| after one login + one page render | 25.0 MiB |
| 1 session, 600 req burst (30 workers) | 35.0 MiB |
| 1 session, 3000 req burst (60 workers) | 35.4 MiB |
| 25 concurrent distinct sessions | 36.2 MiB |
| sustained: 5 more full 25-session rounds | 36.4–37.4 MiB (plateaued, not still climbing) |
| 10s after the heaviest round, idle | 37.1 MiB (no further release observed) |

Peak under sustained heavy synthetic load stayed under **37.5 MiB**.
`MemoryHigh` is ~4.8x that and `MemoryMax` ~5.9x — enough margin that normal
operation and a genuine traffic spike never approach the ceiling, while still
capping the worst case (a bug, a runaway client) at under a quarter of a 1 GiB
host, far below what would threaten a co-located `netbird-server`'s headroom.

**Two caveats that matter.**

The dataset was small — a handful of peers, groups and policies. Peers are
fetched in full and filtered in-process (see above), so RSS scales with account
size in a way this measurement does not capture. A large account will sit
higher.

And it was measured in a 2-core CI sandbox with `GOMAXPROCS` pinned to 1, not
on a real single-vCPU VPS. That is a reasonable proxy — same Go runtime, same
binary, no cgo — but it is a proxy. Re-check against your own deployment:

```sh
systemctl status nbdash | grep Memory
ps -o rss -p "$(pgrep nbdash)"
```
