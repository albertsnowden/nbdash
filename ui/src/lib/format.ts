// Mirrors internal/web/render.go's funcMap ("relTime"/"osIcon") exactly, so
// the SPA and the still-live htmx pages read identically during coexistence.

// Go's zero time.Time marshals to this exact RFC3339 string — that's what
// "never happened" (e.g. a peer that has never connected) looks like on the
// wire, not an empty string or null.
const ZERO_TIME = "0001-01-01T00:00:00Z";

export function relTime(iso: string): string {
  if (!iso || iso === ZERO_TIME) return "never";

  const diffMs = Date.now() - new Date(iso).getTime();
  if (diffMs < 0) return "just now";

  const sec = diffMs / 1000;
  if (sec < 60) return `${Math.floor(sec)}s ago`;
  const min = sec / 60;
  if (min < 60) return `${Math.floor(min)}m ago`;
  const hr = min / 60;
  if (hr < 24) return `${Math.floor(hr)}h ago`;
  const day = hr / 24;
  if (day < 30) return `${Math.floor(day)}d ago`;

  return new Date(iso).toLocaleDateString(undefined, {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

export function absTime(iso: string): string {
  if (!iso || iso === ZERO_TIME) return "—";
  return new Date(iso).toLocaleString(undefined, {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function osGlyph(os: string): string {
  const lower = os.toLowerCase();
  if (lower.includes("darwin") || lower.includes("mac")) return "◍";
  if (lower.includes("windows")) return "▦";
  if (lower.includes("android")) return "▲";
  if (lower.includes("ios")) return "◈";
  if (
    lower.includes("linux") ||
    lower.includes("ubuntu") ||
    lower.includes("debian") ||
    lower.includes("fedora")
  ) {
    return "◆";
  }
  return "○";
}
