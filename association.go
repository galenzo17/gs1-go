package gs1

import (
	"fmt"
	"strings"
)

// ValidateAssociations checks the mandatory and invalid Application
// Identifier pairings that the GS1 Barcode Syntax Dictionary defines for
// every AI present: each "req" group needs at least one of its alternatives
// (an alternative such as "01+21" needs every listed AI), and an "ex"
// pattern must not match any other AI. Patterns use "n" as a digit
// wildcard, so 3102 with 3103 fails on "310n". Elements are checked in
// order and the first violation is returned.
func (b Barcode) ValidateAssociations() error {
	for _, element := range b.Elements {
		spec, ok := aiTable[element.AI]
		if !ok {
			continue
		}
		for _, pattern := range spec.Ex {
			for _, other := range b.Elements {
				if other.AI != element.AI && matchAIPattern(pattern, other.AI) {
					return associationError("AI (%s) excludes AI (%s)", element.AI, other.AI)
				}
			}
		}
		for _, group := range spec.Req {
			if !b.anyAlternativePresent(group) {
				return associationError("AI (%s) requires %s", element.AI, describeAlternatives(group))
			}
		}
	}
	return nil
}

// anyAlternativePresent reports whether one alternative of a req group is
// satisfied: every "+"-joined pattern of that alternative matches some AI.
func (b Barcode) anyAlternativePresent(group []string) bool {
	for _, alternative := range group {
		satisfied := true
		for _, pattern := range strings.Split(alternative, "+") {
			if !b.hasAIMatching(pattern) {
				satisfied = false
				break
			}
		}
		if satisfied {
			return true
		}
	}
	return false
}

func (b Barcode) hasAIMatching(pattern string) bool {
	for _, element := range b.Elements {
		if matchAIPattern(pattern, element.AI) {
			return true
		}
	}
	return false
}

// matchAIPattern compares an AI with a dictionary pattern of the same
// length, where "n" matches any digit.
func matchAIPattern(pattern, ai string) bool {
	if len(pattern) != len(ai) {
		return false
	}
	for i := 0; i < len(pattern); i++ {
		if pattern[i] != ai[i] && pattern[i] != 'n' {
			return false
		}
	}
	return true
}

// describeAlternatives renders a req group as "AI (01), AI (02), or AI (8006)",
// with "+" alternatives as "AI (01) with AI (21)".
func describeAlternatives(group []string) string {
	parts := make([]string, len(group))
	for i, alternative := range group {
		patterns := strings.Split(alternative, "+")
		for j, pattern := range patterns {
			patterns[j] = "AI (" + pattern + ")"
		}
		parts[i] = strings.Join(patterns, " with ")
	}
	switch len(parts) {
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " or " + parts[1]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + ", or " + parts[len(parts)-1]
}

func associationError(message string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidAssociation, fmt.Sprintf(message, args...))
}
