package nbapi

import "time"

// The types below mirror the NetBird Management API schema, whose source of
// truth is shared/management/http/api/openapi.yml in the netbird repo. That
// spec already generates Go types (types.gen.go), but importing them would pull
// the entire netbird module — wireguard-go, gRPC, pion and the rest — into this
// binary's dependency graph for a handful of structs.
//
// So this file carries only the subset the dashboard renders, field-for-field
// with the spec. When the surface grows past a few resources, replace this file
// by running oapi-codegen against the same openapi.yml into this package; the
// names here are chosen to match what it emits.

// GroupMinimum is the group shape embedded in other resources.
type GroupMinimum struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	PeersCount     int     `json:"peers_count"`
	ResourcesCount int     `json:"resources_count"`
	Issued         *string `json:"issued,omitempty"`
}

// PeerLocalFlags are client-side toggles reported by the agent. Every field is
// a pointer in the spec: absent means the agent is too old to report it, which
// is different from false.
type PeerLocalFlags struct {
	BlockInbound          *bool `json:"block_inbound,omitempty"`
	BlockLanAccess        *bool `json:"block_lan_access,omitempty"`
	DisableClientRoutes   *bool `json:"disable_client_routes,omitempty"`
	DisableDNS            *bool `json:"disable_dns,omitempty"`
	DisableFirewall       *bool `json:"disable_firewall,omitempty"`
	DisableServerRoutes   *bool `json:"disable_server_routes,omitempty"`
	LazyConnectionEnabled *bool `json:"lazy_connection_enabled,omitempty"`
	RosenpassEnabled      *bool `json:"rosenpass_enabled,omitempty"`
	RosenpassPermissive   *bool `json:"rosenpass_permissive,omitempty"`
	ServerSSHAllowed      *bool `json:"server_ssh_allowed,omitempty"`
}

// Peer is a machine enrolled in the network.
type Peer struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IP       string `json:"ip"`
	IPv6     string `json:"ipv6,omitempty"`
	Hostname string `json:"hostname"`

	Connected bool      `json:"connected"`
	LastSeen  time.Time `json:"last_seen"`
	CreatedAt time.Time `json:"created_at"`

	OS            string `json:"os"`
	KernelVersion string `json:"kernel_version"`
	Version       string `json:"version"`
	UIVersion     string `json:"ui_version"`
	SerialNumber  string `json:"serial_number"`

	DNSLabel       string   `json:"dns_label"`
	ExtraDNSLabels []string `json:"extra_dns_labels"`

	Groups []GroupMinimum `json:"groups"`

	UserID string `json:"user_id"`

	SSHEnabled bool `json:"ssh_enabled"`
	Ephemeral  bool `json:"ephemeral"`

	LastLogin                   time.Time `json:"last_login"`
	LoginExpired                bool      `json:"login_expired"`
	LoginExpirationEnabled      bool      `json:"login_expiration_enabled"`
	InactivityExpirationEnabled bool      `json:"inactivity_expiration_enabled"`

	ApprovalRequired  bool    `json:"approval_required"`
	DisapprovalReason *string `json:"disapproval_reason,omitempty"`

	CityName     string `json:"city_name"`
	CountryCode  string `json:"country_code"`
	GeonameID    int    `json:"geoname_id"`
	ConnectionIP string `json:"connection_ip"`

	// AccessiblePeersCount only appears on the list endpoint, which returns the
	// spec's PeerBatch. It stays zero on a single-peer fetch.
	AccessiblePeersCount int `json:"accessible_peers_count"`

	LocalFlags *PeerLocalFlags `json:"local_flags,omitempty"`
}

// PeerRequest is the PUT body for /api/peers/{peerId}.
//
// The Management API replaces the whole editable set on every write, so a
// partial body silently clears the fields it omits. Callers must send the
// peer's current values for anything they are not changing.
type PeerRequest struct {
	Name                        string  `json:"name"`
	SSHEnabled                  bool    `json:"ssh_enabled"`
	LoginExpirationEnabled      bool    `json:"login_expiration_enabled"`
	InactivityExpirationEnabled bool    `json:"inactivity_expiration_enabled"`
	ApprovalRequired            *bool   `json:"approval_required,omitempty"`
	IP                          *string `json:"ip,omitempty"`
	IPv6                        *string `json:"ipv6,omitempty"`
}

// SetupKey authenticates a new machine onto the network. Field names and
// enum values were cross-checked against the running server's own handler
// code (management/server/http/handlers/setup_keys), not just the spec:
// Type is "one-off" or "reusable"; State is "valid", "expired", "revoked" or
// "overused", computed server-side from Valid/Revoked and never sent by the
// client.
//
// Key is masked (e.g. "A6160****") on every response except the one returned
// by CreateSetupKey, which carries the plaintext once. There is no separate
// "clear" type: the spec's SetupKeyClear and SetupKey schemas differ only in
// that one field's content, not its shape, so one Go struct serves both —
// same discipline as Peer covering the spec's Peer and PeerBatch.
//
// AutoGroups is a list of group IDs, not embedded group objects the way
// Peer.Groups is — the API never returns group names alongside a setup key.
// This dashboard has no Groups module yet, so the UI can only display raw
// IDs; see the create form's hint text.
type SetupKey struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	Key                 string    `json:"key"`
	Expires             time.Time `json:"expires"`
	Type                string    `json:"type"`
	Valid               bool      `json:"valid"`
	Revoked             bool      `json:"revoked"`
	UsedTimes           int       `json:"used_times"`
	LastUsed            time.Time `json:"last_used"`
	State               string    `json:"state"`
	AutoGroups          []string  `json:"auto_groups"`
	UpdatedAt           time.Time `json:"updated_at"`
	UsageLimit          int       `json:"usage_limit"`
	Ephemeral           bool      `json:"ephemeral"`
	AllowExtraDNSLabels bool      `json:"allow_extra_dns_labels"`
}

// CreateSetupKeyRequest is the POST body for /api/setup-keys. ExpiresIn is
// seconds, bounded by the API to [86400, 31536000] (1–365 days) — the create
// form collects days and does the multiplication.
type CreateSetupKeyRequest struct {
	Name                string   `json:"name"`
	Type                string   `json:"type"`
	ExpiresIn           int      `json:"expires_in"`
	AutoGroups          []string `json:"auto_groups"`
	UsageLimit          int      `json:"usage_limit"`
	Ephemeral           bool     `json:"ephemeral"`
	AllowExtraDNSLabels bool     `json:"allow_extra_dns_labels"`
}

// PeerMinimum is the peer shape embedded in a Group's peer list — a
// different, smaller pair (id, name) than Peer, and unrelated to
// GroupMinimum despite the naming symmetry.
type PeerMinimum struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Group is a named set of peers used to scope policies, routes, DNS zones and
// setup keys' auto_groups. Peers here carries resolved id+name pairs: the
// server's groups handler (toGroupResponse in
// management/server/http/handlers/groups) builds this list by cross
// referencing the account's live peers, not by echoing back the group's raw
// stored ID list — which is also why a peer ID with no matching live peer is
// silently dropped from it server-side, rather than appearing with a blank
// name.
//
// The built-in "All" group — every account has exactly one, every peer joins
// it automatically — cannot be renamed, have its membership edited, or be
// deleted; the server checks this by exact name match (types.Group.IsGroupAll
// in netbird), not a flag in this response, so callers do the same:
// `Name == "All"`.
type Group struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	PeersCount     int           `json:"peers_count"`
	ResourcesCount int           `json:"resources_count"`
	Issued         string        `json:"issued,omitempty"`
	Peers          []PeerMinimum `json:"peers"`
}

// GroupRequest is the POST/PUT body for /api/groups[/{groupId}].
//
// Peers replaces the group's whole membership on every write — the update
// handler diffs the submitted list against the group's current membership and
// removes anything missing, the same full-replace discipline PeerRequest and
// SetupKeyRequest document. Unlike SetupKeyRequest.AutoGroups, a nil or
// omitted Peers is not rejected here; the handler normalises it to an empty
// slice, which means an omitted Peers on an update clears the group entirely.
// The one place this dashboard sends this type for an existing group
// (updateGroup) always populates Peers from a form whose checkboxes were
// pre-checked from a fresh GetGroup fetch, so "whatever is checked" already is
// the complete desired membership — there is no separate merge step, because
// the full state was already in hand before the user changed anything.
//
// Resources is never set by this dashboard: there is no Networks module yet
// to manage them, and omitting the field leaves a group with none, which is
// the only state this app ever creates or edits one into.
type GroupRequest struct {
	Name  string   `json:"name"`
	Peers []string `json:"peers,omitempty"`
}

// SetupKeyRequest is the PUT body for /api/setup-keys/{keyId}.
//
// Unlike PeerRequest this is not the key's whole editable set — only revoked
// and auto_groups can be changed after creation. But the same full-replace
// rule applies to AutoGroups specifically: the handler rejects a nil value
// outright ("AutoGroups field is invalid"), so revoking a key means resending
// its current groups, not just flipping Revoked. Callers must never leave
// AutoGroups as a nil slice — encoding/json marshals that as JSON null, which
// is exactly the value the API rejects; an empty-but-non-nil slice is fine.
type SetupKeyRequest struct {
	Revoked    bool     `json:"revoked"`
	AutoGroups []string `json:"auto_groups"`
}

// PolicyRule is one access rule within a Policy, as the API returns it.
// Sources and Destinations are resolved id+name+count groups (GroupMinimum,
// the same shape Group.Peers uses), not bare IDs — see PolicyRuleRequest for
// the write side, where they are.
//
// The real spec also allows a rule to target a Resource instead of Sources —
// mutually exclusive with it, enforced server-side — but that's a Networks
// concept with no module here, so this type and everything that writes it
// only ever deals with group-based rules. Likewise AuthorizedGroups
// (netbird-ssh's per-source-group authorized-username list) has no
// supporting module and is never populated by this dashboard.
type PolicyRule struct {
	ID            string         `json:"id,omitempty"`
	Name          string         `json:"name"`
	Description   string         `json:"description,omitempty"`
	Enabled       bool           `json:"enabled"`
	Action        string         `json:"action"` // "accept" | "drop"
	Bidirectional bool           `json:"bidirectional"`
	Protocol      string         `json:"protocol"` // "all" | "tcp" | "udp" | "icmp" | "netbird-ssh"
	Sources       []GroupMinimum `json:"sources"`
	Destinations  []GroupMinimum `json:"destinations"`
	Ports         []string       `json:"ports,omitempty"`
}

// Policy is a named, ordered set of access rules controlling which peer
// groups may reach which other peer groups. SourcePostureChecks are posture
// check IDs applied to the policy's source groups — a peer that fails one
// loses access even if its group membership would otherwise allow it.
type Policy struct {
	ID                  string       `json:"id"`
	Name                string       `json:"name"`
	Description         string       `json:"description"`
	Enabled             bool         `json:"enabled"`
	Rules               []PolicyRule `json:"rules"`
	SourcePostureChecks []string     `json:"source_posture_checks"`
}

// PolicyRuleRequest is one rule within PolicyRequest.Rules — the write shape,
// where Sources and Destinations are bare group IDs rather than resolved
// GroupMinimum objects. Setting ID targets an existing rule when the
// enclosing PolicyRequest is a PUT; leaving it empty creates a new one.
type PolicyRuleRequest struct {
	ID            string   `json:"id,omitempty"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	Enabled       bool     `json:"enabled"`
	Action        string   `json:"action"`
	Bidirectional bool     `json:"bidirectional"`
	Protocol      string   `json:"protocol"`
	Sources       []string `json:"sources"`
	Destinations  []string `json:"destinations"`
	Ports         []string `json:"ports,omitempty"`
}

// PolicyRequest is the POST/PUT body for /api/policies[/{policyId}]. Both
// endpoints run through the exact same handler logic server-side (savePolicy
// in management/server/http/handlers/policies decides create-vs-update from
// a bool parameter, not from the request shape), so one Go type serves both —
// unlike CreateSetupKeyRequest/SetupKeyRequest, which really do differ.
//
// Rules replaces the policy's whole rule set on every write: savePolicy
// rebuilds policy.Rules from req.Rules unconditionally, so a write that omits
// an existing rule deletes it — the same full-replace discipline
// GroupRequest.Peers documents. At least one rule is required; the server
// rejects an empty list both on save and, defensively, on every read. This
// dashboard never sends a PolicyRequest built from scratch on every edit: any
// write to just the policy's name/description/enabled, or to a single rule,
// re-fetches the current Policy first and carries its other rules forward
// unchanged — see the handlers, not this type, for that discipline.
type PolicyRequest struct {
	Name                string              `json:"name"`
	Description         string              `json:"description,omitempty"`
	Enabled             bool                `json:"enabled"`
	Rules               []PolicyRuleRequest `json:"rules"`
	SourcePostureChecks []string            `json:"source_posture_checks,omitempty"`
}

// Network is a named container linking one or more routers (routing peers or
// peer groups) to one or more resources (hosts, subnets or domains) reachable
// through them. Unlike Policy.Rules or Group.Peers, Routers and Resources here
// are bare ID lists, not nested resolved objects — routers and resources are
// independent sub-resources with their own endpoints
// (/networks/{id}/routers, /networks/{id}/resources), each fetched
// separately when the detail page needs the full picture.
type Network struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Routers     []string `json:"routers"`
	Resources   []string `json:"resources"`
	// Policies lists every policy ID whose rule targets this network via a
	// Resource (mutually exclusive with a group-based Sources/Destinations
	// rule — see PolicyRule's doc comment). This dashboard's own policy rules
	// never populate it, since group-based rules are the only kind it
	// writes, but another client (the CLI, Terraform, the Next.js dashboard)
	// may have, so it is still read and shown.
	Policies          []string `json:"policies"`
	RoutingPeersCount int      `json:"routing_peers_count"`
}

// NetworkRequest is the POST/PUT body for /api/networks[/{networkId}]. A
// network has no nested editable set to full-replace the way Policy or Group
// does — Routers and Resources are independent sub-resources with their own
// endpoints, so renaming a network never touches them.
type NetworkRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// NetworkResourceType is derived by the server from a resource's Address —
// "host" for a bare IP or a /32 (or IPv6 /128) prefix, "subnet" for any other
// prefix, "domain" for a hostname or wildcard like *.example.com
// (GetResourceType in management/server/networks/resources/types). This
// dashboard never sends it; it only ever reads it back.
type NetworkResourceType string

// NetworkResource is one address reachable through a network's routers,
// scoped to whichever groups it belongs to — those groups are what a
// group-based Policy rule's Sources/Destinations actually reach. Unlike
// NetworkRouter's response shape, Groups here is already resolved to
// id+name+count pairs by the server, the same GroupMinimum shape
// Peer.Groups and PolicyRule.Sources use.
type NetworkResource struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Address     string              `json:"address"`
	Type        NetworkResourceType `json:"type"`
	Enabled     bool                `json:"enabled"`
	Groups      []GroupMinimum      `json:"groups"`
}

// NetworkResourceRequest is the POST/PUT body for a network's
// /resources[/{resourceId}]. Groups replaces the resource's whole group
// membership on every write, the same full-replace discipline
// GroupRequest.Peers documents — but since this is the entire mutable state
// of one independent resource rather than one array shared by a parent that
// also has other fields, there is no "carry the rest forward" concern the
// way Policy.Rules has.
type NetworkResourceRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Address     string   `json:"address"`
	Enabled     bool     `json:"enabled"`
	Groups      []string `json:"groups"`
}

// NetworkRouter directs traffic for a network through either a single peer or
// a set of peer groups — mutually exclusive, required, and enforced
// server-side (Validate in management/server/networks/routers/types).
// Metric follows "lower wins" routing convention; Masquerade controls source
// NAT for traffic sent through this router.
//
// Unlike NetworkResource.Groups, PeerGroups here is NOT resolved to names by
// the server — ToAPIResponse in the real server just echoes back the same
// raw IDs the request carried, and Peer is a bare peer ID too. This
// dashboard resolves both itself for display, the same way SetupKey.AutoGroups
// is resolved via pageData.GroupNames, plus a parallel PeerNames map for Peer.
type NetworkRouter struct {
	ID         string   `json:"id"`
	Peer       string   `json:"peer,omitempty"`
	PeerGroups []string `json:"peer_groups,omitempty"`
	Metric     int      `json:"metric"`
	Masquerade bool     `json:"masquerade"`
	Enabled    bool     `json:"enabled"`
}

// NetworkRouterRequest is the POST/PUT body for a network's
// /routers[/{routerId}]. Peer and PeerGroups are mutually exclusive: set
// exactly one, never both, never neither — the server rejects any other
// combination with "peer and peer_groups cannot be set at the same time" or
// "either peer or peer_groups must be provided".
//
// The real server also silently forces Enabled to true on create regardless
// of what this sends (createRouter in
// management/server/http/handlers/networks/routers_handler.go sets
// router.Enabled = true unconditionally, after decoding the request) — this
// dashboard's create form does not offer the toggle at all, since showing
// one the server ignores would be a promise it doesn't keep. Update has no
// such override and honours whatever this sends.
type NetworkRouterRequest struct {
	Peer       string   `json:"peer,omitempty"`
	PeerGroups []string `json:"peer_groups,omitempty"`
	Metric     int      `json:"metric"`
	Masquerade bool     `json:"masquerade"`
	Enabled    bool     `json:"enabled"`
}

// Nameserver is one DNS server within a NameserverGroup. NSType is always
// "udp" — the only value the spec's enum permits.
type Nameserver struct {
	IP     string `json:"ip"`
	NSType string `json:"ns_type"`
	Port   int    `json:"port"`
}

// NameserverGroup routes DNS queries for its Domains (or, if Primary, every
// query not otherwise matched) to its Nameservers, for whichever peers are
// in Groups. Primary and Domains are mutually exclusive and one is required
// (validateDomainInput in management/server/nameserver.go): Primary means
// "resolve everything else", Domains means "resolve only these" — the same
// either-or shape NetworkRouterRequest's Peer/PeerGroups uses, just with
// different field names. SearchDomainsEnabled is rejected outright when
// Primary is true (there is nothing to append a search domain to on a
// catch-all resolver).
type NameserverGroup struct {
	ID                   string       `json:"id"`
	Name                 string       `json:"name"`
	Description          string       `json:"description,omitempty"`
	Nameservers          []Nameserver `json:"nameservers"`
	Enabled              bool         `json:"enabled"`
	Groups               []string     `json:"groups"`
	Primary              bool         `json:"primary"`
	Domains              []string     `json:"domains,omitempty"`
	SearchDomainsEnabled bool         `json:"search_domains_enabled"`
}

// NameserverGroupRequest is the POST/PUT body for
// /api/dns/nameservers[/{nsgroupId}]. Name must be unique among the
// account's nameserver groups (validateNSGroupName) and 1-40 characters
// (nbdns.MaxGroupNameChar) — the server's own "should be 1 or 3" message on
// Nameservers is misleading: validateNSList only rejects 0 or more than 3,
// so 1, 2 or 3 are all valid, not just the two the message names.
type NameserverGroupRequest struct {
	Name                 string       `json:"name"`
	Description          string       `json:"description,omitempty"`
	Nameservers          []Nameserver `json:"nameservers"`
	Enabled              bool         `json:"enabled"`
	Groups               []string     `json:"groups"`
	Primary              bool         `json:"primary"`
	Domains              []string     `json:"domains,omitempty"`
	SearchDomainsEnabled bool         `json:"search_domains_enabled"`
}

// DNSSettings is a single account-wide object — GET/PUT only, no create or
// delete, unlike everything else in this file. DisabledManagementGroups
// lists groups whose peers skip NetBird's own DNS management entirely
// (their own OS resolver settings are left untouched), separate from and
// unrelated to which nameserver group resolves what.
type DNSSettings struct {
	DisabledManagementGroups []string `json:"disabled_management_groups"`
}

// Zone is a custom DNS zone: a domain this account serves its own records
// for, to whichever peers are in DistributionGroups. Domain is immutable
// once created — UpdateZone in management/internals/modules/zones/manager
// rejects a write that changes it ("zone domain cannot be updated") — and
// must be unique account-wide, checked against every other zone's domain
// and the account's own peer DNS domain alike.
type Zone struct {
	ID                 string      `json:"id"`
	Name               string      `json:"name"`
	Domain             string      `json:"domain"`
	Enabled            bool        `json:"enabled"`
	EnableSearchDomain bool        `json:"enable_search_domain"`
	DistributionGroups []string    `json:"distribution_groups"`
	Records            []DNSRecord `json:"records"`
}

// ZoneRequest is the POST/PUT body for /api/dns/zones[/{zoneId}]. At least
// one distribution group is required (Zone.Validate); Domain is only
// meaningful on create, since an update that changes it is rejected
// server-side (see Zone's doc comment) — this dashboard's own edit form
// never offers to change it, rather than build one whose value is silently
// ignored by the server the way NetworkRouterRequest's create-time Enabled
// briefly was misread to behave.
type ZoneRequest struct {
	Name               string   `json:"name"`
	Domain             string   `json:"domain"`
	Enabled            bool     `json:"enabled"`
	EnableSearchDomain bool     `json:"enable_search_domain"`
	DistributionGroups []string `json:"distribution_groups"`
}

// DNSRecord is one entry within a Zone. Despite the OpenAPI spec's
// description text claiming Name "must be a subdomain within or match the
// zone's domain", the real server's Record.Validate
// (management/internals/modules/zones/records/record.go) only checks it is
// a syntactically valid domain — there is no check that it relates to the
// owning zone's Domain at all. This dashboard does not invent a stricter
// client-side rule the server itself does not enforce.
type DNSRecord struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"` // "A" | "AAAA" | "CNAME"
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
}

// DNSRecordRequest is the POST/PUT body for a zone's
// /records[/{recordId}]. Content's expected format depends on Type: an
// IPv4 address for "A", an IPv6 address for "AAAA", a domain (no wildcard)
// for "CNAME" — validated server-side per type, not just "any string".
type DNSRecordRequest struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
}

// User is an account member — either a regular, IdP-backed user or a
// service user (an API-only identity that exists to hold
// PersonalAccessTokens). Both shapes share this one type, the same
// discipline SetupKey uses for its spec's SetupKey/SetupKeyClear split.
//
// The Management API has no single-user GET — only GET /users (the whole
// list), PUT and DELETE. Every "detail" page in this dashboard fetches the
// full list via ListUsers and finds the one it wants by ID, rather than a
// GetUser call that does not exist.
type User struct {
	ID              string          `json:"id"`
	Email           string          `json:"email,omitempty"`
	Name            string          `json:"name"`
	Role            string          `json:"role"`
	Status          string          `json:"status"` // "active" | "invited" | "blocked"
	AutoGroups      []string        `json:"auto_groups"`
	IsCurrent       bool            `json:"is_current,omitempty"`
	IsServiceUser   bool            `json:"is_service_user,omitempty"`
	IsBlocked       bool            `json:"is_blocked"`
	PendingApproval bool            `json:"pending_approval"`
	LastLogin       time.Time       `json:"last_login,omitempty"`
	Permissions     UserPermissions `json:"permissions,omitempty"`
}

// UserPermissions is what this user's role actually grants them, returned
// alongside the user object rather than derived client-side from the role
// name — the server is the source of truth for what each module's
// read/create/update/delete booleans resolve to for a given role.
type UserPermissions struct {
	IsRestricted bool                       `json:"is_restricted"`
	Modules      map[string]map[string]bool `json:"modules"`
}

// UserRequest is the PUT body for /api/users/{userId}. Role, AutoGroups and
// IsBlocked are the only fields the server accepts an update to
// (processUserUpdate's own comment: "only auto groups, revoked status, and
// integration reference can be updated for now") — Name and Email come from
// the IdP and cannot be changed through this dashboard.
//
// Setting Role to "owner" does not just change a permission level — it
// transfers account ownership to this user (handleOwnerRoleTransfer in
// management/server/user.go), and only the current owner is allowed to send
// it. This dashboard's role picker never offers "owner" as a selectable
// target for exactly that reason; see user_edit_fields.html's doc comment.
// An admin editing their own account cannot change Role or IsBlocked at all
// (validateUserUpdate rejects both) — see updateUser's doc comment for how
// this dashboard's form avoids submitting a change that would just be
// rejected.
type UserRequest struct {
	Role       string   `json:"role"`
	AutoGroups []string `json:"auto_groups"`
	IsBlocked  bool     `json:"is_blocked"`
}

// UserCreateRequest is the POST body for /api/users. IsServiceUser decides
// which of two different server-side operations happens: true creates a
// service user directly (Name/Role/AutoGroups only, no Email, no IdP
// involved — createServiceUser); false sends a real invite through the
// configured IdP (Name and Email both required — inviteNewUser, which
// fails outright if no IdP manager is configured at all). Role can never be
// "owner" here either — invite/create both reject it server-side
// (validateUserInvite, createServiceUser).
type UserCreateRequest struct {
	Email         string   `json:"email,omitempty"`
	Name          string   `json:"name,omitempty"`
	Role          string   `json:"role"`
	AutoGroups    []string `json:"auto_groups"`
	IsServiceUser bool     `json:"is_service_user"`
}

// PersonalAccessToken authenticates API calls as the user it belongs to —
// typically a service user, though the API does not restrict it to one.
// Name and ExpirationDate are the only fields ever written; CreatedBy,
// CreatedAt and LastUsed are server-computed.
type PersonalAccessToken struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	ExpirationDate time.Time `json:"expiration_date"`
	CreatedBy      string    `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
	LastUsed       time.Time `json:"last_used,omitempty"`
}

// PersonalAccessTokenGenerated is CreatePAT's response — the only place the
// plaintext token ever exists in this codebase. Every other read
// (ListPATs) returns just PersonalAccessToken, with no token value at all —
// there is no masked form the way SetupKey.Key has one, because a PAT is
// never displayed again after creation, not even partially.
type PersonalAccessTokenGenerated struct {
	PlainToken          string              `json:"plain_token"`
	PersonalAccessToken PersonalAccessToken `json:"personal_access_token"`
}

// PersonalAccessTokenRequest is the POST body for a user's
// /tokens[/{tokenId}]. ExpiresIn is days, bounded [1, 365] server-side
// (account.PATMinExpireDays/PATMaxExpireDays) — unlike
// CreateSetupKeyRequest.ExpiresIn, which is seconds.
type PersonalAccessTokenRequest struct {
	Name      string `json:"name"`
	ExpiresIn int    `json:"expires_in"`
}

// IdentityProviderType is one of the SSO connector types the Management
// server's embedded Dex instance can drive. "oidc" is generic; the rest map
// to one of Dex's built-in connectors.
//
// The checked-in OpenAPI spec (shared/management/http/api/openapi.yml) only
// lists 8 of these — it is missing Authentik and Keycloak, both of which are
// nonetheless fully implemented server-side
// (management/server/types/identity_provider.go's own const block defines
// all 10, and dex/connector.go's buildOIDCConnectorConfig switches on both).
// Treat the Go source, not the spec, as authoritative here.
type IdentityProviderType string

const (
	IdentityProviderOIDC      IdentityProviderType = "oidc"
	IdentityProviderZitadel   IdentityProviderType = "zitadel"
	IdentityProviderEntra     IdentityProviderType = "entra"
	IdentityProviderGoogle    IdentityProviderType = "google"
	IdentityProviderOkta      IdentityProviderType = "okta"
	IdentityProviderPocketID  IdentityProviderType = "pocketid"
	IdentityProviderMicrosoft IdentityProviderType = "microsoft"
	IdentityProviderADFS      IdentityProviderType = "adfs"
	IdentityProviderAuthentik IdentityProviderType = "authentik"
	IdentityProviderKeycloak  IdentityProviderType = "keycloak"
)

// IdentityProviderTypes lists every supported type in the order offered on
// the create/edit form.
var IdentityProviderTypes = []IdentityProviderType{
	IdentityProviderOIDC,
	IdentityProviderGoogle,
	IdentityProviderMicrosoft,
	IdentityProviderEntra,
	IdentityProviderOkta,
	IdentityProviderZitadel,
	IdentityProviderPocketID,
	IdentityProviderAuthentik,
	IdentityProviderKeycloak,
	IdentityProviderADFS,
}

// HasBuiltInIssuer reports whether the server ignores any user-supplied
// Issuer for this type. Google and Microsoft use Dex connectors with
// hardcoded endpoints (dex/connector.go's buildOAuth2ConnectorConfig) — the
// server's own Validate() skips the "issuer required" check for exactly
// these two, and a value submitted anyway is silently ignored, not stored.
func (t IdentityProviderType) HasBuiltInIssuer() bool {
	return t == IdentityProviderGoogle || t == IdentityProviderMicrosoft
}

// IdentityProvider is an SSO connector configured on the account, as the API
// returns it. ClientSecret is deliberately absent — the server never sends
// it back (management/server/http/handlers/idp/idp_handler.go's
// toAPIResponse strips it explicitly), so there is no masked form the way
// SetupKey.Key has one.
type IdentityProvider struct {
	ID       string               `json:"id"`
	Type     IdentityProviderType `json:"type"`
	Name     string               `json:"name"`
	Issuer   string               `json:"issuer"`
	ClientID string               `json:"client_id"`
}

// IdentityProviderRequest is the POST (create) and PUT (update, at
// /identity-providers/{idpId}) body — both use the same shape.
//
// On update, an empty ClientSecret is not a no-op write of a blank secret:
// the server's storage layer does a partial-overlay update
// (idp/dex/connector.go's overlayConnectorConfig) that only writes a field
// when it is non-empty, so leaving ClientSecret blank preserves whatever
// secret is already stored. Issuer and ClientID get the same overlay
// treatment. Callers building an edit form should surface this as "leave
// blank to keep the existing secret," not require re-entry on every save.
//
// Issuer is required unless Type.HasBuiltInIssuer() — and when it is
// required, the server does a live outbound HTTP GET to
// {issuer}/.well-known/openid-configuration (10s timeout) and rejects the
// write if the discovery document's own issuer field doesn't match what was
// submitted. A create/update call can therefore take a few seconds and can
// fail for reasons no client-side validation catches.
type IdentityProviderRequest struct {
	Type         IdentityProviderType `json:"type"`
	Name         string               `json:"name"`
	Issuer       string               `json:"issuer"`
	ClientID     string               `json:"client_id"`
	ClientSecret string               `json:"client_secret,omitempty"`
}
