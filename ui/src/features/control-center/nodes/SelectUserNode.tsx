import { Handle, Position, type NodeProps } from "@xyflow/react";
import type { User } from "@/api/team";
import Combobox from "../Combobox";

export interface SelectUserNodeData {
  current: string;
  users: User[];
  onChange: (id: string) => void;
  [key: string]: unknown;
}

// Deterministic hue from a user id/email so the same user always gets the
// same avatar color across renders, without needing to store one.
function avatarHue(seed: string): number {
  let hash = 0;
  for (let i = 0; i < seed.length; i++) hash = (hash * 31 + seed.charCodeAt(i)) >>> 0;
  return hash % 360;
}

export default function SelectUserNode({ data }: NodeProps & { data: SelectUserNodeData }) {
  const user = data.users.find((u) => u.id === data.current);

  return (
    <div className="w-[260px]">
      <Combobox
        value={data.current}
        onChange={data.onChange}
        placeholder="Search users…"
        options={data.users.map((u) => ({ value: u.id, label: u.name || u.email }))}
        trigger={
          user ? (
            <div className="flex min-w-0 items-center gap-2.5 px-3 py-2">
              <div
                className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-sm font-medium uppercase text-white"
                style={{ backgroundColor: `hsl(${avatarHue(user.id || user.email)}, 45%, 35%)` }}
              >
                {(user.name || user.email || "?").charAt(0)}
              </div>
              <div className="min-w-0 leading-tight">
                <div className="truncate text-sm text-nb-gray-100">{user.name || user.email}</div>
                {user.email && <div className="truncate text-xs text-nb-gray-500">{user.email}</div>}
              </div>
            </div>
          ) : (
            <span className="px-3 py-2.5 text-sm text-nb-gray-500">Select a user…</span>
          )
        }
      />
      <Handle type="source" position={Position.Right} id="sr" className="opacity-0" />
    </div>
  );
}
