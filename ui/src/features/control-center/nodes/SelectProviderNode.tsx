import { Handle, Position, type NodeProps } from "@xyflow/react";
import { Bot } from "lucide-react";
import type { AgentNetworkProvider } from "@/api/agentNetwork";
import { PROVIDER_TYPE_LABELS } from "@/api/agentNetwork";
import { cn } from "@/lib/utils";
import Combobox from "../Combobox";

export interface SelectProviderNodeData {
  current: string;
  providers: AgentNetworkProvider[];
  onChange: (id: string) => void;
  [key: string]: unknown;
}

export default function SelectProviderNode({ data }: NodeProps & { data: SelectProviderNodeData }) {
  const provider = data.providers.find((p) => p.id === data.current);

  return (
    <div className="w-[260px]">
      <Combobox
        value={data.current}
        onChange={data.onChange}
        placeholder="Search providers…"
        options={data.providers.map((p) => ({ value: p.id, label: p.name }))}
        trigger={
          provider ? (
            <div className="flex min-w-0 items-center gap-3 px-3 py-2.5">
              <div className="relative flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-nb-gray-850 text-nb-gray-300">
                <Bot size={16} />
                <span
                  className={cn(
                    "absolute -bottom-0.5 -right-0.5 h-2.5 w-2.5 rounded-full border-2 border-nb-gray-940",
                    provider.enabled ? "bg-green-500" : "bg-nb-gray-600",
                  )}
                />
              </div>
              <div className="min-w-0">
                <div className="truncate text-sm text-nb-gray-100">{provider.name}</div>
                <div className="truncate text-xs text-nb-gray-500">
                  {PROVIDER_TYPE_LABELS[provider.provider_id] ?? provider.provider_id}
                </div>
              </div>
            </div>
          ) : (
            <span className="px-3 py-2.5 text-sm text-nb-gray-500">Select a provider…</span>
          )
        }
      />
      <Handle type="source" position={Position.Right} id="sr" className="opacity-0" />
    </div>
  );
}
