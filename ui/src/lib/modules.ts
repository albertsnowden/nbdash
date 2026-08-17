// Management API permission module names — verified against the netbird
// server source (management/server/permissions/modules/module.go), not
// guessed. Each dashboard feature is gated by whichever module its
// underlying handlers actually call `ValidateUserPermissions` with,
// confirmed by grepping every handler/manager in the netbird repo:
//
//   Peers                       -> "peers"
//   Groups                      -> "groups"
//   Setup Keys                  -> "setup_keys"
//   Access Control (Policies)   -> "policies"
//   Posture Checks              -> "policies"   (posture_checks.go shares it — no dedicated module)
//   Networks (+Resources+Routers) -> "networks"
//   DNS Nameservers tab         -> "nameservers"
//   DNS Zones + Settings tabs   -> "dns"
//   Team Users + Service Users  -> "users"
//   Team's PAT sub-resource     -> "pats"
//   Identity Providers          -> "identity_providers"
//   Account/Danger Zone/JWT sync -> "accounts"
//   Reverse Proxy (all 4 tabs)  -> "services"
//   Agent Network Providers     -> "agent_network.providers"
//   Agent Network Policies      -> "agent_network.policies"
//   Activity                    -> "events"
//
// Dotted submodules (agent_network.*) cascade from their parent
// ("agent_network") when a role grants the parent directly — the server's
// GetPermissionsByRole already resolves that cascade before this dashboard
// ever sees the response, so no cascade logic is needed here.
export const MODULE = {
  peers: "peers",
  groups: "groups",
  setupKeys: "setup_keys",
  policies: "policies",
  networks: "networks",
  dns: "dns",
  nameservers: "nameservers",
  users: "users",
  pats: "pats",
  identityProviders: "identity_providers",
  accounts: "accounts",
  services: "services",
  agentNetworkProviders: "agent_network.providers",
  agentNetworkPolicies: "agent_network.policies",
  events: "events",
} as const;
