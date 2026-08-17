import type { ColumnDef } from "@tanstack/react-table";
import { Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";
import { type Group, PROTECTED_GROUP_NAME, useCreateGroup, useDeleteGroup, useGroups } from "@/api/groups";
import { usePeers } from "@/api/peers";
import Button from "@/components/ui/Button";
import DataTable from "@/components/table/DataTable";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/Dialog";
import Input from "@/components/ui/Input";
import Label from "@/components/ui/Label";
import PeerCheckboxList from "@/components/PeerCheckboxList";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";

export default function GroupsList() {
  const { data, isLoading, error } = useGroups();
  const deleteGroup = useDeleteGroup();
  const [showCreate, setShowCreate] = useState(false);
  const { canCreate, canDelete } = usePermissionsContext();

  const columns: ColumnDef<Group, any>[] = [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <Link to={`/groups/${row.original.id}`} className="font-medium text-nb-gray-50 hover:text-netbird-400">
          {row.original.name}
        </Link>
      ),
    },
    {
      accessorKey: "peers_count",
      header: "Peers",
      cell: ({ getValue }) => <span className="font-mono text-xs">{getValue<number>()}</span>,
    },
  ];

  if (canDelete(MODULE.groups)) {
    columns.push({
      id: "actions",
      header: "",
      cell: ({ row }) => {
        const group = row.original;
        if (group.name === PROTECTED_GROUP_NAME) return null;
        return (
          <div className="flex justify-end">
            <Button
              variant="ghost"
              size="icon"
              className="text-red-500 hover:bg-red-950 hover:text-red-400"
              disabled={deleteGroup.isPending}
              onClick={() => {
                if (confirm(`Delete group "${group.name}"?`)) {
                  deleteGroup.mutate(group.id, { onError: (err) => toast.error(err.message) });
                }
              }}
              aria-label="Delete group"
            >
              <Trash2 size={16} />
            </Button>
          </div>
        );
      },
    });
  }

  return (
    <section className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-nb-gray-50">Groups</h1>
          <p className="text-sm text-nb-gray-500">
            {data ? `${data.total} group${data.total === 1 ? "" : "s"}` : " "}
          </p>
        </div>
        {canCreate(MODULE.groups) && (
          <Button variant="primary" onClick={() => setShowCreate(true)}>
            <Plus size={16} /> New group
          </Button>
        )}
      </div>

      {error && (
        <div className="rounded-md border border-red-900 bg-red-950/40 px-4 py-3 text-sm text-red-300">
          {error.message}
        </div>
      )}

      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New group</DialogTitle>
          </DialogHeader>
          {showCreate && <CreateGroupForm onDone={() => setShowCreate(false)} />}
        </DialogContent>
      </Dialog>

      <DataTable
        columns={columns}
        data={data?.groups ?? []}
        isLoading={isLoading}
        emptyState={<span className="text-nb-gray-500">No groups yet. Create one above.</span>}
      />
    </section>
  );
}

function CreateGroupForm({ onDone }: { onDone: () => void }) {
  const { data: peersData } = usePeers("");
  const createGroup = useCreateGroup();
  const [name, setName] = useState("");
  const [selected, setSelected] = useState<string[]>([]);

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        createGroup.mutate(
          { name, peers: selected },
          { onSuccess: onDone, onError: (err) => toast.error(err.message) },
        );
      }}
    >
      <DialogBody className="flex flex-col gap-4">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="group-name">Name</Label>
          <Input id="group-name" required value={name} onChange={(e) => setName(e.target.value)} />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label>Peers</Label>
          <PeerCheckboxList peers={peersData?.peers ?? []} selected={selected} onChange={setSelected} />
        </div>
      </DialogBody>

      <DialogFooter>
        <Button variant="primary" type="submit" disabled={createGroup.isPending}>
          Create group
        </Button>
      </DialogFooter>
    </form>
  );
}
