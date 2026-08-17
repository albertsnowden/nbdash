import { ArrowLeft, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { toast } from "sonner";
import { useDeletePeer, usePeer, useUpdatePeer } from "@/api/peers";
import Button from "@/components/ui/Button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/Card";
import Input from "@/components/ui/Input";
import Label from "@/components/ui/Label";
import Switch from "@/components/ui/Switch";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";

export default function PeerDetail() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const { data: peer, isLoading, error } = usePeer(id);
  const updatePeer = useUpdatePeer(id);
  const deletePeer = useDeletePeer();
  const { canUpdate, canDelete } = usePermissionsContext();
  const editable = canUpdate(MODULE.peers);

  const [name, setName] = useState("");
  const [sshEnabled, setSshEnabled] = useState(false);
  const [loginExpirationEnabled, setLoginExpirationEnabled] = useState(false);
  const [inactivityExpirationEnabled, setInactivityExpirationEnabled] = useState(false);

  // Forms here are uncontrolled-from-the-server, controlled-from-the-user:
  // seed local state once the fetch resolves, then the user's edits own it.
  useEffect(() => {
    if (!peer) return;
    setName(peer.name);
    setSshEnabled(peer.ssh_enabled);
    setLoginExpirationEnabled(peer.login_expiration_enabled);
    setInactivityExpirationEnabled(peer.inactivity_expiration_enabled);
  }, [peer]);

  if (isLoading) {
    return <p className="text-sm text-nb-gray-500">Loading…</p>;
  }
  if (error) {
    return (
      <div className="rounded-md border border-red-900 bg-red-950/40 px-4 py-3 text-sm text-red-300">
        {error.message}
      </div>
    );
  }
  if (!peer) return null;

  return (
    <section className="flex max-w-2xl flex-col gap-4">
      <Link to="/peers" className="inline-flex items-center gap-1 text-sm text-nb-gray-500 hover:text-nb-gray-200">
        <ArrowLeft size={14} /> All peers
      </Link>

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-nb-gray-50">{peer.name}</h1>
          <p className="font-mono text-sm text-nb-gray-500">{peer.ip}</p>
        </div>
        {canDelete(MODULE.peers) && (
          <Button
            variant="danger"
            disabled={deletePeer.isPending}
            onClick={() => {
              if (confirm(`Delete peer "${peer.name}"? It will lose access to the network immediately.`)) {
                deletePeer.mutate(id, {
                  onSuccess: () => navigate("/peers"),
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
          updatePeer.mutate(
            {
              name,
              ssh_enabled: sshEnabled,
              login_expiration_enabled: loginExpirationEnabled,
              inactivity_expiration_enabled: inactivityExpirationEnabled,
            },
            {
              onSuccess: () => toast.success("Peer updated."),
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
              <Label htmlFor="peer-name">Name</Label>
              <Input
                id="peer-name"
                required
                disabled={!editable}
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>

            <label htmlFor="peer-ssh" className="flex items-center justify-between gap-4">
              <span className="text-sm text-nb-gray-100">SSH access enabled</span>
              <Switch
                id="peer-ssh"
                checked={sshEnabled}
                disabled={!editable}
                onCheckedChange={setSshEnabled}
              />
            </label>
            <label htmlFor="peer-login-exp" className="flex items-center justify-between gap-4">
              <span className="text-sm text-nb-gray-100">Login expiration enabled</span>
              <Switch
                id="peer-login-exp"
                checked={loginExpirationEnabled}
                disabled={!editable}
                onCheckedChange={setLoginExpirationEnabled}
              />
            </label>
            <label htmlFor="peer-inactivity-exp" className="flex items-center justify-between gap-4">
              <span className="text-sm text-nb-gray-100">Inactivity expiration enabled</span>
              <Switch
                id="peer-inactivity-exp"
                checked={inactivityExpirationEnabled}
                disabled={!editable}
                onCheckedChange={setInactivityExpirationEnabled}
              />
            </label>
          </CardContent>
          {editable && (
            <CardFooter>
              <Button variant="primary" type="submit" disabled={updatePeer.isPending}>
                Save
              </Button>
            </CardFooter>
          )}
        </Card>
      </form>

      <Card>
        <CardContent className="grid grid-cols-2 gap-x-6 gap-y-3 pt-5 text-sm">
          <span className="text-nb-gray-500">Hostname</span>
          <span className="font-mono text-nb-gray-100">{peer.hostname}</span>
          <span className="text-nb-gray-500">OS</span>
          <span className="text-nb-gray-100">{peer.os || "unknown"}</span>
          <span className="text-nb-gray-500">Version</span>
          <span className="text-nb-gray-100">{peer.version || "—"}</span>
          <span className="text-nb-gray-500">Connected</span>
          <span className="text-nb-gray-100">{peer.connected ? "Yes" : "No"}</span>
        </CardContent>
      </Card>
    </section>
  );
}
