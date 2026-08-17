import type { ColumnDef } from "@tanstack/react-table";
import { Search } from "lucide-react";
import { useMemo, useState } from "react";
import type { AuditEvent } from "@/api/events";
import { useAuditEvents } from "@/api/events";
import DataTable from "@/components/table/DataTable";
import Badge from "@/components/ui/Badge";
import Input from "@/components/ui/Input";
import { absTime } from "@/lib/format";

const columns: ColumnDef<AuditEvent, any>[] = [
  {
    accessorKey: "timestamp",
    header: "Time",
    cell: ({ getValue }) => <span title={getValue<string>()}>{absTime(getValue<string>())}</span>,
  },
  {
    accessorKey: "activity",
    header: "Activity",
    cell: ({ row }) => (
      <div className="flex flex-col">
        <span className="text-nb-gray-50">{row.original.activity}</span>
        <span className="font-mono text-xs text-nb-gray-500">{row.original.activity_code}</span>
      </div>
    ),
  },
  {
    id: "initiator",
    header: "Initiator",
    cell: ({ row }) => (
      <div className="flex flex-col">
        <span>{row.original.initiator_name || row.original.initiator_id}</span>
        {row.original.initiator_email && (
          <span className="text-xs text-nb-gray-500">{row.original.initiator_email}</span>
        )}
      </div>
    ),
  },
  {
    id: "meta",
    header: "Details",
    cell: ({ row }) => {
      const entries = Object.entries(row.original.meta ?? {});
      if (entries.length === 0) return <span className="text-nb-gray-600">—</span>;
      return (
        <div className="flex flex-wrap gap-1">
          {entries.map(([k, v]) => (
            <Badge key={k}>
              {k}: {v}
            </Badge>
          ))}
        </div>
      );
    },
  },
];

export default function ActivityList() {
  const { data, isLoading, error } = useAuditEvents();
  const [query, setQuery] = useState("");

  const filtered = useMemo(() => {
    const events = data?.events ?? [];
    const q = query.trim().toLowerCase();
    if (!q) return events;
    return events.filter((e) =>
      [e.activity, e.activity_code, e.initiator_name, e.initiator_email, e.target_id]
        .join(" ")
        .toLowerCase()
        .includes(q),
    );
  }, [data, query]);

  return (
    <section className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-nb-gray-50">Activity</h1>
          <p className="text-sm text-nb-gray-500">{data ? `${data.total} events` : " "}</p>
        </div>
        <div className="w-80">
          <Input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search activity, initiator…"
            prefix={<Search size={15} />}
          />
        </div>
      </div>

      {error && (
        <div className="rounded-md border border-red-900 bg-red-950/40 px-4 py-3 text-sm text-red-300">
          {error.message}
        </div>
      )}

      <DataTable
        columns={columns}
        data={filtered}
        isLoading={isLoading}
        emptyState={<span className="text-nb-gray-500">No activity yet.</span>}
      />
    </section>
  );
}
