import { toast } from "sonner";
import { useDeleteProxyCluster, useProxyClusters } from "@/api/reverseProxy";
import Badge from "@/components/ui/Badge";
import Button from "@/components/ui/Button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/Table";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { MODULE } from "@/lib/modules";

export default function ClustersTab() {
  const { data, isLoading, error } = useProxyClusters();
  const deleteCluster = useDeleteProxyCluster();
  const { canDelete } = usePermissionsContext();
  const clusters = data?.clusters ?? [];

  return (
    <div className="flex flex-col gap-4">
      <p className="text-sm text-nb-gray-500">
        {data ? `${data.total} proxy cluster${data.total === 1 ? "" : "s"} available` : " "}
      </p>

      {error && (
        <div className="rounded-md border border-red-900 bg-red-950/40 px-4 py-3 text-sm text-red-300">
          {error.message}
        </div>
      )}

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Address</TableHead>
            <TableHead>Type</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Connected proxies</TableHead>
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
          ) : data && clusters.length === 0 ? (
            <TableRow>
              <TableCell colSpan={5} className="py-8 text-center text-nb-gray-500">
                No proxy clusters available.
              </TableCell>
            </TableRow>
          ) : (
            clusters.map((cluster) => (
              <TableRow key={cluster.id}>
                <TableCell className="font-mono text-xs">{cluster.address}</TableCell>
                <TableCell>
                  <Badge variant={cluster.type === "shared" ? "brand" : "default"}>{cluster.type}</Badge>
                </TableCell>
                <TableCell>
                  <Badge variant={cluster.online ? "success" : "default"}>{cluster.online ? "online" : "offline"}</Badge>
                </TableCell>
                <TableCell className="font-mono text-xs">{cluster.connected_proxies}</TableCell>
                <TableCell>
                  {cluster.type === "account" && canDelete(MODULE.services) && (
                    <div className="flex justify-end">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-red-500 hover:bg-red-950 hover:text-red-400"
                        disabled={deleteCluster.isPending}
                        onClick={() => {
                          if (confirm(`Remove self-hosted cluster "${cluster.address}"?`))
                            deleteCluster.mutate(cluster.address, { onError: (err) => toast.error(err.message) });
                        }}
                      >
                        Remove
                      </Button>
                    </div>
                  )}
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </div>
  );
}
