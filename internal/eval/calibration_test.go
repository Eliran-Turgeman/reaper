package eval

import "testing"

func TestCalibrationRespectsRecallAndRejectsHiddenTest(t *testing.T) {
	report := Report{Rules: []RuleReport{{Rule: "r"}}, Cases: []ScoredCase{{Rule: "r", Score: .8, Expected: "positive", Split: "dev"}, {Rule: "r", Score: .7, Expected: "positive", Split: "dev"}, {Rule: "r", Score: .2, Expected: "negative", Split: "dev"}}}
	result, err := Calibrate(report, .6)
	if err != nil || len(result[0].Sweep) != 101 || result[0].Candidate.Threshold != .7 || result[0].Candidate.Precision != 1 {
		t.Fatalf("%+v %v", result, err)
	}
	report.Cases[0].Split = "hidden-test"
	if _, err := Calibrate(report, .6); err == nil {
		t.Fatal("hidden-test leakage")
	}
}
