import { Plus, Trash2 } from "lucide-react";
import type { ServiceTarget } from "@/api/reverseProxy";
import Button from "@/components/ui/Button";
import Checkbox from "@/components/ui/Checkbox";
import InfoTooltip from "@/components/ui/InfoTooltip";
import Input from "@/components/ui/Input";
import Label from "@/components/ui/Label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";

const EMPTY_TARGET: ServiceTarget = {
  target_id: "",
  target_type: "host",
  protocol: "http",
  host: "",
  port: 80,
  enabled: true,
};

export default function ServiceTargetsInput({
  targets,
  onChange,
}: {
  targets: ServiceTarget[];
  onChange: (next: ServiceTarget[]) => void;
}) {
  const update = (i: number, patch: Partial<ServiceTarget>) => {
    onChange(targets.map((t, idx) => (idx === i ? { ...t, ...patch } : t)));
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-1.5">
        <Label>Targets</Label>
        <InfoTooltip>
          Where traffic gets forwarded once it reaches this service. Host/Domain dial a plain address; Peer,
          Subnet and Cluster route through the NetBird network by ID — find those IDs on the Peers, Networks and
          Reverse Proxy → Clusters pages.
        </InfoTooltip>
      </div>
      {targets.map((t, i) => (
        <div key={i} className="flex flex-col gap-3 rounded-md border border-nb-gray-900 p-3">
          <div className="grid grid-cols-2 gap-3">
            <div className="flex flex-col gap-1.5">
              <Label>Target type</Label>
              <Select
                value={t.target_type}
                onValueChange={(v) => update(i, { target_type: v as ServiceTarget["target_type"] })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="host">Host / IP</SelectItem>
                  <SelectItem value="domain">Domain</SelectItem>
                  <SelectItem value="peer">Peer ID</SelectItem>
                  <SelectItem value="subnet">Subnet ID</SelectItem>
                  <SelectItem value="cluster">Cluster</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>Protocol</Label>
              <Select value={t.protocol} onValueChange={(v) => update(i, { protocol: v as ServiceTarget["protocol"] })}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="http">HTTP</SelectItem>
                  <SelectItem value="https">HTTPS</SelectItem>
                  <SelectItem value="tcp">TCP</SelectItem>
                  <SelectItem value="udp">UDP</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="grid grid-cols-[2fr_1fr] gap-3">
            <div className="flex flex-col gap-1.5">
              <Label>
                {t.target_type === "host" || t.target_type === "domain" ? "Host / IP" : "Target ID"}
              </Label>
              <Input
                value={t.target_type === "host" || t.target_type === "domain" ? (t.host ?? "") : t.target_id}
                onChange={(e) =>
                  t.target_type === "host" || t.target_type === "domain"
                    ? update(i, { host: e.target.value })
                    : update(i, { target_id: e.target.value })
                }
                placeholder={t.target_type === "host" ? "10.0.0.5" : t.target_type === "domain" ? "internal.example.com" : "ID"}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>Port</Label>
              <Input
                type="number"
                min={1}
                max={65535}
                value={t.port}
                onChange={(e) => update(i, { port: Number(e.target.value) })}
              />
            </div>
          </div>

          <div className="flex items-center justify-between">
            <label className="flex items-center gap-2 text-sm text-nb-gray-300">
              <Checkbox checked={t.enabled} onCheckedChange={(v) => update(i, { enabled: !!v })} />
              Enabled
            </label>
            {targets.length > 1 && (
              <Button
                variant="ghost"
                size="sm"
                type="button"
                className="text-red-500 hover:bg-red-950 hover:text-red-400"
                onClick={() => onChange(targets.filter((_, idx) => idx !== i))}
              >
                <Trash2 size={14} /> Remove
              </Button>
            )}
          </div>
        </div>
      ))}
      <Button
        variant="secondary"
        size="sm"
        type="button"
        className="w-fit"
        onClick={() => onChange([...targets, { ...EMPTY_TARGET, target_id: crypto.randomUUID() }])}
      >
        <Plus size={14} /> Add target
      </Button>
    </div>
  );
}
