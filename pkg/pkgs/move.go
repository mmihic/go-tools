package pkgs

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mmihic/go-tools/pkg/path"
)

// A Move describes a move of a package from one location to another.
type Move struct {
	From path.Path `yaml:"from"`
	To   path.Path `yaml:"to"`
}

// UnmarshalYAML unmarshals the package move from YAML
func (mv *Move) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	parsed, err := ParseMove(s)
	if err != nil {
		return err
	}

	*mv = *parsed
	return nil
}

// ParseMove parses a package move.
func ParseMove(s string) (*Move, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid package move %s", s)
	}

	var (
		from = path.NewPath(parts[0])
		to   = path.NewPath(parts[1])
	)

	return &Move{
		From: from,
		To:   to,
	}, nil
}

// String returns the string form of the package move.
func (mv *Move) String() string {
	return fmt.Sprintf("%30s -> %30s", mv.From, mv.To)
}

// Rewrite rewrites the given path from the source package layout to the target
// package layout.
func (mv *Move) Rewrite(path path.Path) (path.Path, error) {
	if !mv.From.Contains(path) {
		return nil, fmt.Errorf("%s does not contain %s", mv.From, path)
	}

	return mv.To.Append(path[len(mv.From):]), nil
}

// ApplyPrefix applies a prefix to the rules.
func (mv *Move) ApplyPrefix(prefix path.Path) *Move {
	return &Move{
		From: prefix.Append(mv.From),
		To:   prefix.Append(mv.To),
	}
}

// Moves is a list of package moves.
type Moves []*Move

// ParseMoves parses a set of package moves.
func ParseMoves(mvList []string) (Moves, error) {
	moves := make(Moves, 0, len(mvList))
	for _, r := range mvList {
		rule, err := ParseMove(r)
		if err != nil {
			return nil, err
		}

		moves = append(moves, rule)
	}

	sort.Sort(moves)
	return moves, nil
}

// UnmarshalYAML unmarshals a set of rules from YAML.
func (moves *Moves) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var mvList []string
	if err := unmarshal(&mvList); err != nil {
		return err
	}

	parsed, err := ParseMoves(mvList)
	if err != nil {
		return err
	}

	*moves = parsed
	return nil
}

// Len returns the number of rules.
func (moves Moves) Len() int { return len(moves) }

// Swap swaps two rules.
func (moves Moves) Swap(i, j int) { moves[i], moves[j] = moves[j], moves[i] }

// Less compares two rules.
func (moves Moves) Less(i, j int) bool {
	if len(moves[i].From) < len(moves[j].From) {
		return true
	}

	if len(moves[i].From) > len(moves[j].From) {
		return false
	}

	return moves[i].String() < moves[j].String()
}

// BestMatch returns the rule that most specifically matches the given path, or nil
// if no rules match.
func (moves Moves) BestMatch(p path.Path) *Move {
	var matches Moves
	for _, mv := range moves {
		if mv.From.Contains(p) {
			matches = append(matches, mv)
		}
	}

	if len(matches) == 0 {
		return nil
	}

	sort.Sort(matches)
	return matches[len(matches)-1]
}

// ExactMatch returns the rule that exactly matches the given path, or nil if no
// rule matches.
func (moves Moves) ExactMatch(p path.Path) *Move {
	for _, mv := range moves {
		if mv.From.Equal(p) {
			return mv
		}
	}

	return nil
}

// ApplyPrefix applies a prefix to all rules, returning a new set of rules
func (moves Moves) ApplyPrefix(prefix path.Path) Moves {
	newMoves := make(Moves, len(moves))
	for i, mv := range moves {
		newMoves[i] = mv.ApplyPrefix(prefix)
	}

	return newMoves
}
