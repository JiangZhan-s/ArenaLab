package measurement

import "testing"

func TestIncludesErrorsAndDrainTime(t *testing.T) {
	r := Run{Commit: "test", Environment: "synthetic", Variant: "test", WindowMS: 100, Expected: 3, Samples: []Sample{{"a", 0, 10, "success"}, {"b", 10, 110, "success"}, {"c", 20, 220, "timeout"}}}
	s, e := Summarize(r)
	if e != nil {
		t.Fatal(e)
	}
	if s.AllP99MS != 200 || s.Timeouts != 1 || s.Offered != 3 || s.CompletedPerSecond != 2000.0/220 {
		t.Fatalf("%+v", s)
	}
	r.Expected = 4
	if _, e = Summarize(r); e == nil {
		t.Fatal("missing timeout sample accepted")
	}
}
func TestInvalidSamples(t *testing.T) {
	for _, samples := range [][]Sample{{{"a", 0, -1, "success"}}, {{"a", 0, 1, "unknown"}}, {{"a", 0, 1, "success"}, {"a", 1, 2, "success"}}} {
		r := Run{Commit: "x", Environment: "x", Variant: "x", WindowMS: 100, Expected: len(samples), Samples: samples}
		if _, e := Summarize(r); e == nil {
			t.Fatal(samples)
		}
	}
}
