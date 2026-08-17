import { useEffect, useState } from "react";
import { toast } from "sonner";
import { useDNSSettings, useUpdateDNSSettings } from "@/api/dns";
import { useGroups } from "@/api/groups";
import GroupCheckboxList from "@/components/GroupCheckboxList";
import Button from "@/components/ui/Button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/Card";
import Label from "@/components/ui/Label";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";

export default function DNSSettingsPage() {
  const { data, isLoading, error } = useDNSSettings();
  const { data: groupsData } = useGroups();
  const update = useUpdateDNSSettings();
  const [disabledGroups, setDisabledGroups] = useState<string[]>([]);
  const { canUpdate } = usePermissionsContext();
  const editable = canUpdate(MODULE.dns);

  useEffect(() => {
    if (data) setDisabledGroups(data.disabled_management_groups);
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
    <form
      onSubmit={(e) => {
        e.preventDefault();
        update.mutate(
          { disabled_management_groups: disabledGroups },
          { onSuccess: () => toast.success("DNS settings updated."), onError: (err) => toast.error(err.message) },
        );
      }}
      className="max-w-2xl"
    >
      <Card>
        <CardHeader>
          <CardTitle>Management</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col gap-1.5">
            <Label>Groups with DNS management disabled</Label>
            <p className="text-xs text-nb-gray-500">
              Peers in these groups skip NetBird's own DNS management — their OS resolver settings are left
              untouched.
            </p>
            <GroupCheckboxList
              groups={groupsData?.groups ?? []}
              selected={disabledGroups}
              onChange={editable ? setDisabledGroups : () => {}}
            />
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
  );
}
