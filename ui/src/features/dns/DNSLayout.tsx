import { NavLink, Outlet } from "react-router";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";
import { cn } from "@/lib/utils";

const TABS = [
  { to: "/dns/nameservers", label: "Nameservers", modules: [MODULE.nameservers] },
  { to: "/dns/zones", label: "Zones", modules: [MODULE.dns] },
  { to: "/dns/settings", label: "Settings", modules: [MODULE.dns] },
];

export default function DNSLayout() {
  const { canReadAny } = usePermissionsContext();
  const visibleTabs = TABS.filter((tab) => canReadAny(tab.modules));

  return (
    <section className="flex flex-col gap-4">
      <h1 className="text-xl font-semibold text-nb-gray-50">DNS</h1>

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
