/// <reference types="vitest/config" />
import { fileURLToPath, URL } from "node:url";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// The Go binary embeds internal/web/static/ wholesale (see
// internal/web/render.go's `//go:embed templates static`), and serves it
// under /static/. Building straight into internal/web/static/app/ means
// that embed directive needs no change — Vite's output just becomes part
// of the tree that's already embedded. `base` has to match where the app
// is actually served from so the built index.html's asset references
// resolve correctly.
export default defineConfig({
  base: "/static/app/",
  plugins: [react()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  build: {
    outDir: "../internal/web/static/app",
    emptyOutDir: true,
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test-setup.ts"],
  },
});
