import { Handle, Position, type NodeProps } from "@xyflow/react";
import { Link } from "react-router";
import type { AgentNetworkPolicy } from "@/api/agentNetwork";
import { cn } from "@/lib/utils";

export interface AgentPolicyNodeData {
  policy: AgentNetworkPolicy;
  [key: string]: unknown;
}

export default function AgentPolicyNode({ data }: NodeProps & { data: AgentPolicyNodeData }) {
  const { policy } = data;

  return (
    <Link
      to="/agent-network"
      className={cn(
        "flex items-center gap-2 overflow-hidden rounded-full border border-nb-gray-800 bg-nb-gray-940 py-2 pl-3 pr-4 transition-colors hover:bg-nb-gray-930",
        !policy.enabled && "opacity-60",
      )}
    >
      <span className={cn("h-2 w-2 shrink-0 rounded-full", policy.enabled ? "bg-green-400" : "bg-nb-gray-500")} />
      <span className="truncate text-[0.8rem] text-nb-gray-200">{policy.name}</span>
      <Handle type="target" position={Position.Left} id="tl" className="opacity-0" />
      <Handle type="source" position={Position.Right} id="sr" className="opacity-0" />
    </Link>
  );
}
