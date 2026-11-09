// Package coursegate runs fixed assertions against a separate test driver.
// The driver must exercise real transports/processes; it is not the application.
package coursegate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"
)

type Step struct {
	Op       string           `json:"op,omitempty"`
	Args     map[string]any   `json:"args,omitempty"`
	Want     map[string]any   `json:"want,omitempty"`
	Bounds   map[string]Bound `json:"bounds,omitempty"`
	Parallel []Step           `json:"parallel,omitempty"`
}
type Bound struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}
type Case struct {
	Name  string `json:"name"`
	Steps []Step `json:"steps"`
}
type Suite struct {
	Lab   string `json:"lab"`
	Cases []Case `json:"cases"`
}

func Load(path string) (Suite, error) {
	var s Suite
	b, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err = d.Decode(&s); err != nil {
		return s, err
	}
	if d.Decode(new(any)) != io.EOF {
		return s, fmt.Errorf("trailing JSON")
	}
	if s.Lab == "" || len(s.Cases) == 0 {
		return s, fmt.Errorf("empty suite")
	}
	seen := map[string]bool{}
	for _, c := range s.Cases {
		if c.Name == "" || seen[c.Name] || len(c.Steps) == 0 {
			return s, fmt.Errorf("empty/duplicate case")
		}
		seen[c.Name] = true
		for _, st := range c.Steps {
			if err := validate(st); err != nil {
				return s, err
			}
		}
	}
	return s, nil
}
func validate(s Step) error {
	if len(s.Parallel) > 0 {
		if s.Op != "" || len(s.Want) != 0 || len(s.Args) != 0 || len(s.Bounds) != 0 {
			return fmt.Errorf("mixed parallel step")
		}
		for _, p := range s.Parallel {
			if len(p.Parallel) > 0 {
				return fmt.Errorf("nested parallel")
			}
			if err := validate(p); err != nil {
				return err
			}
		}
		return nil
	}
	if s.Op == "" || len(s.Want) == 0 {
		return fmt.Errorf("operation and independent assertions required")
	}
	for k, b := range s.Bounds {
		if k == "" || b.Min > b.Max {
			return fmt.Errorf("invalid numerical bound")
		}
	}
	return nil
}

// Match compares objects as subsets; arrays, scalar types and lengths are exact.
func Match(want, got any) error {
	if w, ok := want.(map[string]any); ok {
		g, ok := got.(map[string]any)
		if !ok {
			return fmt.Errorf("expected object, got %T", got)
		}
		for k, v := range w {
			actual, exists := g[k]
			if !exists {
				return fmt.Errorf("missing field %s", k)
			}
			if err := Match(v, actual); err != nil {
				return fmt.Errorf("%s: %w", k, err)
			}
		}
		return nil
	}
	if !reflect.DeepEqual(want, got) {
		return fmt.Errorf("want %v (%T), got %v (%T)", want, want, got, got)
	}
	return nil
}

type Runner struct {
	URL, Token string
	Client     *http.Client
}

func New(raw, token string) (*Runner, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, fmt.Errorf("driver must be an http loopback origin")
	}
	if u.Hostname() != "127.0.0.1" && u.Hostname() != "::1" {
		return nil, fmt.Errorf("test driver must use literal loopback address")
	}
	if token == "" {
		return nil, fmt.Errorf("ARENALAB_TEST_TOKEN required; use an isolated disposable database")
	}
	return &Runner{strings.TrimRight(raw, "/"), token, &http.Client{Timeout: 45 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (r *Runner) step(s Step) error {
	if len(s.Parallel) > 0 {
		errs := make([]error, len(s.Parallel))
		var wg sync.WaitGroup
		start := make(chan struct{})
		for i, p := range s.Parallel {
			wg.Add(1)
			go func(i int, p Step) { defer wg.Done(); <-start; errs[i] = r.step(p) }(i, p)
		}
		close(start)
		wg.Wait()
		for i, e := range errs {
			if e != nil {
				return fmt.Errorf("parallel[%d]: %w", i, e)
			}
		}
		return nil
	}
	// Never send Want to the driver: it returns observations, not a pass verdict.
	b, err := json.Marshal(struct {
		Op   string         `json:"op"`
		Args map[string]any `json:"args"`
	}{s.Op, s.Args})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, r.URL+"/execute", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.Client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", s.Op, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: driver HTTP %d", s.Op, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20+1))
	if err != nil {
		return err
	}
	if len(data) > 1<<20 {
		return fmt.Errorf("response too large")
	}
	var got map[string]any
	if err = json.Unmarshal(data, &got); err != nil {
		return fmt.Errorf("%s: invalid observation: %w", s.Op, err)
	}
	if err = Match(s.Want, got); err != nil {
		return fmt.Errorf("%s: %w", s.Op, err)
	}
	for k, b := range s.Bounds {
		v, ok := got[k].(float64)
		if !ok || v < b.Min || v > b.Max {
			return fmt.Errorf("%s: %s outside [%g,%g] or missing: %v", s.Op, k, b.Min, b.Max, got[k])
		}
	}
	return nil
}
func (r *Runner) Run(s Suite, w io.Writer) error {
	failed := 0
	for _, c := range s.Cases {
		var failure error
		for i, st := range c.Steps {
			if err := r.step(st); err != nil {
				failure = fmt.Errorf("step %d: %w", i+1, err)
				break
			}
		}
		if failure != nil {
			failed++
			fmt.Fprintf(w, "FAIL %s/%s: %v\n", s.Lab, c.Name, failure)
		} else {
			fmt.Fprintf(w, "PASS %s/%s\n", s.Lab, c.Name)
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d/%d executable scenarios failed; no business pass", failed, len(s.Cases))
	}
	return nil
}
