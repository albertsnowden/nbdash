import { Handle, Position, type NodeProps } from "@xyflow/react";
import { LayoutGrid } from "lucide-react";
import type { Group } from "@/api/groups";
import { cn } from "@/lib/utils";

export interface GroupNodeData {
  group: Group;
  enabled: boolean;
  expanded?: boolean;
  onClick?: () => void;
  [key: string]: unknown;
}

export default function GroupNode({ data }: NodeProps & { data: GroupNodeData }) {
  const { group, enabled, expanded, onClick } = data;
  const peerCount = group.peers_count ?? 0;
  const resourceCount = group.resources_count ?? 0;
  const countLabel =
    resourceCount === 0
      ? `${peerCount} peer${peerCount === 1 ? "" : "s"}`
      : peerCount === 0
        ? `${resourceCount} resource${resourceCount === 1 ? "" : "s"}`
        : `${peerCount} peers, ${resourceCount} resources`;

  return (
    <div
      onClick={onClick}
      className={cn(
        "flex w-[260px] cursor-pointer items-center gap-3 rounded-lg border bg-nb-gray-940 px-3 py-2.5 transition-colors hover:bg-nb-gray-930",
        expanded ? "border-netbird-500" : "border-nb-gray-800",
        !enabled && "opacity-60",
      )}
    >
      <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-nb-gray-850 text-nb-gray-300">
        <LayoutGrid size={16} />
      </div>
      <div className="min-w-0">
        <div className="truncate text-sm text-nb-gray-100">{group.name}</div>
        <div className="truncate text-xs text-nb-gray-500">{countLabel}</div>
      </div>
      <Handle type="target" position={Position.Left} id="tl" className="opacity-0" />
      <Handle type="source" position={Position.Right} id="sr" className="opacity-0" />
    </div>
  );
}
