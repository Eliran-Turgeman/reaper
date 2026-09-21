package eval

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

type ReviewPackOptions struct {
	Seed  string
	Limit int
}

type ReviewAnswers struct {
	FactualOutcome string `json:"factual_outcome"`
	Permission     string `json:"permission"`
	Evidence       string `json:"evidence"`
	Notes          string `json:"notes"`
}

type ReviewCase struct {
	BeforeFiles map[string]string `json:"before_files"`
	AfterFiles  map[string]string `json:"after_files"`
	ID          string            `json:"id"`
	Rule        string            `json:"rule"`
	Task        string            `json:"task"`
	Review      ReviewAnswers     `json:"review"`
}

type ReviewPack struct {
	Rubric              map[string][]string `json:"rubric"`
	Cases               []ReviewCase        `json:"cases"`
	Protocol            string              `json:"protocol"`
	SourceCorpusSHA256  string              `json:"source_corpus_sha256"`
	SelectedCasesSHA256 string              `json:"selected_cases_sha256"`
	BlindedCasesSHA256  string              `json:"blinded_cases_sha256"`
	Version             int                 `json:"version"`
	SourceCases         int                 `json:"source_cases"`
	SelectedCases       int                 `json:"selected_cases"`
	CompleteCorpus      bool                `json:"complete_corpus"`
}

func BuildReviewPack(dir string, options ReviewPackOptions) (ReviewPack, error) {
	if options.Limit < 0 {
		return ReviewPack{}, fmt.Errorf("review pack limit cannot be negative")
	}
	cases, err := LoadGitCases(dir)
	if err != nil {
		return ReviewPack{}, err
	}
	sort.SliceStable(cases, func(i, j int) bool {
		return reviewOrder(options.Seed, cases[i].ID) < reviewOrder(options.Seed, cases[j].ID)
	})
	selected := cases
	if options.Limit > 0 && options.Limit < len(selected) {
		selected = selected[:options.Limit]
	}
	blinded := make([]ReviewCase, 0, len(selected))
	for _, c := range selected {
		blinded = append(blinded, ReviewCase{
			ID: c.ID, Rule: c.Rule, Task: c.Task,
			BeforeFiles: cloneFiles(c.BeforeFiles), AfterFiles: cloneFiles(c.AfterFiles),
			Review: ReviewAnswers{},
		})
	}
	return ReviewPack{
		Version: 1, Protocol: "blind-review-v1",
		SourceCorpusSHA256: fingerprint(cases), SelectedCasesSHA256: fingerprint(selected),
		BlindedCasesSHA256: fingerprint(blinded),
		SourceCases:        len(cases), SelectedCases: len(selected), CompleteCorpus: len(selected) == len(cases),
		Rubric: map[string][]string{
			"factual_outcome": {"violation", "no-violation", "uncertain"},
			"permission":      {"allowed", "disallowed", "uncertain"},
			"evidence":        {"sufficient", "insufficient"},
		},
		Cases: blinded,
	}, nil
}

func WriteReviewPack(w io.Writer, pack ReviewPack) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(pack)
}

func reviewOrder(seed, id string) string {
	sum := sha256.Sum256([]byte(seed + "\x00" + id))
	return hex.EncodeToString(sum[:])
}

func cloneFiles(files map[string]string) map[string]string {
	out := make(map[string]string, len(files))
	for name, content := range files {
		out[name] = content
	}
	return out
}
