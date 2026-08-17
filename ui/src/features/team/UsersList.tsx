import { useState } from "react";
import { toast } from "sonner";
import { PROTECTED_GROUP_NAME, useGroups } from "@/api/groups";
import type { InviteUserRequest } from "@/api/team";
import {
  VALID_ROLES,
  useApproveUser,
  useDeleteUser,
  useInviteUser,
  useRejectUser,
  useResendInvite,
  useUsers,
} from "@/api/team";
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

const EMPTY_INVITE: InviteUserRequest = { name: "", email: "", role: "user", auto_groups: [] };

export default function UsersList() {
  const { data, isLoading, error } = useUsers();
  const deleteUser = useDeleteUser(false);
  const resendInvite = useResendInvite();
  const approveUser = useApproveUser();
  const rejectUser = useRejectUser();
  const [editingId, setEditingId] = useState<string | null>(null);
  const [showInvite, setShowInvite] = useState(false);
  const { canCreate, canUpdate, canDelete } = usePermissionsContext();
  const canEdit = canUpdate(MODULE.users);
  const canRemove = canDelete(MODULE.users);

  const users = data?.users ?? [];
  const editingUser = users.find((u) => u.id === editingId);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-nb-gray-500">{data ? `${data.total} user${data.total === 1 ? "" : "s"}` : " "}</p>
        {canCreate(MODULE.users) && (
          <Button variant="primary" onClick={() => setShowInvite(true)}>
            Invite user
          </Button>
        )}
      </div>

      {error && (
        <div className="rounded-md border border-red-900 bg-red-950/40 px-4 py-3 text-sm text-red-300">
          {error.message}
        </div>
      )}

      <Dialog open={showInvite} onOpenChange={setShowInvite}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Invite user</DialogTitle>
          </DialogHeader>
          {showInvite && <InviteUserForm onDone={() => setShowInvite(false)} />}
        </DialogContent>
      </Dialog>

      {editingUser && (
        <Card>
          <CardHeader>
            <CardTitle>Edit {editingUser.name || editingUser.email}</CardTitle>
          </CardHeader>
          <EditUserForm user={editingUser} onDone={() => setEditingId(null)} onCancel={() => setEditingId(null)} />
        </Card>
      )}

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Email</TableHead>
            <TableHead>Role</TableHead>
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
          ) : data && users.length === 0 ? (
            <TableRow>
              <TableCell colSpan={5} className="py-8 text-center text-nb-gray-500">
                No users yet. Invite one above.
              </TableCell>
            </TableRow>
          ) : (
            users.map((user) => (
              <TableRow key={user.id}>
                <TableCell>{user.name || "—"}</TableCell>
                <TableCell className="font-mono text-xs">{user.email}</TableCell>
                <TableCell className="font-mono text-xs">{user.role}</TableCell>
                <TableCell>
                  {user.pending_approval ? (
                    <Badge variant="warning">pending approval</Badge>
                  ) : user.is_blocked ? (
                    <Badge variant="danger">blocked</Badge>
                  ) : user.status === "invited" ? (
                    <Badge variant="warning">invited</Badge>
                  ) : (
                    <Badge variant="success">active</Badge>
                  )}
                </TableCell>
                <TableCell>
                  <div className="flex justify-end gap-2">
                    {user.pending_approval ? (
                      <>
                        {canEdit && (
                          <Button
                            variant="ghost"
                            size="sm"
                            disabled={approveUser.isPending}
                            onClick={() => approveUser.mutate(user.id, { onError: (err) => toast.error(err.message) })}
                          >
                            Approve
                          </Button>
                        )}
                        {canRemove && (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="text-red-500 hover:bg-red-950 hover:text-red-400"
                            disabled={rejectUser.isPending}
                            onClick={() => {
                              if (confirm(`Reject "${user.name || user.email}"?`))
                                rejectUser.mutate(user.id, { onError: (err) => toast.error(err.message) });
                            }}
                          >
                            Reject
                          </Button>
                        )}
                      </>
                    ) : (
                      <>
                        {user.status === "invited" && canEdit && (
                          <Button
                            variant="ghost"
                            size="sm"
                            disabled={resendInvite.isPending}
                            onClick={() => resendInvite.mutate(user.id, { onError: (err) => toast.error(err.message) })}
                          >
                            Resend invite
                          </Button>
                        )}
                        {canEdit && (
                          <Button variant="ghost" size="sm" onClick={() => setEditingId(user.id)}>
                            Edit
                          </Button>
                        )}
                        {!user.is_current && canRemove && (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="text-red-500 hover:bg-red-950 hover:text-red-400"
                            disabled={deleteUser.isPending}
                            onClick={() => {
                              if (confirm(`Delete user "${user.name || user.email}"?`))
                                deleteUser.mutate(user.id, { onError: (err) => toast.error(err.message) });
                            }}
                          >
                            Delete
                          </Button>
                        )}
                      </>
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

function InviteUserForm({ onDone }: { onDone: () => void }) {
  const { data: groupsData } = useGroups();
  const inviteUser = useInviteUser();
  const [value, setValue] = useState<InviteUserRequest>(EMPTY_INVITE);

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        inviteUser.mutate(value, { onSuccess: onDone, onError: (err) => toast.error(err.message) });
      }}
    >
      <DialogBody className="flex flex-col gap-4">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="invite-name">Name</Label>
          <Input id="invite-name" required value={value.name} onChange={(e) => setValue({ ...value, name: e.target.value })} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="invite-email">Email</Label>
          <Input
            id="invite-email"
            type="email"
            required
            value={value.email}
            onChange={(e) => setValue({ ...value, email: e.target.value })}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="invite-role">Role</Label>
          <Select value={value.role} onValueChange={(v) => setValue({ ...value, role: v })}>
            <SelectTrigger id="invite-role">
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
        <Button variant="primary" type="submit" disabled={inviteUser.isPending}>
          Send invite
        </Button>
      </DialogFooter>
    </form>
  );
}
