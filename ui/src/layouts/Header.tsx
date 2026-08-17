import { LogOut, Menu, PanelLeftClose, PanelLeftOpen, User } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/DropdownMenu";
import Button from "@/components/ui/Button";

interface HeaderProps {
  collapsed: boolean;
  onToggleCollapsed: () => void;
  onOpenMobileNav: () => void;
}

export default function Header({ collapsed, onToggleCollapsed, onOpenMobileNav }: HeaderProps) {
  return (
    <header className="flex h-16 shrink-0 items-center gap-2 border-b border-nb-gray-900 bg-nb-gray-950 px-4">
      <Button
        variant="ghost"
        size="icon"
        className="md:hidden"
        onClick={onOpenMobileNav}
        aria-label="Open navigation"
      >
        <Menu size={18} />
      </Button>

      <Button
        variant="ghost"
        size="icon"
        className="hidden md:inline-flex"
        onClick={onToggleCollapsed}
        aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
      >
        {collapsed ? <PanelLeftOpen size={18} /> : <PanelLeftClose size={18} />}
      </Button>

      <div className="flex-1" />

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            className="flex h-8 w-8 items-center justify-center rounded-full bg-netbird-500 text-white transition-opacity hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-netbird-500 focus-visible:ring-offset-2 focus-visible:ring-offset-nb-gray-950"
            aria-label="Account menu"
          >
            <User size={16} />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <form method="post" action="/logout">
            <DropdownMenuItem asChild>
              <button type="submit" className="w-full">
                <LogOut size={14} />
                Sign out
              </button>
            </DropdownMenuItem>
          </form>
        </DropdownMenuContent>
      </DropdownMenu>
    </header>
  );
}
