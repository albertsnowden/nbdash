import { ArrowLeft, Copy, Plus } from "lucide-react";
import { useState } from "react";
import { Link, useParams } from "react-router";
import { toast } from "sonner";
import type { PATGenerated } from "@/api/team";
import { useCreatePAT, useDeletePAT, usePATs } from "@/api/team";
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

export default function ServiceUserDetail() {
  const { id = "" } = useParams();
  const { data, isLoading, error } = usePATs(id);
  const deletePAT = useDeletePAT(id);
  const [showCreate, setShowCreate] = useState(false);
  const [reveal, setReveal] = useState<PATGenerated | null>(null);
  const { canCreate, canDelete } = usePermissionsContext();
  const tokens = data?.tokens ?? [];

  return (
    <section className="flex flex-col gap-4">
      <Link
        to="/team/service-users"
        className="inline-flex items-center gap-1 text-sm text-nb-gray-500 hover:text-nb-gray-200"
      >
        <ArrowLeft size={14} /> All service users
      </Link>

      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold text-nb-gray-50">Personal access tokens</h1>
        {canCreate(MODULE.pats) && (
          <Button variant="primary" onClick={() => setShowCreate(true)}>
            <Plus size={16} /> New token
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
            <DialogTitle>Token created</DialogTitle>
          </DialogHeader>
          {reveal && <PlaintextReveal setupKey={reveal} onDone={() => setReveal(null)} />}
        </DialogContent>
      </Dialog>

      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New token</DialogTitle>
          </DialogHeader>
          {showCreate && (
            <CreateTokenForm
              userId={id}
              onDone={(generated) => {
                setShowCreate(false);
                setReveal(generated);
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
          ) : data && tokens.length === 0 ? (
            <TableRow>
              <TableCell colSpan={5} className="py-8 text-center text-nb-gray-500">
                No tokens yet. Create one above.
              </TableCell>
            </TableRow>
          ) : (
            tokens.map((tok) => (
              <TableRow key={tok.id}>
                <TableCell>{tok.name}</TableCell>
                <TableCell>{absTime(tok.created_at)}</TableCell>
                <TableCell>{absTime(tok.expiration_date)}</TableCell>
                <TableCell>{absTime(tok.last_used)}</TableCell>
                <TableCell>
                  {canDelete(MODULE.pats) && (
                    <div className="flex justify-end">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-red-500 hover:bg-red-950 hover:text-red-400"
                        disabled={deletePAT.isPending}
                        onClick={() => {
                          if (confirm(`Delete token "${tok.name}"?`))
                            deletePAT.mutate(tok.id, { onError: (err) => toast.error(err.message) });
                        }}
                      >
                        Delete
                      </Button>
                    </div>
                  )}
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </section>
  );
}

function PlaintextReveal({ setupKey, onDone }: { setupKey: PATGenerated; onDone: () => void }) {
  return (
    <>
      <DialogBody className="flex flex-col gap-4">
        <div className="rounded-md border border-yellow-900 bg-yellow-950/30 px-3 py-2 text-sm text-yellow-300">
          This is the only time this token's value is shown — copy it now.
        </div>
        <div className="flex items-center gap-2">
          <Input className="font-mono" readOnly value={setupKey.plain_token} onFocus={(e) => e.target.select()} />
          <Button
            variant="secondary"
            type="button"
            onClick={() => void navigator.clipboard.writeText(setupKey.plain_token)}
          >
            <Copy size={14} /> Copy
          </Button>
        </div>
      </DialogBody>
      <DialogFooter>
        <Button variant="primary" type="button" onClick={onDone}>
          Done
        </Button>
      </DialogFooter>
    </>
  );
}

function CreateTokenForm({ userId, onDone }: { userId: string; onDone: (generated: PATGenerated) => void }) {
  const createPAT = useCreatePAT(userId);
  const [name, setName] = useState("");
  const [expiresIn, setExpiresIn] = useState(90);

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        createPAT.mutate(
          { name, expires_in: expiresIn },
          { onSuccess: onDone, onError: (err) => toast.error(err.message) },
        );
      }}
    >
      <DialogBody className="flex flex-col gap-4">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="pat-name">Name</Label>
          <Input id="pat-name" required value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="pat-expires">Expires in (days)</Label>
          <Input
            id="pat-expires"
            type="number"
            min={1}
            max={365}
            required
            value={expiresIn}
            onChange={(e) => setExpiresIn(Number(e.target.value))}
          />
        </div>
      </DialogBody>
      <DialogFooter>
        <Button variant="primary" type="submit" disabled={createPAT.isPending}>
          Create token
        </Button>
      </DialogFooter>
    </form>
  );
}
