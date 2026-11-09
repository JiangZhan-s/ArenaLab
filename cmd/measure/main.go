package main

import (
	"encoding/json"
	"fmt"
	"github.com/JiangZhan-s/ArenaLab/internal/measurement"
	"io"
	"os"
)

func run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: go run ./cmd/measure artifacts/raw/run.json")
	}
	f, e := os.Open(args[0])
	if e != nil {
		return e
	}
	defer f.Close()
	var r measurement.Run
	d := json.NewDecoder(f)
	d.DisallowUnknownFields()
	if e = d.Decode(&r); e != nil {
		return e
	}
	if d.Decode(new(any)) != io.EOF {
		return fmt.Errorf("trailing input")
	}
	s, e := measurement.Summarize(r)
	if e != nil {
		return e
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(s)
}
func main() {
	if e := run(os.Args[1:]); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
