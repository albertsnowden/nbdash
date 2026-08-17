import { useState } from "react";
import { toast } from "sonner";
import { PROTECTED_GROUP_NAME, useGroups } from "@/api/groups";
import type { User } from "@/api/team";
import { VALID_ROLES, useUpdateUser } from "@/api/team";
import GroupCheckboxList from "@/components/GroupCheckboxList";
import Button from "@/components/ui/Button";
import { CardContent, CardFooter } from "@/components/ui/Card";
import Label from "@/components/ui/Label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import Switch from "@/components/ui/Switch";

// Shared by Users and Service Users — the server accepts the exact same PUT
// body for either (see internal/api/team.go's updateUser doc comment).
export default function EditUserForm({
  user,
  onDone,
  onCancel,
}: {
  user: User;
  onDone: () => void;
  onCancel: () => void;
}) {
  const { data: groupsData } = useGroups();
  const updateUser = useUpdateUser(user.id, user.is_service_user);
  const [role, setRole] = useState(user.role);
  const [autoGroups, setAutoGroups] = useState<string[]>(user.auto_groups);
  const [isBlocked, setIsBlocked] = useState(user.is_blocked);

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        updateUser.mutate(
          { role, auto_groups: autoGroups, is_blocked: isBlocked },
          { onSuccess: onDone, onError: (err) => toast.error(err.message) },
        );
      }}
    >
      <CardContent className="flex flex-col gap-4">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="edit-role">Role</Label>
          <Select value={role} onValueChange={setRole}>
            <SelectTrigger id="edit-role">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {VALID_ROLES.map((r) => (
                <SelectItem key={r} value={r}>
                  {r}
                </SelectItem>
              ))}
              {user.role === "owner" && <SelectItem value="owner">owner</SelectItem>}
            </SelectContent>
          </Select>
        </div>
        <label htmlFor="edit-blocked" className="flex items-center justify-between gap-4">
          <span className="text-sm text-nb-gray-100">Blocked</span>
          <Switch id="edit-blocked" checked={isBlocked} onCheckedChange={setIsBlocked} />
        </label>
        <div className="flex flex-col gap-1.5">
          <Label>Auto-assign groups</Label>
          <GroupCheckboxList
            groups={groupsData?.groups.filter((g) => g.name !== PROTECTED_GROUP_NAME) ?? []}
            selected={autoGroups}
            onChange={setAutoGroups}
          />
        </div>
      </CardContent>

      <CardFooter>
        <Button variant="secondary" type="button" onClick={onCancel}>
          Cancel
        </Button>
        <Button variant="primary" type="submit" disabled={updateUser.isPending}>
          Save
        </Button>
      </CardFooter>
    </form>
  );
}
