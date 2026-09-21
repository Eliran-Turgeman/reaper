package git

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var ErrSourceTooLarge = errors.New("source exceeds evidence byte limit")

// SourceSnapshot holds immutable blob IDs, so later index/worktree edits cannot
// change source returned from this snapshot. Only regular staged files qualify.
type SourceSnapshot struct {
	collector   CommandCollector
	blobs       map[string]string
	Fingerprint string
}

func (c CommandCollector) SnapshotTree(ctx context.Context, ref string) (*SourceSnapshot, error) {
	resolved, err := c.command(ctx, "-C", c.Dir, "rev-parse", "--verify", "--end-of-options", ref+"^{tree}").Output()
	if err != nil {
		return nil, fmt.Errorf("resolve evidence base %q: %w", ref, err)
	}
	id := strings.TrimSpace(string(resolved))
	data, err := c.command(ctx, "-C", c.Dir, "ls-tree", "-r", "-z", "--full-tree", id).Output()
	if err != nil {
		return nil, fmt.Errorf("list evidence base: %w", err)
	}
	s := &SourceSnapshot{collector: c, blobs: map[string]string{}, Fingerprint: id}
	for _, entry := range strings.Split(string(data), "\x00") {
		if entry == "" {
			continue
		}
		metadata, name, ok := strings.Cut(entry, "\t")
		fields := strings.Fields(metadata)
		if !ok || len(fields) != 3 {
			return nil, fmt.Errorf("invalid tree entry")
		}
		if fields[1] == "blob" && (fields[0] == "100644" || fields[0] == "100755") {
			s.blobs[name] = fields[2]
		}
	}
	return s, nil
}

func (c CommandCollector) SnapshotIndex(ctx context.Context) (*SourceSnapshot, error) {
	data, err := c.command(ctx, "-C", c.Dir, "ls-files", "--full-name", "--stage", "-z").Output()
	if err != nil {
		return nil, fmt.Errorf("capture index: %w", err)
	}
	s := &SourceSnapshot{collector: c, blobs: map[string]string{}, Fingerprint: fmt.Sprintf("%x", sha256.Sum256(data))}
	for _, entry := range strings.Split(string(data), "\x00") {
		if entry == "" {
			continue
		}
		metadata, name, ok := strings.Cut(entry, "\t")
		fields := strings.Fields(metadata)
		if !ok || len(fields) != 3 {
			return nil, fmt.Errorf("invalid index entry")
		}
		if fields[2] != "0" {
			return nil, fmt.Errorf("unmerged index entry: %s", name)
		}
		if fields[0] == "100644" || fields[0] == "100755" {
			s.blobs[name] = fields[1]
		}
	}
	return s, nil
}

func (s *SourceSnapshot) Files() []string {
	files := make([]string, 0, len(s.blobs))
	for file := range s.blobs {
		files = append(files, file)
	}
	sort.Strings(files)
	return files
}

func (s *SourceSnapshot) Read(ctx context.Context, name string, limit int64) ([]byte, error) {
	id, ok := s.blobs[name]
	if !ok {
		return nil, fmt.Errorf("no regular staged source for %s", name)
	}
	if limit > 0 {
		data, err := s.collector.command(ctx, "-C", s.collector.Dir, "cat-file", "-s", id).Output()
		if err != nil {
			return nil, fmt.Errorf("read staged size for %s: %w", name, err)
		}
		size, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		if err != nil || size < 0 {
			return nil, fmt.Errorf("invalid staged size for %s", name)
		}
		if size > limit {
			return nil, ErrSourceTooLarge
		}
	}
	data, err := s.collector.command(ctx, "-C", s.collector.Dir, "cat-file", "blob", id).Output()
	if err != nil {
		return nil, fmt.Errorf("read captured staged blob %s: %w", name, err)
	}
	if data == nil {
		data = []byte{}
	}
	return data, nil
}
