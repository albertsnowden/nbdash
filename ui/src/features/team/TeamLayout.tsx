import { NavLink, Outlet } from "react-router";
import { cn } from "@/lib/utils";

const TABS = [
  { to: "/team/users", label: "Users" },
  { to: "/team/service-users", label: "Service Users" },
];

export default function TeamLayout() {
  return (
    <section className="flex flex-col gap-4">
      <h1 className="text-xl font-semibold text-nb-gray-50">Team</h1>

      <div className="inline-flex w-fit items-center gap-1 rounded-md border border-nb-gray-900 bg-nb-gray-940 p-1">
        {TABS.map((tab) => (
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
