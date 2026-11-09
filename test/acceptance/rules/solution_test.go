//go:build acceptance

package acceptance

import (
	"github.com/JiangZhan-s/ArenaLab/internal/rules"
	"testing"
)

func TestRuleAcceptance(t *testing.T) {
	for _, f := range cases(t) {
		t.Run(f.Name, func(t *testing.T) {
			got, err := rules.Apply(f.Before, f.Action)
			if !matches(f, got, err) {
				t.Fatalf("state=%+v error=%v; want=%+v error=%q", got, err, f.After, f.Error)
			}
		})
	}
}
