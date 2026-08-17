import { Check, X } from "lucide-react";
import { usePermissions } from "@/api/permissions";
import Badge from "@/components/ui/Badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/Table";

const OPERATIONS = ["read", "create", "update", "delete"] as const;

export default function PermissionsPage() {
  const { data, isLoading, error } = usePermissions();

  if (isLoading) return <p className="text-sm text-nb-gray-500">Loading…</p>;
  if (error) {
    return (
      <div className="rounded-md border border-red-900 bg-red-950/40 px-4 py-3 text-sm text-red-300">
        {error.message}
      </div>
    );
  }
  if (!data) return null;

  const modules = Object.entries(data.permissions.modules).sort(([a], [b]) => a.localeCompare(b));

  return (
    <section className="flex flex-col gap-4">
      <p className="text-sm text-nb-gray-500">
        What your account role ({data.role}) can do in this account, resolved by the server.
      </p>

      <div className="flex items-center gap-2">
        <Badge variant="brand">{data.role}</Badge>
        {data.permissions.is_restricted && <Badge variant="warning">Restricted peers view</Badge>}
      </div>

      {modules.length === 0 ? (
        <p className="text-sm text-nb-gray-500">No per-module permission data returned for this role.</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Module</TableHead>
              {OPERATIONS.map((op) => (
                <TableHead key={op} className="text-center capitalize">
                  {op}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {modules.map(([module, ops]) => (
              <TableRow key={module}>
                <TableCell className="capitalize">{module.replace(/_/g, " ")}</TableCell>
                {OPERATIONS.map((op) => (
                  <TableCell key={op} className="text-center">
                    {ops[op] ? (
                      <Check size={16} className="mx-auto text-green-500" />
                    ) : (
                      <X size={16} className="mx-auto text-nb-gray-700" />
                    )}
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </section>
  );
}
