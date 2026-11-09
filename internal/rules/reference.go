//go:build reference

package rules

import "errors"

func Apply(s State, a Action) (State, error) {
	reject := func(code string) (State, error) { return s, errors.New(code) }
	if s.Finished {
		return reject("finished")
	}
	if a.Player != s.Turn {
		return reject("not_turn")
	}
	abs := func(n int) int {
		if n < 0 {
			return -n
		}
		return n
	}
	n := s
	switch a.Kind {
	case "move":
		// Range checks precede abs to avoid signed integer overflow.
		if a.DX < -1 || a.DX > 1 || a.DY < -1 || a.DY > 1 || abs(a.DX)+abs(a.DY) != 1 {
			return reject("invalid_action")
		}
		p := Pos{s.Positions[a.Player].X + a.DX, s.Positions[a.Player].Y + a.DY}
		if p.X < 0 || p.X > 4 || p.Y < 0 || p.Y > 4 || p == s.Positions[1-a.Player] {
			return reject("illegal_move")
		}
		n.Positions[a.Player] = p
	case "attack", "pass":
		if a.DX != 0 || a.DY != 0 {
			return reject("invalid_action")
		}
		if a.Kind == "attack" {
			p, q := s.Positions[a.Player], s.Positions[1-a.Player]
			if abs(p.X-q.X)+abs(p.Y-q.Y) != 1 {
				return reject("not_adjacent")
			}
			n.HP[1-a.Player]--
		}
	default:
		return reject("invalid_action")
	}
	n.Ply++
	n.Version++
	if n.HP[1-a.Player] == 0 {
		n.Finished = true
		n.Winner = a.Player
	} else if n.Ply >= 40 {
		n.Finished = true
		n.Winner = -1
		if n.HP[0] > n.HP[1] {
			n.Winner = 0
		} else if n.HP[1] > n.HP[0] {
			n.Winner = 1
		}
	}
	if !n.Finished {
		n.Turn = 1 - n.Turn
	}
	return n, nil
}

func Initial() State {
	return State{Positions: [2]Pos{{0, 0}, {4, 4}}, HP: [2]int{3, 3}, Version: 1, Winner: -1}
}
