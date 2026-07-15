// Package chaos provides fault-injection primitives for crashdummy:
// randomized latency and probabilistic error injection. A Chaos value is
// safe for concurrent use, so request handlers can read it while the admin
// API updates it at runtime.
package chaos

import (
	"math/rand/v2"
	"sync"
	"time"
)

const defaultErrorStatus = 500

// Spec is the externally settable fault configuration for a single route.
type Spec struct {
	LatencyInMilliseconds int64   `json:"latencyInMilliseconds"`
	JitterInMilliseconds  int64   `json:"jitterInMilliseconds"`
	ErrorRate             float64 `json:"errorRate"`
	ErrorStatus           int     `json:"errorStatus"`
}

// Chaos applies randomized latency and probabilistic error injection to a
// route.
type Chaos struct {
	mu   sync.RWMutex
	spec Spec
}

// New returns a Chaos configured from spec, normalized to safe defaults.
func New(spec Spec) *Chaos {
	return &Chaos{spec: normalize(spec)}
}

// normalize clamps a Spec to safe ranges: non-negative delays, an error
// rate in [0,1], and an error status in the valid HTTP range.
func normalize(spec Spec) Spec {
	if spec.LatencyInMilliseconds < 0 {
		spec.LatencyInMilliseconds = 0
	}
	if spec.JitterInMilliseconds < 0 {
		spec.JitterInMilliseconds = 0
	}
	if spec.ErrorRate < 0 {
		spec.ErrorRate = 0
	}
	if spec.ErrorRate > 1 {
		spec.ErrorRate = 1
	}
	if spec.ErrorStatus < 100 || spec.ErrorStatus > 599 {
		spec.ErrorStatus = defaultErrorStatus
	}
	return spec
}

// Update replaces the fault configuration at runtime.
func (c *Chaos) Update(spec Spec) {
	spec = normalize(spec)
	c.mu.Lock()
	c.spec = spec
	c.mu.Unlock()
}

// Snapshot returns the current fault configuration.
func (c *Chaos) Snapshot() Spec {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.spec
}

// Duration returns the base delay shifted by a random jitter in
// [-jitter, +jitter). A zero or negative jitter yields the base delay
// unchanged, and the result never goes below zero.
func (c *Chaos) Duration() time.Duration {
	c.mu.RLock()
	delay := c.spec.LatencyInMilliseconds
	jitter := c.spec.JitterInMilliseconds
	c.mu.RUnlock()

	if jitter > 0 {
		// #nosec G404 -- fault-injection jitter is not security-sensitive; a
		// non-cryptographic RNG is intended here.
		delay += rand.Int64N(jitter*2) - jitter
	}
	if delay < 0 {
		delay = 0
	}
	return time.Duration(delay) * time.Millisecond
}

// ShouldFail reports whether this request should be failed, and with which
// status. It returns false when the error rate is zero or the random draw
// falls outside it.
func (c *Chaos) ShouldFail() (bool, int) {
	c.mu.RLock()
	rate := c.spec.ErrorRate
	status := c.spec.ErrorStatus
	c.mu.RUnlock()

	if rate <= 0 {
		return false, 0
	}
	// #nosec G404 -- fault-injection sampling is not security-sensitive; a
	// non-cryptographic RNG is intended here.
	if rand.Float64() < rate {
		return true, status
	}
	return false, 0
}
