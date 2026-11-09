//go:build reference

package realtime

import (
	"sync"
	"testing"
)

func TestSequenceAndCapacity(t *testing.T) {
	g := New()
	if g.Input(0, Input{1, 1, 0}) != "" {
		t.Fatal("input")
	}
	if g.Input(0, Input{1, 1, 0}) != "duplicate_input" {
		t.Fatal("duplicate")
	}
	if g.Input(0, Input{2, 100, 0}) != "invalid_input" {
		t.Fatal("speed")
	}
	s := g.Step()
	if s.X[0] != 1 || s.Processed[0] != 1 {
		t.Fatal(s)
	}
	for n := 2; n <= 129; n++ {
		if g.Input(0, Input{n, 0, 0}) != "" {
			t.Fatal(n)
		}
	}
	if g.Input(0, Input{130, 0, 0}) != "input_window_exceeded" {
		t.Fatal("unbounded")
	}
}
func TestPredictionDoesNotReplayAcknowledgedInput(t *testing.T) {
	r := Reconciler{}
	r.Predict(Input{1, 1, 0})
	r.Predict(Input{2, 1, 0})
	r.Install(Snapshot{Tick: 1, X: [2]int{1, 99}, Processed: [2]int{1, 0}}, 0)
	if r.X != 2 || len(r.Pending) != 1 {
		t.Fatal(r)
	}
	r.Install(Snapshot{Tick: 2, X: [2]int{2, 99}, Processed: [2]int{2, 0}}, 0)
	if r.X != 2 || len(r.Pending) != 0 {
		t.Fatal(r)
	}
}
func TestConcurrentTickAndInput(t *testing.T) {
	g := New()
	var wg sync.WaitGroup
	for p := 0; p < 2; p++ {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			for seq := 1; seq < 500; seq++ {
				g.Input(p, Input{seq, 0, 0})
				g.Snapshot()
			}
		}(p)
	}
	for i := 0; i < 600; i++ {
		g.Step()
	}
	wg.Wait()
	if !g.Snapshot().Finished {
		t.Fatal("not finished")
	}
}
