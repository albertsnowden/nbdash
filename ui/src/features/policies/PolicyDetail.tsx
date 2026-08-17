import type { ColumnDef } from "@tanstack/react-table";
import { ArrowLeft, Plus, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { toast } from "sonner";
import type { Group } from "@/api/groups";
import { useGroups } from "@/api/groups";
import type { PolicyRule } from "@/api/policies";
import {
  useCreateRule,
  useDeletePolicy,
  useDeleteRule,
  usePolicy,
  useUpdatePolicyMeta,
  useUpdateRule,
} from "@/api/policies";
import { usePostureChecks } from "@/api/postureChecks";
import PostureCheckCheckboxList from "@/components/PostureCheckCheckboxList";
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
import InfoTooltip from "@/components/ui/InfoTooltip";
import Input from "@/components/ui/Input";
import Label from "@/components/ui/Label";
import Switch from "@/components/ui/Switch";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";
import RuleFieldsInputs, { parsePorts, type RuleFormState } from "./RuleFieldsInputs";

function ruleToFormState(rule: PolicyRule): RuleFormState {
  return {
    name: rule.name,
    action: rule.action,
    protocol: rule.protocol === "netbird-ssh" ? "all" : rule.protocol,
    bidirectional: rule.bidirectional,
    enabled: rule.enabled,
    sources: (rule.sources ?? []).map((g) => g.id),
    destinations: (rule.destinations ?? []).map((g) => g.id),
    portsText: (rule.ports ?? []).join(", "),
  };
}

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

export default function PolicyDetail() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const { data: policy, isLoading, error } = usePolicy(id);
  const { data: groupsData } = useGroups();
  const { data: postureChecksData } = usePostureChecks();
  const updateMeta = useUpdatePolicyMeta(id);
  const deletePolicy = useDeletePolicy();
  const deleteRule = useDeleteRule(id);
  const { canCreate, canUpdate, canDelete } = usePermissionsContext();
  const editable = canUpdate(MODULE.policies);
  const canAddRule = canCreate(MODULE.policies);
  const canEditRule = canUpdate(MODULE.policies);
  const canRemoveRule = canDelete(MODULE.policies);

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [enabled, setEnabled] = useState(true);
  const [sourcePostureChecks, setSourcePostureChecks] = useState<string[]>([]);
  const [editingRuleId, setEditingRuleId] = useState<string | null>(null);
  const [showAddRule, setShowAddRule] = useState(false);

  useEffect(() => {
    if (!policy) return;
    setName(policy.name);
    setDescription(policy.description);
    setEnabled(policy.enabled);
    setSourcePostureChecks(policy.source_posture_checks ?? []);
  }, [policy]);

  if (isLoading) return <p className="text-sm text-nb-gray-500">Loading…</p>;
  if (error) {
    return (
      <div className="rounded-md border border-red-900 bg-red-950/40 px-4 py-3 text-sm text-red-300">
        {error.message}
      </div>
    );
  }
  if (!policy) return null;

  const groups = groupsData?.groups ?? [];

  const columns: ColumnDef<PolicyRule, any>[] = [
    { accessorKey: "name", header: "Name" },
    {
      accessorKey: "action",
      header: "Action",
      cell: ({ getValue }) => (
        <Badge variant={getValue<string>() === "accept" ? "success" : "danger"}>{getValue<string>()}</Badge>
      ),
    },
    {
      accessorKey: "protocol",
      header: "Protocol",
      cell: ({ getValue }) => <span className="font-mono text-xs">{getValue<string>()}</span>,
    },
    {
      accessorKey: "bidirectional",
      header: "Direction",
      cell: ({ getValue }) => (getValue<boolean>() ? "↔" : "→"),
    },
  ];

  if (canEditRule || canRemoveRule) {
    columns.push({
      id: "actions",
      header: "",
      cell: ({ row }) => {
        const rule = row.original;
        return (
          <div className="flex justify-end gap-2">
            {canEditRule && (
              <Button variant="ghost" size="sm" onClick={() => setEditingRuleId(rule.id)}>
                Edit
              </Button>
            )}
            {canRemoveRule && (
              <Button
                variant="ghost"
                size="sm"
                className="text-red-500 hover:bg-red-950 hover:text-red-400"
                disabled={deleteRule.isPending || policy.rules.length <= 1}
                title={policy.rules.length <= 1 ? "A policy needs at least one rule" : undefined}
                onClick={() => {
                  if (confirm(`Delete rule "${rule.name}"?`))
                    deleteRule.mutate(rule.id, { onError: (err) => toast.error(err.message) });
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

  const editingRule = policy.rules.find((r) => r.id === editingRuleId);
  // netbird's own management API doesn't reliably persist more than one
  // rule per policy: a rule submitted without its own ID (the only way a
  // genuinely new rule can be submitted — the API rejects any ID it doesn't
  // already recognize) gets silently assigned the whole policy's ID rather
  // than a fresh one, and the database layer then upserts by that ID, so a
  // second rule overwrites the first instead of adding to it. Confirmed
  // against netbird v0.76.3, v0.77.0 (latest release) and main as of
  // 2026-08-16 — this is netbirdio/netbird#3583, open since March 2025 with
  // no fix in progress, not something this dashboard can work around
  // client-side. Blocking "Add rule" here is the only way to stop someone
  // from believing a second rule saved when it silently didn't.
  const canAddMoreRules = policy.rules.length === 0;

  return (
    <section className="flex max-w-2xl flex-col gap-4">
      <Link to="/policies" className="inline-flex items-center gap-1 text-sm text-nb-gray-500 hover:text-nb-gray-200">
        <ArrowLeft size={14} /> All policies
      </Link>

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-nb-gray-50">{policy.name}</h1>
          <p className="text-sm text-nb-gray-500">{policy.rules.length} rules</p>
        </div>
        {canDelete(MODULE.policies) && (
          <Button
            variant="danger"
            disabled={deletePolicy.isPending}
            onClick={() => {
              if (confirm(`Delete policy "${policy.name}"?`)) {
                deletePolicy.mutate(id, {
                  onSuccess: () => navigate("/policies"),
                  onError: (err) => toast.error(err.message),
                });
              }
            }}
          >
            <Trash2 size={14} /> Delete
          </Button>
        )}
      </div>

      <form
        onSubmit={(e) => {
          e.preventDefault();
          updateMeta.mutate(
            { name, description, enabled, source_posture_checks: sourcePostureChecks },
            { onSuccess: () => toast.success("Policy updated."), onError: (err) => toast.error(err.message) },
          );
        }}
      >
        <Card>
          <CardHeader>
            <CardTitle>Settings</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="policy-name">Name</Label>
              <Input
                id="policy-name"
                required
                disabled={!editable}
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="policy-description">Description</Label>
              <Input
                id="policy-description"
                disabled={!editable}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            </div>
            <label htmlFor="policy-enabled" className="flex items-center justify-between gap-4">
              <span className="text-sm text-nb-gray-100">Enabled</span>
              <Switch id="policy-enabled" checked={enabled} disabled={!editable} onCheckedChange={setEnabled} />
            </label>
            <div className="flex flex-col gap-1.5">
              <div className="flex items-center gap-1.5">
                <Label>Posture checks</Label>
                <InfoTooltip>
                  Applied to this policy's source groups — a peer that fails an attached check loses access
                  even though its group membership would otherwise allow it. Optional.
                </InfoTooltip>
              </div>
              <PostureCheckCheckboxList
                checks={postureChecksData?.posture_checks ?? []}
                selected={sourcePostureChecks}
                onChange={setSourcePostureChecks}
              />
            </div>
          </CardContent>
          {editable && (
            <CardFooter>
              <Button variant="primary" type="submit" disabled={updateMeta.isPending}>
                Save
              </Button>
            </CardFooter>
          )}
        </Card>
      </form>

      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-nb-gray-200">Rules</h2>
        {canAddRule && (
          <Button
            variant="secondary"
            size="sm"
            disabled={!canAddMoreRules}
            title={
              canAddMoreRules
                ? undefined
                : "netbird doesn't reliably save more than one rule per policy yet (netbirdio/netbird#3583) — create a separate policy instead."
            }
            onClick={() => setShowAddRule(true)}
          >
            <Plus size={14} /> Add rule
          </Button>
        )}
      </div>

      {canAddRule && !canAddMoreRules && (
        <div className="rounded-md border border-yellow-900 bg-yellow-950/30 px-3 py-2 text-xs text-yellow-300">
          netbird doesn't reliably save more than one rule per policy yet — a second rule silently
          fails to persist instead of erroring (see{" "}
          <a
            href="https://github.com/netbirdio/netbird/issues/3583"
            target="_blank"
            rel="noopener noreferrer"
            className="underline hover:text-yellow-200"
          >
            netbirdio/netbird#3583
          </a>
          ). Create a separate policy for additional rules instead.
        </div>
      )}

      {editingRule && (
        <Card>
          <CardHeader>
            <CardTitle>Edit rule</CardTitle>
          </CardHeader>
          <RuleEditForm
            policyId={id}
            rule={editingRule}
            groups={groups}
            onDone={() => setEditingRuleId(null)}
            onCancel={() => setEditingRuleId(null)}
          />
        </Card>
      )}

      <DataTable columns={columns} data={policy.rules} />

      <Dialog open={showAddRule} onOpenChange={setShowAddRule}>
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>Add rule</DialogTitle>
          </DialogHeader>
          {showAddRule && <AddRuleForm policyId={id} groups={groups} onDone={() => setShowAddRule(false)} />}
        </DialogContent>
      </Dialog>
    </section>
  );
}

function AddRuleForm({ policyId, groups, onDone }: { policyId: string; groups: Group[]; onDone: () => void }) {
  const createRule = useCreateRule(policyId);
  const [rule, setRule] = useState<RuleFormState>(EMPTY_RULE);

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        createRule.mutate(
          { ...rule, ports: parsePorts(rule.portsText), description: "" },
          { onSuccess: onDone, onError: (err) => toast.error(err.message) },
        );
      }}
    >
      <DialogBody className="max-h-[70vh] overflow-y-auto">
        <RuleFieldsInputs groups={groups} value={rule} onChange={setRule} />
      </DialogBody>
      <DialogFooter>
        <Button variant="primary" type="submit" disabled={createRule.isPending}>
          Add rule
        </Button>
      </DialogFooter>
    </form>
  );
}

function RuleEditForm({
  policyId,
  rule,
  groups,
  onDone,
  onCancel,
}: {
  policyId: string;
  rule: PolicyRule;
  groups: Group[];
  onDone: () => void;
  onCancel: () => void;
}) {
  const updateRule = useUpdateRule(policyId, rule.id);
  const [value, setValue] = useState<RuleFormState>(() => ruleToFormState(rule));

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        updateRule.mutate(
          { ...value, ports: parsePorts(value.portsText), description: "" },
          { onSuccess: onDone, onError: (err) => toast.error(err.message) },
        );
      }}
    >
      <CardContent>
        <RuleFieldsInputs groups={groups} value={value} onChange={setValue} />
      </CardContent>
      <CardFooter>
        <Button variant="secondary" type="button" onClick={onCancel}>
          Cancel
        </Button>
        <Button variant="primary" type="submit" disabled={updateRule.isPending}>
          Save rule
        </Button>
      </CardFooter>
    </form>
  );
}
