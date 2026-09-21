package eval

import (
	"fmt"
	"testing"
)

func TestBlockingCandidateBundleBindsExactArtifacts(t *testing.T) {
	for _, version := range []int{1, 2, 3} {
		bundle, err := LoadCandidateBundle(fmt.Sprintf("../../benchmarks/candidates/blocking-v%d.json", version))
		if err != nil {
			t.Fatal(err)
		}
		if bundle.ReleaseEligible || len(bundle.Rules) != 5 {
			t.Fatalf("invalid frozen candidate bundle: %+v", bundle)
		}
	}
}
