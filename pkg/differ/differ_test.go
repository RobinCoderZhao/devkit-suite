package differ

import "testing"

func TestTextDiff_NoChanges(t *testing.T) {
	result := TextDiff("hello\nworld", "hello\nworld")
	if result.HasChanges {
		t.Fatal("expected no changes")
	}
	if result.Summary() != "No changes detected" {
		t.Fatalf("unexpected summary: %s", result.Summary())
	}
}

func TestTextDiff_WithChanges(t *testing.T) {
	old := "line1\nline2\nline3"
	new := "line1\nline2modified\nline3\nline4"
	result := TextDiff(old, new)

	if !result.HasChanges {
		t.Fatal("expected changes")
	}
	if result.Stats.Additions == 0 {
		t.Fatal("expected additions")
	}
	if result.Stats.Deletions == 0 {
		t.Fatal("expected deletions")
	}
	if result.Unified == "" {
		t.Fatal("expected unified diff output")
	}
}

func TestTextDiff_AllNew(t *testing.T) {
	result := TextDiff("", "new content\nhere")
	if !result.HasChanges {
		t.Fatal("expected changes")
	}
	if len(result.Added) < 1 {
		t.Fatal("expected added lines")
	}
}
