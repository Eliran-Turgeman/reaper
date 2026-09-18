package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandCollectorWorkingStagedAndReferenceDiffs(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "--quiet")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
	file := filepath.Join(dir, "value.txt")
	if err := os.WriteFile(file, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "value.txt")
	runGit(t, dir, "commit", "--quiet", "-m", "initial")
	if err := os.WriteFile(file, []byte("two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	collector := CommandCollector{Dir: dir}
	working, root, err := collector.Diff(context.Background(), Options{})
	if err != nil || !strings.Contains(working, "+two") || root == "" {
		t.Fatalf("working diff failed: root=%q err=%v diff=%s", root, err, working)
	}
	staged, _, err := collector.Diff(context.Background(), Options{Staged: true})
	if err != nil || staged != "" {
		t.Fatalf("expected empty staged diff: err=%v diff=%s", err, staged)
	}
	runGit(t, dir, "add", "value.txt")
	staged, _, err = collector.Diff(context.Background(), Options{Staged: true})
	if err != nil || !strings.Contains(staged, "+two") {
		t.Fatalf("staged diff failed: err=%v diff=%s", err, staged)
	}
	againstHead, _, err := collector.Diff(context.Background(), Options{Ref: "HEAD"})
	if err != nil || !strings.Contains(againstHead, "+two") {
		t.Fatalf("reference diff failed: err=%v diff=%s", err, againstHead)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}
