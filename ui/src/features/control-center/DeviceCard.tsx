import type { Peer } from "@/api/peers";
import { osGlyph } from "@/lib/format";
import { cn } from "@/lib/utils";

// Shared leaf renderer for a peer inside the Control Center graph — the
// node types (PeerNode) and the select-peer combobox trigger both use it,
// same as DataTable rows use a common Cell pattern elsewhere in this app.
export default function DeviceCard({ peer, className }: { peer: Peer; className?: string }) {
  return (
    <div className={cn("flex w-[220px] shrink-0 items-center gap-2.5 px-3 py-2 text-left", className)}>
      <div className="relative flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-nb-gray-900 text-nb-gray-400">
        <span className="text-base leading-none">{osGlyph(peer.os)}</span>
        <span
          className={cn(
            "absolute -bottom-0.5 -right-0.5 h-2.5 w-2.5 rounded-full border-2 border-nb-gray-940",
            peer.connected ? "bg-green-500" : "bg-nb-gray-600",
          )}
          title={peer.connected ? "Connected" : "Disconnected"}
        />
      </div>
      <div className="min-w-0 leading-tight">
        <div className="truncate text-sm text-nb-gray-100">{peer.name}</div>
        <div className="truncate text-xs text-nb-gray-500">{peer.ip}</div>
      </div>
    </div>
  );
}
