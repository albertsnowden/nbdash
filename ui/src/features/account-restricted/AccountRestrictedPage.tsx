import { Ban, Clock } from "lucide-react";
import { useSearchParams } from "react-router";
import Button from "@/components/ui/Button";
import { Card, CardContent } from "@/components/ui/Card";

const COPY: Record<string, { icon: typeof Clock; title: string; description: string }> = {
  pending_approval: {
    icon: Clock,
    title: "Your account is pending approval",
    description:
      "An administrator needs to approve your account before you can use this dashboard. Please contact your NetBird administrator and ask them to approve you under Team → Users.",
  },
  blocked: {
    icon: Ban,
    title: "Your account has been blocked",
    description: "An administrator has blocked this account. Please contact your NetBird administrator.",
  },
};

// Reached whenever apiFetch() sees an account-wide lockout reason (see
// ui/src/api/client.ts) — every other page would hit the exact same 403 no
// matter what it fetches, so this is a dedicated, chrome-free destination
// (no sidebar/topbar — nothing there would work either) rather than an
// inline error on whatever page the user happened to land on.
export default function AccountRestrictedPage() {
  const [params] = useSearchParams();
  const reason = params.get("reason") ?? "";
  const copy = COPY[reason] ?? COPY.blocked;
  const Icon = copy.icon;

  return (
    <div className="flex min-h-screen items-center justify-center bg-nb-gray-950 px-4">
      <Card className="w-full max-w-md">
        <CardContent className="flex flex-col items-center gap-4 pt-8 text-center">
          <img className="h-6 w-auto" src="/static/netbird-full.svg" alt="NetBird" />
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-nb-gray-900 text-nb-gray-400">
            <Icon size={22} />
          </div>
          <h1 className="text-lg font-semibold text-nb-gray-50">{copy.title}</h1>
          <p className="text-sm text-nb-gray-400">{copy.description}</p>
          <form method="post" action="/logout" className="w-full pt-2">
            <Button variant="secondary" type="submit" className="w-full">
              Sign out
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
