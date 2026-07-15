package chaos

import (
	"testing"
	"time"
)

func TestDurationNoJitterReturnsBase(t *testing.T) {
	c := New(Spec{LatencyInMilliseconds: 250})
	for range 100 {
		if got := c.Duration(); got != 250*time.Millisecond {
			t.Fatalf("Duration() = %v, want 250ms", got)
		}
	}
}

func TestDurationStaysWithinJitterBand(t *testing.T) {
	c := New(Spec{LatencyInMilliseconds: 200, JitterInMilliseconds: 50})
	low, high := 150*time.Millisecond, 250*time.Millisecond
	for range 1000 {
		got := c.Duration()
		if got < low || got >= high {
			t.Fatalf("Duration() = %v, want in [%v, %v)", got, low, high)
		}
	}
}

func TestDurationNeverNegative(t *testing.T) {
	// Jitter wider than the base could push a naive delay below zero.
	c := New(Spec{LatencyInMilliseconds: 10, JitterInMilliseconds: 100})
	for range 1000 {
		if got := c.Duration(); got < 0 {
			t.Fatalf("Duration() = %v, want >= 0", got)
		}
	}
}

func TestShouldFailRateZeroNeverFails(t *testing.T) {
	c := New(Spec{ErrorRate: 0, ErrorStatus: 503})
	for range 1000 {
		if fail, _ := c.ShouldFail(); fail {
			t.Fatal("ShouldFail() = true at rate 0, want false")
		}
	}
}

func TestShouldFailRateOneAlwaysFails(t *testing.T) {
	c := New(Spec{ErrorRate: 1, ErrorStatus: 503})
	for range 1000 {
		fail, status := c.ShouldFail()
		if !fail {
			t.Fatal("ShouldFail() = false at rate 1, want true")
		}
		if status != 503 {
			t.Fatalf("ShouldFail() status = %d, want 503", status)
		}
	}
}

func TestShouldFailRateIsRoughlyProportional(t *testing.T) {
	c := New(Spec{ErrorRate: 0.3, ErrorStatus: 500})
	const n = 10000
	fails := 0
	for range n {
		if fail, _ := c.ShouldFail(); fail {
			fails++
		}
	}
	rate := float64(fails) / n
	if rate < 0.25 || rate > 0.35 {
		t.Fatalf("observed fail rate = %.3f, want ~0.30", rate)
	}
}

func TestNormalizeClampsSpec(t *testing.T) {
	c := New(Spec{
		LatencyInMilliseconds: -100,
		JitterInMilliseconds:  -5,
		ErrorRate:             2.5,
		ErrorStatus:           99,
	})
	got := c.Snapshot()
	want := Spec{
		LatencyInMilliseconds: 0,
		JitterInMilliseconds:  0,
		ErrorRate:             1,
		ErrorStatus:           defaultErrorStatus,
	}
	if got != want {
		t.Fatalf("normalized spec = %+v, want %+v", got, want)
	}
}

func TestUpdateReplacesSpec(t *testing.T) {
	c := New(Spec{})
	c.Update(Spec{LatencyInMilliseconds: 500, ErrorRate: 0.5, ErrorStatus: 502})
	got := c.Snapshot()
	if got.LatencyInMilliseconds != 500 || got.ErrorRate != 0.5 || got.ErrorStatus != 502 {
		t.Fatalf("after Update, snapshot = %+v", got)
	}
}
