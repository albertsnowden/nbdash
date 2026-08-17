import type { ColumnDef } from "@tanstack/react-table";
import { Plus } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";
import type { PostureCheck, PostureCheckRequest } from "@/api/postureChecks";
import {
  useCreatePostureCheck,
  useDeletePostureCheck,
  usePostureChecks,
  useUpdatePostureCheck,
} from "@/api/postureChecks";
import DataTable from "@/components/table/DataTable";
import Badge from "@/components/ui/Badge";
import Button from "@/components/ui/Button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/Card";
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
import PostureCheckFieldsInputs, {
  checksToFormState,
  EMPTY_CHECKS_FORM,
  formStateToChecks,
  type ChecksFormState,
} from "./PostureCheckFieldsInputs";

function checkLabels(check: PostureCheck): string[] {
  const labels: string[] = [];
  if (check.checks.nb_version_check) labels.push("NetBird version");
  if (check.checks.os_version_check) labels.push("OS version");
  if (check.checks.geo_location_check) labels.push("Geo location");
  if (check.checks.peer_network_range_check) labels.push("Network range");
  if (check.checks.process_check) labels.push("Processes");
  return labels;
}

export default function PostureChecksList() {
  const { data, isLoading, error } = usePostureChecks();
  const deleteCheck = useDeletePostureCheck();
  const [editingId, setEditingId] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const { canCreate, canUpdate, canDelete } = usePermissionsContext();
  const canEdit = canUpdate(MODULE.policies);
  const canRemove = canDelete(MODULE.policies);

  const editing = data?.posture_checks?.find((c) => c.id === editingId);

  const columns: ColumnDef<PostureCheck, any>[] = [
    { accessorKey: "name", header: "Name" },
    {
      accessorKey: "description",
      header: "Description",
      cell: ({ getValue }) => <span className="text-nb-gray-400">{getValue<string>() || "—"}</span>,
    },
    {
      id: "checks",
      header: "Checks",
      cell: ({ row }) => {
        const labels = checkLabels(row.original);
        if (labels.length === 0) return <span className="text-nb-gray-600">None configured</span>;
        return (
          <div className="flex flex-wrap gap-1">
            {labels.map((l) => (
              <Badge key={l}>{l}</Badge>
            ))}
          </div>
        );
      },
    },
  ];

  if (canEdit || canRemove) {
    columns.push({
      id: "actions",
      header: "",
      cell: ({ row }) => {
        const check = row.original;
        return (
          <div className="flex justify-end gap-2">
            {canEdit && (
              <Button variant="ghost" size="sm" onClick={() => setEditingId(check.id)}>
                Edit
              </Button>
            )}
            {canRemove && (
              <Button
                variant="ghost"
                size="sm"
                className="text-red-500 hover:bg-red-950 hover:text-red-400"
                disabled={deleteCheck.isPending}
                onClick={() => {
                  if (confirm(`Delete posture check "${check.name}"?`))
                    deleteCheck.mutate(check.id, { onError: (err) => toast.error(err.message) });
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
          <h1 className="text-xl font-semibold text-nb-gray-50">Posture Checks</h1>
          <p className="text-sm text-nb-gray-500">
            {data ? `${data.total} posture check${data.total === 1 ? "" : "s"}` : " "}
          </p>
        </div>
        {canCreate(MODULE.policies) && (
          <Button variant="primary" onClick={() => setShowCreate(true)}>
            <Plus size={16} /> New posture check
          </Button>
        )}
      </div>

      <p className="text-sm text-nb-gray-500">
        Attach a posture check to a policy from that policy's Settings on the{" "}
        <Link to="/policies" className="text-netbird-400 hover:underline">
          Access Control
        </Link>{" "}
        page — a peer in the policy's source groups that fails an attached check loses access even though its
        group membership would otherwise allow it.
      </p>

      {error && (
        <div className="rounded-md border border-red-900 bg-red-950/40 px-4 py-3 text-sm text-red-300">
          {error.message}
        </div>
      )}

      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>New posture check</DialogTitle>
          </DialogHeader>
          {showCreate && <PostureCheckForm onDone={() => setShowCreate(false)} />}
        </DialogContent>
      </Dialog>

      {editing && (
        <Card>
          <CardHeader>
            <CardTitle>Edit {editing.name}</CardTitle>
          </CardHeader>
          <PostureCheckForm
            checkId={editing.id}
            initial={{ name: editing.name, description: editing.description, checks: editing.checks }}
            onDone={() => setEditingId(null)}
            onCancel={() => setEditingId(null)}
          />
        </Card>
      )}

      <DataTable
        columns={columns}
        data={data?.posture_checks ?? []}
        isLoading={isLoading}
        emptyState={<span className="text-nb-gray-500">No posture checks yet. Create one above.</span>}
      />
    </section>
  );
}

function PostureCheckForm({
  checkId,
  initial,
  onDone,
  onCancel,
}: {
  checkId?: string;
  initial?: PostureCheckRequest;
  onDone: () => void;
  onCancel?: () => void;
}) {
  const create = useCreatePostureCheck();
  const update = useUpdatePostureCheck(checkId ?? "");
  const mutation = checkId ? update : create;

  const [name, setName] = useState(initial?.name ?? "");
  const [description, setDescription] = useState(initial?.description ?? "");
  const [checksForm, setChecksForm] = useState<ChecksFormState>(
    initial ? checksToFormState(initial.checks) : EMPTY_CHECKS_FORM,
  );

  const inDialog = !checkId;
  const Body = inDialog ? DialogBody : CardContent;
  const Foot = inDialog ? DialogFooter : CardFooter;

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        mutation.mutate(
          { name, description, checks: formStateToChecks(checksForm) },
          { onSuccess: onDone, onError: (err) => toast.error(err.message) },
        );
      }}
    >
      <Body className="flex max-h-[70vh] flex-col gap-4 overflow-y-auto">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="pc-name">Name</Label>
          <Input id="pc-name" required value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="pc-description">Description</Label>
          <Input id="pc-description" value={description} onChange={(e) => setDescription(e.target.value)} />
        </div>

        <PostureCheckFieldsInputs value={checksForm} onChange={setChecksForm} />
      </Body>
      <Foot>
        {onCancel && (
          <Button variant="secondary" type="button" onClick={onCancel}>
            Cancel
          </Button>
        )}
        <Button variant="primary" type="submit" disabled={mutation.isPending}>
          {checkId ? "Save" : "Create posture check"}
        </Button>
      </Foot>
    </form>
  );
}
