package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var generatedTask = regexp.MustCompile(`(?is)<!--\s*(?:reaper:)?generated:start\s*-->.*?<!--\s*(?:reaper:)?generated:end\s*-->`)
var taskComments = regexp.MustCompile(`(?s)<!--.*?-->`)

func resolveTask(options checkOptions, explicit bool, getenv func(string) string) (string, string, error) {
	if explicit {
		return options.task, "--task", nil
	}
	if options.taskFile != "" {
		data, err := os.ReadFile(options.taskFile)
		if err != nil {
			return "", "", fmt.Errorf("read task file: %w", err)
		}
		return strings.TrimSpace(string(data)), "--task-file", nil
	}
	if task := getenv("REAPER_TASK"); task != "" {
		return task, "REAPER_TASK", nil
	}
	if !options.taskFromPR {
		return "", "none", nil
	}
	path := getenv("GITHUB_EVENT_PATH")
	if path == "" {
		return "", "none (no GitHub event)", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("read GitHub event: %w", err)
	}
	var event struct {
		PR *struct {
			Title string `json:"title"`
			Body  string `json:"body"`
		} `json:"pull_request"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		return "", "", fmt.Errorf("parse GitHub event: %w", err)
	}
	if event.PR == nil {
		return "", "none (not a pull request)", nil
	}
	task := event.PR.Title + "\n\n" + event.PR.Body
	task = generatedTask.ReplaceAllString(task, "")
	task = strings.TrimSpace(taskComments.ReplaceAllString(task, ""))
	text := []rune(task)
	if len(text) > 4000 {
		task = string(text[:4000])
	}
	if task == "" {
		return "", "none (empty pull request)", nil
	}
	return task, "GitHub PR title and body", nil
}
