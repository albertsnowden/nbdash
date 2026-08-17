// Package api is nbdash's whole HTTP surface: the JSON backend for the
// React SPA embedded at / (see internal/web.AppShell), plus the OIDC
// login/callback/logout routes and static asset serving every other route
// sits behind. It replaced internal/handlers' HTML/htmx router at the
// project's M3 cutover (see the migration plan) — before that, the two
// coexisted with the SPA reachable only under /app while it was being
// built out feature by feature.
//
// It deliberately does not proxy the Management API blindly: every
// /api/bff/... route calls a specific internal/nbapi method and shapes a
// specific JSON response, the same "types and endpoints hand-mirrored from
// openapi.yml" discipline internal/nbapi itself already follows (see its
// package doc comment). A generic pass-through would re-expose the entire
// upstream API surface unreviewed and have nowhere to hang the business
// logic several resources carry (e.g. Identity Providers' "blank secret
// preserves the stored one" in internal/nbapi/identityproviders.go).
package api

import (
	"log/slog"
	"net/http"

	"github.com/albertsnowden/nbdash/internal/auth"
	"github.com/albertsnowden/nbdash/internal/config"
	"github.com/albertsnowden/nbdash/internal/nbapi"
	"github.com/albertsnowden/nbdash/internal/web"
)

type Server struct {
	cfg  *config.Config
	auth *auth.Authenticator
	api  *nbapi.Client
	log  *slog.Logger
}

func New(cfg *config.Config, authenticator *auth.Authenticator, api *nbapi.Client, log *slog.Logger) *Server {
	return &Server{cfg: cfg, auth: authenticator, api: api, log: log}
}

// Routes builds the whole router.
//
// The SPA shell (index.html) is deliberately public, like /static/ — it's
// inert UI code with no user data. Everything it actually needs is behind
// /api/bff/..., gated by Middleware below, not by a check on the page
// request itself; the frontend's own apiFetch() wrapper redirects to
// /login on a 401 from that layer. This means there is no server-side
// "you must be logged in to load the app" check the way the old htmx
// pages had one per route — the SPA always loads, and decides for itself
// whether it has a session by asking the API.
func (s *Server) Routes() (http.Handler, error) {
	static, err := web.Static()
	if err != nil {
		return nil, err
	}
	appShell, err := web.AppShell()
	if err != nil {
		return nil, err
	}

	bffMux := http.NewServeMux()
	s.RegisterRoutes(bffMux)

	mux := http.NewServeMux()
	mux.Handle("GET /static/", static)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /login", s.auth.Login)
	mux.HandleFunc("GET /auth/callback", s.auth.Callback)
	// POST, not GET — see Authenticator.Logout. Signing out is a state change,
	// and requireSameSiteOrigin only checks unsafe methods.
	mux.HandleFunc("POST /logout", s.auth.Logout)
	mux.Handle("/api/bff/", s.Middleware(bffMux))
	// Catch-all: every other path — including "/" — serves the SPA shell,
	// which owns its own client-side routing from there (react-router,
	// basename "/"). This is deliberately the least specific pattern
	// registered; Go's mux always prefers a more specific match (/static/,
	// /login, ...) over this regardless of registration order.
	//
	// No method prefix, even though only GET ever reaches it in practice:
	// "GET /{path...}" and the method-agnostic "/api/bff/" pattern above
	// each dominate the other in one dimension (method vs. path specificity)
	// and neither in both, which net/http.ServeMux treats as a genuine
	// registration-time conflict and panics on. Dropping the method here
	// matches /api/bff/'s style, so path specificity alone orders every
	// pattern in this mux unambiguously.
	mux.HandleFunc("/{path...}", appShell)

	return limitRequestBody(securityHeaders(s.cfg.Secure, requireSameSiteOrigin(mux))), nil
}

// RegisterRoutes adds every /api/bff/... route onto mux. The caller is
// expected to wrap the result of this in Middleware (below) before mounting
// it — these handlers all call s.token(w,r), which reads the session
// auth.NewContext attaches to the request, and nothing here attaches one
// itself.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/bff/healthz", s.healthz)

	mux.HandleFunc("GET /api/bff/peers", s.listPeers)
	mux.HandleFunc("GET /api/bff/peers/{id}", s.getPeer)
	mux.HandleFunc("PUT /api/bff/peers/{id}", s.updatePeer)
	mux.HandleFunc("DELETE /api/bff/peers/{id}", s.deletePeer)

	mux.HandleFunc("GET /api/bff/groups", s.listGroups)
	mux.HandleFunc("GET /api/bff/groups/{id}", s.getGroup)
	mux.HandleFunc("POST /api/bff/groups", s.createGroup)
	mux.HandleFunc("PUT /api/bff/groups/{id}", s.updateGroup)
	mux.HandleFunc("DELETE /api/bff/groups/{id}", s.deleteGroup)

	mux.HandleFunc("GET /api/bff/setup-keys", s.listSetupKeys)
	mux.HandleFunc("POST /api/bff/setup-keys", s.createSetupKey)
	mux.HandleFunc("POST /api/bff/setup-keys/{id}/revoke", s.revokeSetupKey)
	mux.HandleFunc("DELETE /api/bff/setup-keys/{id}", s.deleteSetupKey)

	mux.HandleFunc("GET /api/bff/policies", s.listPolicies)
	mux.HandleFunc("GET /api/bff/policies/{id}", s.getPolicy)
	mux.HandleFunc("POST /api/bff/policies", s.createPolicy)
	mux.HandleFunc("PUT /api/bff/policies/{id}", s.updatePolicyMeta)
	mux.HandleFunc("DELETE /api/bff/policies/{id}", s.deletePolicy)
	mux.HandleFunc("POST /api/bff/policies/{id}/rules", s.createRule)
	mux.HandleFunc("PUT /api/bff/policies/{id}/rules/{ruleID}", s.updateRule)
	mux.HandleFunc("DELETE /api/bff/policies/{id}/rules/{ruleID}", s.deleteRule)

	mux.HandleFunc("GET /api/bff/networks", s.listNetworks)
	mux.HandleFunc("GET /api/bff/networks/{id}", s.getNetwork)
	mux.HandleFunc("POST /api/bff/networks", s.createNetwork)
	mux.HandleFunc("PUT /api/bff/networks/{id}", s.updateNetworkMeta)
	mux.HandleFunc("DELETE /api/bff/networks/{id}", s.deleteNetwork)
	mux.HandleFunc("GET /api/bff/networks/{id}/resources", s.listResources)
	mux.HandleFunc("POST /api/bff/networks/{id}/resources", s.createResource)
	mux.HandleFunc("PUT /api/bff/networks/{id}/resources/{resourceID}", s.updateResource)
	mux.HandleFunc("DELETE /api/bff/networks/{id}/resources/{resourceID}", s.deleteResource)
	mux.HandleFunc("GET /api/bff/networks/{id}/routers", s.listRouters)
	mux.HandleFunc("POST /api/bff/networks/{id}/routers", s.createRouter)
	mux.HandleFunc("PUT /api/bff/networks/{id}/routers/{routerID}", s.updateRouter)
	mux.HandleFunc("DELETE /api/bff/networks/{id}/routers/{routerID}", s.deleteRouter)

	mux.HandleFunc("GET /api/bff/dns/nameservers", s.listNameserverGroups)
	mux.HandleFunc("GET /api/bff/dns/nameservers/{id}", s.getNameserverGroup)
	mux.HandleFunc("POST /api/bff/dns/nameservers", s.createNameserverGroup)
	mux.HandleFunc("PUT /api/bff/dns/nameservers/{id}", s.updateNameserverGroup)
	mux.HandleFunc("DELETE /api/bff/dns/nameservers/{id}", s.deleteNameserverGroup)
	mux.HandleFunc("GET /api/bff/dns/settings", s.getDNSSettings)
	mux.HandleFunc("PUT /api/bff/dns/settings", s.updateDNSSettings)
	mux.HandleFunc("GET /api/bff/dns/zones", s.listZones)
	mux.HandleFunc("GET /api/bff/dns/zones/{id}", s.getZone)
	mux.HandleFunc("POST /api/bff/dns/zones", s.createZone)
	mux.HandleFunc("PUT /api/bff/dns/zones/{id}", s.updateZoneMeta)
	mux.HandleFunc("DELETE /api/bff/dns/zones/{id}", s.deleteZone)
	mux.HandleFunc("POST /api/bff/dns/zones/{id}/records", s.createRecord)
	mux.HandleFunc("PUT /api/bff/dns/zones/{id}/records/{recordID}", s.updateRecord)
	mux.HandleFunc("DELETE /api/bff/dns/zones/{id}/records/{recordID}", s.deleteRecord)

	mux.HandleFunc("GET /api/bff/team/users", s.listUsers)
	mux.HandleFunc("POST /api/bff/team/users", s.inviteUser)
	mux.HandleFunc("PUT /api/bff/team/users/{id}", s.updateUser)
	mux.HandleFunc("DELETE /api/bff/team/users/{id}", s.deleteUser)
	mux.HandleFunc("POST /api/bff/team/users/{id}/invite", s.resendInvite)
	mux.HandleFunc("POST /api/bff/team/users/{id}/approve", s.approveUser)
	mux.HandleFunc("DELETE /api/bff/team/users/{id}/reject", s.rejectUser)

	mux.HandleFunc("GET /api/bff/team/service-users", s.listServiceUsers)
	mux.HandleFunc("POST /api/bff/team/service-users", s.createServiceUser)
	mux.HandleFunc("PUT /api/bff/team/service-users/{id}", s.updateUser)
	mux.HandleFunc("DELETE /api/bff/team/service-users/{id}", s.deleteUser)
	mux.HandleFunc("GET /api/bff/team/service-users/{id}/tokens", s.listPATs)
	mux.HandleFunc("POST /api/bff/team/service-users/{id}/tokens", s.createPAT)
	mux.HandleFunc("DELETE /api/bff/team/service-users/{id}/tokens/{tokenID}", s.deletePAT)

	mux.HandleFunc("GET /api/bff/settings/identity-providers", s.listIdentityProviders)
	mux.HandleFunc("GET /api/bff/settings/identity-providers/{id}", s.getIdentityProvider)
	mux.HandleFunc("POST /api/bff/settings/identity-providers", s.createIdentityProvider)
	mux.HandleFunc("PUT /api/bff/settings/identity-providers/{id}", s.updateIdentityProvider)
	mux.HandleFunc("DELETE /api/bff/settings/identity-providers/{id}", s.deleteIdentityProvider)

	mux.HandleFunc("GET /api/bff/events/audit", s.listAuditEvents)

	mux.HandleFunc("GET /api/bff/posture-checks", s.listPostureChecks)
	mux.HandleFunc("GET /api/bff/posture-checks/{id}", s.getPostureCheck)
	mux.HandleFunc("POST /api/bff/posture-checks", s.createPostureCheck)
	mux.HandleFunc("PUT /api/bff/posture-checks/{id}", s.updatePostureCheck)
	mux.HandleFunc("DELETE /api/bff/posture-checks/{id}", s.deletePostureCheck)

	mux.HandleFunc("GET /api/bff/permissions", s.getCurrentUserPermissions)

	mux.HandleFunc("GET /api/bff/account", s.getCurrentAccount)
	mux.HandleFunc("DELETE /api/bff/account", s.deleteAccount)

	mux.HandleFunc("GET /api/bff/reverse-proxy/clusters", s.listProxyClusters)
	mux.HandleFunc("DELETE /api/bff/reverse-proxy/clusters/{address}", s.deleteProxyCluster)
	mux.HandleFunc("GET /api/bff/reverse-proxy/proxy-tokens", s.listProxyTokens)
	mux.HandleFunc("POST /api/bff/reverse-proxy/proxy-tokens", s.createProxyToken)
	mux.HandleFunc("DELETE /api/bff/reverse-proxy/proxy-tokens/{id}", s.deleteProxyToken)
	mux.HandleFunc("GET /api/bff/reverse-proxy/domains", s.listReverseProxyDomains)
	mux.HandleFunc("POST /api/bff/reverse-proxy/domains", s.createReverseProxyDomain)
	mux.HandleFunc("DELETE /api/bff/reverse-proxy/domains/{id}", s.deleteReverseProxyDomain)
	mux.HandleFunc("GET /api/bff/reverse-proxy/domains/{id}/validate", s.validateReverseProxyDomain)
	mux.HandleFunc("GET /api/bff/reverse-proxy/services", s.listServices)
	mux.HandleFunc("GET /api/bff/reverse-proxy/services/{id}", s.getService)
	mux.HandleFunc("POST /api/bff/reverse-proxy/services", s.createService)
	mux.HandleFunc("PUT /api/bff/reverse-proxy/services/{id}", s.updateService)
	mux.HandleFunc("DELETE /api/bff/reverse-proxy/services/{id}", s.deleteService)

	mux.HandleFunc("GET /api/bff/agent-network/providers", s.listAgentNetworkProviders)
	mux.HandleFunc("POST /api/bff/agent-network/providers", s.createAgentNetworkProvider)
	mux.HandleFunc("PUT /api/bff/agent-network/providers/{id}", s.updateAgentNetworkProvider)
	mux.HandleFunc("DELETE /api/bff/agent-network/providers/{id}", s.deleteAgentNetworkProvider)
	mux.HandleFunc("GET /api/bff/agent-network/policies", s.listAgentNetworkPolicies)
	mux.HandleFunc("POST /api/bff/agent-network/policies", s.createAgentNetworkPolicy)
	mux.HandleFunc("PUT /api/bff/agent-network/policies/{id}", s.updateAgentNetworkPolicy)
	mux.HandleFunc("DELETE /api/bff/agent-network/policies/{id}", s.deleteAgentNetworkPolicy)

	mux.HandleFunc("GET /api/bff/settings/jwt-group-sync", s.getJWTGroupSync)
	mux.HandleFunc("PUT /api/bff/settings/jwt-group-sync", s.updateJWTGroupSync)
}

// Middleware is the auth gate for /api/bff/.... It answers a missing or
// expired session with a plain JSON 401 body, deliberately not an HTML
// redirect — a redirect response to a fetch() call gets followed silently
// by the browser, and the caller ends up trying to JSON-parse the login
// page. It is the frontend's own apiFetch() wrapper (ui/src/api/client.ts)
// that turns a 401 from this layer into an actual navigation to /login;
// this middleware's only job is to resolve the session into the request
// context (auth.Authenticate) so every handler below can call s.token(w,r).
func (s *Server) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := s.auth.Authenticate(r)
		if !ok {
			s.writeError(w, http.StatusUnauthorized, "unauthenticated")
			return
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
