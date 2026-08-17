import type { ColumnDef } from "@tanstack/react-table";
import { Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";
import { useGroups } from "@/api/groups";
import type { Policy } from "@/api/policies";
import { useCreatePolicy, useDeletePolicy, usePolicies } from "@/api/policies";
import { usePostureChecks } from "@/api/postureChecks";
import PostureCheckCheckboxList from "@/components/PostureCheckCheckboxList";
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
import Switch from "@/components/ui/Switch";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";
import RuleFieldsInputs, { parsePorts, type RuleFormState } from "./RuleFieldsInputs";

const EMPTY_RULE: RuleFormState = {
  name: "",
  action: "accept",
  protocol: "all",
  bidirectional: true,
  enabled: true,
  sources: [],
  destinations: [],
  portsText: "",
};

export default function PoliciesList() {
  const { data, isLoading, error } = usePolicies();
  const deletePolicy = useDeletePolicy();
  const [showCreate, setShowCreate] = useState(false);
  const { canCreate, canDelete } = usePermissionsContext();

  const columns: ColumnDef<Policy, any>[] = [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <Link to={`/policies/${row.original.id}`} className="font-medium text-nb-gray-50 hover:text-netbird-400">
          {row.original.name}
        </Link>
      ),
    },
    {
      id: "rules",
      header: "Rules",
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.rules.length}</span>,
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

  if (canDelete(MODULE.policies)) {
    columns.push({
      id: "actions",
      header: "",
      cell: ({ row }) => {
        const policy = row.original;
        return (
          <div className="flex justify-end">
            <Button
              variant="ghost"
              size="icon"
              className="text-red-500 hover:bg-red-950 hover:text-red-400"
              disabled={deletePolicy.isPending}
              onClick={() => {
                if (confirm(`Delete policy "${policy.name}"?`))
                  deletePolicy.mutate(policy.id, { onError: (err) => toast.error(err.message) });
              }}
              aria-label="Delete policy"
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
          <h1 className="text-xl font-semibold text-nb-gray-50">Access Control</h1>
          <p className="text-sm text-nb-gray-500">
            {data ? `${data.enabled} of ${data.total} policies enabled` : " "}
          </p>
        </div>
        {canCreate(MODULE.policies) && (
          <Button variant="primary" onClick={() => setShowCreate(true)}>
            <Plus size={16} /> New policy
          </Button>
        )}
      </div>

      {error && (
        <div className="rounded-md border border-red-900 bg-red-950/40 px-4 py-3 text-sm text-red-300">
          {error.message}
        </div>
      )}

      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>New policy</DialogTitle>
          </DialogHeader>
          {showCreate && <CreatePolicyForm onDone={() => setShowCreate(false)} />}
        </DialogContent>
      </Dialog>

      <DataTable
        columns={columns}
        data={data?.policies ?? []}
        isLoading={isLoading}
        emptyState={<span className="text-nb-gray-500">No policies yet. Create one above.</span>}
      />
    </section>
  );
}

function CreatePolicyForm({ onDone }: { onDone: () => void }) {
  const { data: groupsData } = useGroups();
  const { data: postureChecksData } = usePostureChecks();
  const createPolicy = useCreatePolicy();

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [enabled, setEnabled] = useState(true);
  const [rule, setRule] = useState<RuleFormState>(EMPTY_RULE);
  const [sourcePostureChecks, setSourcePostureChecks] = useState<string[]>([]);

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        createPolicy.mutate(
          {
            name,
            description,
            enabled,
            rule: { ...rule, ports: parsePorts(rule.portsText), description: "" },
            source_posture_checks: sourcePostureChecks,
          },
          { onSuccess: onDone, onError: (err) => toast.error(err.message) },
        );
      }}
    >
      <DialogBody className="flex max-h-[70vh] flex-col gap-4 overflow-y-auto">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="policy-name">Name</Label>
          <Input id="policy-name" required value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="policy-description">Description</Label>
          <Input id="policy-description" value={description} onChange={(e) => setDescription(e.target.value)} />
        </div>
        <label htmlFor="policy-enabled" className="flex items-center justify-between gap-4">
          <span className="text-sm text-nb-gray-100">Enabled</span>
          <Switch id="policy-enabled" checked={enabled} onCheckedChange={setEnabled} />
        </label>

        <div className="flex flex-col gap-1.5">
          <div className="flex items-center gap-1.5">
            <Label>Posture checks</Label>
            <InfoTooltip>
              Applied to this policy's source groups — a peer that fails an attached check loses access even
              though its group membership would otherwise allow it. Optional.
            </InfoTooltip>
          </div>
          <PostureCheckCheckboxList
            checks={postureChecksData?.posture_checks ?? []}
            selected={sourcePostureChecks}
            onChange={setSourcePostureChecks}
          />
        </div>

        <h2 className="mt-2 text-xs font-semibold uppercase tracking-wide text-nb-gray-500">First rule</h2>
        <RuleFieldsInputs groups={groupsData?.groups ?? []} value={rule} onChange={setRule} />
      </DialogBody>

      <DialogFooter>
        <Button variant="primary" type="submit" disabled={createPolicy.isPending}>
          Create policy
        </Button>
      </DialogFooter>
    </form>
  );
}
