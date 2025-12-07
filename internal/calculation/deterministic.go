package calculation

import "time"

// nowFunc returns the current time (override in tests for determinism).
var nowFunc = time.Now

// SetNowFunc overrides the time provider (use only in tests).
func SetNowFunc(f func() time.Time) { nowFunc = f }

// seedFunc returns a pseudo-random seed (override for deterministic Monte Carlo tests).
var seedFunc = func() int64 { return time.Now().UnixNano() }

// SetSeedFunc overrides the seed provider (use only in tests).
func SetSeedFunc(f func() int64) { seedFunc = f }
