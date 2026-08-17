import { Plus } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import type { Group } from "@/api/groups";
import type { NetworkRouter, RouterRequest } from "@/api/networks";
import { useCreateRouter, useDeleteRouter, useRouters, useUpdateRouter } from "@/api/networks";
import { usePeers } from "@/api/peers";
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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import Switch from "@/components/ui/Switch";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/Table";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";

const EMPTY: RouterRequest = { peer: "", peer_groups: [], metric: 100, masquerade: false, enabled: true };

function toRequest(router: NetworkRouter): RouterRequest {
  return {
    peer: router.peer,
    // peer_groups comes back JSON `null` for a router that targets a single
    // peer instead — same nil-slice-from-server shape as GroupDetail's
    // group.peers, so this must not assume an array.
    peer_groups: router.peer_groups ?? [],
    metric: router.metric,
    masquerade: router.masquerade,
    enabled: router.enabled,
  };
}

export default function RoutersPanel({ networkId, groups }: { networkId: string; groups: Group[] }) {
  const { data, isLoading } = useRouters(networkId);
  const deleteRouter = useDeleteRouter(networkId);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [showAdd, setShowAdd] = useState(false);
  const { canCreate, canUpdate, canDelete } = usePermissionsContext();
  const canEdit = canUpdate(MODULE.networks);
  const canRemove = canDelete(MODULE.networks);

  const routers = data?.routers ?? [];
  const editing = routers.find((r) => r.id === editingId);

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-nb-gray-200">Routers</h2>
        {canCreate(MODULE.networks) && (
          <Button variant="secondary" size="sm" onClick={() => setShowAdd(true)}>
            <Plus size={14} /> Add router
          </Button>
        )}
      </div>

      {editing && (
        <Card>
          <CardHeader>
            <CardTitle>Edit router</CardTitle>
          </CardHeader>
          <RouterForm
            networkId={networkId}
            groups={groups}
            routerId={editing.id}
            initial={toRequest(editing)}
            onDone={() => setEditingId(null)}
            onCancel={() => setEditingId(null)}
          />
        </Card>
      )}

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Routing target</TableHead>
            <TableHead>Metric</TableHead>
            <TableHead>Masquerade</TableHead>
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
          ) : data && routers.length === 0 ? (
            <TableRow>
              <TableCell colSpan={5} className="py-8 text-center text-nb-gray-500">
                No routers yet.
              </TableCell>
            </TableRow>
          ) : (
            routers.map((router) => (
              <TableRow key={router.id}>
                <TableCell className="font-mono text-xs">
                  {router.peer || `${(router.peer_groups ?? []).length} group(s)`}
                </TableCell>
                <TableCell className="font-mono text-xs">{router.metric}</TableCell>
                <TableCell>{router.masquerade ? "✓" : "—"}</TableCell>
                <TableCell>
                  <Badge variant={router.enabled ? "success" : "default"}>
                    {router.enabled ? "enabled" : "disabled"}
                  </Badge>
                </TableCell>
                <TableCell>
                  <div className="flex justify-end gap-2">
                    {canEdit && (
                      <Button variant="ghost" size="sm" onClick={() => setEditingId(router.id)}>
                        Edit
                      </Button>
                    )}
                    {canRemove && (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-red-500 hover:bg-red-950 hover:text-red-400"
                        disabled={deleteRouter.isPending}
                        onClick={() => {
                          if (confirm("Delete this router?"))
                            deleteRouter.mutate(router.id, { onError: (err) => toast.error(err.message) });
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
            <DialogTitle>Add router</DialogTitle>
          </DialogHeader>
          {showAdd && <RouterForm networkId={networkId} groups={groups} onDone={() => setShowAdd(false)} inDialog />}
        </DialogContent>
      </Dialog>
    </div>
  );
}

function RouterForm({
  networkId,
  groups,
  routerId,
  initial,
  onDone,
  onCancel,
  inDialog,
}: {
  networkId: string;
  groups: Group[];
  routerId?: string;
  initial?: RouterRequest;
  onDone: () => void;
  onCancel?: () => void;
  inDialog?: boolean;
}) {
  const { data: peersData } = usePeers("");
  const create = useCreateRouter(networkId);
  const update = useUpdateRouter(networkId, routerId ?? "");
  const mutation = routerId ? update : create;

  const [value, setValue] = useState<RouterRequest>(initial ?? EMPTY);
  const [targetType, setTargetType] = useState<"peer" | "groups">(
    (initial?.peer_groups.length ?? 0) > 0 ? "groups" : "peer",
  );

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
          <Label htmlFor="router-target-type">Routing target</Label>
          <Select
            value={targetType}
            onValueChange={(v) => {
              const next = v as "peer" | "groups";
              setTargetType(next);
              setValue(next === "peer" ? { ...value, peer_groups: [] } : { ...value, peer: "" });
            }}
          >
            <SelectTrigger id="router-target-type">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="peer">Single routing peer</SelectItem>
              <SelectItem value="groups">Peer group(s)</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {targetType === "peer" ? (
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="router-peer">Peer</Label>
            <Select value={value.peer} onValueChange={(v) => setValue({ ...value, peer: v })}>
              <SelectTrigger id="router-peer">
                <SelectValue placeholder="Select a peer…" />
              </SelectTrigger>
              <SelectContent>
                {peersData?.peers.map((p) => (
                  <SelectItem key={p.id} value={p.id}>
                    {p.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        ) : (
          <div className="flex flex-col gap-1.5">
            <Label>Peer groups</Label>
            <GroupCheckboxList
              groups={groups}
              selected={value.peer_groups}
              onChange={(ids) => setValue({ ...value, peer_groups: ids })}
            />
          </div>
        )}

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="router-metric">Metric</Label>
          <Input
            id="router-metric"
            type="number"
            min={1}
            max={9999}
            required
            value={value.metric}
            onChange={(e) => setValue({ ...value, metric: Number(e.target.value) })}
          />
          <p className="text-xs text-nb-gray-500">
            Lower wins when more than one router can reach the same destination.
          </p>
        </div>

        <label htmlFor="router-masquerade" className="flex items-center justify-between gap-4">
          <span className="text-sm text-nb-gray-100">Masquerade (source NAT)</span>
          <Switch
            id="router-masquerade"
            checked={value.masquerade}
            onCheckedChange={(v) => setValue({ ...value, masquerade: v })}
          />
        </label>

        {routerId && (
          <label htmlFor="router-enabled" className="flex items-center justify-between gap-4">
            <span className="text-sm text-nb-gray-100">Enabled</span>
            <Switch
              id="router-enabled"
              checked={value.enabled}
              onCheckedChange={(v) => setValue({ ...value, enabled: v })}
            />
          </label>
        )}
      </Body>

      <Foot>
        {onCancel && (
          <Button variant="secondary" type="button" onClick={onCancel}>
            Cancel
          </Button>
        )}
        <Button variant="primary" type="submit" disabled={mutation.isPending}>
          {routerId ? "Save router" : "Add router"}
        </Button>
      </Foot>
    </form>
  );
}
