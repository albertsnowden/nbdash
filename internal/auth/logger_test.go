package auth

import "log/slog"

// testLogger is the discarding logger the tests here hand to New. The
// Authenticator logs failed logins and refresh failures; none of these tests
// assert on that output, so it goes nowhere rather than into the test log.
func testLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }
