import { Handle, Position, type NodeProps } from "@xyflow/react";
import { Link } from "react-router";
import type { Policy, PolicyRule } from "@/api/policies";
import { cn } from "@/lib/utils";

export interface PolicyNodeData {
  policy: Policy;
  enabled: boolean;
  [key: string]: unknown;
}

function protocolLabel(rule?: PolicyRule): string {
  if (!rule) return "";
  if (rule.protocol === "all") return "All";
  if (rule.protocol === "icmp") return "ICMP";
  const proto = rule.protocol.toUpperCase();
  const ports = rule.ports ?? [];
  if (ports.length === 0) return proto;
  const shown = ports.slice(0, 4);
  const suffix = ports.length > shown.length ? ", …" : "";
  return `${proto}:${shown.join(",")}${suffix}`;
}

export default function PolicyNode({ data }: NodeProps & { data: PolicyNodeData }) {
  const { policy, enabled } = data;
  const label = protocolLabel(policy.rules[0]);

  return (
    <Link
      to={`/policies/${policy.id}`}
      className={cn(
        "flex items-center overflow-hidden rounded-full border border-nb-gray-800 bg-nb-gray-940 transition-colors hover:bg-nb-gray-930",
        !enabled && "opacity-60",
      )}
    >
      <span className={cn("ml-3 mr-2 h-2 w-2 shrink-0 rounded-full", enabled ? "bg-green-400" : "bg-nb-gray-500")} />
      <span className="truncate py-2 pr-3 text-[0.8rem] text-nb-gray-200">{policy.name}</span>
      {label && (
        <span className="border-l border-nb-gray-800 px-2.5 py-2 font-mono text-[0.65rem] text-nb-gray-400">
          {label}
        </span>
      )}
      <Handle type="target" position={Position.Left} id="tl" className="opacity-0" />
      <Handle type="source" position={Position.Right} id="sr" className="opacity-0" />
    </Link>
  );
}
