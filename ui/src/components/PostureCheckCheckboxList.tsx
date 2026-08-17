import type { PostureCheck } from "@/api/postureChecks";
import Checkbox from "@/components/ui/Checkbox";

interface Props {
  checks: PostureCheck[];
  selected: string[];
  onChange: (ids: string[]) => void;
}

// The Posture Checks counterpart to GroupCheckboxList — attaches posture
// checks to a policy's source groups.
export default function PostureCheckCheckboxList({ checks, selected, onChange }: Props) {
  const toggle = (id: string) => {
    onChange(selected.includes(id) ? selected.filter((x) => x !== id) : [...selected, id]);
  };

  if (checks.length === 0) {
    return (
      <p className="text-sm text-nb-gray-500">
        No posture checks yet — create one on the Posture Checks page first.
      </p>
    );
  }

  return (
    <div className="flex max-h-64 flex-col gap-0.5 overflow-y-auto rounded-md border border-nb-gray-900 p-2">
      {checks.map((check) => (
        <label
          key={check.id}
          className="flex cursor-pointer items-center gap-2 rounded-sm px-2 py-1.5 text-sm text-nb-gray-100 hover:bg-nb-gray-900"
        >
          <Checkbox checked={selected.includes(check.id)} onCheckedChange={() => toggle(check.id)} />
          <span>{check.name}</span>
        </label>
      ))}
    </div>
  );
}
