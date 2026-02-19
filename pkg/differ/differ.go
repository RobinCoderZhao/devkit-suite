// Package differ provides text diffing utilities for comparing document versions.
package differ

import (
	"fmt"
	"strings"
)

// DiffResult holds the result of comparing two text versions.
type DiffResult struct {
	HasChanges bool     `json:"has_changes"`
	Added      []string `json:"added"`
	Removed    []string `json:"removed"`
	Unified    string   `json:"unified"`
	Stats      Stats    `json:"stats"`
}

// Stats holds counts of changes.
type Stats struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
}

// TextDiff computes a line-by-line diff between old and new text.
func TextDiff(oldText, newText string) DiffResult {
	if oldText == newText {
		return DiffResult{HasChanges: false}
	}

	oldLines := strings.Split(oldText, "\n")
	newLines := strings.Split(newText, "\n")

	oldSet := make(map[string]bool, len(oldLines))
	newSet := make(map[string]bool, len(newLines))
	for _, line := range oldLines {
		oldSet[line] = true
	}
	for _, line := range newLines {
		newSet[line] = true
	}

	var added, removed []string
	for _, line := range oldLines {
		if !newSet[line] && strings.TrimSpace(line) != "" {
			removed = append(removed, line)
		}
	}
	for _, line := range newLines {
		if !oldSet[line] && strings.TrimSpace(line) != "" {
			added = append(added, line)
		}
	}

	// Build unified diff
	var sb strings.Builder
	sb.WriteString("--- old\n+++ new\n")
	for _, line := range removed {
		sb.WriteString(fmt.Sprintf("-%s\n", line))
	}
	for _, line := range added {
		sb.WriteString(fmt.Sprintf("+%s\n", line))
	}

	return DiffResult{
		HasChanges: true,
		Added:      added,
		Removed:    removed,
		Unified:    sb.String(),
		Stats: Stats{
			Additions: len(added),
			Deletions: len(removed),
		},
	}
}

// Summary returns a human-readable summary of the diff.
func (d DiffResult) Summary() string {
	if !d.HasChanges {
		return "No changes detected"
	}
	return fmt.Sprintf("%d additions, %d deletions", d.Stats.Additions, d.Stats.Deletions)
}
