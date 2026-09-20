package eval

import (
	"path/filepath"
	"testing"

	"github.com/Eliran-Turgeman/reaper/internal/rules"
)

func TestSeedCorpusHasTwentyBalancedExamplesPerRule(t *testing.T) {
	dir := filepath.Join("..", "..", "evals")
	for _, rule := range rules.All() {
		examples, err := Load(dir, rule.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(examples) != 20 {
			t.Fatalf("%s has %d examples, want 20", rule.ID, len(examples))
		}
		positive := 0
		languages := map[string]bool{}
		for _, example := range examples {
			if example.Expected == "positive" {
				positive++
			}
			languages[example.Language] = true
		}
		if positive != 10 || len(languages) < 5 {
			t.Fatalf("%s is not balanced and varied: positives=%d languages=%d", rule.ID, positive, len(languages))
		}
	}
}
