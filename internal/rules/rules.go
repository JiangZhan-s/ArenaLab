// Package rules contains shared rule types. Apply is selected by build tag.
package rules

import "errors"

var ErrNotImplemented = errors.New("lab01: Apply not implemented")

type Pos struct {
	X int `json:"x"`
	Y int `json:"y"`
}
type State struct {
	Positions [2]Pos `json:"positions"`
	HP        [2]int `json:"hp"`
	Turn      int    `json:"turn"`
	Ply       int    `json:"ply"`
	Version   int    `json:"version"`
	Finished  bool   `json:"finished"`
	Winner    int    `json:"winner"`
}
type Action struct {
	Player int    `json:"player"`
	Kind   string `json:"kind"`
	DX     int    `json:"dx"`
	DY     int    `json:"dy"`
}
