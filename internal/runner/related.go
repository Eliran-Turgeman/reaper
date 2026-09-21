package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"

	"github.com/Eliran-Turgeman/reaper/internal/diff"
	"github.com/Eliran-Turgeman/reaper/internal/evidence"
	repogit "github.com/Eliran-Turgeman/reaper/internal/git"
	"github.com/Eliran-Turgeman/reaper/internal/rules"
	"github.com/Eliran-Turgeman/reaper/internal/semantic"
)

func (r *Runner) needsTargetedContext(rule rules.Rule) bool {
	return r.TargetedContext && (rule.ID == "removed-authorization-check" || rule.ID == "removed-validation")
}

// addRelatedEvidence reuses the fixture collector over bounded repository source.
// Each rule applies its own path policy before any helper source is read.
func (r *Runner) addRelatedEvidence(ctx context.Context, units []semantic.Unit, rule rules.Rule) error {
	u := units[0]
	if u.Language != "go" {
		return evidence.Add(nil, nil, r.ContextPatch, units, true)
	}
	if r.BeforeSnapshot == nil {
		return fmt.Errorf("targeted evidence requires a captured before snapshot")
	}
	patches, err := diff.Parse(r.ContextPatch)
	if err != nil {
		return err
	}
	dirs := map[string]bool{path.Dir(u.FilePath): true}
	focal := map[string]bool{u.FilePath: true}
	for _, p := range patches {
		if p.NewPath == u.FilePath || p.OldPath == u.FilePath {
			if p.OldPath != "/dev/null" {
				dirs[path.Dir(p.OldPath)] = true
				focal[p.OldPath] = true
			}
			if p.NewPath != "/dev/null" {
				dirs[path.Dir(p.NewPath)] = true
				focal[p.NewPath] = true
			}
		}
	}
	before := map[string]string{}
	after := map[string]string{}
	var limitations []string
	for _, side := range []string{"before", "after"} {
		snapshot := r.BeforeSnapshot
		destination := before
		if side == "after" {
			snapshot = r.IndexSnapshot
			destination = after
		}
		var files []string
		if snapshot != nil {
			files = snapshot.Files()
		} else {
			index, err := (repogit.CommandCollector{Dir: r.Root, Env: r.GitEnv}).SnapshotIndex(ctx)
			if err != nil {
				return err
			}
			files = index.Files()
		}
		sort.Strings(files)
		sort.SliceStable(files, func(i, j int) bool { return focal[files[i]] && !focal[files[j]] })
		bytesRead, scanned := 0, 0
		for _, file := range files {
			if path.Ext(file) != ".go" || !dirs[path.Dir(file)] || !r.matches(rule, file) {
				continue
			}
			if scanned >= 128 {
				limitations = append(limitations, side+": repository source file limit reached")
				break
			}
			scanned++
			if bytesRead >= 1048576 {
				limitations = append(limitations, side+": repository source byte limit reached")
				break
			}
			limit := int64(min(262144, 1048576-bytesRead))
			var data []byte
			if snapshot != nil {
				data, err = snapshot.Read(ctx, file, limit)
			} else {
				data, err = r.contextSource(ctx, file, limit)
			}
			if errors.Is(err, repogit.ErrSourceTooLarge) {
				limitations = append(limitations, side+": repository source exceeds remaining file/byte budget: "+file)
				continue
			}
			if err != nil {
				return err
			}
			if data == nil {
				limitations = append(limitations, side+": repository source unavailable: "+file)
				continue
			}
			bytesRead += len(data)
			destination[file] = string(data)
		}
	}
	if err := evidence.Add(before, after, r.ContextPatch, units, true); err != nil {
		return err
	}
	if len(limitations) > 0 {
		var report evidence.Context
		if err := json.Unmarshal(units[0].RelatedEvidence, &report); err != nil {
			return err
		}
		report.Limitations = append(report.Limitations, limitations...)
		data, err := json.Marshal(report)
		if err != nil {
			return err
		}
		if len(data) > 16384 {
			return fmt.Errorf("targeted context with repository limits exceeds 16 KiB")
		}
		units[0].RelatedEvidence = data
	}
	return nil
}
