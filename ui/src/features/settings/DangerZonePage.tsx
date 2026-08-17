import { AlertTriangle } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { useCurrentAccount, useDeleteAccount } from "@/api/account";
import Button from "@/components/ui/Button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/Card";
import Input from "@/components/ui/Input";
import Label from "@/components/ui/Label";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";

// The Management API's session cookie stops being usable the instant the
// account is gone, so a plain client-side redirect isn't enough — this
// submits the same POST-only /logout form the topbar's sign-out button
// uses (see ui/src/layouts/Header.tsx), which is what actually clears the
// session and lands back on the login page.
function submitLogout() {
  const form = document.createElement("form");
  form.method = "post";
  form.action = "/logout";
  document.body.appendChild(form);
  form.submit();
}

export default function DangerZonePage() {
  const { data: account } = useCurrentAccount();
  const deleteAccount = useDeleteAccount();
  const [confirmText, setConfirmText] = useState("");
  const { canDelete: hasDeletePermission } = usePermissionsContext();

  const expected = account?.domain || account?.id || "";
  const confirmed = expected !== "" && confirmText === expected;

  if (!hasDeletePermission(MODULE.accounts)) {
    return (
      <section className="flex max-w-2xl flex-col gap-4">
        <p className="text-sm text-nb-gray-500">Irreversible actions for this account.</p>
        <p className="text-sm text-nb-gray-500">Your role doesn't have permission to delete this account.</p>
      </section>
    );
  }

  return (
    <section className="flex max-w-2xl flex-col gap-4">
      <p className="text-sm text-nb-gray-500">Irreversible actions for this account.</p>

      <Card className="border-red-900">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-red-400">
            <AlertTriangle size={18} /> Delete this account
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <p className="text-sm text-nb-gray-300">
            Permanently deletes this account and every peer, group, policy, network, DNS zone, and user in it.
            This cannot be undone.
          </p>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="confirm-delete">
              Type <span className="font-mono text-nb-gray-100">{expected || "…"}</span> to confirm
            </Label>
            <Input
              id="confirm-delete"
              value={confirmText}
              onChange={(e) => setConfirmText(e.target.value)}
              autoComplete="off"
              disabled={!expected}
            />
          </div>
        </CardContent>
        <CardFooter>
          <Button
            variant="danger"
            disabled={!confirmed || deleteAccount.isPending}
            onClick={() => {
              deleteAccount.mutate(undefined, {
                onSuccess: submitLogout,
                onError: (err) => toast.error(err.message),
              });
            }}
          >
            Delete account permanently
          </Button>
        </CardFooter>
      </Card>
    </section>
  );
}
