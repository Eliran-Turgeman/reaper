package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

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
