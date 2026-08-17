import { ArrowLeft, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { toast } from "sonner";
import { useGroups } from "@/api/groups";
import { useDeleteNetwork, useNetwork, useUpdateNetwork } from "@/api/networks";
import Button from "@/components/ui/Button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/Card";
import Input from "@/components/ui/Input";
import Label from "@/components/ui/Label";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";
import ResourcesPanel from "./ResourcesPanel";
import RoutersPanel from "./RoutersPanel";

export default function NetworkDetail() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const { data: network, isLoading, error } = useNetwork(id);
  const { data: groupsData } = useGroups();
  const updateNetwork = useUpdateNetwork(id);
  const deleteNetwork = useDeleteNetwork();
  const { canUpdate, canDelete } = usePermissionsContext();
  const editable = canUpdate(MODULE.networks);

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");

  useEffect(() => {
    if (!network) return;
    setName(network.name);
    setDescription(network.description);
  }, [network]);

  if (isLoading) return <p className="text-sm text-nb-gray-500">Loading…</p>;
  if (error) {
    return (
      <div className="rounded-md border border-red-900 bg-red-950/40 px-4 py-3 text-sm text-red-300">
        {error.message}
      </div>
    );
  }
  if (!network) return null;

  return (
    <section className="flex max-w-3xl flex-col gap-6">
      <Link to="/networks" className="inline-flex items-center gap-1 text-sm text-nb-gray-500 hover:text-nb-gray-200">
        <ArrowLeft size={14} /> All networks
      </Link>

      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold text-nb-gray-50">{network.name}</h1>
        {canDelete(MODULE.networks) && (
          <Button
            variant="danger"
            disabled={deleteNetwork.isPending}
            onClick={() => {
              if (confirm(`Delete network "${network.name}"? Its resources and routers go with it.`)) {
                deleteNetwork.mutate(id, {
                  onSuccess: () => navigate("/networks"),
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
          updateNetwork.mutate(
            { name, description },
            { onSuccess: () => toast.success("Network updated."), onError: (err) => toast.error(err.message) },
          );
        }}
      >
        <Card>
          <CardHeader>
            <CardTitle>Settings</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="network-name">Name</Label>
              <Input
                id="network-name"
                required
                disabled={!editable}
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="network-description">Description</Label>
              <Input
                id="network-description"
                disabled={!editable}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            </div>
          </CardContent>
          {editable && (
            <CardFooter>
              <Button variant="primary" type="submit" disabled={updateNetwork.isPending}>
                Save
              </Button>
            </CardFooter>
          )}
        </Card>
      </form>

      <ResourcesPanel networkId={id} groups={groupsData?.groups ?? []} />
      <RoutersPanel networkId={id} groups={groupsData?.groups ?? []} />
    </section>
  );
}
