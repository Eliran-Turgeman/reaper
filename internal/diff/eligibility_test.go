package diff_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Eliran-Turgeman/reaper/internal/diff"
	"github.com/Eliran-Turgeman/reaper/internal/rules"
)

func TestExistingFileEligibilityUsesGitFileHistory(t *testing.T) {
	before := "package service\nfunc Read(fast bool) error {\n if !allowed() { return denied }\n return read()\n}\n"
	for _, tc := range []struct {
		name, oldPath, newPath, before, after string
		wantExisting                          bool
	}{
		{"inserted bypass", "service.go", "service.go", before, "package service\nfunc Read(fast bool) error {\n if fast { return read() }\n if !allowed() { return denied }\n return read()\n}\n", true},
		{"guard moved after operation", "service.go", "service.go", before, "package service\nfunc Read(fast bool) error {\n err := read()\n if !allowed() { return denied }\n return err\n}\n", true},
		{"new file", "", "service.go", "", before, false},
		{"existing empty file", "service.go", "service.go", "", before, true},
		{"modified rename", "old.go", "service.go", before, before + "// retained operation\n", true},
		{"deleted file", "service.go", "", before, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			run := func(args ...string) []byte {
				t.Helper()
				cmd := exec.Command("git", append([]string{"-C", root, "-c", "core.autocrlf=false"}, args...)...)
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("git %v: %v: %s", args, err, out)
				}
				return out
			}
			write := func(name, content string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			run("init", "--quiet")
			if tc.oldPath != "" {
				write(tc.oldPath, tc.before)
				run("add", "--", tc.oldPath)
			}
			if tc.oldPath != "" && tc.oldPath != tc.newPath {
				if err := os.Remove(filepath.Join(root, tc.oldPath)); err != nil {
					t.Fatal(err)
				}
			}
			if tc.newPath != "" {
				write(tc.newPath, tc.after)
				if tc.newPath != tc.oldPath {
					run("add", "--intent-to-add", "--", tc.newPath)
				}
			}
			patch := run("diff", "--no-ext-diff", "--find-renames", "--unified=6")
			units, err := diff.Units(root, string(patch), 6)
			if err != nil {
				t.Fatal(err)
			}
			if len(units) != 1 {
				t.Fatalf("got %d units from %s", len(units), patch)
			}
			unit := units[0]
			if unit.ExistingModified != tc.wantExisting {
				t.Fatalf("existing modification = %v, want %v", unit.ExistingModified, tc.wantExisting)
			}
			for _, id := range []string{"removed-authorization-check", "removed-validation", "swallowed-cancellation"} {
				rule, _ := rules.Get(id)
				got, reason := rule.Applicable(unit, "Preserve existing contracts")
				if got != tc.wantExisting {
					t.Fatalf("%s applicable=%v: %s", id, got, reason)
				}
			}
		})
	}
}
