import type { ColumnDef } from "@tanstack/react-table";
import { Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";
import type { Zone, ZoneRequest } from "@/api/dns";
import { useCreateZone, useDeleteZone, useZones } from "@/api/dns";
import { useGroups } from "@/api/groups";
import GroupCheckboxList from "@/components/GroupCheckboxList";
import DataTable from "@/components/table/DataTable";
import Badge from "@/components/ui/Badge";
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
import Switch from "@/components/ui/Switch";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";

const EMPTY: ZoneRequest = {
  name: "",
  domain: "",
  enabled: true,
  enable_search_domain: false,
  distribution_groups: [],
};

export default function ZonesList() {
  const { data, isLoading, error } = useZones();
  const deleteZone = useDeleteZone();
  const [showCreate, setShowCreate] = useState(false);
  const { canCreate, canDelete } = usePermissionsContext();

  const columns: ColumnDef<Zone, any>[] = [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <Link to={`/dns/zones/${row.original.id}`} className="font-medium text-nb-gray-50 hover:text-netbird-400">
          {row.original.name}
        </Link>
      ),
    },
    {
      accessorKey: "domain",
      header: "Domain",
      cell: ({ getValue }) => <span className="font-mono text-xs">{getValue<string>()}</span>,
    },
    {
      id: "records",
      header: "Records",
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.records.length}</span>,
    },
    {
      accessorKey: "enabled",
      header: "Status",
      cell: ({ getValue }) => (
        <Badge variant={getValue<boolean>() ? "success" : "default"}>
          {getValue<boolean>() ? "enabled" : "disabled"}
        </Badge>
      ),
    },
  ];

  if (canDelete(MODULE.dns)) {
    columns.push({
      id: "actions",
      header: "",
      cell: ({ row }) => {
        const zone = row.original;
        return (
          <div className="flex justify-end">
            <Button
              variant="ghost"
              size="icon"
              className="text-red-500 hover:bg-red-950 hover:text-red-400"
              disabled={deleteZone.isPending}
              onClick={() => {
                if (confirm(`Delete DNS zone "${zone.name}"?`))
                  deleteZone.mutate(zone.id, { onError: (err) => toast.error(err.message) });
              }}
              aria-label="Delete zone"
            >
              <Trash2 size={16} />
            </Button>
          </div>
        );
      },
    });
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-nb-gray-500">{data ? `${data.total} zone${data.total === 1 ? "" : "s"}` : " "}</p>
        {canCreate(MODULE.dns) && (
          <Button variant="primary" onClick={() => setShowCreate(true)}>
            <Plus size={16} /> New zone
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
            <DialogTitle>New zone</DialogTitle>
          </DialogHeader>
          {showCreate && <CreateZoneForm onDone={() => setShowCreate(false)} />}
        </DialogContent>
      </Dialog>

      <DataTable
        columns={columns}
        data={data?.zones ?? []}
        isLoading={isLoading}
        emptyState={<span className="text-nb-gray-500">No DNS zones yet. Create one above.</span>}
      />
    </div>
  );
}

function CreateZoneForm({ onDone }: { onDone: () => void }) {
  const { data: groupsData } = useGroups();
  const createZone = useCreateZone();
  const [value, setValue] = useState<ZoneRequest>(EMPTY);

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        createZone.mutate(value, { onSuccess: onDone, onError: (err) => toast.error(err.message) });
      }}
    >
      <DialogBody className="flex flex-col gap-4">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="zone-name">Name</Label>
          <Input id="zone-name" required value={value.name} onChange={(e) => setValue({ ...value, name: e.target.value })} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="zone-domain">Domain</Label>
          <Input
            id="zone-domain"
            required
            placeholder="internal.example.com"
            value={value.domain}
            onChange={(e) => setValue({ ...value, domain: e.target.value })}
          />
          <p className="text-xs text-nb-gray-500">Cannot be changed after the zone is created.</p>
        </div>
        <label htmlFor="zone-enabled" className="flex items-center justify-between gap-4">
          <span className="text-sm text-nb-gray-100">Enabled</span>
          <Switch id="zone-enabled" checked={value.enabled} onCheckedChange={(v) => setValue({ ...value, enabled: v })} />
        </label>
        <label htmlFor="zone-search-domain" className="flex items-center justify-between gap-4">
          <span className="text-sm text-nb-gray-100">Use as search domain</span>
          <Switch
            id="zone-search-domain"
            checked={value.enable_search_domain}
            onCheckedChange={(v) => setValue({ ...value, enable_search_domain: v })}
          />
        </label>
        <div className="flex flex-col gap-1.5">
          <Label>Distribution groups</Label>
          <GroupCheckboxList
            groups={groupsData?.groups ?? []}
            selected={value.distribution_groups}
            onChange={(ids) => setValue({ ...value, distribution_groups: ids })}
          />
        </div>
      </DialogBody>
      <DialogFooter>
        <Button variant="primary" type="submit" disabled={createZone.isPending}>
          Create zone
        </Button>
      </DialogFooter>
    </form>
  );
}
