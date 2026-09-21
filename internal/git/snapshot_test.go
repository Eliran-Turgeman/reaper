package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestIndexSnapshotIncludesWholeRepositoryFromNestedDirectory(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "--quiet")
	nested := filepath.Join(dir, "service")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"root.go", "service/handler.go"} {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	runGit(t, dir, "add", ".")
	rootSnapshot, err := (CommandCollector{Dir: dir}).SnapshotIndex(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	nestedSnapshot, err := (CommandCollector{Dir: nested}).SnapshotIndex(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(nestedSnapshot.Files(), rootSnapshot.Files()) || nestedSnapshot.Fingerprint != rootSnapshot.Fingerprint {
		t.Fatalf("nested snapshot omitted repository evidence: %v versus %v", nestedSnapshot.Files(), rootSnapshot.Files())
	}
	data, err := nestedSnapshot.Read(context.Background(), "root.go", 100)
	if err != nil || string(data) != "root.go" {
		t.Fatal("cannot read captured root source from nested directory", string(data), err)
	}
}

func TestIndexSnapshotPinsRegularBlobContentsAndBoundsReads(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "--quiet")
	file := filepath.Join(dir, "source.go")
	if err := os.WriteFile(file, []byte("staged source"), 0600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "source.go")
	collector := CommandCollector{Dir: dir}
	snapshot, err := collector.SnapshotIndex(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "-c", "user.name=Test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "before")
	if err := os.WriteFile(file, []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "source.go")
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	data, err := snapshot.Read(context.Background(), "source.go", 100)
	if err != nil || string(data) != "staged source" {
		t.Fatal("snapshot drifted", string(data), err)
	}
	changed, err := collector.SnapshotIndex(context.Background())
	if err != nil || changed.Fingerprint == snapshot.Fingerprint {
		t.Fatal("index drift not detected", err)
	}
	tree, err := collector.SnapshotTree(context.Background(), "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	historical, err := tree.Read(context.Background(), "source.go", 100)
	if err != nil || string(historical) != "staged source" {
		t.Fatal("wrong before tree", string(historical), err)
	}
	if _, err := snapshot.Read(context.Background(), "source.go", 2); !errors.Is(err, ErrSourceTooLarge) {
		t.Fatal("oversized blob read", err)
	}
	if _, err := snapshot.Read(context.Background(), "missing.go", 0); err == nil {
		t.Fatal("missing staged source accepted")
	}
	// A Git symlink blob is data in the index, regardless of OS symlink support.
	runGit(t, dir, "update-index", "--add", "--cacheinfo", "120000,"+snapshot.blobs["source.go"]+",link.go")
	withLink, err := collector.SnapshotIndex(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(withLink.Files()) != 1 {
		t.Fatal("symlink included in regular source evidence", withLink.Files())
	}
}
