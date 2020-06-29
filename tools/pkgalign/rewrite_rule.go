package pkgalign

import (
	"fmt"
	"sort"
	"strings"
)

// RewriteRule tells us which imports to rewrite.
type RewriteRule struct {
	From Path `yaml:"from"`
	To   Path `yaml:"to"`
}

// ParseRewriteRule parses a rewrite rule.
func ParseRewriteRule(pathPrefix Path, s string) (*RewriteRule, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid rewrite rule %s", s)
	}

	var (
		from = NewPath(parts[0])
		to   = NewPath(parts[1])
	)
	if len(pathPrefix) != 0 {
		from = pathPrefix.Append(from)
		to = pathPrefix.Append(to)
	}

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
func (rule *RewriteRule) Rewrite(path Path) (Path, error) {
	if !rule.From.Contains(path) {
		return nil, fmt.Errorf("%s does not contain %s", rule.From, path)
	}

	return rule.To.Append(path[len(rule.From):]), nil
}

// RewriteRules is a list of rewrite rules.
type RewriteRules []*RewriteRule

// ParseRewriteRules parses a set of rewrite rules.
func ParseRewriteRules(pathPrefix Path, rulesList []string) (RewriteRules, error) {
	rules := make(RewriteRules, 0, len(rulesList))
	for _, r := range rulesList {
		rule, err := ParseRewriteRule(pathPrefix, r)
		if err != nil {
			return nil, err
		}

		rules = append(rules, rule)
	}

	sort.Sort(rules)
	return rules, nil
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
func (rules RewriteRules) BestMatch(p Path) *RewriteRule {
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
func (rules RewriteRules) ExactMatch(p Path) *RewriteRule {
	for _, rule := range rules {
		if rule.From.Equal(p) {
			return rule
		}
	}

	return nil
}
