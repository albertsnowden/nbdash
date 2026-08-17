package auth

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

const (
	sessionCookie = "nb_session"
	stateCookie   = "nb_auth_state"

	// hostPrefix is prepended to both cookie names on an https deployment.
	//
	// __Host- is enforced by the browser, not by us: it refuses to store the
	// cookie unless it is Secure, Path=/, and carries no Domain attribute. The
	// value of that guarantee here is the no-Domain part. Without it, a sibling
	// host under a shared parent domain (dev.example.com writing for
	// .example.com) can set a cookie the browser will then send to the
	// dashboard, overwriting the session — a fixation vector that SameSite does
	// nothing about, since SameSite treats siblings as same-site. That is the
	// exact gap requireSameSiteOrigin's doc comment calls out.
	//
	// It cannot be used on a plain-http dev instance, because the prefix
	// requires Secure and a browser would silently drop the cookie — hence
	// cookieName picking per deployment rather than a constant.
	hostPrefix = "__Host-"

	// pendingTTL bounds how long a half-finished login (redirect issued, no
	// callback yet) is remembered.
	pendingTTL = 15 * time.Minute

	// sessionLifetime caps a session from the moment it is created and is never
	// extended. The cookie's MaxAge is only a hint to a cooperating browser —
	// anyone holding a stolen session ID simply sets the cookie without an
	// expiry — so this is the sole server-side bound that a thief cannot evade.
	sessionLifetime = 12 * time.Hour

	// sessionIdleTimeout ends a session that has gone unused this long, sliding
	// forward on each use. It shortens the window in which a session left open
	// on an unattended machine is still worth stealing.
	sessionIdleTimeout = 2 * time.Hour

	// sessionReapInterval is how often expired sessions are swept out. get()
	// already evicts on access, so this exists only for sessions nobody will
	// touch again; without it they pin memory until the process restarts, which
	// matters under the systemd MemoryMax ceiling.
	sessionReapInterval = 10 * time.Minute
)

// Session is a logged-in user. Tokens are held server-side and never reach the
// browser — the browser only ever carries the opaque session ID. This is the
// main security difference from the Next.js dashboard, where the access token
// lives in the browser and is attached to every XHR.
type Session struct {
	// mu guards Token and IDToken. Store.mu is no help there: it protects the
	// map, not the values the map points at, and store.get hands the same
	// *Session to every concurrent request. The SPA's React Query hooks
	// routinely fire several /api/bff/... calls in parallel on one page
	// load, so two handlers routinely refresh one session at the same
	// instant.
	//
	// created and lastUsed are deliberately outside this lock — they are
	// guarded by Store.mu, which the store already holds whenever it stamps
	// them. Two locks over one struct is only safe because the field sets do
	// not overlap; keep it that way.
	mu sync.Mutex

	ID      string
	Token   *oauth2.Token
	IDToken string
	Email   string
	Name    string
	Sub     string

	// created and lastUsed drive expiry. They are unexported and stamped by the
	// store because when a session dies is the store's business — a caller that
	// could set them could also extend its own session.
	created  time.Time
	lastUsed time.Time
}

// expired reports whether a session has passed either bound: the absolute
// lifetime measured from creation, or the idle timeout since it was last used.
func (s *Session) expired(now time.Time) bool {
	return now.Sub(s.created) >= sessionLifetime ||
		now.Sub(s.lastUsed) >= sessionIdleTimeout
}

// pending is the in-flight state of an authorization request, held between the
// redirect to the provider and the callback.
type pending struct {
	verifier string
	nonce    string
	redirect string
	created  time.Time
}

// Store keeps sessions in memory.
//
// This means sessions do not survive a restart and do not work across multiple
// replicas. That is an acceptable trade for a single-binary deployment; moving
// to Redis or signed cookies is a drop-in replacement for this type.
type Store struct {
	mu       sync.Mutex
	sessions map[string]*Session
	pending  map[string]*pending

	// now is the clock. Tests replace it to reach a 12-hour expiry without
	// waiting; NewStore is the only production writer, and the reapers do not
	// read it until their first tick, long after any test has finished.
	now func() time.Time
}

func NewStore() *Store {
	s := &Store{
		sessions: make(map[string]*Session),
		pending:  make(map[string]*pending),
		now:      time.Now,
	}
	go s.reapPending()
	go s.reapSessions()
	return s
}

// reapPending drops abandoned authorization attempts so the map cannot grow
// without bound from users who start a login and never finish it.
func (s *Store) reapPending() {
	for range time.Tick(pendingTTL) {
		cutoff := time.Now().Add(-pendingTTL)
		s.mu.Lock()
		for k, p := range s.pending {
			if p.created.Before(cutoff) {
				delete(s.pending, k)
			}
		}
		s.mu.Unlock()
	}
}

// reapSessions drops expired sessions so the map cannot grow without bound from
// users who close the browser and never come back. Mirrors reapPending.
func (s *Store) reapSessions() {
	for range time.Tick(sessionReapInterval) {
		s.sweepSessions()
	}
}

// sweepSessions removes every expired session in one pass. It is split out from
// the goroutine so expiry can be tested without waiting for a tick.
func (s *Store) sweepSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	for id, sess := range s.sessions {
		if sess.expired(now) {
			delete(s.sessions, id)
		}
	}
}

func (s *Store) savePending(state string, p *pending) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.created = time.Now()
	s.pending[state] = p
}

// takePending returns and removes the pending request for state, so a state
// value cannot be replayed.
func (s *Store) takePending(state string) (*pending, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.pending[state]
	delete(s.pending, state)
	if !ok || time.Since(p.created) > pendingTTL {
		return nil, false
	}
	return p, true
}

func (s *Store) save(sess *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	// created is stamped once. save runs again after every token refresh, and
	// renewing the absolute lifetime there would mean an actively refreshed
	// session never ages out — which is exactly what it exists to prevent.
	if sess.created.IsZero() {
		sess.created = now
	}
	sess.lastUsed = now
	s.sessions[sess.ID] = sess
}

func (s *Store) get(id string) (*Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[id]
	if !ok {
		return nil, false
	}

	now := s.now()
	if sess.expired(now) {
		// Evict here as well as in the reaper so a stolen ID stops working the
		// moment it is presented, rather than at the next sweep.
		delete(s.sessions, id)
		return nil, false
	}

	sess.lastUsed = now
	return sess, true
}

func (s *Store) delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

func randomID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// setCookie writes one of the two auth cookies.
//
// SameSite is Lax, not Strict, and that is deliberate. Strict's defining
// behaviour — the property that distinguishes it from Lax — is that the cookie
// is withheld on cross-site top-level navigations. The provider's redirect to
// /auth/callback is exactly such a navigation, so under Strict nb_auth_state
// never arrives, Callback cannot match the state parameter, and every login
// ends at "login state missing or expired". Strict was tried and reverted; see
// TestLoginFailsWhenStateCookieIsWithheldAtCallback in internal/handlers.
//
// Lax therefore carries no CSRF weight here. That job belongs to
// requireSameSiteOrigin in internal/handlers/router.go, which is a server-side
// check on unsafe methods and does not depend on browser cookie behaviour at
// all.
func setCookie(w http.ResponseWriter, name, value string, secure bool, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
		// No Domain: required by the __Host- prefix (see hostPrefix), and
		// correct regardless — the dashboard is reached on exactly one host.
	})
}

// cookieName returns the session cookie's name for this deployment, prefixed
// with __Host- when the deployment is served over TLS. See hostPrefix.
func (a *Authenticator) cookieName() string { return prefixed(sessionCookie, a.cfg.Secure) }

// stateCookieName is cookieName's counterpart for the short-lived login-state
// cookie. It gets the same treatment: it is what binds a callback to the
// browser that started the flow, so a sibling host being able to overwrite it
// would undo that binding.
func (a *Authenticator) stateCookieName() string { return prefixed(stateCookie, a.cfg.Secure) }

func prefixed(name string, secure bool) string {
	if secure {
		return hostPrefix + name
	}
	return name
}

// clearCookie expires a cookie. It mirrors setCookie's attributes because a
// browser matches a deletion against name, path and domain — a Set-Cookie with
// a different Path would add a second expired cookie and leave the live one in
// place.
func (a *Authenticator) clearCookie(w http.ResponseWriter, name string) {
	setCookie(w, name, "", a.cfg.Secure, -1)
}
