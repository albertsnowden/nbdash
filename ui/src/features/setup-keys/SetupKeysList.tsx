import type { ColumnDef } from "@tanstack/react-table";
import { Copy, Plus } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { useGroups } from "@/api/groups";
import type { SetupKey } from "@/api/setupKeys";
import { useCreateSetupKey, useDeleteSetupKey, useRevokeSetupKey, useSetupKeys } from "@/api/setupKeys";
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
import InfoTooltip from "@/components/ui/InfoTooltip";
import Input from "@/components/ui/Input";
import Label from "@/components/ui/Label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import Switch from "@/components/ui/Switch";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { absTime } from "@/lib/format";
import { MODULE } from "@/lib/modules";

const STATE_VARIANT: Record<string, "success" | "danger" | "warning"> = {
  valid: "success",
  revoked: "danger",
  expired: "warning",
  overused: "warning",
};

export default function SetupKeysList() {
  const { data, isLoading, error } = useSetupKeys();
  const revokeKey = useRevokeSetupKey();
  const deleteKey = useDeleteSetupKey();
  const [showCreate, setShowCreate] = useState(false);
  const [reveal, setReveal] = useState<SetupKey | null>(null);
  const { canCreate, canUpdate, canDelete } = usePermissionsContext();
  const canRevoke = canUpdate(MODULE.setupKeys);
  const canRemove = canDelete(MODULE.setupKeys);

  const columns: ColumnDef<SetupKey, any>[] = [
    { accessorKey: "name", header: "Name" },
    {
      accessorKey: "type",
      header: "Type",
      cell: ({ getValue }) => <span className="font-mono text-xs">{getValue<string>()}</span>,
    },
    {
      id: "uses",
      header: "Uses",
      cell: ({ row }) => (
        <span className="font-mono text-xs">
          {row.original.used_times}
          {row.original.usage_limit > 0 ? ` / ${row.original.usage_limit}` : " / ∞"}
        </span>
      ),
    },
    {
      accessorKey: "expires",
      header: "Expires",
      cell: ({ getValue }) => absTime(getValue<string>()),
    },
    {
      accessorKey: "state",
      header: "Status",
      cell: ({ getValue }) => {
        const state = getValue<string>();
        return <Badge variant={STATE_VARIANT[state] ?? "default"}>{state}</Badge>;
      },
    },
  ];

  if (canRevoke || canRemove) {
    columns.push({
      id: "actions",
      header: "",
      cell: ({ row }) => {
        const k = row.original;
        return (
          <div className="flex justify-end gap-2">
            {k.valid && canRevoke && (
              <Button
                variant="ghost"
                size="sm"
                disabled={revokeKey.isPending}
                onClick={() => {
                  if (confirm(`Revoke setup key "${k.name}"?`))
                    revokeKey.mutate(k.id, { onError: (err) => toast.error(err.message) });
                }}
              >
                Revoke
              </Button>
            )}
            {canRemove && (
              <Button
                variant="ghost"
                size="sm"
                className="text-red-500 hover:bg-red-950 hover:text-red-400"
                disabled={deleteKey.isPending}
                onClick={() => {
                  if (confirm(`Delete setup key "${k.name}"?`))
                    deleteKey.mutate(k.id, { onError: (err) => toast.error(err.message) });
                }}
              >
                Delete
              </Button>
            )}
          </div>
        );
      },
    });
  }

  return (
    <section className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-nb-gray-50">Setup Keys</h1>
          <p className="text-sm text-nb-gray-500">{data ? `${data.valid} of ${data.total} valid` : " "}</p>
        </div>
        {canCreate(MODULE.setupKeys) && (
          <Button variant="primary" onClick={() => setShowCreate(true)}>
            <Plus size={16} /> New setup key
          </Button>
        )}
      </div>

      {error && (
        <div className="rounded-md border border-red-900 bg-red-950/40 px-4 py-3 text-sm text-red-300">
          {error.message}
        </div>
      )}

      <Dialog open={!!reveal} onOpenChange={(open) => !open && setReveal(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Setup key created</DialogTitle>
          </DialogHeader>
          {reveal && <PlaintextReveal setupKey={reveal} onDone={() => setReveal(null)} />}
        </DialogContent>
      </Dialog>

      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New setup key</DialogTitle>
          </DialogHeader>
          {showCreate && (
            <CreateSetupKeyForm
              onDone={(created) => {
                setShowCreate(false);
                setReveal(created);
              }}
            />
          )}
        </DialogContent>
      </Dialog>

      <DataTable
        columns={columns}
        data={data?.setup_keys ?? []}
        isLoading={isLoading}
        emptyState={<span className="text-nb-gray-500">No setup keys yet. Create one above.</span>}
      />
    </section>
  );
}

function PlaintextReveal({ setupKey, onDone }: { setupKey: SetupKey; onDone: () => void }) {
  return (
    <>
      <DialogBody className="flex flex-col gap-4">
        <div className="rounded-md border border-yellow-900 bg-yellow-950/30 px-3 py-2 text-sm text-yellow-300">
          This is the only time this key's value is shown — copy it now.
        </div>
        <div className="flex items-center gap-2">
          <Input className="font-mono" readOnly value={setupKey.key} onFocus={(e) => e.target.select()} />
          <Button
            variant="secondary"
            type="button"
            onClick={() => void navigator.clipboard.writeText(setupKey.key)}
          >
            <Copy size={14} /> Copy
          </Button>
        </div>
      </DialogBody>
      <DialogFooter>
        <Button variant="primary" type="button" onClick={onDone}>
          Done
        </Button>
      </DialogFooter>
    </>
  );
}

function CreateSetupKeyForm({ onDone }: { onDone: (created: SetupKey) => void }) {
  const { data: groupsData } = useGroups();
  const createKey = useCreateSetupKey();

  const [name, setName] = useState("");
  const [type, setType] = useState<"one-off" | "reusable">("reusable");
  const [expiresInDays, setExpiresInDays] = useState(30);
  const [usageLimit, setUsageLimit] = useState(0);
  const [ephemeral, setEphemeral] = useState(false);
  const [autoGroups, setAutoGroups] = useState<string[]>([]);

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        createKey.mutate(
          { name, type, expires_in_days: expiresInDays, usage_limit: usageLimit, ephemeral, auto_groups: autoGroups },
          { onSuccess: onDone, onError: (err) => toast.error(err.message) },
        );
      }}
    >
      <DialogBody className="flex flex-col gap-4">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sk-name">Name</Label>
          <Input id="sk-name" required value={name} onChange={(e) => setName(e.target.value)} />
        </div>

        <div className="flex flex-col gap-1.5">
          <div className="flex items-center gap-1.5">
            <Label htmlFor="sk-type">Type</Label>
            <InfoTooltip>
              Reusable keys can enroll any number of peers until they expire or hit their usage limit. One-off
              keys are consumed by the first peer that uses them.
            </InfoTooltip>
          </div>
          <Select value={type} onValueChange={(v) => setType(v as "one-off" | "reusable")}>
            <SelectTrigger id="sk-type">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="reusable">Reusable</SelectItem>
              <SelectItem value="one-off">One-off</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sk-expires">Expires in (days)</Label>
          <Input
            id="sk-expires"
            type="number"
            min={1}
            max={365}
            required
            value={expiresInDays}
            onChange={(e) => setExpiresInDays(Number(e.target.value))}
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sk-usage-limit">Usage limit</Label>
          <Input
            id="sk-usage-limit"
            type="number"
            min={0}
            value={usageLimit}
            onChange={(e) => setUsageLimit(Number(e.target.value))}
          />
          <p className="text-xs text-nb-gray-500">0 means unlimited.</p>
        </div>

        <div className="flex items-center justify-between gap-4">
          <span className="inline-flex items-center gap-1.5 text-sm text-nb-gray-100">
            <label htmlFor="sk-ephemeral">Ephemeral peers</label>
            <InfoTooltip>
              Peers enrolled with this key are automatically removed a short time after they disconnect —
              useful for CI runners and other short-lived machines that shouldn't clutter the peer list.
            </InfoTooltip>
          </span>
          <Switch id="sk-ephemeral" checked={ephemeral} onCheckedChange={setEphemeral} />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label>Auto-assign groups</Label>
          <GroupCheckboxList groups={groupsData?.groups ?? []} selected={autoGroups} onChange={setAutoGroups} />
        </div>
      </DialogBody>

      <DialogFooter>
        <Button variant="primary" type="submit" disabled={createKey.isPending}>
          Create setup key
        </Button>
      </DialogFooter>
    </form>
  );
}
