import { ArrowLeft, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { toast } from "sonner";
import { PROTECTED_GROUP_NAME, useDeleteGroup, useGroup, useUpdateGroup } from "@/api/groups";
import { usePeers } from "@/api/peers";
import PeerCheckboxList from "@/components/PeerCheckboxList";
import Button from "@/components/ui/Button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/Card";
import Input from "@/components/ui/Input";
import Label from "@/components/ui/Label";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";

export default function GroupDetail() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const { data: group, isLoading, error } = useGroup(id);
  const { data: peersData } = usePeers("");
  const updateGroup = useUpdateGroup(id);
  const deleteGroup = useDeleteGroup();
  const { canUpdate, canDelete } = usePermissionsContext();
  const editable = canUpdate(MODULE.groups);

  const [name, setName] = useState("");
  const [selected, setSelected] = useState<string[]>([]);

  useEffect(() => {
    if (!group) return;
    setName(group.name);
    // group.peers comes back as JSON `null` (not `[]`) for a group with zero
    // peers — the server only appends to it, never initialises it — so this
    // must not assume an array.
    setSelected((group.peers ?? []).map((p) => p.id));
  }, [group]);

  if (isLoading) return <p className="text-sm text-nb-gray-500">Loading…</p>;
  if (error) {
    return (
      <div className="rounded-md border border-red-900 bg-red-950/40 px-4 py-3 text-sm text-red-300">
        {error.message}
      </div>
    );
  }
  if (!group) return null;

  const isProtected = group.name === PROTECTED_GROUP_NAME;

  return (
    <section className="flex max-w-2xl flex-col gap-4">
      <Link to="/groups" className="inline-flex items-center gap-1 text-sm text-nb-gray-500 hover:text-nb-gray-200">
        <ArrowLeft size={14} /> All groups
      </Link>

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-nb-gray-50">{group.name}</h1>
          <p className="text-sm text-nb-gray-500">{group.peers_count} peers</p>
        </div>
        {!isProtected && canDelete(MODULE.groups) && (
          <Button
            variant="danger"
            disabled={deleteGroup.isPending}
            onClick={() => {
              if (confirm(`Delete group "${group.name}"?`)) {
                deleteGroup.mutate(id, {
                  onSuccess: () => navigate("/groups"),
                  onError: (err) => toast.error(err.message),
                });
              }
            }}
          >
            <Trash2 size={14} /> Delete
          </Button>
        )}
      </div>

      {isProtected ? (
        <div className="rounded-md border border-green-900 bg-green-950/30 px-4 py-3 text-sm text-green-300">
          “All” is the built-in group every peer joins automatically — it can’t be renamed, have
          its membership edited, or be deleted.
        </div>
      ) : (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            updateGroup.mutate(
              { name, peers: selected },
              {
                onSuccess: () => toast.success("Group updated."),
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
                <Label htmlFor="group-name">Name</Label>
                <Input
                  id="group-name"
                  required
                  disabled={!editable}
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </div>

              <div className="flex flex-col gap-1.5">
                <Label>Peers</Label>
                <PeerCheckboxList peers={peersData?.peers ?? []} selected={selected} onChange={setSelected} />
              </div>
            </CardContent>
            {editable && (
              <CardFooter>
                <Button variant="primary" type="submit" disabled={updateGroup.isPending}>
                  Save
                </Button>
              </CardFooter>
            )}
          </Card>
        </form>
      )}
    </section>
  );
}
