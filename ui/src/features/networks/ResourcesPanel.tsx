import { Plus } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import type { Group } from "@/api/groups";
import type { NetworkResource, ResourceRequest } from "@/api/networks";
import { useCreateResource, useDeleteResource, useResources, useUpdateResource } from "@/api/networks";
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

const EMPTY: ResourceRequest = { name: "", description: "", address: "", enabled: true, groups: [] };

function toRequest(resource: NetworkResource): ResourceRequest {
  return {
    name: resource.name,
    description: resource.description,
    address: resource.address,
    enabled: resource.enabled,
    // resource.groups comes back as JSON `null` when the server's group-info
    // lookup for this resource misses (same root cause as GroupDetail's
    // group.peers), so this must not assume an array.
    groups: (resource.groups ?? []).map((g) => g.id),
  };
}

export default function ResourcesPanel({ networkId, groups }: { networkId: string; groups: Group[] }) {
  const { data, isLoading } = useResources(networkId);
  const deleteResource = useDeleteResource(networkId);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [showAdd, setShowAdd] = useState(false);
  const { canCreate, canUpdate, canDelete } = usePermissionsContext();
  const canEdit = canUpdate(MODULE.networks);
  const canRemove = canDelete(MODULE.networks);

  const resources = data?.resources ?? [];
  const editing = resources.find((r) => r.id === editingId);

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-nb-gray-200">Resources</h2>
        {canCreate(MODULE.networks) && (
          <Button variant="secondary" size="sm" onClick={() => setShowAdd(true)}>
            <Plus size={14} /> Add resource
          </Button>
        )}
      </div>

      {editing && (
        <Card>
          <CardHeader>
            <CardTitle>Edit resource</CardTitle>
          </CardHeader>
          <ResourceForm
            networkId={networkId}
            groups={groups}
            resourceId={editing.id}
            initial={toRequest(editing)}
            onDone={() => setEditingId(null)}
            onCancel={() => setEditingId(null)}
          />
        </Card>
      )}

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Address</TableHead>
            <TableHead>Type</TableHead>
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
          ) : data && resources.length === 0 ? (
            <TableRow>
              <TableCell colSpan={5} className="py-8 text-center text-nb-gray-500">
                No resources yet.
              </TableCell>
            </TableRow>
          ) : (
            resources.map((resource) => (
              <TableRow key={resource.id}>
                <TableCell>{resource.name}</TableCell>
                <TableCell className="font-mono text-xs">{resource.address}</TableCell>
                <TableCell className="font-mono text-xs">{resource.type}</TableCell>
                <TableCell>
                  <Badge variant={resource.enabled ? "success" : "default"}>
                    {resource.enabled ? "enabled" : "disabled"}
                  </Badge>
                </TableCell>
                <TableCell>
                  <div className="flex justify-end gap-2">
                    {canEdit && (
                      <Button variant="ghost" size="sm" onClick={() => setEditingId(resource.id)}>
                        Edit
                      </Button>
                    )}
                    {canRemove && (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-red-500 hover:bg-red-950 hover:text-red-400"
                        disabled={deleteResource.isPending}
                        onClick={() => {
                          if (confirm(`Delete resource "${resource.name}"?`))
                            deleteResource.mutate(resource.id, { onError: (err) => toast.error(err.message) });
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

      <Dialog open={showAdd} onOpenChange={setShowAdd}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add resource</DialogTitle>
          </DialogHeader>
          {showAdd && <ResourceForm networkId={networkId} groups={groups} onDone={() => setShowAdd(false)} inDialog />}
        </DialogContent>
      </Dialog>
    </div>
  );
}

function ResourceForm({
  networkId,
  groups,
  resourceId,
  initial,
  onDone,
  onCancel,
  inDialog,
}: {
  networkId: string;
  groups: Group[];
  resourceId?: string;
  initial?: ResourceRequest;
  onDone: () => void;
  onCancel?: () => void;
  inDialog?: boolean;
}) {
  const create = useCreateResource(networkId);
  const update = useUpdateResource(networkId, resourceId ?? "");
  const mutation = resourceId ? update : create;

  const [value, setValue] = useState<ResourceRequest>(initial ?? EMPTY);

  const Body = inDialog ? DialogBody : CardContent;
  const Foot = inDialog ? DialogFooter : CardFooter;

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        mutation.mutate(value, { onSuccess: onDone, onError: (err) => toast.error(err.message) });
      }}
    >
      <Body className="flex flex-col gap-4">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="resource-name">Name</Label>
          <Input
            id="resource-name"
            required
            value={value.name}
            onChange={(e) => setValue({ ...value, name: e.target.value })}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="resource-address">Address</Label>
          <Input
            id="resource-address"
            required
            placeholder="10.0.0.1, 10.0.0.0/24, or example.com"
            value={value.address}
            onChange={(e) => setValue({ ...value, address: e.target.value })}
          />
          <p className="text-xs text-nb-gray-500">
            Type (host / subnet / domain) is derived automatically from what you enter.
          </p>
        </div>
        <label htmlFor="resource-enabled" className="flex items-center justify-between gap-4">
          <span className="text-sm text-nb-gray-100">Enabled</span>
          <Switch
            id="resource-enabled"
            checked={value.enabled}
            onCheckedChange={(v) => setValue({ ...value, enabled: v })}
          />
        </label>
        <div className="flex flex-col gap-1.5">
          <Label>Groups</Label>
          <GroupCheckboxList
            groups={groups}
            selected={value.groups}
            onChange={(ids) => setValue({ ...value, groups: ids })}
          />
        </div>
      </Body>

      <Foot>
        {onCancel && (
          <Button variant="secondary" type="button" onClick={onCancel}>
            Cancel
          </Button>
        )}
        <Button variant="primary" type="submit" disabled={mutation.isPending}>
          {resourceId ? "Save resource" : "Add resource"}
        </Button>
      </Foot>
    </form>
  );
}
