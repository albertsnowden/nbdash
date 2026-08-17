import { ArrowLeft, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { toast } from "sonner";
import {
  IDENTITY_PROVIDER_LABELS,
  hasBuiltInIssuer,
  useDeleteIdentityProvider,
  useIdentityProvider,
  useUpdateIdentityProvider,
} from "@/api/identityProviders";
import Button from "@/components/ui/Button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/Card";
import Input from "@/components/ui/Input";
import Label from "@/components/ui/Label";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";
import { IDENTITY_PROVIDER_HELP } from "./identityProviderHelp";

export default function IdentityProviderDetail() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const { data: provider, isLoading, error } = useIdentityProvider(id);
  const updateProvider = useUpdateIdentityProvider(id);
  const deleteProvider = useDeleteIdentityProvider();
  const { canUpdate, canDelete } = usePermissionsContext();
  const editable = canUpdate(MODULE.identityProviders);

  const [name, setName] = useState("");
  const [issuer, setIssuer] = useState("");
  const [clientID, setClientID] = useState("");
  const [clientSecret, setClientSecret] = useState("");

  useEffect(() => {
    if (!provider) return;
    setName(provider.name);
    setIssuer(provider.issuer);
    setClientID(provider.client_id);
  }, [provider]);

  if (isLoading) return <p className="text-sm text-nb-gray-500">Loading…</p>;
  if (error) {
    return (
      <div className="rounded-md border border-red-900 bg-red-950/40 px-4 py-3 text-sm text-red-300">
        {error.message}
      </div>
    );
  }
  if (!provider) return null;

  return (
    <section className="flex max-w-2xl flex-col gap-4">
      <Link
        to="/settings/identity-providers"
        className="inline-flex items-center gap-1 text-sm text-nb-gray-500 hover:text-nb-gray-200"
      >
        <ArrowLeft size={14} /> All identity providers
      </Link>

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-nb-gray-50">{provider.name}</h1>
          <p className="text-sm text-nb-gray-500">{IDENTITY_PROVIDER_LABELS[provider.type]}</p>
        </div>
        {canDelete(MODULE.identityProviders) && (
          <Button
            variant="danger"
            disabled={deleteProvider.isPending}
            onClick={() => {
              if (
                confirm(
                  `Delete identity provider "${provider.name}"? Anyone signed in through it will need another way to sign in.`,
                )
              ) {
                deleteProvider.mutate(id, {
                  onSuccess: () => navigate("/settings/identity-providers"),
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
          updateProvider.mutate(
            { type: provider.type, name, issuer, client_id: clientID, client_secret: clientSecret },
            {
              onSuccess: () => toast.success("Identity provider updated."),
              onError: (err) => toast.error(err.message),
            },
          );
        }}
      >
        <Card>
          <CardHeader>
            <CardTitle>Settings</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="idp-name">Name</Label>
              <Input
                id="idp-name"
                required
                disabled={!editable}
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <Label>Provider type</Label>
              <Input readOnly value={IDENTITY_PROVIDER_LABELS[provider.type]} />
              <p className="text-xs text-nb-gray-500">Provider type cannot be changed after creation.</p>
            </div>

            <div className="flex flex-col gap-1.5 rounded-md border border-nb-gray-900 bg-nb-gray-950 p-3">
              <p className="text-xs font-medium text-nb-gray-200">
                {IDENTITY_PROVIDER_HELP[provider.type].summary}
              </p>
              <ol className="list-decimal space-y-1 pl-4 text-xs text-nb-gray-400 marker:text-nb-gray-600">
                {IDENTITY_PROVIDER_HELP[provider.type].steps}
              </ol>
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="idp-issuer">Issuer URL</Label>
              <Input id="idp-issuer" disabled={!editable} value={issuer} onChange={(e) => setIssuer(e.target.value)} />
              {hasBuiltInIssuer(provider.type) && (
                <p className="text-xs text-nb-gray-500">
                  Not needed for this provider type — it uses a fixed built-in issuer.
                </p>
              )}
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="idp-client-id">Client ID</Label>
              <Input
                id="idp-client-id"
                required
                disabled={!editable}
                value={clientID}
                onChange={(e) => setClientID(e.target.value)}
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="idp-client-secret">Client secret</Label>
              <Input
                id="idp-client-secret"
                type="password"
                placeholder="••••••••"
                autoComplete="new-password"
                disabled={!editable}
                value={clientSecret}
                onChange={(e) => setClientSecret(e.target.value)}
              />
              <p className="text-xs text-nb-gray-500">Leave blank to keep the existing secret.</p>
            </div>
          </CardContent>
          {editable && (
            <CardFooter>
              <Button variant="primary" type="submit" disabled={updateProvider.isPending}>
                Save
              </Button>
            </CardFooter>
          )}
        </Card>
      </form>
    </section>
  );
}
