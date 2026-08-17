import { createContext, useContext, useMemo, type ReactNode } from "react";
import { usePermissions } from "@/api/permissions";

// Matches management/server/permissions/operations' Operation type in the
// netbird repo — the server never sends any value outside this set.
type Operation = "create" | "read" | "update" | "delete";

interface PermissionsContextValue {
  // True until the first /permissions response lands. Callers that gate
  // rendering on a module should treat "loading" as "allow" — hiding
  // everything for a flash before data arrives is worse than briefly
  // showing something the very next render may hide, and every access is
  // re-checked server-side regardless of what this UI shows.
  isLoading: boolean;
  role: string;
  can: (module: string, operation: Operation) => boolean;
  canRead: (module: string) => boolean;
  canReadAny: (modules: string[]) => boolean;
  canCreate: (module: string) => boolean;
  canUpdate: (module: string) => boolean;
  canDelete: (module: string) => boolean;
}

const PermissionsContext = createContext<PermissionsContextValue | null>(null);

export function PermissionsProvider({ children }: { children: ReactNode }) {
  const { data, isLoading } = usePermissions();

  const value = useMemo<PermissionsContextValue>(() => {
    const modules = data?.permissions.modules ?? {};
    const can = (module: string, operation: Operation) => {
      if (isLoading) return true;
      const perm = modules[module];
      // Missing entry is treated as denied — GetPermissionsByRole
      // (server-side) always returns every module key, so an absent key on
      // a loaded response is a real denial, not missing data.
      return perm ? perm[operation] === true : false;
    };
    const canRead = (module: string) => can(module, "read");
    return {
      isLoading,
      role: data?.role ?? "",
      can,
      canRead,
      canReadAny: (mods) => mods.some(canRead),
      canCreate: (module: string) => can(module, "create"),
      canUpdate: (module: string) => can(module, "update"),
      canDelete: (module: string) => can(module, "delete"),
    };
  }, [data, isLoading]);

  return <PermissionsContext.Provider value={value}>{children}</PermissionsContext.Provider>;
}

export function usePermissionsContext() {
  const ctx = useContext(PermissionsContext);
  if (!ctx) {
    throw new Error("usePermissionsContext must be used within PermissionsProvider");
  }
  return ctx;
}
