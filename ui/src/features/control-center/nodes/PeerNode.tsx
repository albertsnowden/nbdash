import { Handle, Position, type NodeProps } from "@xyflow/react";
import type { Peer } from "@/api/peers";
import { cn } from "@/lib/utils";
import DeviceCard from "../DeviceCard";

export interface PeerNodeData {
  peer: Peer;
  enabled: boolean;
  [key: string]: unknown;
}

export default function PeerNode({ data }: NodeProps & { data: PeerNodeData }) {
  return (
    <div className={cn("rounded-lg border border-nb-gray-800 bg-nb-gray-940", !data.enabled && "opacity-60")}>
      <DeviceCard peer={data.peer} />
      <Handle type="target" position={Position.Left} id="tl" className="opacity-0" />
      <Handle type="source" position={Position.Right} id="sr" className="opacity-0" />
    </div>
  );
}
