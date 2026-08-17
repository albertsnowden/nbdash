import type { ColumnDef } from "@tanstack/react-table";
import { Search, Trash2 } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";
import { type Peer, useDeletePeer, usePeers } from "@/api/peers";
import Button from "@/components/ui/Button";
import Input from "@/components/ui/Input";
import DataTable from "@/components/table/DataTable";
import { usePermissionsContext } from "@/contexts/PermissionsContext";
import { osGlyph, relTime } from "@/lib/format";
import { MODULE } from "@/lib/modules";

const columns: ColumnDef<Peer, any>[] = [
  {
    accessorKey: "name",
    header: "Name",
    cell: ({ row }) => {
      const peer = row.original;
      return (
        <Link to={`/peers/${peer.id}`} className="flex items-center gap-3 hover:text-netbird-400">
          <span
            className={`h-2 w-2 shrink-0 rounded-full ${peer.connected ? "bg-green-500" : "bg-nb-gray-600"}`}
            title={peer.connected ? "Connected" : "Disconnected"}
          />
          <span className="flex flex-col">
            <span className="font-medium text-nb-gray-50">{peer.name}</span>
            <span className="text-xs text-nb-gray-500">{peer.hostname}</span>
          </span>
        </Link>
      );
    },
  },
  {
    accessorKey: "ip",
    header: "Address",
    cell: ({ getValue }) => <span className="font-mono text-xs">{getValue<string>()}</span>,
  },
  {
    accessorKey: "os",
    header: "OS",
    cell: ({ getValue }) => {
      const os = getValue<string>();
      return (
        <span className="inline-flex items-center gap-2">
          <span className="text-nb-gray-500">{osGlyph(os)}</span>
          {os || "unknown"}
        </span>
      );
    },
  },
  {
    accessorKey: "last_seen",
    header: "Last seen",
    cell: ({ getValue }) => {
      const val = getValue<string>();
      return <span title={val}>{relTime(val)}</span>;
    },
  },
];

export default function PeersList() {
  const [query, setQuery] = useState("");
  const { data, isLoading, isFetching, error } = usePeers(query);
  const deletePeer = useDeletePeer();
  const { canDelete } = usePermissionsContext();

  const actionColumn: ColumnDef<Peer, any> = {
    id: "actions",
    header: "",
    cell: ({ row }) => {
      const peer = row.original;
      return (
        <div className="flex justify-end">
          <Button
            variant="ghost"
            size="icon"
            className="text-red-500 hover:bg-red-950 hover:text-red-400"
            disabled={deletePeer.isPending}
            onClick={(e) => {
              e.stopPropagation();
              if (confirm(`Delete peer "${peer.name}"? It will lose access to the network immediately.`)) {
                deletePeer.mutate(peer.id, {
                  onError: (err) => toast.error(err.message),
                });
              }
            }}
            aria-label="Delete peer"
          >
            <Trash2 size={16} />
          </Button>
        </div>
      );
    },
  };

  return (
    <section className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-nb-gray-50">Peers</h1>
          <p className="text-sm text-nb-gray-500">
            {data
              ? `${data.connected_count} of ${data.total} peer${data.total === 1 ? "" : "s"} connected`
              : " "}
          </p>
        </div>

        <div className="w-80">
          <Input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search name, address, OS or group…"
            autoComplete="off"
            spellCheck={false}
            prefix={<Search size={15} />}
            suffix={isFetching ? <span className="h-3 w-3 animate-spin rounded-full border-2 border-nb-gray-600 border-t-netbird-500" /> : undefined}
          />
        </div>
      </div>

      {error && (
        <div className="rounded-md border border-red-900 bg-red-950/40 px-4 py-3 text-sm text-red-300">
          {error.message}
        </div>
      )}

      <DataTable
        columns={canDelete(MODULE.peers) ? [...columns, actionColumn] : columns}
        data={data?.peers ?? []}
        isLoading={isLoading}
        emptyState={<span className="text-nb-gray-500">No peers match your search.</span>}
      />
    </section>
  );
}
