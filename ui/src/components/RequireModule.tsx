import { ShieldOff } from "lucide-react";
import type { ReactNode } from "react";
import { usePermissionsContext } from "@/contexts/PermissionsContext";

// Blocks direct navigation to a page the current role can't read — the
// sidebar already hides the link (see Sidebar.tsx/navItems.ts), but a
// bookmarked or typed URL would otherwise still render the page and let it
// hammer the Management API with 403s on every query. Wrap the page's
// element with this instead of guessing what error state the page itself
// would show.
export default function RequireModule({ modules, children }: { modules: string[]; children: ReactNode }) {
  const { isLoading, canReadAny } = usePermissionsContext();

  if (isLoading) return null;

  if (!canReadAny(modules)) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 rounded-lg border border-nb-gray-900 bg-nb-gray-950 py-16 text-center">
        <ShieldOff size={28} className="text-nb-gray-600" />
        <p className="text-sm font-medium text-nb-gray-200">You don't have access to this section</p>
        <p className="max-w-sm text-xs text-nb-gray-500">
          Your role doesn't grant read access here. Contact an account admin if you think this is wrong.
        </p>
      </div>
    );
  }

  return <>{children}</>;
}
