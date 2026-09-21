package runner

import "github.com/Eliran-Turgeman/reaper/internal/semantic"

// stateFields contains the same evidence as buildState, without parsing source
// text for header delimiters (which can legitimately occur inside code).
func stateFields(unit semantic.Unit, task string, audit bool, retrieved string) map[string]any {
	fields := map[string]any{"file": unit.FilePath, "language": unit.Language, "current_code": unit.NewContent}
	if len(unit.RelatedEvidence) > 0 {
		fields["matched_snapshot_evidence"] = unit.RelatedEvidence
	}
	if audit {
		fields["mode"] = "EXISTING CODE AUDIT"
	} else {
		fields["diff"] = unit.Diff
	}
	if task != "" {
		fields["task"] = task
	}
	if unit.OldContent != "" {
		fields["previous_code"] = unit.OldContent
	}
	if unit.SurroundingCode != "" && unit.SurroundingCode != unit.NewContent {
		fields["surrounding_context"] = unit.SurroundingCode
	}
	if retrieved != "" {
		fields["repository_search_evidence"] = retrieved
	}
	return fields
}
