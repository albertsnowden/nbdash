import { CheckCircle2, Plus, RefreshCw } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import type { ReverseProxyDomainRequest } from "@/api/reverseProxy";
import {
  useCreateReverseProxyDomain,
  useDeleteReverseProxyDomain,
  useReverseProxyDomains,
  useValidateReverseProxyDomain,
} from "@/api/reverseProxy";
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
import Input from "@/components/ui/Input";
import Label from "@/components/ui/Label";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/Table";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";

export default function DomainsTab() {
  const { data, isLoading, error } = useReverseProxyDomains();
  const deleteDomain = useDeleteReverseProxyDomain();
  const validateDomain = useValidateReverseProxyDomain();
  const [showCreate, setShowCreate] = useState(false);
  const { canCreate, canUpdate, canDelete } = usePermissionsContext();
  const canValidate = canUpdate(MODULE.services);
  const canRemove = canDelete(MODULE.services);
  const domains = data?.domains ?? [];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-nb-gray-500">
          {data ? `${data.total} domain${data.total === 1 ? "" : "s"}` : " "}
        </p>
        {canCreate(MODULE.services) && (
          <Button variant="primary" onClick={() => setShowCreate(true)}>
            <Plus size={16} /> Add custom domain
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
            <DialogTitle>Add custom domain</DialogTitle>
          </DialogHeader>
          {showCreate && <CreateDomainForm onDone={() => setShowCreate(false)} />}
        </DialogContent>
      </Dialog>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Domain</TableHead>
            <TableHead>Type</TableHead>
            <TableHead>Target cluster</TableHead>
            <TableHead>Validated</TableHead>
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
          ) : data && domains.length === 0 ? (
            <TableRow>
              <TableCell colSpan={5} className="py-8 text-center text-nb-gray-500">
                No custom domains yet.
              </TableCell>
            </TableRow>
          ) : (
            domains.map((domain) => (
              <TableRow key={domain.id}>
                <TableCell className="font-mono text-xs">{domain.domain}</TableCell>
                <TableCell>
                  <Badge>{domain.type}</Badge>
                </TableCell>
                <TableCell className="font-mono text-xs">{domain.target_cluster || "—"}</TableCell>
                <TableCell>
                  {domain.validated ? (
                    <Badge variant="success">
                      <CheckCircle2 size={12} /> Validated
                    </Badge>
                  ) : (
                    <Badge variant="warning">Pending</Badge>
                  )}
                </TableCell>
                <TableCell>
                  <div className="flex justify-end gap-2">
                    {!domain.validated && domain.type === "custom" && canValidate && (
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={validateDomain.isPending}
                        onClick={() =>
                          validateDomain.mutate(domain.id, {
                            onSuccess: () => toast.success("Validation triggered."),
                            onError: (err) => toast.error(err.message),
                          })
                        }
                      >
                        <RefreshCw size={14} /> Validate
                      </Button>
                    )}
                    {domain.type === "custom" && canRemove && (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-red-500 hover:bg-red-950 hover:text-red-400"
                        disabled={deleteDomain.isPending}
                        onClick={() => {
                          if (confirm(`Delete domain "${domain.domain}"?`))
                            deleteDomain.mutate(domain.id, { onError: (err) => toast.error(err.message) });
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

function CreateDomainForm({ onDone }: { onDone: () => void }) {
  const createDomain = useCreateReverseProxyDomain();
  const [value, setValue] = useState<ReverseProxyDomainRequest>({ domain: "", target_cluster: "" });

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        createDomain.mutate(value, { onSuccess: onDone, onError: (err) => toast.error(err.message) });
      }}
    >
      <DialogBody className="flex flex-col gap-4">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="domain-name">Domain</Label>
          <Input
            id="domain-name"
            required
            placeholder="myapp.example.com"
            value={value.domain}
            onChange={(e) => setValue({ ...value, domain: e.target.value })}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="domain-cluster">Target proxy cluster</Label>
          <Input
            id="domain-cluster"
            required
            placeholder="eu.proxy.netbird.io"
            value={value.target_cluster}
            onChange={(e) => setValue({ ...value, target_cluster: e.target.value })}
          />
          <p className="text-xs text-nb-gray-500">
            See the Clusters tab for available cluster addresses.
          </p>
        </div>
      </DialogBody>
      <DialogFooter>
        <Button variant="primary" type="submit" disabled={createDomain.isPending}>
          Add domain
        </Button>
      </DialogFooter>
    </form>
  );
}
