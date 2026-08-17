import { AnimatePresence, motion } from "framer-motion";
import { Suspense, useState } from "react";
import { Outlet } from "react-router";
import { PermissionsProvider } from "@/contexts/PermissionsContext";
import Header from "./Header";
import Sidebar from "./Sidebar";

// Every route under this layout is a lazy-loaded chunk (see App.tsx) — this
// is the one Suspense boundary that catches all of them, including a
// second Outlet's worth nested inside DNS/Team/Settings' own layouts, so
// the sidebar/header chrome stays mounted and only the content area shows
// the fallback while a route's code downloads.
function RouteLoadingFallback() {
  return (
    <div className="flex h-full items-center justify-center py-24 text-sm text-nb-gray-500">Loading…</div>
  );
}

export default function DashboardLayout() {
  const [collapsed, setCollapsed] = useState(false);
  const [mobileNavOpen, setMobileNavOpen] = useState(false);

  return (
    <PermissionsProvider>
      <div className="flex h-screen bg-nb-gray-950 text-nb-gray-50">
        <Sidebar collapsed={collapsed} className="hidden md:flex" />

        <AnimatePresence>
          {mobileNavOpen && (
            <>
              <motion.div
                key="backdrop"
                className="fixed inset-0 z-40 bg-black/60 md:hidden"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                onClick={() => setMobileNavOpen(false)}
              />
              <motion.div
                key="drawer"
                className="fixed inset-y-0 left-0 z-50 md:hidden"
                initial={{ x: "-100%" }}
                animate={{ x: 0 }}
                exit={{ x: "-100%" }}
                transition={{ type: "spring", stiffness: 320, damping: 32 }}
              >
                <Sidebar collapsed={false} onNavigate={() => setMobileNavOpen(false)} />
              </motion.div>
            </>
          )}
        </AnimatePresence>

        <div className="flex min-w-0 flex-1 flex-col">
          <Header
            collapsed={collapsed}
            onToggleCollapsed={() => setCollapsed((v) => !v)}
            onOpenMobileNav={() => setMobileNavOpen(true)}
          />
          <main className="flex-1 overflow-y-auto p-6">
            <Suspense fallback={<RouteLoadingFallback />}>
              <Outlet />
            </Suspense>
          </main>
        </div>
      </div>
    </PermissionsProvider>
  );
}
