package tcp

import (
	"testing"
	"time"
)

// TestCapRTOBackoff pins the overlay RTO backoff cap: it must only shorten an
// RTO that exceeds the cap, and a zero cap must preserve RFC behavior.
func TestCapRTOBackoff(t *testing.T) {
	for _, tc := range []struct {
		name string
		rto  time.Duration
		cap  time.Duration
		want time.Duration
	}{
		{"disabled", 2 * time.Second, 0, 2 * time.Second},
		{"clamped", 2 * time.Second, time.Second, time.Second},
		{"below", 500 * time.Millisecond, time.Second, 500 * time.Millisecond},
		{"equal", time.Second, time.Second, time.Second},
		{"zero-rto", 0, time.Second, 0},
	} {
		if got := capRTOBackoff(tc.rto, tc.cap); got != tc.want {
			t.Errorf("%s: capRTOBackoff(%v, %v) = %v, want %v", tc.name, tc.rto, tc.cap, got, tc.want)
		}
	}
}
