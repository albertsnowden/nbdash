import { useEffect, useState } from "react";
import { toast } from "sonner";
import { useJWTGroupSync, useUpdateJWTGroupSync } from "@/api/jwtGroupSync";
import Button from "@/components/ui/Button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/Card";
import Input from "@/components/ui/Input";
import Label from "@/components/ui/Label";
import Switch from "@/components/ui/Switch";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";

export default function JWTGroupSyncPage() {
  const { data, isLoading, error } = useJWTGroupSync();
  const update = useUpdateJWTGroupSync();
  const { canUpdate } = usePermissionsContext();
  const editable = canUpdate(MODULE.accounts);

  const [enabled, setEnabled] = useState(false);
  const [claimName, setClaimName] = useState("");
  const [allowGroupsText, setAllowGroupsText] = useState("");

  useEffect(() => {
    if (!data) return;
    setEnabled(data.jwt_groups_enabled);
    setClaimName(data.jwt_groups_claim_name);
    setAllowGroupsText(data.jwt_allow_groups.join(", "));
  }, [data]);

  if (isLoading) return <p className="text-sm text-nb-gray-500">Loading…</p>;
  if (error) {
    return (
      <div className="rounded-md border border-red-900 bg-red-950/40 px-4 py-3 text-sm text-red-300">
        {error.message}
      </div>
    );
  }

  return (
    <section className="flex max-w-2xl flex-col gap-4">
      <p className="text-sm text-nb-gray-500">
        Extract group names from an identity token claim and map them onto NetBird account groups.
      </p>

      <form
        onSubmit={(e) => {
          e.preventDefault();
          const jwt_allow_groups = allowGroupsText
            .split(/[,\n]+/)
            .map((s) => s.trim())
            .filter(Boolean);
          update.mutate(
            { jwt_groups_enabled: enabled, jwt_groups_claim_name: claimName, jwt_allow_groups },
            { onSuccess: () => toast.success("JWT group sync updated."), onError: (err) => toast.error(err.message) },
          );
        }}
      >
        <Card>
          <CardHeader>
            <CardTitle>Settings</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <label htmlFor="jgs-enabled" className="flex items-center justify-between gap-4">
              <span className="text-sm text-nb-gray-100">Enable JWT group sync</span>
              <Switch id="jgs-enabled" checked={enabled} disabled={!editable} onCheckedChange={setEnabled} />
            </label>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="jgs-claim">Claim name</Label>
              <Input
                id="jgs-claim"
                placeholder="e.g. roles"
                value={claimName}
                onChange={(e) => setClaimName(e.target.value)}
                disabled={!enabled || !editable}
              />
              <p className="text-xs text-nb-gray-500">
                The JWT claim NetBird reads group names from on every login.
              </p>
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="jgs-allow-groups">Allowed groups</Label>
              <Input
                id="jgs-allow-groups"
                placeholder="Administrators, Engineering"
                value={allowGroupsText}
                onChange={(e) => setAllowGroupsText(e.target.value)}
                disabled={!enabled || !editable}
              />
              <p className="text-xs text-nb-gray-500">
                Comma-separated group names from the claim. Leave empty to allow every group the claim carries.
              </p>
            </div>
          </CardContent>
          {editable && (
            <CardFooter>
              <Button variant="primary" type="submit" disabled={update.isPending}>
                Save
              </Button>
            </CardFooter>
          )}
        </Card>
      </form>
    </section>
  );
}
