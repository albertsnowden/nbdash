import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App.tsx";
// Sonner (toast library) also injects this same CSS at runtime via a raw
// `document.createElement('style')`, which the CSP's `style-src 'self'`
// blocks — see securityHeaders' doc comment in internal/api/middleware.go.
// This static import is a real same-origin stylesheet, so it's unaffected
// by that block and is what actually styles toasts; the blocked runtime
// injection is harmless, redundant noise once this is in place.
import "sonner/dist/styles.css";
import "./styles/globals.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
