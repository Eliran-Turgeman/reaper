package eval

import "testing"

func TestPatchCorpusAndBaseRate(t *testing.T) {
	cases, err := LoadPatches("../../benchmarks/patches")
	if err != nil {
		t.Fatal(err)
	}
	negative := 0
	for _, c := range cases {
		if c.Expected == "negative" {
			negative++
		}
	}
	if len(cases) < 20 || negative*4 < len(cases)*3 {
		t.Fatal("benchmark needs a majority of clean patches")
	}
}

func TestBenchmarkMetrics(t *testing.T) {
	cases := []ScoredCase{{Rule: "r", Expected: "positive", Score: .95}, {Rule: "r", Expected: "positive", Score: .2}, {Rule: "r", Expected: "negative", Score: .99}, {Rule: "r", Expected: "negative", Score: .1}}
	m := Metrics("r", cases, .9)
	if m.TruePositive != 1 || m.FalsePositive != 1 || m.TrueNegative != 1 || m.FalseNegative != 1 || m.Precision != .5 || m.Recall != .5 || m.FalsePositiveRate != .5 {
		t.Fatal(m)
	}
}
