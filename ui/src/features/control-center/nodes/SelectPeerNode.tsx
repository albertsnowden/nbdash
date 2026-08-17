import { Handle, Position, type NodeProps } from "@xyflow/react";
import type { Peer } from "@/api/peers";
import Combobox from "../Combobox";
import DeviceCard from "../DeviceCard";

export interface SelectPeerNodeData {
  current: string;
  peers: Peer[];
  onChange: (id: string) => void;
  [key: string]: unknown;
}

export default function SelectPeerNode({ data }: NodeProps & { data: SelectPeerNodeData }) {
  const peer = data.peers.find((p) => p.id === data.current);

  return (
    <div className="w-[260px]">
      <Combobox
        value={data.current}
        onChange={data.onChange}
        placeholder="Search peers…"
        options={data.peers.map((p) => ({ value: p.id, label: p.name }))}
        trigger={
          peer ? (
            <DeviceCard peer={peer} />
          ) : (
            <span className="px-3 py-2.5 text-sm text-nb-gray-500">Select a peer…</span>
          )
        }
      />
      <Handle type="source" position={Position.Right} id="sr" className="opacity-0" />
    </div>
  );
}
