import { Handle, Position, type NodeProps } from "@xyflow/react";
import { Globe2 } from "lucide-react";
import type { NetworkResource } from "@/api/networks";
import { cn } from "@/lib/utils";

export interface ResourceNodeData {
  resource: NetworkResource;
  [key: string]: unknown;
}

export default function ResourceNode({ data }: NodeProps & { data: ResourceNodeData }) {
  const { resource } = data;

  return (
    <div
      className={cn(
        "flex w-[240px] items-center gap-2.5 rounded-lg border border-nb-gray-800 bg-nb-gray-940 px-3 py-2",
        !resource.enabled && "opacity-60",
      )}
    >
      <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-nb-gray-850 text-nb-gray-300">
        <Globe2 size={16} />
      </div>
      <div className="min-w-0 leading-tight">
        <div className="truncate text-sm text-nb-gray-100">{resource.name}</div>
        <div className="truncate text-xs text-nb-gray-500">{resource.address}</div>
      </div>
      <Handle type="target" position={Position.Left} id="tl" className="opacity-0" />
      <Handle type="source" position={Position.Right} id="sr" className="opacity-0" />
    </div>
  );
}
