import { MODULE } from "@/lib/modules";

// A /permissions response granting every operation on every module this
// dashboard checks — the default for component tests that aren't
// specifically exercising permission-gated UI, so existing assertions
// (written before role-based hiding existed) don't need to know about it.
export const FULL_ACCESS_PERMISSIONS_RESPONSE = {
  role: "admin",
  permissions: {
    is_restricted: false,
    modules: Object.fromEntries(
      Object.values(MODULE).map((module) => [
        module,
        { read: true, create: true, update: true, delete: true },
      ]),
    ),
  },
};
