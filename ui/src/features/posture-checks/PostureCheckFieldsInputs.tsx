import { Plus, Trash2 } from "lucide-react";
import type { Checks, Process } from "@/api/postureChecks";
import Button from "@/components/ui/Button";
import Input from "@/components/ui/Input";
import Label from "@/components/ui/Label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import Switch from "@/components/ui/Switch";

export interface ChecksFormState {
  nbVersionEnabled: boolean;
  nbMinVersion: string;

  osVersionEnabled: boolean;
  androidMinVersion: string;
  darwinMinVersion: string;
  iosMinVersion: string;
  linuxMinKernelVersion: string;
  windowsMinKernelVersion: string;

  geoEnabled: boolean;
  geoAction: "allow" | "deny";
  geoLocationsText: string; // "US:New York, DE" — one per line/comma, city optional

  rangeEnabled: boolean;
  rangeAction: "allow" | "deny";
  rangesText: string; // comma/space separated CIDRs

  processEnabled: boolean;
  processes: Process[];
}

export const EMPTY_CHECKS_FORM: ChecksFormState = {
  nbVersionEnabled: false,
  nbMinVersion: "",
  osVersionEnabled: false,
  androidMinVersion: "",
  darwinMinVersion: "",
  iosMinVersion: "",
  linuxMinKernelVersion: "",
  windowsMinKernelVersion: "",
  geoEnabled: false,
  geoAction: "allow",
  geoLocationsText: "",
  rangeEnabled: false,
  rangeAction: "allow",
  rangesText: "",
  processEnabled: false,
  processes: [],
};

export function checksToFormState(checks: Checks): ChecksFormState {
  const geo = checks.geo_location_check;
  const range = checks.peer_network_range_check;
  return {
    nbVersionEnabled: !!checks.nb_version_check,
    nbMinVersion: checks.nb_version_check?.min_version ?? "",
    osVersionEnabled: !!checks.os_version_check,
    androidMinVersion: checks.os_version_check?.android?.min_version ?? "",
    darwinMinVersion: checks.os_version_check?.darwin?.min_version ?? "",
    iosMinVersion: checks.os_version_check?.ios?.min_version ?? "",
    linuxMinKernelVersion: checks.os_version_check?.linux?.min_kernel_version ?? "",
    windowsMinKernelVersion: checks.os_version_check?.windows?.min_kernel_version ?? "",
    geoEnabled: !!geo,
    geoAction: geo?.action ?? "allow",
    geoLocationsText: (geo?.locations ?? [])
      .map((l) => (l.city_name ? `${l.country_code}:${l.city_name}` : l.country_code))
      .join(", "),
    rangeEnabled: !!range,
    rangeAction: range?.action ?? "allow",
    rangesText: (range?.ranges ?? []).join(", "),
    processEnabled: !!checks.process_check,
    processes: checks.process_check?.processes ?? [],
  };
}

export function formStateToChecks(v: ChecksFormState): Checks {
  const checks: Checks = {};

  if (v.nbVersionEnabled && v.nbMinVersion.trim()) {
    checks.nb_version_check = { min_version: v.nbMinVersion.trim() };
  }

  if (v.osVersionEnabled) {
    checks.os_version_check = {
      android: v.androidMinVersion.trim() ? { min_version: v.androidMinVersion.trim() } : undefined,
      darwin: v.darwinMinVersion.trim() ? { min_version: v.darwinMinVersion.trim() } : undefined,
      ios: v.iosMinVersion.trim() ? { min_version: v.iosMinVersion.trim() } : undefined,
      linux: v.linuxMinKernelVersion.trim() ? { min_kernel_version: v.linuxMinKernelVersion.trim() } : undefined,
      windows: v.windowsMinKernelVersion.trim()
        ? { min_kernel_version: v.windowsMinKernelVersion.trim() }
        : undefined,
    };
  }

  if (v.geoEnabled) {
    const locations = v.geoLocationsText
      .split(/[,\n]+/)
      .map((s) => s.trim())
      .filter(Boolean)
      .map((entry) => {
        const [country, city] = entry.split(":").map((s) => s.trim());
        return city ? { country_code: country, city_name: city } : { country_code: country };
      });
    checks.geo_location_check = { locations, action: v.geoAction };
  }

  if (v.rangeEnabled) {
    const ranges = v.rangesText
      .split(/[,\s]+/)
      .map((s) => s.trim())
      .filter(Boolean);
    checks.peer_network_range_check = { ranges, action: v.rangeAction };
  }

  if (v.processEnabled && v.processes.length > 0) {
    checks.process_check = { processes: v.processes };
  }

  return checks;
}

interface Props {
  value: ChecksFormState;
  onChange: (next: ChecksFormState) => void;
}

export default function PostureCheckFieldsInputs({ value, onChange }: Props) {
  const set = <K extends keyof ChecksFormState>(key: K, v: ChecksFormState[K]) =>
    onChange({ ...value, [key]: v });

  const updateProcess = (i: number, patch: Partial<Process>) => {
    const next = value.processes.map((p, idx) => (idx === i ? { ...p, ...patch } : p));
    set("processes", next);
  };

  return (
    <div className="flex flex-col gap-5">
      <CheckSection title="NetBird version" enabled={value.nbVersionEnabled} onEnabledChange={(v) => set("nbVersionEnabled", v)}>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="pc-nb-version">Minimum version</Label>
          <Input
            id="pc-nb-version"
            placeholder="e.g. 0.30.0"
            value={value.nbMinVersion}
            onChange={(e) => set("nbMinVersion", e.target.value)}
          />
        </div>
      </CheckSection>

      <CheckSection title="Operating system version" enabled={value.osVersionEnabled} onEnabledChange={(v) => set("osVersionEnabled", v)}>
        <div className="grid grid-cols-2 gap-3">
          <div className="flex flex-col gap-1.5">
            <Label>Android min version</Label>
            <Input value={value.androidMinVersion} onChange={(e) => set("androidMinVersion", e.target.value)} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>macOS min version</Label>
            <Input value={value.darwinMinVersion} onChange={(e) => set("darwinMinVersion", e.target.value)} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>iOS min version</Label>
            <Input value={value.iosMinVersion} onChange={(e) => set("iosMinVersion", e.target.value)} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>Linux min kernel version</Label>
            <Input value={value.linuxMinKernelVersion} onChange={(e) => set("linuxMinKernelVersion", e.target.value)} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>Windows min kernel version</Label>
            <Input value={value.windowsMinKernelVersion} onChange={(e) => set("windowsMinKernelVersion", e.target.value)} />
          </div>
        </div>
      </CheckSection>

      <CheckSection title="Geo location" enabled={value.geoEnabled} onEnabledChange={(v) => set("geoEnabled", v)}>
        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="pc-geo-action">Action</Label>
            <Select value={value.geoAction} onValueChange={(v) => set("geoAction", v as "allow" | "deny")}>
              <SelectTrigger id="pc-geo-action">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="allow">Allow</SelectItem>
                <SelectItem value="deny">Deny</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="pc-geo-locations">Locations</Label>
            <Input
              id="pc-geo-locations"
              placeholder="US:New York, DE"
              value={value.geoLocationsText}
              onChange={(e) => set("geoLocationsText", e.target.value)}
            />
            <p className="text-xs text-nb-gray-500">
              Comma-separated. Each entry is a 2-letter country code, optionally followed by :City.
            </p>
          </div>
        </div>
      </CheckSection>

      <CheckSection title="Peer network range" enabled={value.rangeEnabled} onEnabledChange={(v) => set("rangeEnabled", v)}>
        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="pc-range-action">Action</Label>
            <Select value={value.rangeAction} onValueChange={(v) => set("rangeAction", v as "allow" | "deny")}>
              <SelectTrigger id="pc-range-action">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="allow">Allow</SelectItem>
                <SelectItem value="deny">Deny</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="pc-ranges">Ranges (CIDR)</Label>
            <Input
              id="pc-ranges"
              placeholder="10.0.0.0/8, 2001:db8::/32"
              value={value.rangesText}
              onChange={(e) => set("rangesText", e.target.value)}
            />
          </div>
        </div>
      </CheckSection>

      <CheckSection title="Required processes" enabled={value.processEnabled} onEnabledChange={(v) => set("processEnabled", v)}>
        <div className="flex flex-col gap-3">
          {value.processes.map((p, i) => (
            <div key={i} className="flex items-end gap-2 rounded-md border border-nb-gray-900 p-3">
              <div className="flex flex-1 flex-col gap-2">
                <Input
                  placeholder="Linux path, e.g. /usr/local/bin/netbird"
                  value={p.linux_path ?? ""}
                  onChange={(e) => updateProcess(i, { linux_path: e.target.value })}
                />
                <Input
                  placeholder="macOS path"
                  value={p.mac_path ?? ""}
                  onChange={(e) => updateProcess(i, { mac_path: e.target.value })}
                />
                <Input
                  placeholder="Windows path"
                  value={p.windows_path ?? ""}
                  onChange={(e) => updateProcess(i, { windows_path: e.target.value })}
                />
              </div>
              <Button
                variant="ghost"
                size="icon"
                type="button"
                className="text-red-500 hover:bg-red-950 hover:text-red-400"
                onClick={() => set("processes", value.processes.filter((_, idx) => idx !== i))}
                aria-label="Remove process"
              >
                <Trash2 size={16} />
              </Button>
            </div>
          ))}
          <Button
            variant="secondary"
            size="sm"
            type="button"
            className="w-fit"
            onClick={() => set("processes", [...value.processes, {}])}
          >
            <Plus size={14} /> Add process
          </Button>
        </div>
      </CheckSection>
    </div>
  );
}

function CheckSection({
  title,
  enabled,
  onEnabledChange,
  children,
}: {
  title: string;
  enabled: boolean;
  onEnabledChange: (v: boolean) => void;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-md border border-nb-gray-900 p-4">
      <label className="flex cursor-pointer items-center justify-between gap-4">
        <span className="text-sm font-medium text-nb-gray-100">{title}</span>
        <Switch checked={enabled} onCheckedChange={onEnabledChange} />
      </label>
      {enabled && <div className="mt-4">{children}</div>}
    </div>
  );
}
