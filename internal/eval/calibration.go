package eval

import "fmt"

type Calibration struct {
	Rule      string       `json:"rule"`
	Candidate *RuleReport  `json:"candidate"`
	Sweep     []RuleReport `json:"sweep"`
}

func Calibrate(report Report, minimumRecall float64) ([]Calibration, error) {
	if minimumRecall < 0 || minimumRecall > 1 {
		return nil, fmt.Errorf("minimum recall must be between 0 and 1")
	}
	for _, c := range report.Cases {
		if c.Split == "hidden-test" {
			return nil, fmt.Errorf("cannot calibrate on hidden-test cases")
		}
	}
	if len(report.Cases) == 0 {
		return nil, fmt.Errorf("report has no scored cases")
	}
	var out []Calibration
	for _, rule := range report.Rules {
		result := Calibration{Rule: rule.Rule}
		for i := 0; i <= 100; i++ {
			metric := Metrics(rule.Rule, report.Cases, float64(i)/100)
			result.Sweep = append(result.Sweep, metric)
			if metric.Recall >= minimumRecall && metric.TruePositive+metric.FalsePositive > 0 && (result.Candidate == nil || metric.Precision > result.Candidate.Precision || metric.Precision == result.Candidate.Precision && metric.Recall >= result.Candidate.Recall) {
				copy := metric
				result.Candidate = &copy
			}
		}
		out = append(out, result)
	}
	return out, nil
}
