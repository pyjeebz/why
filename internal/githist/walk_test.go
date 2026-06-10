package githist

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pyjeebz/why/internal/target"
)

// fixtureRepo builds a tiny repo where line 2 of notes.txt changes in the
// second commit, so a line-log walk should find both commits.
func fixtureRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=t@example.com",
			"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=t@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	write := func(content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	git("init", "-b", "main")
	write("alpha\ntimeout=10\nomega\n")
	git("add", ".")
	git("commit", "-m", "initial notes")
	write("alpha\ntimeout=30\nomega\n")
	git("add", ".")
	git("commit", "-m", "raise timeout after flaky CI", "-m", "10s was too tight on slow runners.")
	return dir
}

func TestWalkLineRange(t *testing.T) {
	dir := fixtureRepo(t)

	commits, err := Walk(dir, target.Target{Path: "notes.txt", Start: 2, End: 2}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 2 {
		t.Fatalf("got %d commits, want 2: %+v", len(commits), commits)
	}
	if commits[0].Subject != "raise timeout after flaky CI" {
		t.Errorf("newest commit subject = %q", commits[0].Subject)
	}
	if commits[0].Body != "10s was too tight on slow runners." {
		t.Errorf("newest commit body = %q", commits[0].Body)
	}
	if commits[1].Subject != "initial notes" {
		t.Errorf("oldest commit subject = %q", commits[1].Subject)
	}
}

func TestWalkWholeFile(t *testing.T) {
	dir := fixtureRepo(t)

	commits, err := Walk(dir, target.Target{Path: "notes.txt"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 2 {
		t.Fatalf("got %d commits, want 2", len(commits))
	}
}

func TestWalkMaxHops(t *testing.T) {
	dir := fixtureRepo(t)

	commits, err := Walk(dir, target.Target{Path: "notes.txt"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 1 {
		t.Fatalf("got %d commits, want 1", len(commits))
	}
}

func TestWalkUntrackedFile(t *testing.T) {
	dir := fixtureRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Walk(dir, target.Target{Path: "new.txt", Start: 1, End: 1}, 10)
	if err == nil {
		t.Fatal("expected error for untracked file")
	}
}

func TestRepoRoot(t *testing.T) {
	dir := fixtureRepo(t)
	root, err := RepoRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if evalRoot, _ := filepath.EvalSymlinks(root); evalRoot == "" {
		t.Fatalf("empty repo root")
	}

	if _, err := RepoRoot(os.TempDir()); err == nil {
		t.Skip("os.TempDir unexpectedly inside a git repo")
	}
}
