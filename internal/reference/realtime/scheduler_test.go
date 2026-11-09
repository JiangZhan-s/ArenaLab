//go:build reference

package realtime

import (
	"testing"
	"time"
)

func TestBoundedCatchup(t *testing.T) {
	now := time.Unix(0, 0)
	var s Scheduler
	s.Reset(now)
	if n, _, _ := s.Due(now.Add(49 * time.Millisecond)); n != 0 {
		t.Fatal(n)
	}
	if n, _, drop := s.Due(now.Add(50 * time.Millisecond)); n != 1 || drop != 0 {
		t.Fatal(n, drop)
	}
	if n, lag, drop := s.Due(now.Add(time.Second)); n != 3 || drop != 16 || lag != 900*time.Millisecond {
		t.Fatal(n, lag, drop)
	}
	if n, _, _ := s.Due(now.Add(time.Second)); n != 0 {
		t.Fatal("unbounded catchup", n)
	}
}
