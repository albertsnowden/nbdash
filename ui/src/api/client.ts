// Thin fetch wrapper for the Go backend's JSON BFF layer at /api/bff/...
//
// There is no bearer token to attach here — auth is a session cookie the
// browser already sends automatically (see internal/api's design notes).
// `credentials: "same-origin"` is set explicitly rather than relied on as a
// fetch-spec default, since that default has changed across spec revisions.
const BASE = "/api/bff";

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, {
    ...init,
    credentials: "same-origin",
    headers: {
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
  });

  // The BFF has no HX-Redirect-style header the browser follows automatically
  // for a fetch() — this is the one place that translates "session expired
  // or missing" into an actual navigation, mirroring what
  // internal/handlers' redirectToLogin does for the htmx app.
  if (res.status === 401) {
    window.location.href = "/login";
    throw new ApiError(401, "unauthenticated");
  }

  if (!res.ok) {
    let message = res.statusText;
    let reason: string | undefined;
    try {
      const body = (await res.json()) as { error?: { message?: string; reason?: string } };
      if (body.error?.message) message = body.error.message;
      reason = body.error?.reason;
    } catch {
      // Non-JSON error body (e.g. a proxy's own error page) — keep statusText.
    }

    // pending_approval/blocked are account-wide, not specific to whatever
    // page happened to make this request — every other page would hit the
    // exact same error, so this redirects to one dedicated screen instead
    // of leaking a raw 403 into each page's own inline error state. Same
    // shape as the 401 handling above, just to a different destination.
    if (reason === "pending_approval" || reason === "blocked") {
      window.location.href = `/account-restricted?reason=${reason}`;
      throw new ApiError(res.status, message);
    }

    throw new ApiError(res.status, message);
  }

  if (res.status === 204) {
    return undefined as T;
  }
  return (await res.json()) as T;
}

export const apiGet = <T>(path: string): Promise<T> => request<T>(path);

export const apiPost = <T>(path: string, body: unknown): Promise<T> =>
  request<T>(path, { method: "POST", body: JSON.stringify(body) });

export const apiPut = <T>(path: string, body: unknown): Promise<T> =>
  request<T>(path, { method: "PUT", body: JSON.stringify(body) });

export const apiDelete = (path: string): Promise<void> =>
  request<void>(path, { method: "DELETE" });
