package eval

import (
	"bytes"
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

func TestCandidateArtifactHashingIsLineEndingStable(t *testing.T) {
	lf := []byte("{\n  \"version\": 1\n}\n")
	crlf := []byte("{\r\n  \"version\": 1\r\n}\r\n")
	if !bytes.Equal(canonicalCandidateArtifact(lf), canonicalCandidateArtifact(crlf)) {
		t.Fatal("candidate artifact hash input depends on checkout line endings")
	}
}
