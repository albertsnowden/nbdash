import type { Group } from "@/api/groups";
import GroupCheckboxList from "@/components/GroupCheckboxList";
import Input from "@/components/ui/Input";
import Label from "@/components/ui/Label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import Switch from "@/components/ui/Switch";

export interface RuleFormState {
  name: string;
  action: "accept" | "drop";
  protocol: "all" | "tcp" | "udp" | "icmp";
  bidirectional: boolean;
  enabled: boolean;
  sources: string[];
  destinations: string[];
  portsText: string;
}

export function parsePorts(text: string): string[] {
  return text
    .split(/[,\s]+/)
    .map((s) => s.trim())
    .filter(Boolean);
}

interface Props {
  groups: Group[];
  value: RuleFormState;
  onChange: (next: RuleFormState) => void;
}

// Shared by the initial rule on a new policy and the standalone add/edit
// rule form on a policy's detail page — the field set is identical in both
// places (see internal/api/policies.go's parseRuleForm, which every one of
// those write paths shares server-side too).
export default function RuleFieldsInputs({ groups, value, onChange }: Props) {
  const set = <K extends keyof RuleFormState>(key: K, v: RuleFormState[K]) =>
    onChange({ ...value, [key]: v });

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="rule-name">Rule name</Label>
        <Input id="rule-name" required value={value.name} onChange={(e) => set("name", e.target.value)} />
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="rule-action">Action</Label>
          <Select value={value.action} onValueChange={(v) => set("action", v as RuleFormState["action"])}>
            <SelectTrigger id="rule-action">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="accept">Accept</SelectItem>
              <SelectItem value="drop">Drop</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="rule-protocol">Protocol</Label>
          <Select value={value.protocol} onValueChange={(v) => set("protocol", v as RuleFormState["protocol"])}>
            <SelectTrigger id="rule-protocol">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All</SelectItem>
              <SelectItem value="tcp">TCP</SelectItem>
              <SelectItem value="udp">UDP</SelectItem>
              <SelectItem value="icmp">ICMP</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      <label htmlFor="rule-bidirectional" className="flex items-center justify-between gap-4">
        <span className="text-sm text-nb-gray-100">Bidirectional</span>
        <Switch id="rule-bidirectional" checked={value.bidirectional} onCheckedChange={(v) => set("bidirectional", v)} />
      </label>

      <label htmlFor="rule-enabled" className="flex items-center justify-between gap-4">
        <span className="text-sm text-nb-gray-100">Enabled</span>
        <Switch id="rule-enabled" checked={value.enabled} onCheckedChange={(v) => set("enabled", v)} />
      </label>

      <div className="grid grid-cols-2 gap-4">
        <div className="flex flex-col gap-1.5">
          <Label>Sources</Label>
          <GroupCheckboxList groups={groups} selected={value.sources} onChange={(ids) => set("sources", ids)} />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label>Destinations</Label>
          <GroupCheckboxList
            groups={groups}
            selected={value.destinations}
            onChange={(ids) => set("destinations", ids)}
          />
        </div>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="rule-ports">Ports</Label>
        <Input
          id="rule-ports"
          placeholder="e.g. 22, 443"
          value={value.portsText}
          onChange={(e) => set("portsText", e.target.value)}
        />
        <p className="text-xs text-nb-gray-500">Comma-separated. Not allowed for All or ICMP.</p>
      </div>
    </div>
  );
}
