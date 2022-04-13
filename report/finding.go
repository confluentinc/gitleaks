package report

import (
	"strings"
)

// Finding contains information about strings that
// have been captured by a tree-sitter query.
type Finding struct {
	Description string
	StartLine   int
	EndLine     int
	StartColumn int
	EndColumn   int

	Match string

	// Secret contains the full content of what is matched in
	// the tree-sitter query.
	Secret string

	// File is the name of the file containing the finding
	File string

	Commit string

	// Entropy is the shannon entropy of Value
	Entropy float32

	Author  string
	Email   string
	Date    string
	Message string
	Tags    []string

	// Rule is the name of the rule that was matched
	RuleID string
}

// Redact removes sensitive information from a finding.
func (f *Finding) Redact() {
	f.Match = strings.Replace(f.Match, f.Secret, "REDACTED", -1)
	f.Secret = "REDACT"
}
