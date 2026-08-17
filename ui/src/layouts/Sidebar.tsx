import { NavLink } from "react-router";
import Badge from "@/components/ui/Badge";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { cn } from "@/lib/utils";
import { BOTTOM_NAV_ITEMS, NAV_ITEMS, type NavItem } from "./navItems";

interface SidebarProps {
  collapsed: boolean;
  className?: string;
  onNavigate?: () => void;
}

function NavRow({ item, collapsed, onNavigate }: { item: NavItem; collapsed: boolean; onNavigate?: () => void }) {
  const Icon = item.icon;
  return (
    <NavLink
      to={item.to}
      onClick={onNavigate}
      title={collapsed ? item.label : undefined}
      className={({ isActive }) =>
        cn(
          "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-nb-gray-300 transition-colors",
          "hover:bg-nb-gray-900 hover:text-nb-gray-50",
          isActive && "bg-nb-gray-900 text-nb-gray-50",
          collapsed && "justify-center px-0",
        )
      }
    >
      <Icon size={18} className="shrink-0" />
      {!collapsed && (
        <span className="flex flex-1 items-center justify-between gap-2 truncate">
          {item.label}
          {item.beta && (
            <Badge variant="beta" className="px-1.5 py-0 text-[10px] leading-4">
              Beta
            </Badge>
          )}
        </span>
      )}
    </NavLink>
  );
}

export default function Sidebar({ collapsed, className, onNavigate }: SidebarProps) {
  const { canReadAny } = usePermissionsContext();
  const visible = (item: NavItem) => !item.modules || canReadAny(item.modules);

  return (
    <aside
      className={cn(
        "flex h-full flex-col border-r border-nb-gray-900 bg-nb-gray-950 transition-[width] duration-200",
        collapsed ? "w-[68px]" : "w-60",
        className,
      )}
    >
      <div className={cn("flex h-16 shrink-0 items-center px-5", collapsed && "justify-center px-0")}>
        <img
          className={cn("h-6 w-auto", collapsed && "hidden")}
          src="/static/netbird-full.svg"
          alt="NetBird"
        />
        <img
          className={cn("hidden h-6 w-6", collapsed && "block")}
          src="/static/favicon.ico"
          alt="NetBird"
        />
      </div>

      <nav className="flex-1 space-y-0.5 overflow-y-auto px-3 py-2">
        {NAV_ITEMS.filter(visible).map((item) => (
          <NavRow key={item.key} item={item} collapsed={collapsed} onNavigate={onNavigate} />
        ))}
      </nav>

      <nav className="space-y-0.5 border-t border-nb-gray-900 px-3 py-2">
        {BOTTOM_NAV_ITEMS.filter(visible).map((item) => (
          <NavRow key={item.key} item={item} collapsed={collapsed} onNavigate={onNavigate} />
        ))}
      </nav>
    </aside>
  );
}
