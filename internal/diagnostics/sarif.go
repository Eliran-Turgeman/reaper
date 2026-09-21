package diagnostics

import (
	"encoding/json"
	"io"
	"net/url"
	"strings"
)

func WriteSARIF(w io.Writer, report Report) error {
	results := make([]any, 0, len(report.Diagnostics))
	for _, item := range report.Diagnostics {
		result := map[string]any{
			"ruleId": item.Rule, "level": item.Severity,
			"message":    map[string]any{"text": item.Message},
			"properties": map[string]any{"confidence": item.Confidence, "threshold": item.Threshold},
		}
		if item.File != "<patch>" {
			if item.Fingerprint != "" {
				result["partialFingerprints"] = map[string]string{"reaper/v1": item.Fingerprint}
			}
			uri := (&url.URL{Path: strings.ReplaceAll(item.File, "\\", "/")}).String()
			result["locations"] = []any{map[string]any{"physicalLocation": map[string]any{
				"artifactLocation": map[string]any{"uri": uri},
				"region":           map[string]any{"startLine": max(1, item.StartLine), "endLine": max(1, item.StartLine, item.EndLine)},
			}}}
		}
		results = append(results, result)
	}
	notifications := make([]any, 0, len(report.Skipped))
	for _, skipped := range report.Skipped {
		notifications = append(notifications, map[string]any{"level": "error", "message": map[string]any{"text": skipped.File + ": " + skipped.Reason}})
	}
	return json.NewEncoder(w).Encode(map[string]any{
		"version": "2.1.0", "$schema": "https://json.schemastore.org/sarif-2.1.0.json",
		"runs": []any{map[string]any{
			"tool":        map[string]any{"driver": map[string]any{"name": "Reaper", "informationUri": "https://github.com/Eliran-Turgeman/reaper"}},
			"results":     results,
			"invocations": []any{map[string]any{"executionSuccessful": report.Complete, "toolExecutionNotifications": notifications}},
			"properties":  map[string]any{"status": report.Status, "provider": report.Provider, "model": report.Model},
		}},
	})
}
