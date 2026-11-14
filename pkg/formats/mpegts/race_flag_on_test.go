//go:build race

package mpegts_test

// raceEnabled reports whether the test binary was built with the race
// detector. Large in-memory streams are dramatically slower under
// instrumentation, so tests use smaller frames in that case: the code paths
// under test are identical, while the full-size coverage always runs in the
// regular (non-race) build.
func raceEnabled() bool {
	return true
}
