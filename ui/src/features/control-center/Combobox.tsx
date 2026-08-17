import { ChevronsUpDown, Search } from "lucide-react";
import { type ReactNode, useMemo, useState } from "react";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/Popover";
import { cn } from "@/lib/utils";

export interface ComboboxOption {
  value: string;
  label: string;
}

// The anchor picker for every Control Center view (group/peer/user) — a
// searchable dropdown whose trigger renders a caller-supplied summary
// (e.g. a DeviceCard) instead of plain text. Radix's own Select doesn't
// support a free-form trigger + in-list search, so this wraps Popover
// instead, matching the shape (not the code) of the reference dashboard's
// combobox-as-graph-node pattern.
export default function Combobox({
  value,
  onChange,
  options,
  placeholder,
  trigger,
  className,
}: {
  value: string;
  onChange: (value: string) => void;
  options: ComboboxOption[];
  placeholder?: string;
  trigger: ReactNode;
  className?: string;
}) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return options;
    return options.filter((o) => o.label.toLowerCase().includes(q));
  }, [options, query]);

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) setQuery("");
      }}
    >
      <PopoverTrigger asChild>
        <button
          type="button"
          className={cn(
            "flex w-full items-center justify-between gap-2 rounded-lg border border-nb-gray-800 bg-nb-gray-930 text-left transition-colors hover:bg-nb-gray-910",
            className,
          )}
        >
          {trigger}
          <ChevronsUpDown size={16} className="mr-3 shrink-0 text-nb-gray-500" />
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-72 p-0">
        <div className="flex items-center gap-2 border-b border-nb-gray-900 px-3 py-2">
          <Search size={14} className="shrink-0 text-nb-gray-500" />
          <input
            autoFocus
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={placeholder ?? "Search…"}
            className="w-full bg-transparent text-sm text-nb-gray-100 outline-none placeholder:text-nb-gray-600"
          />
        </div>
        <div className="max-h-72 overflow-y-auto p-1">
          {filtered.length === 0 ? (
            <p className="px-2 py-3 text-center text-sm text-nb-gray-500">No results.</p>
          ) : (
            filtered.map((o) => (
              <button
                key={o.value}
                type="button"
                onClick={() => {
                  onChange(o.value);
                  setOpen(false);
                  setQuery("");
                }}
                className={cn(
                  "block w-full truncate rounded-sm px-2 py-1.5 text-left text-sm text-nb-gray-100 hover:bg-nb-gray-900",
                  o.value === value && "bg-nb-gray-900",
                )}
              >
                {o.label}
              </button>
            ))
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}
