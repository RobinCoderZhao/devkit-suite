// Package git provides Git operations used by DevKit CLI commands.
package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Repo represents a Git repository.
type Repo struct {
	dir string
}

// Open opens the Git repository at the given directory.
func Open(dir string) (*Repo, error) {
	r := &Repo{dir: dir}
	// Verify it's a git repo
	if _, err := r.run(context.Background(), "rev-parse", "--git-dir"); err != nil {
		return nil, fmt.Errorf("not a git repository: %s", dir)
	}
	return r, nil
}

// OpenCurrent opens the Git repository for the current working directory.
func OpenCurrent() (*Repo, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil, fmt.Errorf("not in a git repository")
	}
	return Open(strings.TrimSpace(string(out)))
}

// Diff returns the staged diff (or working tree diff if nothing staged).
type DiffOptions struct {
	Staged    bool
	FilePaths []string
}

// Diff returns the diff output for the repository.
func (r *Repo) Diff(ctx context.Context, opts DiffOptions) (string, error) {
	args := []string{"diff"}
	if opts.Staged {
		args = append(args, "--cached")
	}
	args = append(args, "--")
	args = append(args, opts.FilePaths...)
	return r.run(ctx, args...)
}

// StagedDiff returns the diff of staged changes.
func (r *Repo) StagedDiff(ctx context.Context) (string, error) {
	return r.Diff(ctx, DiffOptions{Staged: true})
}

// WorkingDiff returns the diff of unstaged changes.
func (r *Repo) WorkingDiff(ctx context.Context) (string, error) {
	return r.Diff(ctx, DiffOptions{Staged: false})
}

// Status returns the short status output.
func (r *Repo) Status(ctx context.Context) (string, error) {
	return r.run(ctx, "status", "--short")
}

// StagedFiles returns a list of staged file paths.
func (r *Repo) StagedFiles(ctx context.Context) ([]string, error) {
	out, err := r.run(ctx, "diff", "--cached", "--name-only")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(strings.TrimSpace(out), "\n"), nil
}

// Log returns recent commit messages.
func (r *Repo) Log(ctx context.Context, n int) (string, error) {
	return r.run(ctx, "log", fmt.Sprintf("-%d", n), "--oneline")
}

// Commit creates a new commit with the given message.
func (r *Repo) Commit(ctx context.Context, message string) error {
	_, err := r.run(ctx, "commit", "-m", message)
	return err
}

// AddAll stages all changes.
func (r *Repo) AddAll(ctx context.Context) error {
	_, err := r.run(ctx, "add", "-A")
	return err
}

// CurrentBranch returns the current branch name.
func (r *Repo) CurrentBranch(ctx context.Context) (string, error) {
	out, err := r.run(ctx, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// HasStagedChanges returns true if there are staged changes.
func (r *Repo) HasStagedChanges(ctx context.Context) (bool, error) {
	files, err := r.StagedFiles(ctx)
	if err != nil {
		return false, err
	}
	return len(files) > 0, nil
}

func (r *Repo) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %s %w", strings.Join(args, " "), stderr.String(), err)
	}
	return stdout.String(), nil
}
