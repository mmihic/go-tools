package pkgs

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mmihic/go-tools/pkg/path"
)

// A Package move describes a move of a package from one location to another.
type Package struct {
	From path.Path `yaml:"from"`
	To   path.Path `yaml:"to"`
}

// UnmarshalYAML unmarshals the package move from YAML
func (mv *Package) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	parsed, err := ParsePackageMove(s)
	if err != nil {
		return err
	}

	*mv = *parsed
	return nil
}

// ParsePackageMove parses a package move.
func ParsePackageMove(s string) (*Package, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid package move %s", s)
	}

	var (
		from = path.NewPath(parts[0])
		to   = path.NewPath(parts[1])
	)

	return &Package{
		From: from,
		To:   to,
	}, nil
}

// String returns the string form of the package move.
func (mv *Package) String() string {
	return fmt.Sprintf("%30s -> %30s", mv.From, mv.To)
}

// Rewrite rewrites the given path from the source package layout to the target
// package layout.
func (mv *Package) Rewrite(path path.Path) (path.Path, error) {
	if !mv.From.Contains(path) {
		return nil, fmt.Errorf("%s does not contain %s", mv.From, path)
	}

	return mv.To.Append(path[len(mv.From):]), nil
}

// ApplyPrefix applies a prefix to the rules.
func (mv *Package) ApplyPrefix(prefix path.Path) *Package {
	return &Package{
		From: prefix.Append(mv.From),
		To:   prefix.Append(mv.To),
	}
}

// Packages is a list of package moves.
type Packages []*Package

// ParsePackageMoves parses a set of package moves.
func ParsePackageMoves(rulesList []string) (Packages, error) {
	rules := make(Packages, 0, len(rulesList))
	for _, r := range rulesList {
		rule, err := ParsePackageMove(r)
		if err != nil {
			return nil, err
		}

		rules = append(rules, rule)
	}

	sort.Sort(rules)
	return rules, nil
}

// UnmarshalYAML unmarshals a set of rules from YAML.
func (moves *Packages) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var rulesList []string
	if err := unmarshal(&rulesList); err != nil {
		return err
	}

	parsed, err := ParsePackageMoves(rulesList)
	if err != nil {
		return err
	}

	*moves = parsed
	return nil
}

// Len returns the number of rules.
func (moves Packages) Len() int { return len(moves) }

// Swap swaps two rules.
func (moves Packages) Swap(i, j int) { moves[i], moves[j] = moves[j], moves[i] }

// Less compares two rules.
func (moves Packages) Less(i, j int) bool {
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
func (moves Packages) BestMatch(p path.Path) *Package {
	var matches Packages
	for _, rule := range moves {
		if rule.From.Contains(p) {
			matches = append(matches, rule)
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
func (moves Packages) ExactMatch(p path.Path) *Package {
	for _, rule := range moves {
		if rule.From.Equal(p) {
			return rule
		}
	}

	return nil
}

// ApplyPrefix applies a prefix to all rules, returning a new set of rules
func (moves Packages) ApplyPrefix(prefix path.Path) Packages {
	newRules := make(Packages, len(moves))
	for i, rule := range moves {
		newRules[i] = rule.ApplyPrefix(prefix)
	}

	return newRules
}