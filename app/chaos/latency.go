// Package chaos provides fault-injection primitives for crashdummy.
package chaos

import (
	"math/rand/v2"
	"time"
)

// Latency produces randomized delays around a base duration.
type Latency struct {
	DelayInMilliseconds  int64
	JitterInMilliseconds int64
}

// New returns a Latency with the given base delay and jitter, both in
// milliseconds.
func New(delayInMilliseconds, jitterInMilliseconds int64) *Latency {
	return &Latency{
		DelayInMilliseconds:  delayInMilliseconds,
		JitterInMilliseconds: jitterInMilliseconds,
	}
}

// Duration returns the base delay shifted by a random jitter in
// [-jitter, +jitter). A zero or negative jitter yields the base delay
// unchanged, and the result never goes below zero.
func (l *Latency) Duration() time.Duration {
	delay := l.DelayInMilliseconds
	if l.JitterInMilliseconds > 0 {
		delay += rand.Int64N(l.JitterInMilliseconds*2) - l.JitterInMilliseconds
	}
	if delay < 0 {
		delay = 0
	}
	return time.Duration(delay) * time.Millisecond
}
