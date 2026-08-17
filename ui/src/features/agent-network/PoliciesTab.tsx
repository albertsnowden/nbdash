import { Plus } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import type { AgentNetworkPolicyRequest } from "@/api/agentNetwork";
import {
  useAgentNetworkPolicies,
  useAgentNetworkProviders,
  useCreateAgentNetworkPolicy,
  useDeleteAgentNetworkPolicy,
  useUpdateAgentNetworkPolicy,
} from "@/api/agentNetwork";
import { useGroups } from "@/api/groups";
import GroupCheckboxList from "@/components/GroupCheckboxList";
import Badge from "@/components/ui/Badge";
import Button from "@/components/ui/Button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/Card";
import Checkbox from "@/components/ui/Checkbox";
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
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/Table";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";

const EMPTY: AgentNetworkPolicyRequest = {
  name: "",
  description: "",
  enabled: true,
  source_groups: [],
  destination_provider_ids: [],
};

export default function PoliciesTab() {
  const { data, isLoading, error } = useAgentNetworkPolicies();
  const { data: providersData } = useAgentNetworkProviders();
  const deletePolicy = useDeleteAgentNetworkPolicy();
  const [editingId, setEditingId] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const { canCreate, canUpdate, canDelete } = usePermissionsContext();
  const canEdit = canUpdate(MODULE.agentNetworkPolicies);
  const canRemove = canDelete(MODULE.agentNetworkPolicies);

  const providerName = (id: string) => providersData?.providers?.find((p) => p.id === id)?.name ?? id;
  const policies = data?.policies ?? [];
  const editing = policies.find((p) => p.id === editingId);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-nb-gray-500">{data ? `${data.total} polic${data.total === 1 ? "y" : "ies"}` : " "}</p>
        {canCreate(MODULE.agentNetworkPolicies) && (
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
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New policy</DialogTitle>
          </DialogHeader>
          {showCreate && <PolicyForm onDone={() => setShowCreate(false)} />}
        </DialogContent>
      </Dialog>

      {editing && (
        <Card>
          <CardHeader>
            <CardTitle>Edit {editing.name}</CardTitle>
          </CardHeader>
          <PolicyForm
            policyId={editing.id}
            initial={{
              name: editing.name,
              description: editing.description,
              enabled: editing.enabled,
              source_groups: editing.source_groups,
              destination_provider_ids: editing.destination_provider_ids,
            }}
            onDone={() => setEditingId(null)}
            onCancel={() => setEditingId(null)}
          />
        </Card>
      )}

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Source groups</TableHead>
            <TableHead>Destination providers</TableHead>
            <TableHead>Status</TableHead>
            <TableHead />
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableRow>
              <TableCell colSpan={5} className="py-8 text-center text-nb-gray-500">
                Loading…
              </TableCell>
            </TableRow>
          ) : data && policies.length === 0 ? (
            <TableRow>
              <TableCell colSpan={5} className="py-8 text-center text-nb-gray-500">
                No policies yet. Create one above.
              </TableCell>
            </TableRow>
          ) : (
            policies.map((policy) => (
              <TableRow key={policy.id}>
                <TableCell>{policy.name}</TableCell>
                <TableCell className="font-mono text-xs">{policy.source_groups.length}</TableCell>
                <TableCell>
                  <div className="flex flex-wrap gap-1">
                    {policy.destination_provider_ids.map((id) => (
                      <Badge key={id}>{providerName(id)}</Badge>
                    ))}
                  </div>
                </TableCell>
                <TableCell>
                  <Badge variant={policy.enabled ? "success" : "default"}>
                    {policy.enabled ? "enabled" : "disabled"}
                  </Badge>
                </TableCell>
                <TableCell>
                  <div className="flex justify-end gap-2">
                    {canEdit && (
                      <Button variant="ghost" size="sm" onClick={() => setEditingId(policy.id)}>
                        Edit
                      </Button>
                    )}
                    {canRemove && (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-red-500 hover:bg-red-950 hover:text-red-400"
                        disabled={deletePolicy.isPending}
                        onClick={() => {
                          if (confirm(`Delete policy "${policy.name}"?`))
                            deletePolicy.mutate(policy.id, { onError: (err) => toast.error(err.message) });
                        }}
                      >
                        Delete
                      </Button>
                    )}
                  </div>
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </div>
  );
}

function PolicyForm({
  policyId,
  initial,
  onDone,
  onCancel,
}: {
  policyId?: string;
  initial?: AgentNetworkPolicyRequest;
  onDone: () => void;
  onCancel?: () => void;
}) {
  const { data: groupsData } = useGroups();
  const { data: providersData } = useAgentNetworkProviders();
  const providers = providersData?.providers ?? [];
  const create = useCreateAgentNetworkPolicy();
  const update = useUpdateAgentNetworkPolicy(policyId ?? "");
  const mutation = policyId ? update : create;

  const [value, setValue] = useState<AgentNetworkPolicyRequest>(initial ?? EMPTY);

  const inDialog = !policyId;
  const Body = inDialog ? DialogBody : CardContent;
  const Foot = inDialog ? DialogFooter : CardFooter;

  const toggleProvider = (id: string) => {
    setValue({
      ...value,
      destination_provider_ids: value.destination_provider_ids.includes(id)
        ? value.destination_provider_ids.filter((x) => x !== id)
        : [...value.destination_provider_ids, id],
    });
  };

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        mutation.mutate(value, { onSuccess: onDone, onError: (err) => toast.error(err.message) });
      }}
    >
      <Body className="flex max-h-[70vh] flex-col gap-4 overflow-y-auto">
        <p className="text-xs text-nb-gray-500">
          A policy grants members of the selected NetBird groups access to the selected AI providers. A user
          with no matching policy can't reach any provider through the Agent Network gateway.
        </p>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="anp-name">Name</Label>
          <Input id="anp-name" required value={value.name} onChange={(e) => setValue({ ...value, name: e.target.value })} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="anp-description">Description</Label>
          <Input
            id="anp-description"
            value={value.description}
            onChange={(e) => setValue({ ...value, description: e.target.value })}
          />
        </div>
        <label htmlFor="anp-enabled" className="flex items-center justify-between gap-4">
          <span className="text-sm text-nb-gray-100">Enabled</span>
          <Switch id="anp-enabled" checked={value.enabled} onCheckedChange={(v) => setValue({ ...value, enabled: v })} />
        </label>

        <div className="flex flex-col gap-1.5">
          <Label>Source groups</Label>
          <GroupCheckboxList
            groups={groupsData?.groups ?? []}
            selected={value.source_groups}
            onChange={(ids) => setValue({ ...value, source_groups: ids })}
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label>Destination providers</Label>
          {providers.length > 0 ? (
            <div className="flex max-h-64 flex-col gap-0.5 overflow-y-auto rounded-md border border-nb-gray-900 p-2">
              {providers.map((p) => (
                <label
                  key={p.id}
                  className="flex cursor-pointer items-center gap-2 rounded-sm px-2 py-1.5 text-sm text-nb-gray-100 hover:bg-nb-gray-900"
                >
                  <Checkbox
                    checked={value.destination_provider_ids.includes(p.id)}
                    onCheckedChange={() => toggleProvider(p.id)}
                  />
                  <span>{p.name}</span>
                </label>
              ))}
            </div>
          ) : (
            <p className="text-sm text-nb-gray-500">Connect a provider first.</p>
          )}
        </div>
      </Body>
      <Foot>
        {onCancel && (
          <Button variant="secondary" type="button" onClick={onCancel}>
            Cancel
          </Button>
        )}
        <Button variant="primary" type="submit" disabled={mutation.isPending}>
          {policyId ? "Save" : "Create policy"}
        </Button>
      </Foot>
    </form>
  );
}
