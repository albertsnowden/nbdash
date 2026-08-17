package auth

import (
	"testing"
	"time"
)

// Expiry is driven through Store.now rather than by sleeping, so these tests
// exercise 12-hour behaviour in microseconds.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }
func newFakeClock() *fakeClock               { return &fakeClock{t: time.Date(2026, 8, 10, 9, 0, 0, 0, time.UTC)} }

// storeAt returns a store whose clock is the returned fakeClock.
func storeAt(t *testing.T) (*Store, *fakeClock) {
	t.Helper()
	store := NewStore()
	clock := newFakeClock()
	store.now = clock.now
	return store, clock
}

// has reports whether the id is still present in the map, independent of what
// get() would return — the reaper's job is to actually free the memory.
func (s *Store) has(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.sessions[id]
	return ok
}

// The absolute lifetime is the backstop against a stolen session ID: it must
// end the session no matter how recently it was used.
func TestGetRejectsSessionPastAbsoluteLifetime(t *testing.T) {
	store, clock := storeAt(t)
	store.save(&Session{ID: "s1"})

	// Use it every hour so the idle timeout never fires and only the absolute
	// bound can be what ends it.
	for i := range 11 {
		clock.advance(time.Hour)
		if _, ok := store.get("s1"); !ok {
			t.Fatalf("session expired early, after %dh", i+1)
		}
	}

	// 12h30m old, but idle for only 30 minutes.
	clock.advance(90 * time.Minute)

	if _, ok := store.get("s1"); ok {
		t.Error("session past the 12h absolute lifetime should be rejected")
	}
	if store.has("s1") {
		t.Error("a rejected session should be deleted, not left in the map")
	}
}

func TestGetRejectsIdleSession(t *testing.T) {
	store, clock := storeAt(t)
	store.save(&Session{ID: "s1"})

	// Well inside the absolute lifetime; only the idle bound is exceeded.
	clock.advance(2*time.Hour + time.Minute)

	if _, ok := store.get("s1"); ok {
		t.Error("session idle for more than 2h should be rejected")
	}
	if store.has("s1") {
		t.Error("a rejected session should be deleted, not left in the map")
	}
}

// The idle window slides, so an actively used session survives far longer than
// the idle timeout itself.
func TestRepeatedUseKeepsSessionAliveBeyondIdleTimeout(t *testing.T) {
	store, clock := storeAt(t)
	store.save(&Session{ID: "s1"})

	for i := range 5 {
		clock.advance(90 * time.Minute) // under the 2h idle bound each time
		if _, ok := store.get("s1"); !ok {
			t.Fatalf("session died at %dh30m despite continuous use", (i+1)*3/2)
		}
	}

	// 7h30m of continuous use: more than three idle windows, still inside the
	// 12h absolute lifetime.
	if _, ok := store.get("s1"); !ok {
		t.Error("actively used session should still be alive after 7h30m")
	}
}

func TestSweepRemovesExpiredSessions(t *testing.T) {
	store, clock := storeAt(t)

	store.save(&Session{ID: "stale"})
	clock.advance(time.Hour)
	store.save(&Session{ID: "recent"})
	clock.advance(90 * time.Minute)

	// stale is 2h30m idle (expired); recent is 1h30m idle (alive).
	store.sweepSessions()

	if store.has("stale") {
		t.Error("reaper should have removed the expired session")
	}
	if !store.has("recent") {
		t.Error("reaper should have kept the live session")
	}
}

// A session abandoned mid-lifetime is only ever freed by the reaper, since
// nothing will call get() on it again.
func TestSweepRemovesSessionPastAbsoluteLifetime(t *testing.T) {
	store, clock := storeAt(t)
	sess := &Session{ID: "s1"}
	store.save(sess)

	// Keep it in use right up to the absolute bound, then let the reaper run
	// while it is still inside the idle window.
	for range 12 {
		clock.advance(time.Hour)
		store.get("s1")
	}
	store.sweepSessions()

	if store.has("s1") {
		t.Error("reaper should remove a session past its absolute lifetime")
	}
}

// save() runs again after every token refresh; that must not renew the
// absolute lifetime, or a long-lived session could never age out.
func TestSaveDoesNotRenewAbsoluteLifetime(t *testing.T) {
	store, clock := storeAt(t)
	sess := &Session{ID: "s1"}
	store.save(sess)

	clock.advance(11 * time.Hour)
	store.save(sess) // as BearerToken does after refreshing

	clock.advance(90 * time.Minute) // 12h30m since creation

	if _, ok := store.get("s1"); ok {
		t.Error("re-saving a session must not extend its absolute lifetime")
	}
}
