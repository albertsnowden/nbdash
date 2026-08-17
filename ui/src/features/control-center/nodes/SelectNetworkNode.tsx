import { Handle, Position, type NodeProps } from "@xyflow/react";
import { Network as NetworkIcon } from "lucide-react";
import type { Network } from "@/api/networks";
import Combobox from "../Combobox";

export interface SelectNetworkNodeData {
  current: string;
  networks: Network[];
  onChange: (id: string) => void;
  [key: string]: unknown;
}

export default function SelectNetworkNode({ data }: NodeProps & { data: SelectNetworkNodeData }) {
  const network = data.networks.find((n) => n.id === data.current);

  return (
    <div className="w-[260px]">
      <Combobox
        value={data.current}
        onChange={data.onChange}
        placeholder="Search networks…"
        options={data.networks.map((n) => ({ value: n.id, label: n.name }))}
        trigger={
          network ? (
            <div className="flex min-w-0 items-center gap-3 px-3 py-2.5">
              <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-nb-gray-850 text-nb-gray-300">
                <NetworkIcon size={16} />
              </div>
              <div className="min-w-0">
                <div className="truncate text-sm text-nb-gray-100">{network.name}</div>
                <div className="truncate text-xs text-nb-gray-500">
                  {network.routing_peers_count} routing peer{network.routing_peers_count === 1 ? "" : "s"}
                </div>
              </div>
            </div>
          ) : (
            <span className="px-3 py-2.5 text-sm text-nb-gray-500">Select a network…</span>
          )
        }
      />
      <Handle type="source" position={Position.Right} id="sr" className="opacity-0" />
    </div>
  );
}
