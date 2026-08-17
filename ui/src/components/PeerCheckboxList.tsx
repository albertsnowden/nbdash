import type { Peer } from "@/api/peers";
import Checkbox from "@/components/ui/Checkbox";

interface Props {
  peers: Peer[];
  selected: string[];
  onChange: (ids: string[]) => void;
}

// Reused everywhere a form needs to pick a peer subset (Groups' membership
// today; Setup Keys/Policies/Networks/Users' auto-groups-style pickers).
export default function PeerCheckboxList({ peers, selected, onChange }: Props) {
  const toggle = (id: string) => {
    onChange(selected.includes(id) ? selected.filter((x) => x !== id) : [...selected, id]);
  };

  if (peers.length === 0) {
    return <p className="text-sm text-nb-gray-500">No peers yet.</p>;
  }

  return (
    <div className="flex max-h-64 flex-col gap-0.5 overflow-y-auto rounded-md border border-nb-gray-900 p-2">
      {peers.map((peer) => (
        <label
          key={peer.id}
          className="flex cursor-pointer items-center gap-2 rounded-sm px-2 py-1.5 text-sm text-nb-gray-100 hover:bg-nb-gray-900"
        >
          <Checkbox checked={selected.includes(peer.id)} onCheckedChange={() => toggle(peer.id)} />
          <span>{peer.name}</span>
        </label>
      ))}
    </div>
  );
}
