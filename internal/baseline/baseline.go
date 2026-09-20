package baseline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Eliran-Turgeman/reaper/internal/diagnostics"
	"github.com/Eliran-Turgeman/reaper/internal/rules"
)

type Entry struct {
	Fingerprint string `json:"fingerprint"`
	Rule        string `json:"rule"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Reason      string `json:"reason,omitempty"`
}
type File struct {
	Version int     `json:"version"`
	Entries []Entry `json:"entries"`
}

func Create(report diagnostics.Report, reason string) (File, error) {
	if !report.Complete {
		return File{}, fmt.Errorf("cannot baseline incomplete analysis")
	}
	out := File{Version: 1, Entries: []Entry{}}
	for _, d := range report.Diagnostics {
		if d.Fingerprint == "" {
			return File{}, fmt.Errorf("finding lacks fingerprint")
		}
		out.Entries = append(out.Entries, Entry{Fingerprint: d.Fingerprint, Rule: d.Rule, File: d.File, Line: d.StartLine, Reason: reason})
	}
	return out, nil
}

var directive = regexp.MustCompile(`^\s*(?://|#|--)\s*reaper:\s*ignore\s+(\S+)(?:\s+--\s+(.+))?\s*$`)

func Apply(root, baselinePath string, report diagnostics.Report) (diagnostics.Report, error) {
	known := map[string]Entry{}
	if baselinePath != "" {
		data, err := os.ReadFile(baselinePath)
		if err != nil {
			return report, err
		}
		var file File
		if err := json.Unmarshal(data, &file); err != nil {
			return report, err
		}
		if file.Version != 1 {
			return report, fmt.Errorf("unsupported baseline version")
		}
		for _, entry := range file.Entries {
			if entry.Fingerprint == "" {
				return report, fmt.Errorf("baseline entry lacks fingerprint")
			}
			known[entry.Fingerprint] = entry
		}
	}
	var active, suppressed []diagnostics.Diagnostic
	for _, d := range report.Diagnostics {
		if entry, ok := known[d.Fingerprint]; ok && entry.Rule == d.Rule && entry.File == d.File {
			d.SuppressionReason = "baseline: " + entry.Reason
		} else if d.File != "<patch>" {
			data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(d.File)))
			if err != nil && !os.IsNotExist(err) {
				return report, err
			}
			for i, line := range strings.Split(string(data), "\n") {
				if i+1 < d.StartLine-1 || i+1 > d.EndLine {
					continue
				}
				match := directive.FindStringSubmatch(line)
				if match == nil {
					continue
				}
				if _, ok := rules.Get(match[1]); !ok || strings.TrimSpace(match[2]) == "" {
					return report, fmt.Errorf("%s:%d: suppression requires a known rule and -- reason", d.File, i+1)
				}
				if match[1] == d.Rule {
					d.SuppressionReason = strings.TrimSpace(match[2])
				}
			}
		}
		if d.SuppressionReason != "" {
			suppressed = append(suppressed, d)
		} else {
			active = append(active, d)
		}
	}
	updated := diagnostics.New(active, report.Summary)
	updated.Provider, updated.Model = report.Provider, report.Model
	updated.Capabilities = report.Capabilities
	updated.SetIncomplete(report.Skipped, report.IncompletePolicy)
	updated.Suppressed = suppressed
	updated.Summary.Suppressed = len(suppressed)
	return updated, nil
}
