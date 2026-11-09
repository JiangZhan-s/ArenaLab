package measurement

import (
	"fmt"
	"math"
	"sort"
)

// One sample per scheduled request, including queue delay and failures.
type Sample struct {
	ID          string  `json:"id"`
	ScheduledMS float64 `json:"scheduled_ms"`
	FinishedMS  float64 `json:"finished_ms"`
	Outcome     string  `json:"outcome"`
}
type Run struct {
	Commit      string   `json:"commit"`
	Environment string   `json:"environment"`
	Variant     string   `json:"variant"`
	Seed        int64    `json:"seed"`
	WindowMS    float64  `json:"window_ms"`
	Expected    int      `json:"expected_requests"`
	Samples     []Sample `json:"samples"`
}
type Summary struct {
	Commit             string  `json:"commit"`
	Environment        string  `json:"environment"`
	Variant            string  `json:"variant"`
	Seed               int64   `json:"seed"`
	Offered            int     `json:"offered"`
	Succeeded          int     `json:"succeeded"`
	Rejected           int     `json:"rejected"`
	Timeouts           int     `json:"timeouts"`
	Errors             int     `json:"errors"`
	SuccessRatio       float64 `json:"success_ratio"`
	OfferedPerSecond   float64 `json:"offered_per_second"`
	CompletedPerSecond float64 `json:"completed_per_second"`
	AllP50MS           float64 `json:"all_p50_ms"`
	AllP95MS           float64 `json:"all_p95_ms"`
	AllP99MS           float64 `json:"all_p99_ms"`
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func Summarize(r Run) (Summary, error) {
	s := Summary{Commit: r.Commit, Environment: r.Environment, Variant: r.Variant, Seed: r.Seed}
	if r.Commit == "" || r.Environment == "" || r.Variant == "" || !finite(r.WindowMS) || r.WindowMS <= 0 || r.Expected < 1 || len(r.Samples) != r.Expected {
		return s, fmt.Errorf("metadata or complete sample coverage missing")
	}
	lat := make([]float64, 0, len(r.Samples))
	seen := map[string]bool{}
	for _, v := range r.Samples {
		if v.ID == "" || seen[v.ID] || !finite(v.ScheduledMS) || !finite(v.FinishedMS) || v.ScheduledMS < 0 || v.ScheduledMS >= r.WindowMS || v.FinishedMS < v.ScheduledMS {
			return s, fmt.Errorf("invalid or duplicate sample %s", v.ID)
		}
		seen[v.ID] = true
		switch v.Outcome {
		case "success":
			s.Succeeded++
		case "rejected":
			s.Rejected++
		case "timeout":
			s.Timeouts++
		case "error":
			s.Errors++
		default:
			return s, fmt.Errorf("unknown outcome %s", v.Outcome)
		}
		lat = append(lat, v.FinishedMS-v.ScheduledMS)
	}
	sort.Float64s(lat)
	q := func(p float64) float64 { return lat[int(math.Ceil(p*float64(len(lat))))-1] }
	s.Offered = len(lat)
	s.SuccessRatio = float64(s.Succeeded) / float64(s.Offered)
	s.OfferedPerSecond = float64(s.Offered) * 1000 / r.WindowMS
	// Include drain time in throughput; never count requests finishing late as
	// if they completed inside the offered-load window.
	end := r.WindowMS
	for _, v := range r.Samples {
		if v.FinishedMS > end {
			end = v.FinishedMS
		}
	}
	s.CompletedPerSecond = float64(s.Succeeded) * 1000 / end
	s.AllP50MS = q(.5)
	s.AllP95MS = q(.95)
	s.AllP99MS = q(.99)
	return s, nil
}
