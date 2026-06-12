package chaos

import (
	"math/rand"
	"time"
)

type Latency struct {
	DelayInMillieconds  int64
	JitterInMillieconds int64
}

func New(delayInMillieconds int64, jitterInMillieconds int64) *Latency {

	return &Latency{
		DelayInMillieconds:  delayInMillieconds,
		JitterInMillieconds: jitterInMillieconds,
	}
}

func (l *Latency) GetValueInMilliseconds() int64 {

	jitter := rand.Int63n(l.JitterInMillieconds*2) - l.JitterInMillieconds

	delay := time.Duration(l.DelayInMillieconds+jitter) * time.Millisecond

	return delay.Milliseconds()
}
