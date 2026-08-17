import type { ColumnDef } from "@tanstack/react-table";
import { Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";
import type { Network } from "@/api/networks";
import { useCreateNetwork, useDeleteNetwork, useNetworks } from "@/api/networks";
import DataTable from "@/components/table/DataTable";
import Button from "@/components/ui/Button";
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
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";

export default function NetworksList() {
  const { data, isLoading, error } = useNetworks();
  const deleteNetwork = useDeleteNetwork();
  const [showCreate, setShowCreate] = useState(false);
  const { canCreate, canDelete } = usePermissionsContext();

  const columns: ColumnDef<Network, any>[] = [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <Link to={`/networks/${row.original.id}`} className="font-medium text-nb-gray-50 hover:text-netbird-400">
          {row.original.name}
        </Link>
      ),
    },
    {
      id: "resources",
      header: "Resources",
      cell: ({ row }) => <span className="font-mono text-xs">{(row.original.resources ?? []).length}</span>,
    },
    {
      id: "routers",
      header: "Routers",
      cell: ({ row }) => <span className="font-mono text-xs">{(row.original.routers ?? []).length}</span>,
    },
  ];

  if (canDelete(MODULE.networks)) {
    columns.push({
      id: "actions",
      header: "",
      cell: ({ row }) => {
        const network = row.original;
        return (
          <div className="flex justify-end">
            <Button
              variant="ghost"
              size="icon"
              className="text-red-500 hover:bg-red-950 hover:text-red-400"
              disabled={deleteNetwork.isPending}
              onClick={() => {
                if (confirm(`Delete network "${network.name}"? Its resources and routers go with it.`))
                  deleteNetwork.mutate(network.id, { onError: (err) => toast.error(err.message) });
              }}
              aria-label="Delete network"
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
          <h1 className="text-xl font-semibold text-nb-gray-50">Networks</h1>
          <p className="text-sm text-nb-gray-500">
            {data ? `${data.total} network${data.total === 1 ? "" : "s"}` : " "}
          </p>
        </div>
        {canCreate(MODULE.networks) && (
          <Button variant="primary" onClick={() => setShowCreate(true)}>
            <Plus size={16} /> New network
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
            <DialogTitle>New network</DialogTitle>
          </DialogHeader>
          {showCreate && <CreateNetworkForm onDone={() => setShowCreate(false)} />}
        </DialogContent>
      </Dialog>

      <DataTable
        columns={columns}
        data={data?.networks ?? []}
        isLoading={isLoading}
        emptyState={<span className="text-nb-gray-500">No networks yet. Create one above.</span>}
      />
    </section>
  );
}

function CreateNetworkForm({ onDone }: { onDone: () => void }) {
  const createNetwork = useCreateNetwork();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        createNetwork.mutate(
          { name, description },
          { onSuccess: onDone, onError: (err) => toast.error(err.message) },
        );
      }}
    >
      <DialogBody className="flex flex-col gap-4">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="network-name">Name</Label>
          <Input id="network-name" required value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="network-description">Description</Label>
          <Input id="network-description" value={description} onChange={(e) => setDescription(e.target.value)} />
        </div>
      </DialogBody>
      <DialogFooter>
        <Button variant="primary" type="submit" disabled={createNetwork.isPending}>
          Create network
        </Button>
      </DialogFooter>
    </form>
  );
}
