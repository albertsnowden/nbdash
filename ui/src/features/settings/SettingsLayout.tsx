import { NavLink, Outlet } from "react-router";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";
import { cn } from "@/lib/utils";

// Permissions itself has no `modules` entry — it reads the caller's own
// role/permissions, which every authenticated user can see regardless of
// role, so that tab is never hidden.
const TABS = [
  { to: "/settings/identity-providers", label: "Identity Providers", modules: [MODULE.identityProviders] },
  { to: "/settings/jwt-group-sync", label: "JWT Group Sync", modules: [MODULE.accounts] },
  { to: "/settings/permissions", label: "Permissions", modules: undefined },
  { to: "/settings/danger-zone", label: "Danger Zone", modules: [MODULE.accounts] },
];

export default function SettingsLayout() {
  const { canReadAny } = usePermissionsContext();
  const visibleTabs = TABS.filter((tab) => !tab.modules || canReadAny(tab.modules));

  return (
    <section className="flex flex-col gap-4">
      <h1 className="text-xl font-semibold text-nb-gray-50">Settings</h1>

      <div className="inline-flex w-fit items-center gap-1 rounded-md border border-nb-gray-900 bg-nb-gray-940 p-1">
        {visibleTabs.map((tab) => (
          <NavLink
            key={tab.to}
            to={tab.to}
            className={({ isActive }) =>
              cn(
                "rounded-sm px-3 py-1.5 text-sm font-medium text-nb-gray-400 transition-colors hover:text-nb-gray-100",
                isActive && "bg-nb-gray-850 text-nb-gray-50",
              )
            }
          >
            {tab.label}
          </NavLink>
        ))}
      </div>

      <Outlet />
    </section>
  );
}
