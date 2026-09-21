package eval

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/Eliran-Turgeman/reaper/internal/rules"
)

type CandidateRule struct {
	Experiment string  `json:"experiment"`
	Threshold  float64 `json:"threshold"`
	Notes      string  `json:"notes"`
}

type CandidateBundle struct {
	Artifacts       map[string]string        `json:"artifacts"`
	Rules           map[string]CandidateRule `json:"rules"`
	Name            string                   `json:"name"`
	Status          string                   `json:"status"`
	Provider        string                   `json:"provider"`
	RequestedModel  string                   `json:"requested_model"`
	ResolvedModel   string                   `json:"resolved_model"`
	GitProtocol     string                   `json:"git_protocol"`
	Version         int                      `json:"version"`
	ReleaseEligible bool                     `json:"release_eligible"`
}

func LoadCandidateBundle(file string) (CandidateBundle, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return CandidateBundle{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var bundle CandidateBundle
	if err := decoder.Decode(&bundle); err != nil {
		return CandidateBundle{}, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return CandidateBundle{}, fmt.Errorf("candidate bundle must contain one JSON object")
	}
	if err := bundle.validate(filepath.Dir(file)); err != nil {
		return CandidateBundle{}, err
	}
	return bundle, nil
}

func (b CandidateBundle) validate(base string) error {
	if b.Version != 1 || strings.TrimSpace(b.Name) == "" || b.Status != "development-frozen" ||
		b.Provider == "" || b.RequestedModel == "" || b.ResolvedModel == "" || b.GitProtocol == "" {
		return fmt.Errorf("candidate bundle requires version 1, development-frozen status, provider, model identities, and Git protocol")
	}
	if b.ReleaseEligible {
		return fmt.Errorf("development candidate bundle cannot be release eligible")
	}
	for _, rule := range rules.All() {
		if rule.DefaultSeverity != rules.SeverityError {
			continue
		}
		candidate, ok := b.Rules[rule.ID]
		if !ok || candidate.Experiment == "" || math.IsNaN(candidate.Threshold) || candidate.Threshold < 0 || candidate.Threshold > 1 {
			return fmt.Errorf("candidate bundle has invalid or missing blocking rule %s", rule.ID)
		}
		if _, ok := b.Artifacts[candidate.Experiment]; !ok {
			return fmt.Errorf("candidate experiment %s is not hash-bound", candidate.Experiment)
		}
		if _, err := LoadExperiment(filepath.Join(base, filepath.FromSlash(candidate.Experiment))); err != nil {
			return fmt.Errorf("candidate experiment %s: %w", candidate.Experiment, err)
		}
	}
	for name, want := range b.Artifacts {
		expected, err := hex.DecodeString(want)
		if err != nil || len(expected) != sha256.Size {
			return fmt.Errorf("artifact %s has invalid SHA-256", name)
		}
		data, err := os.ReadFile(filepath.Join(base, filepath.FromSlash(name)))
		if err != nil {
			return fmt.Errorf("read candidate artifact %s: %w", name, err)
		}
		got := sha256.Sum256(canonicalCandidateArtifact(data))
		if !bytes.Equal(got[:], expected) {
			return fmt.Errorf("candidate artifact %s SHA-256 mismatch", name)
		}
	}
	return nil
}

func canonicalCandidateArtifact(data []byte) []byte {
	lf := bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	return bytes.ReplaceAll(lf, []byte("\n"), []byte("\r\n"))
}
