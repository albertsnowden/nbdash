import {
  Activity,
  Bot,
  Globe,
  KeyRound,
  LayoutDashboard,
  LayoutGrid,
  Network,
  ShieldAlert,
  Settings as SettingsIcon,
  ShieldCheck,
  Users,
  Monitor,
  Waypoints,
} from "lucide-react";
import type { ComponentType } from "react";
import { MODULE } from "@/lib/modules";

export interface NavItem {
  key: string;
  label: string;
  to: string;
  icon: ComponentType<{ size?: number; className?: string }>;
  beta?: boolean;
  // Modules that grant this item read access — ANY one of them is enough.
  // Undefined means always shown (no server-side module gates it, or it's
  // self-describing regardless of role, like Settings' own Permissions
  // tab).
  modules?: string[];
}

// Order and grouping loosely follow the reference Next.js dashboard's own
// sidebar. Every item here is a real, working route — no dead links. See
// ui/src/lib/modules.ts for how each `modules` entry was verified against
// the netbird server source, not guessed.
export const NAV_ITEMS: NavItem[] = [
  { key: "control-center", label: "Control Center", to: "/control-center", icon: LayoutDashboard, modules: [MODULE.groups] },
  { key: "peers", label: "Peers", to: "/peers", icon: Monitor, modules: [MODULE.peers] },
  { key: "policies", label: "Access Control", to: "/policies", icon: ShieldCheck, modules: [MODULE.policies] },
  { key: "posture-checks", label: "Posture Checks", to: "/posture-checks", icon: ShieldAlert, modules: [MODULE.policies] },
  { key: "groups", label: "Groups", to: "/groups", icon: LayoutGrid, modules: [MODULE.groups] },
  { key: "setup-keys", label: "Setup Keys", to: "/setup-keys", icon: KeyRound, modules: [MODULE.setupKeys] },
  { key: "networks", label: "Networks", to: "/networks", icon: Network, modules: [MODULE.networks] },
  {
    key: "reverse-proxy",
    label: "Reverse Proxy",
    to: "/reverse-proxy",
    icon: Waypoints,
    beta: true,
    modules: [MODULE.services],
  },
  {
    key: "agent-network",
    label: "Agent Network",
    to: "/agent-network",
    icon: Bot,
    beta: true,
    modules: [MODULE.agentNetworkProviders, MODULE.agentNetworkPolicies],
  },
  { key: "dns", label: "DNS", to: "/dns", icon: Globe, modules: [MODULE.dns, MODULE.nameservers] },
  { key: "team", label: "Team", to: "/team", icon: Users, modules: [MODULE.users] },
  { key: "activity", label: "Activity", to: "/activity", icon: Activity, modules: [MODULE.events] },
];

export const BOTTOM_NAV_ITEMS: NavItem[] = [
  // Always shown: Settings' own Permissions tab is readable by any
  // authenticated user regardless of role (it's a read-only view of your
  // own permissions), so the section itself is never worth hiding — the
  // sub-tabs inside it (SettingsLayout.tsx) hide individually instead.
  { key: "settings", label: "Settings", to: "/settings", icon: SettingsIcon },
];
