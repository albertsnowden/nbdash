import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { lazy } from "react";
import { BrowserRouter, Navigate, Route, Routes } from "react-router";
import { Toaster } from "sonner";
import RequireModule from "./components/RequireModule";
import { TooltipProvider } from "./components/ui/Tooltip";
import AccountRestrictedPage from "./features/account-restricted/AccountRestrictedPage";
import DashboardLayout from "./layouts/DashboardLayout";
import { MODULE } from "./lib/modules";

// Every feature page is its own chunk, loaded on first visit instead of
// bundled into the initial download — with ~15 nav destinations (several
// pulling in their own heavy deps, e.g. Control Center's @xyflow/react),
// one static bundle was tripping Vite's 500kB chunk-size warning. The
// Suspense boundary lives once, around DashboardLayout's <Outlet/> below,
// so it catches every route's lazy import no matter how deeply nested
// (DNS/Team/Settings each nest a second Outlet of their own for their
// tabs) without needing a boundary per route.
const ActivityList = lazy(() => import("./features/activity/ActivityList"));
const AgentNetworkPage = lazy(() => import("./features/agent-network/AgentNetworkPage"));
const ControlCenterPage = lazy(() => import("./features/control-center/ControlCenterPage"));
const DNSLayout = lazy(() => import("./features/dns/DNSLayout"));
const DNSSettingsPage = lazy(() => import("./features/dns/DNSSettingsPage"));
const NameserversList = lazy(() => import("./features/dns/NameserversList"));
const ZoneDetail = lazy(() => import("./features/dns/ZoneDetail"));
const ZonesList = lazy(() => import("./features/dns/ZonesList"));
const GroupDetail = lazy(() => import("./features/groups/GroupDetail"));
const GroupsList = lazy(() => import("./features/groups/GroupsList"));
const NetworkDetail = lazy(() => import("./features/networks/NetworkDetail"));
const NetworksList = lazy(() => import("./features/networks/NetworksList"));
const PeerDetail = lazy(() => import("./features/peers/PeerDetail"));
const PeersList = lazy(() => import("./features/peers/PeersList"));
const PoliciesList = lazy(() => import("./features/policies/PoliciesList"));
const PolicyDetail = lazy(() => import("./features/policies/PolicyDetail"));
const PostureChecksList = lazy(() => import("./features/posture-checks/PostureChecksList"));
const ReverseProxyPage = lazy(() => import("./features/reverse-proxy/ReverseProxyPage"));
const DangerZonePage = lazy(() => import("./features/settings/DangerZonePage"));
const IdentityProviderDetail = lazy(() => import("./features/settings/IdentityProviderDetail"));
const IdentityProvidersList = lazy(() => import("./features/settings/IdentityProvidersList"));
const JWTGroupSyncPage = lazy(() => import("./features/settings/JWTGroupSyncPage"));
const PermissionsPage = lazy(() => import("./features/settings/PermissionsPage"));
const SettingsLayout = lazy(() => import("./features/settings/SettingsLayout"));
const SetupKeysList = lazy(() => import("./features/setup-keys/SetupKeysList"));
const ServiceUserDetail = lazy(() => import("./features/team/ServiceUserDetail"));
const ServiceUsersList = lazy(() => import("./features/team/ServiceUsersList"));
const TeamLayout = lazy(() => import("./features/team/TeamLayout"));
const UsersList = lazy(() => import("./features/team/UsersList"));

const queryClient = new QueryClient();

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider delayDuration={200}>
        <Toaster
          theme="dark"
          toastOptions={{
            classNames: {
              toast: "!bg-nb-gray-900 !border-nb-gray-800 !text-nb-gray-50",
            },
          }}
        />
        <BrowserRouter>
          <Routes>
            <Route path="account-restricted" element={<AccountRestrictedPage />} />
            <Route element={<DashboardLayout />}>
              <Route
                index
                element={
                  <RequireModule modules={[MODULE.peers]}>
                    <PeersList />
                  </RequireModule>
                }
              />
              <Route
                path="peers"
                element={
                  <RequireModule modules={[MODULE.peers]}>
                    <PeersList />
                  </RequireModule>
                }
              />
              <Route
                path="peers/:id"
                element={
                  <RequireModule modules={[MODULE.peers]}>
                    <PeerDetail />
                  </RequireModule>
                }
              />
              <Route
                path="groups"
                element={
                  <RequireModule modules={[MODULE.groups]}>
                    <GroupsList />
                  </RequireModule>
                }
              />
              <Route
                path="groups/:id"
                element={
                  <RequireModule modules={[MODULE.groups]}>
                    <GroupDetail />
                  </RequireModule>
                }
              />
              <Route
                path="setup-keys"
                element={
                  <RequireModule modules={[MODULE.setupKeys]}>
                    <SetupKeysList />
                  </RequireModule>
                }
              />
              <Route
                path="policies"
                element={
                  <RequireModule modules={[MODULE.policies]}>
                    <PoliciesList />
                  </RequireModule>
                }
              />
              <Route
                path="policies/:id"
                element={
                  <RequireModule modules={[MODULE.policies]}>
                    <PolicyDetail />
                  </RequireModule>
                }
              />
              <Route
                path="networks"
                element={
                  <RequireModule modules={[MODULE.networks]}>
                    <NetworksList />
                  </RequireModule>
                }
              />
              <Route
                path="networks/:id"
                element={
                  <RequireModule modules={[MODULE.networks]}>
                    <NetworkDetail />
                  </RequireModule>
                }
              />
              <Route
                path="dns"
                element={
                  <RequireModule modules={[MODULE.dns, MODULE.nameservers]}>
                    <DNSLayout />
                  </RequireModule>
                }
              >
                <Route index element={<Navigate to="nameservers" replace />} />
                <Route
                  path="nameservers"
                  element={
                    <RequireModule modules={[MODULE.nameservers]}>
                      <NameserversList />
                    </RequireModule>
                  }
                />
                <Route
                  path="zones"
                  element={
                    <RequireModule modules={[MODULE.dns]}>
                      <ZonesList />
                    </RequireModule>
                  }
                />
                <Route
                  path="zones/:id"
                  element={
                    <RequireModule modules={[MODULE.dns]}>
                      <ZoneDetail />
                    </RequireModule>
                  }
                />
                <Route
                  path="settings"
                  element={
                    <RequireModule modules={[MODULE.dns]}>
                      <DNSSettingsPage />
                    </RequireModule>
                  }
                />
              </Route>
              <Route
                path="team"
                element={
                  <RequireModule modules={[MODULE.users]}>
                    <TeamLayout />
                  </RequireModule>
                }
              >
                <Route index element={<Navigate to="users" replace />} />
                <Route path="users" element={<UsersList />} />
                <Route path="service-users" element={<ServiceUsersList />} />
                <Route path="service-users/:id" element={<ServiceUserDetail />} />
              </Route>
              <Route path="settings" element={<SettingsLayout />}>
                <Route index element={<Navigate to="identity-providers" replace />} />
                <Route
                  path="identity-providers"
                  element={
                    <RequireModule modules={[MODULE.identityProviders]}>
                      <IdentityProvidersList />
                    </RequireModule>
                  }
                />
                <Route
                  path="identity-providers/:id"
                  element={
                    <RequireModule modules={[MODULE.identityProviders]}>
                      <IdentityProviderDetail />
                    </RequireModule>
                  }
                />
                <Route
                  path="jwt-group-sync"
                  element={
                    <RequireModule modules={[MODULE.accounts]}>
                      <JWTGroupSyncPage />
                    </RequireModule>
                  }
                />
                <Route path="permissions" element={<PermissionsPage />} />
                <Route
                  path="danger-zone"
                  element={
                    <RequireModule modules={[MODULE.accounts]}>
                      <DangerZonePage />
                    </RequireModule>
                  }
                />
              </Route>
              <Route
                path="activity"
                element={
                  <RequireModule modules={[MODULE.events]}>
                    <ActivityList />
                  </RequireModule>
                }
              />
              <Route
                path="posture-checks"
                element={
                  <RequireModule modules={[MODULE.policies]}>
                    <PostureChecksList />
                  </RequireModule>
                }
              />
              <Route
                path="reverse-proxy"
                element={
                  <RequireModule modules={[MODULE.services]}>
                    <ReverseProxyPage />
                  </RequireModule>
                }
              />
              <Route
                path="agent-network"
                element={
                  <RequireModule modules={[MODULE.agentNetworkProviders, MODULE.agentNetworkPolicies]}>
                    <AgentNetworkPage />
                  </RequireModule>
                }
              />
              <Route
                path="control-center"
                element={
                  <RequireModule modules={[MODULE.groups]}>
                    <ControlCenterPage />
                  </RequireModule>
                }
              />
            </Route>
          </Routes>
        </BrowserRouter>
      </TooltipProvider>
    </QueryClientProvider>
  );
}
