import { Plus } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import type { NameserverGroup, NameserverGroupRequest } from "@/api/dns";
import {
  useCreateNameserverGroup,
  useDeleteNameserverGroup,
  useNameserverGroups,
  useUpdateNameserverGroup,
} from "@/api/dns";
import { useGroups } from "@/api/groups";
import GroupCheckboxList from "@/components/GroupCheckboxList";
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
import Switch from "@/components/ui/Switch";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/Table";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";

const EMPTY: NameserverGroupRequest = {
  name: "",
  description: "",
  nameservers: [{ ip: "", port: 53 }],
  enabled: true,
  groups: [],
  primary: true,
  domains: [],
  search_domains_enabled: false,
};

function toRequest(group: NameserverGroup): NameserverGroupRequest {
  return {
    name: group.name,
    description: group.description,
    nameservers: group.nameservers.map((ns) => ({ ip: ns.ip, port: ns.port })),
    enabled: group.enabled,
    groups: group.groups,
    primary: group.primary,
    domains: group.domains,
    search_domains_enabled: group.search_domains_enabled,
  };
}

export default function NameserversList() {
  const { data, isLoading, error } = useNameserverGroups();
  const deleteGroup = useDeleteNameserverGroup();
  const [editingId, setEditingId] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const { canCreate, canUpdate, canDelete } = usePermissionsContext();
  const canEdit = canUpdate(MODULE.nameservers);
  const canRemove = canDelete(MODULE.nameservers);

  const nameserverGroups = data?.groups ?? [];
  const editing = nameserverGroups.find((g) => g.id === editingId);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-nb-gray-500">
          {data ? `${data.total} nameserver group${data.total === 1 ? "" : "s"}` : " "}
        </p>
        {canCreate(MODULE.nameservers) && (
          <Button variant="primary" onClick={() => setShowCreate(true)}>
            <Plus size={16} /> New nameserver group
          </Button>
        )}
      </div>

      {error && (
        <div className="rounded-md border border-red-900 bg-red-950/40 px-4 py-3 text-sm text-red-300">
          {error.message}
        </div>
      )}

      {editing && (
        <Card>
          <CardHeader>
            <CardTitle>Edit nameserver group</CardTitle>
          </CardHeader>
          <NameserverGroupForm
            groupId={editing.id}
            initial={toRequest(editing)}
            onDone={() => setEditingId(null)}
            onCancel={() => setEditingId(null)}
          />
        </Card>
      )}

      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New nameserver group</DialogTitle>
          </DialogHeader>
          {showCreate && <NameserverGroupForm onDone={() => setShowCreate(false)} inDialog />}
        </DialogContent>
      </Dialog>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Servers</TableHead>
            <TableHead>Resolves</TableHead>
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
          ) : data && nameserverGroups.length === 0 ? (
            <TableRow>
              <TableCell colSpan={5} className="py-8 text-center text-nb-gray-500">
                No nameserver groups yet. Create one above.
              </TableCell>
            </TableRow>
          ) : (
            nameserverGroups.map((group) => (
              <TableRow key={group.id}>
                <TableCell>{group.name}</TableCell>
                <TableCell className="font-mono text-xs">{group.nameservers.length}</TableCell>
                <TableCell>
                  {group.primary ? <Badge variant="success">primary</Badge> : group.domains.join(", ")}
                </TableCell>
                <TableCell>
                  <Badge variant={group.enabled ? "success" : "default"}>
                    {group.enabled ? "enabled" : "disabled"}
                  </Badge>
                </TableCell>
                <TableCell>
                  <div className="flex justify-end gap-2">
                    {canEdit && (
                      <Button variant="ghost" size="sm" onClick={() => setEditingId(group.id)}>
                        Edit
                      </Button>
                    )}
                    {canRemove && (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-red-500 hover:bg-red-950 hover:text-red-400"
                        disabled={deleteGroup.isPending}
                        onClick={() => {
                          if (confirm(`Delete nameserver group "${group.name}"?`))
                            deleteGroup.mutate(group.id, { onError: (err) => toast.error(err.message) });
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

function NameserverGroupForm({
  groupId,
  initial,
  onDone,
  onCancel,
  inDialog,
}: {
  groupId?: string;
  initial?: NameserverGroupRequest;
  onDone: () => void;
  onCancel?: () => void;
  inDialog?: boolean;
}) {
  const { data: groupsData } = useGroups();
  const create = useCreateNameserverGroup();
  const update = useUpdateNameserverGroup(groupId ?? "");
  const mutation = groupId ? update : create;

  const [value, setValue] = useState<NameserverGroupRequest>(initial ?? EMPTY);
  const [domainsText, setDomainsText] = useState((initial?.domains ?? []).join(", "));

  const Body = inDialog ? DialogBody : CardContent;
  const Foot = inDialog ? DialogFooter : CardFooter;

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        const domains = domainsText
          .split(/[,\s]+/)
          .map((s) => s.trim())
          .filter(Boolean);
        mutation.mutate(
          { ...value, domains },
          { onSuccess: onDone, onError: (err) => toast.error(err.message) },
        );
      }}
    >
      <Body className="flex flex-col gap-4">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="ns-name">Name</Label>
          <Input
            id="ns-name"
            required
            maxLength={40}
            value={value.name}
            onChange={(e) => setValue({ ...value, name: e.target.value })}
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label>Nameservers</Label>
          <p className="text-xs text-nb-gray-500">Up to 3. At least one is required.</p>
          {[0, 1, 2].map((i) => (
            <div className="flex gap-2" key={i}>
              <Input
                className="flex-[2]"
                placeholder={i === 0 ? "e.g. 1.1.1.1" : "optional"}
                value={value.nameservers[i]?.ip ?? ""}
                onChange={(e) => {
                  const next = [...value.nameservers];
                  if (e.target.value === "") {
                    next.splice(i, 1);
                  } else {
                    next[i] = { ip: e.target.value, port: next[i]?.port ?? 53 };
                  }
                  setValue({ ...value, nameservers: next.filter((ns) => ns.ip !== "") });
                }}
              />
              <Input
                type="number"
                className="flex-1"
                min={1}
                max={65535}
                placeholder="53"
                value={value.nameservers[i]?.port ?? ""}
                onChange={(e) => {
                  const next = [...value.nameservers];
                  if (next[i]) next[i] = { ...next[i], port: Number(e.target.value) };
                  setValue({ ...value, nameservers: next });
                }}
              />
            </div>
          ))}
        </div>

        <label htmlFor="ns-enabled" className="flex items-center justify-between gap-4">
          <span className="text-sm text-nb-gray-100">Enabled</span>
          <Switch id="ns-enabled" checked={value.enabled} onCheckedChange={(v) => setValue({ ...value, enabled: v })} />
        </label>

        <div className="flex flex-col gap-1.5">
          <Label>Distribution groups</Label>
          <GroupCheckboxList
            groups={groupsData?.groups ?? []}
            selected={value.groups}
            onChange={(ids) => setValue({ ...value, groups: ids })}
          />
        </div>

        <label htmlFor="ns-primary" className="flex items-center justify-between gap-4">
          <span className="flex flex-col">
            <span className="text-sm text-nb-gray-100">Primary</span>
            <span className="text-xs text-nb-gray-500">Resolves every domain not matched by another group.</span>
          </span>
          <Switch id="ns-primary" checked={value.primary} onCheckedChange={(v) => setValue({ ...value, primary: v })} />
        </label>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="ns-domains">Match domains</Label>
          <Input
            id="ns-domains"
            placeholder="example.com, internal.corp"
            value={domainsText}
            onChange={(e) => setDomainsText(e.target.value)}
          />
          <p className="text-xs text-nb-gray-500">Comma-separated. Required unless Primary is checked.</p>
        </div>

        <label htmlFor="ns-search-domains" className="flex items-center justify-between gap-4">
          <span className="text-sm text-nb-gray-100">Use as search domain</span>
          <Switch
            id="ns-search-domains"
            checked={value.search_domains_enabled}
            onCheckedChange={(v) => setValue({ ...value, search_domains_enabled: v })}
          />
        </label>
      </Body>

      <Foot>
        {onCancel && (
          <Button variant="secondary" type="button" onClick={onCancel}>
            Cancel
          </Button>
        )}
        <Button variant="primary" type="submit" disabled={mutation.isPending}>
          {groupId ? "Save" : "Create nameserver group"}
        </Button>
      </Foot>
    </form>
  );
}
