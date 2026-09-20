package feedback

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/Eliran-Turgeman/reaper/internal/diagnostics"
	"os"
	"sort"
)

type Entry struct {
	ID       string `json:"id"`
	Rule     string `json:"rule"`
	Decision string `json:"decision"`
}
type Metric struct {
	Rule          string `json:"rule"`
	Findings      int    `json:"findings"`
	Suppressed    int    `json:"suppressed"`
	FalsePositive int    `json:"false_positive"`
	Accepted      int    `json:"accepted"`
}

func Record(path string, report diagnostics.Report, id, decision string) error {
	if decision != "useful" && decision != "false-positive" {
		return fmt.Errorf("feedback must be useful or false-positive")
	}
	rule := ""
	for _, d := range append(append([]diagnostics.Diagnostic{}, report.Diagnostics...), report.Suppressed...) {
		if d.Fingerprint == id {
			rule = d.Rule
		}
	}
	if id == "" || rule == "" {
		return fmt.Errorf("finding ID not found in report")
	}
	data, err := json.Marshal(Entry{ID: id, Rule: rule, Decision: decision})
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(append(data, '\n'))
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func Metrics(path string, report diagnostics.Report) ([]Metric, error) {
	entries := map[string]Entry{}
	file, err := os.Open(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var entry Entry
			if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
				return nil, err
			}
			if entry.ID == "" || entry.Rule == "" || (entry.Decision != "useful" && entry.Decision != "false-positive") {
				return nil, fmt.Errorf("invalid feedback entry")
			}
			entries[entry.ID] = entry
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}
	counts := map[string]Metric{}
	for _, d := range append(append([]diagnostics.Diagnostic{}, report.Diagnostics...), report.Suppressed...) {
		metric := counts[d.Rule]
		metric.Rule = d.Rule
		metric.Findings++
		if d.SuppressionReason != "" {
			metric.Suppressed++
		}
		counts[d.Rule] = metric
	}
	for _, entry := range entries {
		metric := counts[entry.Rule]
		metric.Rule = entry.Rule
		if entry.Decision == "useful" {
			metric.Accepted++
		} else {
			metric.FalsePositive++
		}
		counts[entry.Rule] = metric
	}
	out := []Metric{}
	for _, metric := range counts {
		out = append(out, metric)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Rule < out[j].Rule })
	return out, nil
}
