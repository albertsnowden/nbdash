package api

import (
	"net/http"
	"net/url"
)

// maxRequestBodyBytes bounds every request body this server reads. Every
// JSON body this dashboard's SPA posts is a handful of fields — even an
// account with thousands of groups checked into one auto_groups picker
// stays in the tens of kilobytes — so 1 MiB is generous headroom, not a
// tight fit. Without this, a single oversized request could exhaust memory
// before any handler-level validation ever runs; combined with sessions
// living in-memory (see "Sessions are in-memory" in the README's Design
// notes), that would cost every signed-in admin their session, not just the
// request that caused it.
const maxRequestBodyBytes = 1 << 20 // 1 MiB

// limitRequestBody caps every request body at maxRequestBodyBytes. Applied
// to the whole mux rather than only state-changing routes: GET/HEAD
// requests ordinarily carry no body, so this is a no-op for them, and one
// blanket limit is simpler to reason about than remembering to add it to
// every new POST/PUT/DELETE route as the dashboard grows.
func limitRequestBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		next.ServeHTTP(w, r)
	})
}

// requireSameSiteOrigin refuses cross-site state-changing requests.
//
// The session cookie's SameSite=Lax already blocks the common case, but that
// is a promise made by the browser rather than a check made by the server,
// and it treats a sibling subdomain sharing the parent cookie scope as
// same-site. This costs one header comparison and does not need CSRF
// tokens, because every mutation here is a same-origin fetch() call from
// the SPA's own apiFetch() wrapper — see ui/src/api/client.ts.
func requireSameSiteOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Reads are side-effect free, and blocking them would break ordinary
		// inbound links.
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}

		// Sec-Fetch-Site is set by the browser itself and is unreachable from
		// page script, so it decides whenever it is present.
		switch r.Header.Get("Sec-Fetch-Site") {
		case "same-origin", "same-site":
			next.ServeHTTP(w, r)
			return
		case "cross-site":
			http.Error(w, "cross-site request rejected", http.StatusForbidden)
			return
		}

		// Older clients: fall back to Origin. Its absence is not suspicious —
		// a client that sends neither header carries no ambient cookies to
		// abuse in the first place.
		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || u.Host != r.Host {
				http.Error(w, "cross-site request rejected", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// securityHeaders applies a CSP tight enough that the bundled SPA script is
// the only script that can run and no *stylesheet* (a <style> tag or
// <link rel=stylesheet>, i.e. arbitrary CSS rules that could rewrite the
// whole page's presentation or exfiltrate data via attribute selectors) can
// come from anywhere but this app's own bundle: no inline script, no
// external origins, and only two specific pinned exceptions on the style
// side (see below). style-src-attr is the one deliberate exception to "no
// inline style" — see its own paragraph below for why.
//
// style-src's two hashes are Sonner (the toast library, see
// ui/src/main.tsx's Toaster import): it injects its own CSS with a raw
// `document.createElement('style')` at module load regardless of anything
// this app does, with no option to turn that off. The static
// `import "sonner/dist/styles.css"` in main.tsx is what actually styles
// toasts — it's a same-origin stylesheet, unaffected by this CSP — so
// Sonner's own injection attempt is redundant and would otherwise just be
// silently blocked. The two hashes exist only to keep that harmless,
// expected block out of every browser's console instead of hiding it
// behind 'unsafe-inline':
//   - sha256-47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU= is the hash of an
//     empty string — Sonner appends the <style> element to <head> before
//     setting its content, so the browser's CSP check fires once on that
//     empty element first. This one is permanent and library-agnostic:
//     any library that appends-then-fills a <style> tag produces it.
//   - sha256-StEaX+se6YS7pqjzrzMIA0KaX9zF/8zAhvQXZAe5epY= is the hash of
//     Sonner's actual CSS text (the second write, once content lands).
//     This one is tied to the exact bytes Sonner ships and WILL change on
//     a sonner version bump — if its console warning reappears after
//     `npm update sonner`, take the new hash straight from that warning
//     and swap it in here (no need to recompute by hand).
//
// style-src-attr 'unsafe-inline': every Radix primitive built on
// @radix-ui/react-popper (Select, Popover, DropdownMenu, HoverCard,
// Tooltip — used all over ui/src/components/ui/*) positions its floating
// content by writing directly to that element's style property
// (elements.floating.style.setProperty(...) in
// node_modules/@radix-ui/react-popper, called from @floating-ui/dom's
// autoUpdate on every scroll/resize/reposition), not by adding a
// stylesheet rule. There is no nonce or hash escape hatch for this: CSP3
// nonces only cover <style>/<link> elements, and the computed x/y differs
// on every reposition, so no finite hash list could cover it either — the
// only CSP-legal ways to allow it at all are 'unsafe-inline' or CSP3's
// narrower 'unsafe-hashes' (which still requires enumerating hashes, so it
// doesn't actually help for dynamic values). Blocking it doesn't error;
// it just leaves the dropdown/popover/tooltip unpositioned, effectively
// unusable. style-src-attr is a CSP3 directive that governs only the
// style="" HTML attribute — style-src-elem (here, inherited from the
// style-src fallback above, since it's not overridden) still governs
// <style>/<link> and stays exactly as strict as it was, so this doesn't
// reopen the stylesheet-injection hole the rest of this comment is about;
// it only permits an attacker who can already inject a style="" attribute
// on one element they control to restyle that one element, a narrow
// primitive next to full-page CSS rewriting or ex-filtration through
// injected selectors. This app has no dangerouslySetInnerHTML or other
// HTML-injection sink today, so nothing currently exploits even that
// narrower hole — this is a defense-in-depth loosening scoped as tight as
// the CSP3 spec allows, not a response to any known vulnerability. A
// browser too old to understand style-src-attr/style-src-elem falls back
// to the combined style-src above for both, i.e. loses Radix positioning
// but keeps the stylesheet lockdown — a safe direction to fail in.
//
// HSTS is emitted by this service rather than left to the reverse proxy.
// Verify what your own proxy already sends (`curl -sI https://your-dashboard`)
// before relying on either: a proxy that supplies the header makes this line
// redundant, and a duplicate is harmless — browsers take the first — but the
// header should have exactly one owner, so drop it here if your proxy owns it.
//
// It is emitted only when secure is set, i.e. when AUTH_REDIRECT_URI is an
// https URL and the deployment is genuinely served over TLS. Sending it from a
// plain-http local dev instance would be ignored by browsers anyway, and
// omitting it keeps the dev and production responses honest.
//
// max-age is one year. includeSubDomains and preload are deliberately absent:
// both bind sibling hostnames that this service does not own, and preload is
// close to irreversible.
func securityHeaders(secure bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; "+
				"style-src 'self' 'sha256-47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=' "+
				"'sha256-StEaX+se6YS7pqjzrzMIA0KaX9zF/8zAhvQXZAe5epY='; "+
				"style-src-attr 'unsafe-inline'; "+
				"img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("X-Frame-Options", "DENY")
		if secure {
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}
