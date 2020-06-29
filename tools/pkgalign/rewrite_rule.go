package pkgalign

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mmihic/go-tools/pkg/path"
)

// RewriteRule tells us which imports to rewrite.
type RewriteRule struct {
	From path.Path `yaml:"from"`
	To   path.Path `yaml:"to"`
}

// UnmarshalYAML unmarshals the rewrite rule from YAML
func (rule *RewriteRule) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	parsed, err := ParseRewriteRule(s)
	if err != nil {
		return err
	}

	*rule = *parsed
	return nil
}

// ParseRewriteRule parses a rewrite rule.
func ParseRewriteRule(s string) (*RewriteRule, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid rewrite rule %s", s)
	}

	var (
		from = path.NewPath(parts[0])
		to   = path.NewPath(parts[1])
	)

	return &RewriteRule{
		From: from,
		To:   to,
	}, nil
}

// String returns the string form of the rewrite rule.
func (rule *RewriteRule) String() string {
	return fmt.Sprintf("%30s -> %30s", rule.From, rule.To)
}

// Rewrite rewrites the given path from the source package layout to the target
// package layout.
func (rule *RewriteRule) Rewrite(path path.Path) (path.Path, error) {
	if !rule.From.Contains(path) {
		return nil, fmt.Errorf("%s does not contain %s", rule.From, path)
	}

	return rule.To.Append(path[len(rule.From):]), nil
}

// ApplyPrefix applies a prefix to the rules.
func (rule *RewriteRule) ApplyPrefix(prefix path.Path) *RewriteRule {
	return &RewriteRule{
		From: prefix.Append(rule.From),
		To:   prefix.Append(rule.To),
	}
}

// RewriteRules is a list of rewrite rules.
type RewriteRules []*RewriteRule

// ParseRewriteRules parses a set of rewrite rules.
func ParseRewriteRules(rulesList []string) (RewriteRules, error) {
	rules := make(RewriteRules, 0, len(rulesList))
	for _, r := range rulesList {
		rule, err := ParseRewriteRule(r)
		if err != nil {
			return nil, err
		}

		rules = append(rules, rule)
	}

	sort.Sort(rules)
	return rules, nil
}

// UnmarshalYAML unmarshals a set of rules from YAML.
func (rules *RewriteRules) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var rulesList []string
	if err := unmarshal(&rulesList); err != nil {
		return err
	}

	parsed, err := ParseRewriteRules(rulesList)
	if err != nil {
		return err
	}

	*rules = parsed
	return nil
}

// Len returns the number of rules.
func (rules RewriteRules) Len() int { return len(rules) }

// Swap swaps two rules.
func (rules RewriteRules) Swap(i, j int) { rules[i], rules[j] = rules[j], rules[i] }

// Less compares two rules.
func (rules RewriteRules) Less(i, j int) bool {
	if len(rules[i].From) < len(rules[j].From) {
		return true
	}

	if len(rules[i].From) > len(rules[j].From) {
		return false
	}

	return rules[i].String() < rules[j].String()
}

// BestMatch returns the rule that most specifically matches the given path, or nil
// if no rules match.
func (rules RewriteRules) BestMatch(p path.Path) *RewriteRule {
	var matches RewriteRules
	for _, rule := range rules {
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
func (rules RewriteRules) ExactMatch(p path.Path) *RewriteRule {
	for _, rule := range rules {
		if rule.From.Equal(p) {
			return rule
		}
	}

	return nil
}

// ApplyPrefix applies a prefix to all rules, returning a new set of rules
func (rules RewriteRules) ApplyPrefix(prefix path.Path) RewriteRules {
	newRules := make(RewriteRules, len(rules))
	for i, rule := range rules {
		newRules[i] = rule.ApplyPrefix(prefix)
	}

	return newRules
}