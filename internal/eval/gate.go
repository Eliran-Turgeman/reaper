package eval

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
)

func CheckBaseline(w io.Writer, report Report, path string, tolerance float64) error {
	if math.IsNaN(tolerance) || tolerance < 0 || tolerance > 1 {
		return fmt.Errorf("invalid quality-gate tolerance")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var baseline Report
	if err := json.Unmarshal(data, &baseline); err != nil {
		return err
	}
	if baseline.Version != report.Version || baseline.Provider != report.Provider || baseline.Model != report.Model || len(baseline.Rules) == 0 {
		return fmt.Errorf("baseline schema/provider/model mismatch or empty baseline")
	}
	if baseline.Mode != report.Mode || baseline.Grouping != report.Grouping {
		return fmt.Errorf("baseline evaluation mode or grouping mismatch")
	}
	if len(baseline.Rules) != len(report.Rules) {
		return fmt.Errorf("baseline rule set mismatch")
	}
	if len(baseline.Cases) != len(report.Cases) {
		return fmt.Errorf("baseline case set mismatch")
	}
	labels := map[string]string{}
	for _, c := range baseline.Cases {
		labels[c.Rule+":"+c.ID] = c.Expected + ":" + c.Split
	}
	for _, c := range report.Cases {
		if labels[c.Rule+":"+c.ID] != c.Expected+":"+c.Split {
			return fmt.Errorf("baseline case or label mismatch for %s", c.ID)
		}
	}
	current := map[string]RuleReport{}
	for _, r := range report.Rules {
		current[r.Rule] = r
	}
	failed := false
	for _, expected := range baseline.Rules {
		actual, ok := current[expected.Rule]
		if !ok || actual.Examples != expected.Examples || actual.Threshold != expected.Threshold {
			return fmt.Errorf("baseline corpus mismatch for %s", expected.Rule)
		}
		precisionDelta, recallDelta := actual.Precision-expected.Precision, actual.Recall-expected.Recall
		fmt.Fprintf(w, "%s precision_delta=%+.4f recall_delta=%+.4f\n", expected.Rule, precisionDelta, recallDelta)
		if precisionDelta < -tolerance || recallDelta < -tolerance {
			failed = true
		}
	}
	if failed {
		return fmt.Errorf("evaluation quality regressed beyond tolerance %.4f", tolerance)
	}
	return nil
}
