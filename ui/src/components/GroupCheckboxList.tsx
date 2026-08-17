import type { Group } from "@/api/groups";
import Checkbox from "@/components/ui/Checkbox";

interface Props {
  groups: Group[];
  selected: string[];
  onChange: (ids: string[]) => void;
}

// The Groups counterpart to PeerCheckboxList — reused by every
// auto-groups-style picker (Setup Keys, Policies, Networks).
export default function GroupCheckboxList({ groups, selected, onChange }: Props) {
  const toggle = (id: string) => {
    onChange(selected.includes(id) ? selected.filter((x) => x !== id) : [...selected, id]);
  };

  if (groups.length === 0) {
    return <p className="text-sm text-nb-gray-500">No groups yet.</p>;
  }

  return (
    <div className="flex max-h-64 flex-col gap-0.5 overflow-y-auto rounded-md border border-nb-gray-900 p-2">
      {groups.map((group) => (
        <label
          key={group.id}
          className="flex cursor-pointer items-center gap-2 rounded-sm px-2 py-1.5 text-sm text-nb-gray-100 hover:bg-nb-gray-900"
        >
          <Checkbox checked={selected.includes(group.id)} onCheckedChange={() => toggle(group.id)} />
          <span>{group.name}</span>
        </label>
      ))}
    </div>
  );
}
