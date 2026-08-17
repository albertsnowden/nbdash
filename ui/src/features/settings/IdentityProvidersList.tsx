import type { ColumnDef } from "@tanstack/react-table";
import { Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";
import type { IdentityProvider, IdentityProviderRequest, IdentityProviderType } from "@/api/identityProviders";
import {
  IDENTITY_PROVIDER_LABELS,
  IDENTITY_PROVIDER_TYPES,
  hasBuiltInIssuer,
  useCreateIdentityProvider,
  useDeleteIdentityProvider,
  useIdentityProviders,
} from "@/api/identityProviders";
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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import DataTable from "@/components/table/DataTable";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";
import { IDENTITY_PROVIDER_HELP } from "./identityProviderHelp";

const EMPTY: IdentityProviderRequest = {
  type: "oidc",
  name: "",
  issuer: "",
  client_id: "",
  client_secret: "",
};

export default function IdentityProvidersList() {
  const { data, isLoading, error } = useIdentityProviders();
  const deleteProvider = useDeleteIdentityProvider();
  const [showCreate, setShowCreate] = useState(false);
  const { canCreate, canDelete } = usePermissionsContext();

  const columns: ColumnDef<IdentityProvider, any>[] = [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <Link
          to={`/settings/identity-providers/${row.original.id}`}
          className="font-medium text-nb-gray-50 hover:text-netbird-400"
        >
          {row.original.name}
        </Link>
      ),
    },
    {
      accessorKey: "type",
      header: "Type",
      cell: ({ getValue }) => <Badge>{IDENTITY_PROVIDER_LABELS[getValue<IdentityProviderType>()]}</Badge>,
    },
    {
      accessorKey: "issuer",
      header: "Issuer",
      cell: ({ getValue }) => <span className="font-mono text-xs">{getValue<string>() || "—"}</span>,
    },
    {
      accessorKey: "client_id",
      header: "Client ID",
      cell: ({ getValue }) => <span className="font-mono text-xs">{getValue<string>()}</span>,
    },
  ];

  if (canDelete(MODULE.identityProviders)) {
    columns.push({
      id: "actions",
      header: "",
      cell: ({ row }) => {
        const provider = row.original;
        return (
          <div className="flex justify-end">
            <Button
              variant="ghost"
              size="icon"
              className="text-red-500 hover:bg-red-950 hover:text-red-400"
              disabled={deleteProvider.isPending}
              onClick={() => {
                if (
                  confirm(
                    `Delete identity provider "${provider.name}"? Anyone signed in through it will need another way to sign in.`,
                  )
                ) {
                  deleteProvider.mutate(provider.id, { onError: (err) => toast.error(err.message) });
                }
              }}
              aria-label="Delete identity provider"
            >
              <Trash2 size={16} />
            </Button>
          </div>
        );
      },
    });
  }

  return (
    <section className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-nb-gray-500">
          {data ? `${data.total} identity provider${data.total === 1 ? "" : "s"} configured` : " "}
        </p>
        {canCreate(MODULE.identityProviders) && (
          <Button variant="primary" onClick={() => setShowCreate(true)}>
            <Plus size={16} /> Add identity provider
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
            <DialogTitle>Add identity provider</DialogTitle>
          </DialogHeader>
          {showCreate && <IdentityProviderForm onDone={() => setShowCreate(false)} />}
        </DialogContent>
      </Dialog>

      <DataTable
        columns={columns}
        data={data?.providers ?? []}
        isLoading={isLoading}
        emptyState={<span className="text-nb-gray-500">No identity providers configured yet. Add one above.</span>}
      />
    </section>
  );
}

function IdentityProviderForm({ onDone }: { onDone: () => void }) {
  const createProvider = useCreateIdentityProvider();
  const [value, setValue] = useState<IdentityProviderRequest>(EMPTY);

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        createProvider.mutate(value, { onSuccess: onDone, onError: (err) => toast.error(err.message) });
      }}
    >
      <DialogBody className="flex flex-col gap-4">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="idp-name">Name</Label>
          <Input
            id="idp-name"
            required
            placeholder="e.g. Corporate SSO"
            value={value.name}
            onChange={(e) => setValue({ ...value, name: e.target.value })}
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="idp-type">Provider type</Label>
          <Select value={value.type} onValueChange={(v) => setValue({ ...value, type: v as IdentityProviderType })}>
            <SelectTrigger id="idp-type">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {IDENTITY_PROVIDER_TYPES.map((type) => (
                <SelectItem key={type} value={type}>
                  {IDENTITY_PROVIDER_LABELS[type]}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="flex flex-col gap-1.5 rounded-md border border-nb-gray-900 bg-nb-gray-940 p-3">
          <p className="text-xs font-medium text-nb-gray-200">
            Setting up {IDENTITY_PROVIDER_LABELS[value.type]}: {IDENTITY_PROVIDER_HELP[value.type].summary}
          </p>
          <ol className="list-decimal space-y-1 pl-4 text-xs text-nb-gray-400 marker:text-nb-gray-600">
            {IDENTITY_PROVIDER_HELP[value.type].steps}
          </ol>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="idp-issuer">Issuer URL</Label>
          <Input
            id="idp-issuer"
            placeholder="https://login.example.com"
            value={value.issuer}
            onChange={(e) => setValue({ ...value, issuer: e.target.value })}
          />
          <p className="text-xs text-nb-gray-500">
            {hasBuiltInIssuer(value.type)
              ? "Not needed for this provider type — it uses a fixed built-in issuer."
              : "The server validates this by fetching {issuer}/.well-known/openid-configuration, so saving can take a few seconds."}
          </p>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="idp-client-id">Client ID</Label>
          <Input
            id="idp-client-id"
            required
            value={value.client_id}
            onChange={(e) => setValue({ ...value, client_id: e.target.value })}
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="idp-client-secret">Client secret</Label>
          <Input
            id="idp-client-secret"
            type="password"
            required
            autoComplete="new-password"
            value={value.client_secret}
            onChange={(e) => setValue({ ...value, client_secret: e.target.value })}
          />
        </div>
      </DialogBody>
      <DialogFooter>
        <Button variant="primary" type="submit" disabled={createProvider.isPending}>
          Add identity provider
        </Button>
      </DialogFooter>
    </form>
  );
}
