package rules

import "github.com/Eliran-Turgeman/reaper/internal/decision"

// Questions is the common instruction assembly path for checks and evaluations.
func Questions(selected []Rule, audit, retrieved bool) []decision.Question {
	var questions []decision.Question
	for _, rule := range selected {
		for _, signal := range rule.Signals {
			instructions := signal.Instructions
			if retrieved {
				instructions = "Use REPOSITORY SEARCH EVIDENCE as part of the supplied context. Distinct retrieved implementations or usages count as visible behavior. Do not infer absence of uses from a truncated or limited search. " + instructions
			}
			if audit {
				instructions = "Evaluate the current code as an existing-code audit, regardless of when it was introduced. " + instructions
			}
			questions = append(questions, decision.Question{ID: rule.QuestionID(signal), Instructions: instructions, Criteria: signal.Criteria})
		}
	}
	return questions
}
