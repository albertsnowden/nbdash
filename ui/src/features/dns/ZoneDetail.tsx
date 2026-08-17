import { ArrowLeft, Plus, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { toast } from "sonner";
import type { DNSRecord, RecordRequest } from "@/api/dns";
import {
  useCreateRecord,
  useDeleteRecord,
  useDeleteZone,
  useUpdateRecord,
  useUpdateZone,
  useZone,
} from "@/api/dns";
import { useGroups } from "@/api/groups";
import GroupCheckboxList from "@/components/GroupCheckboxList";
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

const EMPTY_RECORD: RecordRequest = { name: "", type: "A", content: "", ttl: 300 };

export default function ZoneDetail() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const { data: zone, isLoading, error } = useZone(id);
  const { data: groupsData } = useGroups();
  const updateZone = useUpdateZone(id);
  const deleteZone = useDeleteZone();
  const deleteRecord = useDeleteRecord(id);
  const { canCreate, canUpdate, canDelete } = usePermissionsContext();
  const editable = canUpdate(MODULE.dns);
  const canAddRecord = canCreate(MODULE.dns);
  const canEditRecord = canUpdate(MODULE.dns);
  const canRemoveRecord = canDelete(MODULE.dns);

  const [name, setName] = useState("");
  const [enabled, setEnabled] = useState(true);
  const [enableSearchDomain, setEnableSearchDomain] = useState(false);
  const [distributionGroups, setDistributionGroups] = useState<string[]>([]);
  const [editingRecordId, setEditingRecordId] = useState<string | null>(null);
  const [showAddRecord, setShowAddRecord] = useState(false);

  useEffect(() => {
    if (!zone) return;
    setName(zone.name);
    setEnabled(zone.enabled);
    setEnableSearchDomain(zone.enable_search_domain);
    setDistributionGroups(zone.distribution_groups);
  }, [zone]);

  if (isLoading) return <p className="text-sm text-nb-gray-500">Loading…</p>;
  if (error) {
    return (
      <div className="rounded-md border border-red-900 bg-red-950/40 px-4 py-3 text-sm text-red-300">
        {error.message}
      </div>
    );
  }
  if (!zone) return null;

  const editingRecord = zone.records.find((r) => r.id === editingRecordId);

  return (
    <section className="flex max-w-2xl flex-col gap-4">
      <Link to="/dns/zones" className="inline-flex items-center gap-1 text-sm text-nb-gray-500 hover:text-nb-gray-200">
        <ArrowLeft size={14} /> All zones
      </Link>

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-nb-gray-50">{zone.name}</h1>
          <p className="font-mono text-sm text-nb-gray-500">{zone.domain}</p>
        </div>
        {canDelete(MODULE.dns) && (
          <Button
            variant="danger"
            disabled={deleteZone.isPending}
            onClick={() => {
              if (confirm(`Delete DNS zone "${zone.name}"?`)) {
                deleteZone.mutate(id, {
                  onSuccess: () => navigate("/dns/zones"),
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
          updateZone.mutate(
            {
              name,
              domain: zone.domain,
              enabled,
              enable_search_domain: enableSearchDomain,
              distribution_groups: distributionGroups,
            },
            { onSuccess: () => toast.success("Zone updated."), onError: (err) => toast.error(err.message) },
          );
        }}
      >
        <Card>
          <CardHeader>
            <CardTitle>Settings</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="zone-name">Name</Label>
              <Input
                id="zone-name"
                required
                disabled={!editable}
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>Domain</Label>
              <Input readOnly value={zone.domain} />
              <p className="text-xs text-nb-gray-500">Cannot be changed after creation.</p>
            </div>
            <label htmlFor="zone-enabled" className="flex items-center justify-between gap-4">
              <span className="text-sm text-nb-gray-100">Enabled</span>
              <Switch id="zone-enabled" checked={enabled} disabled={!editable} onCheckedChange={setEnabled} />
            </label>
            <label htmlFor="zone-search-domain" className="flex items-center justify-between gap-4">
              <span className="text-sm text-nb-gray-100">Use as search domain</span>
              <Switch
                id="zone-search-domain"
                checked={enableSearchDomain}
                disabled={!editable}
                onCheckedChange={setEnableSearchDomain}
              />
            </label>
            <div className="flex flex-col gap-1.5">
              <Label>Distribution groups</Label>
              <GroupCheckboxList
                groups={groupsData?.groups ?? []}
                selected={distributionGroups}
                onChange={setDistributionGroups}
              />
            </div>
          </CardContent>
          {editable && (
            <CardFooter>
              <Button variant="primary" type="submit" disabled={updateZone.isPending}>
                Save
              </Button>
            </CardFooter>
          )}
        </Card>
      </form>

      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-nb-gray-200">Records</h2>
        {canAddRecord && (
          <Button variant="secondary" size="sm" onClick={() => setShowAddRecord(true)}>
            <Plus size={14} /> Add record
          </Button>
        )}
      </div>

      {editingRecord && (
        <Card>
          <CardHeader>
            <CardTitle>Edit record</CardTitle>
          </CardHeader>
          <RecordForm
            zoneId={id}
            recordId={editingRecord.id}
            initial={toRecordRequest(editingRecord)}
            onDone={() => setEditingRecordId(null)}
            onCancel={() => setEditingRecordId(null)}
          />
        </Card>
      )}

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Type</TableHead>
            <TableHead>Content</TableHead>
            <TableHead>TTL</TableHead>
            <TableHead />
          </TableRow>
        </TableHeader>
        <TableBody>
          {zone.records.length === 0 ? (
            <TableRow>
              <TableCell colSpan={5} className="py-8 text-center text-nb-gray-500">
                No records yet.
              </TableCell>
            </TableRow>
          ) : (
            zone.records.map((record) => (
              <TableRow key={record.id}>
                <TableCell>{record.name}</TableCell>
                <TableCell className="font-mono text-xs">{record.type}</TableCell>
                <TableCell className="font-mono text-xs">{record.content}</TableCell>
                <TableCell className="font-mono text-xs">{record.ttl}</TableCell>
                <TableCell>
                  <div className="flex justify-end gap-2">
                    {canEditRecord && (
                      <Button variant="ghost" size="sm" onClick={() => setEditingRecordId(record.id)}>
                        Edit
                      </Button>
                    )}
                    {canRemoveRecord && (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-red-500 hover:bg-red-950 hover:text-red-400"
                        disabled={deleteRecord.isPending}
                        onClick={() => {
                          if (confirm(`Delete record "${record.name}"?`))
                            deleteRecord.mutate(record.id, { onError: (err) => toast.error(err.message) });
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

      <Dialog open={showAddRecord} onOpenChange={setShowAddRecord}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add record</DialogTitle>
          </DialogHeader>
          {showAddRecord && <RecordForm zoneId={id} onDone={() => setShowAddRecord(false)} inDialog />}
        </DialogContent>
      </Dialog>
    </section>
  );
}

function toRecordRequest(record: DNSRecord): RecordRequest {
  return { name: record.name, type: record.type, content: record.content, ttl: record.ttl };
}

function RecordForm({
  zoneId,
  recordId,
  initial,
  onDone,
  onCancel,
  inDialog,
}: {
  zoneId: string;
  recordId?: string;
  initial?: RecordRequest;
  onDone: () => void;
  onCancel?: () => void;
  inDialog?: boolean;
}) {
  const create = useCreateRecord(zoneId);
  const update = useUpdateRecord(zoneId, recordId ?? "");
  const mutation = recordId ? update : create;
  const [value, setValue] = useState<RecordRequest>(initial ?? EMPTY_RECORD);

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
          <Label htmlFor="record-name">Name</Label>
          <Input
            id="record-name"
            required
            value={value.name}
            onChange={(e) => setValue({ ...value, name: e.target.value })}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="record-type">Type</Label>
          <Select value={value.type} onValueChange={(v) => setValue({ ...value, type: v as RecordRequest["type"] })}>
            <SelectTrigger id="record-type">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="A">A</SelectItem>
              <SelectItem value="AAAA">AAAA</SelectItem>
              <SelectItem value="CNAME">CNAME</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="record-content">Content</Label>
          <Input
            id="record-content"
            required
            value={value.content}
            onChange={(e) => setValue({ ...value, content: e.target.value })}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="record-ttl">TTL</Label>
          <Input
            id="record-ttl"
            type="number"
            min={0}
            required
            value={value.ttl}
            onChange={(e) => setValue({ ...value, ttl: Number(e.target.value) })}
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
          {recordId ? "Save record" : "Add record"}
        </Button>
      </Foot>
    </form>
  );
}
