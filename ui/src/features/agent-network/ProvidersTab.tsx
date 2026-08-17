import { Plus } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import type { AgentNetworkProvider, AgentNetworkProviderRequest } from "@/api/agentNetwork";
import {
  useAgentNetworkProviders,
  useCreateAgentNetworkProvider,
  useDeleteAgentNetworkProvider,
  useUpdateAgentNetworkProvider,
} from "@/api/agentNetwork";
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
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/Table";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";

const PROVIDER_TYPES = [
  { value: "openai_api", label: "OpenAI" },
  { value: "anthropic_api", label: "Anthropic" },
  { value: "azure_openai_api", label: "Azure OpenAI" },
  { value: "bedrock_api", label: "AWS Bedrock" },
  { value: "vertex_ai_api", label: "Google Vertex AI" },
  { value: "mistral_api", label: "Mistral" },
  { value: "custom", label: "Custom" },
];

const EMPTY: AgentNetworkProviderRequest = {
  provider_id: "openai_api",
  name: "",
  upstream_url: "",
  api_key: "",
  enabled: true,
  skip_tls_verification: false,
  metadata_disabled: false,
};

export default function ProvidersTab() {
  const { data, isLoading, error } = useAgentNetworkProviders();
  const deleteProvider = useDeleteAgentNetworkProvider();
  const [editingId, setEditingId] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const { canCreate, canUpdate, canDelete } = usePermissionsContext();
  const canEdit = canUpdate(MODULE.agentNetworkProviders);
  const canRemove = canDelete(MODULE.agentNetworkProviders);

  const providers = data?.providers ?? [];
  const editing = providers.find((p) => p.id === editingId);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-nb-gray-500">
          {data ? `${data.total} provider${data.total === 1 ? "" : "s"}` : " "}
        </p>
        {canCreate(MODULE.agentNetworkProviders) && (
          <Button variant="primary" onClick={() => setShowCreate(true)}>
            <Plus size={16} /> Connect provider
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
            <DialogTitle>Connect provider</DialogTitle>
          </DialogHeader>
          {showCreate && <ProviderForm onDone={() => setShowCreate(false)} />}
        </DialogContent>
      </Dialog>

      {editing && (
        <Card>
          <CardHeader>
            <CardTitle>Edit {editing.name}</CardTitle>
          </CardHeader>
          <ProviderForm
            providerId={editing.id}
            initial={{
              provider_id: editing.provider_id,
              name: editing.name,
              upstream_url: editing.upstream_url,
              api_key: "",
              enabled: editing.enabled,
              skip_tls_verification: editing.skip_tls_verification ?? false,
              metadata_disabled: editing.metadata_disabled ?? false,
            }}
            onDone={() => setEditingId(null)}
            onCancel={() => setEditingId(null)}
          />
        </Card>
      )}

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Type</TableHead>
            <TableHead>Upstream URL</TableHead>
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
          ) : data && providers.length === 0 ? (
            <TableRow>
              <TableCell colSpan={5} className="py-8 text-center text-nb-gray-500">
                No providers connected yet.
              </TableCell>
            </TableRow>
          ) : (
            providers.map((provider: AgentNetworkProvider) => (
              <TableRow key={provider.id}>
                <TableCell>{provider.name}</TableCell>
                <TableCell>
                  <Badge>{PROVIDER_TYPES.find((p) => p.value === provider.provider_id)?.label ?? provider.provider_id}</Badge>
                </TableCell>
                <TableCell className="font-mono text-xs">{provider.upstream_url}</TableCell>
                <TableCell>
                  <Badge variant={provider.enabled ? "success" : "default"}>
                    {provider.enabled ? "enabled" : "disabled"}
                  </Badge>
                </TableCell>
                <TableCell>
                  <div className="flex justify-end gap-2">
                    {canEdit && (
                      <Button variant="ghost" size="sm" onClick={() => setEditingId(provider.id)}>
                        Edit
                      </Button>
                    )}
                    {canRemove && (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-red-500 hover:bg-red-950 hover:text-red-400"
                        disabled={deleteProvider.isPending}
                        onClick={() => {
                          if (confirm(`Delete provider "${provider.name}"?`))
                            deleteProvider.mutate(provider.id, { onError: (err) => toast.error(err.message) });
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

function ProviderForm({
  providerId,
  initial,
  onDone,
  onCancel,
}: {
  providerId?: string;
  initial?: AgentNetworkProviderRequest;
  onDone: () => void;
  onCancel?: () => void;
}) {
  const create = useCreateAgentNetworkProvider();
  const update = useUpdateAgentNetworkProvider(providerId ?? "");
  const mutation = providerId ? update : create;

  const [value, setValue] = useState<AgentNetworkProviderRequest>(initial ?? EMPTY);

  const inDialog = !providerId;
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
          <Label htmlFor="an-name">Name</Label>
          <Input id="an-name" required value={value.name} onChange={(e) => setValue({ ...value, name: e.target.value })} />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="an-type">Provider type</Label>
          <Select
            value={value.provider_id}
            onValueChange={(v) => setValue({ ...value, provider_id: v })}
            disabled={!!providerId}
          >
            <SelectTrigger id="an-type">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {PROVIDER_TYPES.map((p) => (
                <SelectItem key={p.value} value={p.value}>
                  {p.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="flex flex-col gap-1.5">
          <div className="flex items-center gap-1.5">
            <Label htmlFor="an-url">Upstream URL</Label>
            <InfoTooltip>
              The provider's real API base URL — NetBird's gateway forwards requests here and injects the API
              key, so your peers never see it. Use the provider's standard API host unless you're pointing at a
              private/self-hosted gateway (Azure OpenAI, Bedrock, Vertex AI, or an internal LiteLLM/Portkey
              instance).
            </InfoTooltip>
          </div>
          <Input
            id="an-url"
            required
            placeholder="https://api.openai.com"
            value={value.upstream_url}
            onChange={(e) => setValue({ ...value, upstream_url: e.target.value })}
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <div className="flex items-center gap-1.5">
            <Label htmlFor="an-key">API key</Label>
            <InfoTooltip>
              Create this key in the provider's own console (e.g. platform.openai.com/api-keys for OpenAI,
              console.anthropic.com for Anthropic). It's sealed at rest on your Management server and never sent
              to peers.
            </InfoTooltip>
          </div>
          <Input
            id="an-key"
            type="password"
            required={!providerId}
            placeholder={providerId ? "Leave blank to keep the existing key" : undefined}
            autoComplete="new-password"
            value={value.api_key}
            onChange={(e) => setValue({ ...value, api_key: e.target.value })}
          />
        </div>

        <label htmlFor="an-enabled" className="flex items-center justify-between gap-4">
          <span className="text-sm text-nb-gray-100">Enabled</span>
          <Switch id="an-enabled" checked={value.enabled} onCheckedChange={(v) => setValue({ ...value, enabled: v })} />
        </label>
        <label htmlFor="an-skip-tls" className="flex items-center justify-between gap-4">
          <span className="text-sm text-nb-gray-100">Skip upstream TLS verification</span>
          <Switch
            id="an-skip-tls"
            checked={value.skip_tls_verification}
            onCheckedChange={(v) => setValue({ ...value, skip_tls_verification: v })}
          />
        </label>
        <label htmlFor="an-no-metadata" className="flex items-center justify-between gap-4">
          <span className="text-sm text-nb-gray-100">Disable identity metadata injection</span>
          <Switch
            id="an-no-metadata"
            checked={value.metadata_disabled}
            onCheckedChange={(v) => setValue({ ...value, metadata_disabled: v })}
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
          {providerId ? "Save" : "Connect provider"}
        </Button>
      </Foot>
    </form>
  );
}
