//go:build !reference

package rules

// Implement in Lab01. Teacher code requires the explicit reference build tag.
func Apply(s State, a Action) (State, error) { return s, ErrNotImplemented }
