import { Handle, Position, type NodeProps } from "@xyflow/react";
import { LayoutGrid } from "lucide-react";
import type { Group } from "@/api/groups";
import Combobox from "../Combobox";

export interface SelectGroupNodeData {
  current: string;
  groups: Group[];
  onChange: (id: string) => void;
  [key: string]: unknown;
}

export default function SelectGroupNode({ data }: NodeProps & { data: SelectGroupNodeData }) {
  const group = data.groups.find((g) => g.id === data.current);

  return (
    <div className="w-[260px]">
      <Combobox
        value={data.current}
        onChange={data.onChange}
        placeholder="Search groups…"
        options={data.groups.map((g) => ({ value: g.id, label: g.name }))}
        trigger={
          group ? (
            <div className="flex min-w-0 items-center gap-3 px-3 py-2.5">
              <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-nb-gray-850 text-nb-gray-300">
                <LayoutGrid size={16} />
              </div>
              <div className="min-w-0">
                <div className="truncate text-sm text-nb-gray-100">{group.name}</div>
                <div className="truncate text-xs text-nb-gray-500">{group.peers_count} peers</div>
              </div>
            </div>
          ) : (
            <span className="px-3 py-2.5 text-sm text-nb-gray-500">Select a group…</span>
          )
        }
      />
      <Handle type="source" position={Position.Right} id="sr" className="opacity-0" />
    </div>
  );
}
