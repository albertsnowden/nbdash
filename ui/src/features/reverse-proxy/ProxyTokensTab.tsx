import { Copy, Plus } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import type { ProxyTokenCreated, ProxyTokenRequest } from "@/api/reverseProxy";
import { useCreateProxyToken, useDeleteProxyToken, useProxyTokens } from "@/api/reverseProxy";
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
import { absTime } from "@/lib/format";
import { MODULE } from "@/lib/modules";

export default function ProxyTokensTab() {
  const { data, isLoading, error } = useProxyTokens();
  const deleteToken = useDeleteProxyToken();
  const [showCreate, setShowCreate] = useState(false);
  const [reveal, setReveal] = useState<ProxyTokenCreated | null>(null);
  const { canCreate, canDelete } = usePermissionsContext();
  const canRevoke = canDelete(MODULE.services);
  const tokens = data?.tokens ?? [];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-nb-gray-500">
          {data ? `${data.total} proxy token${data.total === 1 ? "" : "s"}` : " "}
        </p>
        {canCreate(MODULE.services) && (
          <Button variant="primary" onClick={() => setShowCreate(true)}>
            <Plus size={16} /> New proxy token
          </Button>
        )}
      </div>

      {error && (
        <div className="rounded-md border border-red-900 bg-red-950/40 px-4 py-3 text-sm text-red-300">
          {error.message}
        </div>
      )}

      <Dialog open={!!reveal} onOpenChange={(open) => !open && setReveal(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Proxy token created</DialogTitle>
          </DialogHeader>
          {reveal && (
            <>
              <DialogBody className="flex flex-col gap-4">
                <div className="rounded-md border border-yellow-900 bg-yellow-950/30 px-3 py-2 text-sm text-yellow-300">
                  This is the only time this token's value is shown — copy it now. Use it to register a
                  self-hosted proxy (`netbird proxy`) with this account.
                </div>
                <div className="flex items-center gap-2">
                  <Input
                    className="font-mono"
                    readOnly
                    value={reveal.plain_token}
                    onFocus={(e) => e.target.select()}
                  />
                  <Button
                    variant="secondary"
                    type="button"
                    onClick={() => void navigator.clipboard.writeText(reveal.plain_token)}
                  >
                    <Copy size={14} /> Copy
                  </Button>
                </div>
              </DialogBody>
              <DialogFooter>
                <Button variant="primary" type="button" onClick={() => setReveal(null)}>
                  Done
                </Button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>

      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New proxy token</DialogTitle>
          </DialogHeader>
          {showCreate && (
            <CreateTokenForm
              onDone={(created) => {
                setShowCreate(false);
                setReveal(created);
              }}
            />
          )}
        </DialogContent>
      </Dialog>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Created</TableHead>
            <TableHead>Expires</TableHead>
            <TableHead>Last used</TableHead>
            <TableHead>Status</TableHead>
            <TableHead />
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableRow>
              <TableCell colSpan={6} className="py-8 text-center text-nb-gray-500">
                Loading…
              </TableCell>
            </TableRow>
          ) : data && tokens.length === 0 ? (
            <TableRow>
              <TableCell colSpan={6} className="py-8 text-center text-nb-gray-500">
                No proxy tokens yet.
              </TableCell>
            </TableRow>
          ) : (
            tokens.map((tok) => (
              <TableRow key={tok.id}>
                <TableCell>{tok.name}</TableCell>
                <TableCell>{absTime(tok.created_at)}</TableCell>
                <TableCell>{tok.expires_at ? absTime(tok.expires_at) : "Never"}</TableCell>
                <TableCell>{tok.last_used ? absTime(tok.last_used) : "—"}</TableCell>
                <TableCell>
                  <Badge variant={tok.revoked ? "danger" : "success"}>{tok.revoked ? "revoked" : "active"}</Badge>
                </TableCell>
                <TableCell>
                  {canRevoke && (
                    <div className="flex justify-end">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-red-500 hover:bg-red-950 hover:text-red-400"
                        disabled={deleteToken.isPending || tok.revoked}
                        onClick={() => {
                          if (confirm(`Revoke proxy token "${tok.name}"?`))
                            deleteToken.mutate(tok.id, { onError: (err) => toast.error(err.message) });
                        }}
                      >
                        Revoke
                      </Button>
                    </div>
                  )}
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </div>
  );
}

function CreateTokenForm({ onDone }: { onDone: (created: ProxyTokenCreated) => void }) {
  const createToken = useCreateProxyToken();
  const [name, setName] = useState("");
  const [expiresInDays, setExpiresInDays] = useState(0);

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        const req: ProxyTokenRequest = { name, expires_in: expiresInDays * 86400 };
        createToken.mutate(req, { onSuccess: onDone, onError: (err) => toast.error(err.message) });
      }}
    >
      <DialogBody className="flex flex-col gap-4">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="pt-name">Name</Label>
          <Input id="pt-name" required value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="pt-expires">Expires in (days)</Label>
          <Input
            id="pt-expires"
            type="number"
            min={0}
            value={expiresInDays}
            onChange={(e) => setExpiresInDays(Number(e.target.value))}
          />
          <p className="text-xs text-nb-gray-500">0 means never expires.</p>
        </div>
      </DialogBody>
      <DialogFooter>
        <Button variant="primary" type="submit" disabled={createToken.isPending}>
          Create token
        </Button>
      </DialogFooter>
    </form>
  );
}
