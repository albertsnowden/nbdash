import type { ColumnDef } from "@tanstack/react-table";
import { Plus } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import type { Service, ServiceRequest } from "@/api/reverseProxy";
import { useCreateService, useDeleteService, useServices, useUpdateService } from "@/api/reverseProxy";
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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import Switch from "@/components/ui/Switch";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";
import ServiceTargetsInput from "./ServiceTargetsInput";

const STATUS_VARIANT: Record<string, "success" | "warning" | "danger" | "default"> = {
  active: "success",
  pending: "warning",
  certificate_pending: "warning",
  tunnel_not_created: "warning",
  certificate_failed: "danger",
  error: "danger",
};

const EMPTY_REQUEST: ServiceRequest = {
  name: "",
  domain: "",
  mode: "http",
  listen_port: 0,
  targets: [{ target_id: crypto.randomUUID(), target_type: "host", protocol: "http", host: "", port: 80, enabled: true }],
  enabled: true,
  pass_host_header: false,
  rewrite_redirects: false,
};

function toRequest(service: Service): ServiceRequest {
  return {
    name: service.name,
    domain: service.domain,
    mode: service.mode,
    listen_port: service.listen_port ?? 0,
    targets: service.targets,
    enabled: service.enabled,
    pass_host_header: service.pass_host_header ?? false,
    rewrite_redirects: service.rewrite_redirects ?? false,
  };
}

export default function ServicesTab() {
  const { data, isLoading, error } = useServices();
  const deleteService = useDeleteService();
  const [editingId, setEditingId] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const { canCreate, canUpdate, canDelete } = usePermissionsContext();
  const canEdit = canUpdate(MODULE.services);
  const canRemove = canDelete(MODULE.services);

  const editing = data?.services?.find((s) => s.id === editingId);

  const columns: ColumnDef<Service, any>[] = [
    { accessorKey: "name", header: "Name" },
    {
      accessorKey: "domain",
      header: "Domain",
      cell: ({ getValue }) => <span className="font-mono text-xs">{getValue<string>()}</span>,
    },
    {
      accessorKey: "mode",
      header: "Mode",
      cell: ({ getValue }) => <Badge>{getValue<string>()}</Badge>,
    },
    {
      id: "status",
      header: "Status",
      cell: ({ row }) => <Badge variant={STATUS_VARIANT[row.original.meta.status] ?? "default"}>{row.original.meta.status}</Badge>,
    },
    {
      accessorKey: "enabled",
      header: "Enabled",
      cell: ({ getValue }) => (getValue<boolean>() ? "Yes" : "No"),
    },
  ];

  if (canEdit || canRemove) {
    columns.push({
      id: "actions",
      header: "",
      cell: ({ row }) => {
        const service = row.original;
        return (
          <div className="flex justify-end gap-2">
            {canEdit && (
              <Button variant="ghost" size="sm" onClick={() => setEditingId(service.id)}>
                Edit
              </Button>
            )}
            {canRemove && (
              <Button
                variant="ghost"
                size="sm"
                className="text-red-500 hover:bg-red-950 hover:text-red-400"
                disabled={deleteService.isPending}
                onClick={() => {
                  if (confirm(`Delete service "${service.name}"?`))
                    deleteService.mutate(service.id, { onError: (err) => toast.error(err.message) });
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
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-nb-gray-500">{data ? `${data.total} service${data.total === 1 ? "" : "s"}` : " "}</p>
        {canCreate(MODULE.services) && (
          <Button variant="primary" onClick={() => setShowCreate(true)}>
            <Plus size={16} /> New service
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
            <DialogTitle>New service</DialogTitle>
          </DialogHeader>
          {showCreate && <ServiceForm onDone={() => setShowCreate(false)} />}
        </DialogContent>
      </Dialog>

      {editing && (
        <Card>
          <CardHeader>
            <CardTitle>Edit {editing.name}</CardTitle>
          </CardHeader>
          <ServiceForm
            serviceId={editing.id}
            initial={toRequest(editing)}
            onDone={() => setEditingId(null)}
            onCancel={() => setEditingId(null)}
          />
        </Card>
      )}

      <DataTable
        columns={columns}
        data={data?.services ?? []}
        isLoading={isLoading}
        emptyState={<span className="text-nb-gray-500">No services yet. Create one above.</span>}
      />
    </div>
  );
}

function ServiceForm({
  serviceId,
  initial,
  onDone,
  onCancel,
}: {
  serviceId?: string;
  initial?: ServiceRequest;
  onDone: () => void;
  onCancel?: () => void;
}) {
  const create = useCreateService();
  const update = useUpdateService(serviceId ?? "");
  const mutation = serviceId ? update : create;

  const [value, setValue] = useState<ServiceRequest>(initial ?? EMPTY_REQUEST);

  const inDialog = !serviceId;
  const Body = inDialog ? DialogBody : CardContent;
  const Foot = inDialog ? DialogFooter : CardFooter;

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        mutation.mutate(value, { onSuccess: onDone, onError: (err) => toast.error(err.message) });
      }}
    >
      <Body className="flex max-h-[70vh] flex-col gap-4 overflow-y-auto">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="svc-name">Name</Label>
          <Input id="svc-name" required value={value.name} onChange={(e) => setValue({ ...value, name: e.target.value })} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="svc-domain">Domain</Label>
          <Input
            id="svc-domain"
            required
            placeholder="myapp.example.netbird.app"
            value={value.domain}
            onChange={(e) => setValue({ ...value, domain: e.target.value })}
          />
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div className="flex flex-col gap-1.5">
            <div className="flex items-center gap-1.5">
              <Label htmlFor="svc-mode">Mode</Label>
              <InfoTooltip>
                HTTP terminates and reverse-proxies web traffic (TLS, routing by domain). TCP/UDP/TLS forward raw
                traffic on a dedicated port instead — use these for non-HTTP services like databases or SSH.
              </InfoTooltip>
            </div>
            <Select value={value.mode} onValueChange={(v) => setValue({ ...value, mode: v as ServiceRequest["mode"] })}>
              <SelectTrigger id="svc-mode">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="http">HTTP (L7 reverse proxy)</SelectItem>
                <SelectItem value="tcp">TCP</SelectItem>
                <SelectItem value="udp">UDP</SelectItem>
                <SelectItem value="tls">TLS</SelectItem>
              </SelectContent>
            </Select>
          </div>
          {value.mode !== "http" && (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="svc-port">Listen port</Label>
              <Input
                id="svc-port"
                type="number"
                min={0}
                max={65535}
                value={value.listen_port}
                onChange={(e) => setValue({ ...value, listen_port: Number(e.target.value) })}
              />
              <p className="text-xs text-nb-gray-500">0 = auto-assign.</p>
            </div>
          )}
        </div>

        <ServiceTargetsInput targets={value.targets} onChange={(targets) => setValue({ ...value, targets })} />

        <label htmlFor="svc-enabled" className="flex items-center justify-between gap-4">
          <span className="text-sm text-nb-gray-100">Enabled</span>
          <Switch id="svc-enabled" checked={value.enabled} onCheckedChange={(v) => setValue({ ...value, enabled: v })} />
        </label>

        {value.mode === "http" && (
          <>
            <label htmlFor="svc-pass-host" className="flex items-center justify-between gap-4">
              <span className="text-sm text-nb-gray-100">Pass original Host header</span>
              <Switch
                id="svc-pass-host"
                checked={value.pass_host_header}
                onCheckedChange={(v) => setValue({ ...value, pass_host_header: v })}
              />
            </label>
            <label htmlFor="svc-rewrite-redirects" className="flex items-center justify-between gap-4">
              <span className="text-sm text-nb-gray-100">Rewrite redirect Location headers</span>
              <Switch
                id="svc-rewrite-redirects"
                checked={value.rewrite_redirects}
                onCheckedChange={(v) => setValue({ ...value, rewrite_redirects: v })}
              />
            </label>
          </>
        )}
      </Body>
      <Foot>
        {onCancel && (
          <Button variant="secondary" type="button" onClick={onCancel}>
            Cancel
          </Button>
        )}
        <Button variant="primary" type="submit" disabled={mutation.isPending}>
          {serviceId ? "Save" : "Create service"}
        </Button>
      </Foot>
    </form>
  );
}
