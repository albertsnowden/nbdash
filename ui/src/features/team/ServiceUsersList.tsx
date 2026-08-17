import { useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";
import { PROTECTED_GROUP_NAME, useGroups } from "@/api/groups";
import type { CreateServiceUserRequest } from "@/api/team";
import { VALID_ROLES, useCreateServiceUser, useDeleteUser, useServiceUsers } from "@/api/team";
import GroupCheckboxList from "@/components/GroupCheckboxList";
import Badge from "@/components/ui/Badge";
import Button from "@/components/ui/Button";
import { Card, CardHeader, CardTitle } from "@/components/ui/Card";
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
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/Table";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";
import EditUserForm from "./EditUserForm";

const EMPTY: CreateServiceUserRequest = { name: "", role: "user", auto_groups: [] };

export default function ServiceUsersList() {
  const { data, isLoading, error } = useServiceUsers();
  const deleteUser = useDeleteUser(true);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const { canCreate, canUpdate, canDelete } = usePermissionsContext();
  const canEdit = canUpdate(MODULE.users);
  const canRemove = canDelete(MODULE.users);

  const users = data?.users ?? [];
  const editingUser = users.find((u) => u.id === editingId);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-nb-gray-500">
          {data ? `${data.total} service user${data.total === 1 ? "" : "s"}` : " "}
        </p>
        {canCreate(MODULE.users) && (
          <Button variant="primary" onClick={() => setShowCreate(true)}>
            New service user
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
            <DialogTitle>New service user</DialogTitle>
          </DialogHeader>
          {showCreate && <CreateServiceUserForm onDone={() => setShowCreate(false)} />}
        </DialogContent>
      </Dialog>

      {editingUser && (
        <Card>
          <CardHeader>
            <CardTitle>Edit {editingUser.name}</CardTitle>
          </CardHeader>
          <EditUserForm user={editingUser} onDone={() => setEditingId(null)} onCancel={() => setEditingId(null)} />
        </Card>
      )}

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Role</TableHead>
            <TableHead>Status</TableHead>
            <TableHead />
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableRow>
              <TableCell colSpan={4} className="py-8 text-center text-nb-gray-500">
                Loading…
              </TableCell>
            </TableRow>
          ) : data && users.length === 0 ? (
            <TableRow>
              <TableCell colSpan={4} className="py-8 text-center text-nb-gray-500">
                No service users yet. Create one above.
              </TableCell>
            </TableRow>
          ) : (
            users.map((user) => (
              <TableRow key={user.id}>
                <TableCell>
                  <Link to={`/team/service-users/${user.id}`} className="font-medium text-nb-gray-50 hover:text-netbird-400">
                    {user.name}
                  </Link>
                </TableCell>
                <TableCell className="font-mono text-xs">{user.role}</TableCell>
                <TableCell>
                  <Badge variant={user.is_blocked ? "danger" : "success"}>
                    {user.is_blocked ? "blocked" : "active"}
                  </Badge>
                </TableCell>
                <TableCell>
                  <div className="flex justify-end gap-2">
                    {canEdit && (
                      <Button variant="ghost" size="sm" onClick={() => setEditingId(user.id)}>
                        Edit
                      </Button>
                    )}
                    {canRemove && (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-red-500 hover:bg-red-950 hover:text-red-400"
                        disabled={deleteUser.isPending}
                        onClick={() => {
                          if (confirm(`Delete service user "${user.name}"?`))
                            deleteUser.mutate(user.id, { onError: (err) => toast.error(err.message) });
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

function CreateServiceUserForm({ onDone }: { onDone: () => void }) {
  const { data: groupsData } = useGroups();
  const createServiceUser = useCreateServiceUser();
  const [value, setValue] = useState<CreateServiceUserRequest>(EMPTY);

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        createServiceUser.mutate(value, { onSuccess: onDone, onError: (err) => toast.error(err.message) });
      }}
    >
      <DialogBody className="flex flex-col gap-4">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="su-name">Name</Label>
          <Input id="su-name" required value={value.name} onChange={(e) => setValue({ ...value, name: e.target.value })} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="su-role">Role</Label>
          <Select value={value.role} onValueChange={(v) => setValue({ ...value, role: v })}>
            <SelectTrigger id="su-role">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {VALID_ROLES.map((role) => (
                <SelectItem key={role} value={role}>
                  {role}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Auto-assign groups</Label>
          <GroupCheckboxList
            groups={groupsData?.groups.filter((g) => g.name !== PROTECTED_GROUP_NAME) ?? []}
            selected={value.auto_groups}
            onChange={(ids) => setValue({ ...value, auto_groups: ids })}
          />
        </div>
      </DialogBody>
      <DialogFooter>
        <Button variant="primary" type="submit" disabled={createServiceUser.isPending}>
          Create service user
        </Button>
      </DialogFooter>
    </form>
  );
}
