package semantic

import "testing"

func TestSupportedSourceWhitelist(t *testing.T) {
	tests := []struct {
		path      string
		language  string
		supported bool
	}{
		{path: "main.go", language: "go", supported: true},
		{path: "worker.py", language: "python", supported: true},
		{path: "view.tsx", language: "typescript", supported: true},
		{path: "app.JS", language: "javascript", supported: true},
		{path: "Service.cs", language: "csharp", supported: true},
		{path: "Service.java", language: "java", supported: true},
		{path: "task.rb", language: "ruby", supported: true},
		{path: "lib.rs", language: "rust", supported: true},
		{path: "native.c", language: "c", supported: true},
		{path: "native.hpp", language: "cpp", supported: true},
		{path: "Main.kt", language: "kotlin", supported: true},
		{path: "App.swift", language: "swift", supported: true},
		{path: "index.php", language: "php", supported: true},
		{path: "Job.scala", language: "scala", supported: true},
		{path: "widget.dart", language: "dart", supported: true},
		{path: "worker.ex", language: "elixir", supported: true},
		{path: "plugin.lua", language: "lua", supported: true},
		{path: "ViewController.m", language: "objective-c", supported: true},
		{path: "module.mjs", language: "javascript", supported: true},
		{path: "types.mts", language: "typescript", supported: true},
		{path: "README.md", language: "text", supported: false},
		{path: "emails.txt", language: "text", supported: false},
		{path: "settings.json", language: "text", supported: false},
		{path: "pipeline.yaml", language: "text", supported: false},
		{path: "Dockerfile", language: "text", supported: false},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			if got := Language(test.path); got != test.language {
				t.Fatalf("Language(%q) = %q, want %q", test.path, got, test.language)
			}
			if got := IsSupportedSource(test.path); got != test.supported {
				t.Fatalf("IsSupportedSource(%q) = %t, want %t", test.path, got, test.supported)
			}
		})
	}
}

func TestContainsCommentUsesLanguageSyntax(t *testing.T) {
	tests := []struct {
		language string
		code     string
		want     bool
	}{
		{language: "cpp", code: "// explain why", want: true},
		{language: "swift", code: "value += 1 // required", want: true},
		{language: "elixir", code: "# explain why", want: true},
		{language: "php", code: "# explain why", want: true},
		{language: "php", code: "// explain why", want: true},
		{language: "lua", code: "-- explain why", want: true},
		{language: "lua", code: "value = value - -1", want: false},
		{language: "python", code: "value // 2", want: false},
	}

	for _, test := range tests {
		t.Run(test.language+"_"+test.code, func(t *testing.T) {
			if got := ContainsComment(test.language, test.code); got != test.want {
				t.Fatalf("ContainsComment(%q, %q) = %t, want %t", test.language, test.code, got, test.want)
			}
		})
	}
}

func TestExpandedLanguageTestFiles(t *testing.T) {
	for _, path := range []string{
		"native_test.cpp",
		"widget_test.dart",
		"worker_test.exs",
		"UserServiceTest.kt",
		"PaymentTest.php",
		"ParserTest.scala",
		"FeatureTests.swift",
		"ControllerTests.m",
		"parser.spec.lua",
	} {
		t.Run(path, func(t *testing.T) {
			if !IsTestFile(path) {
				t.Fatalf("IsTestFile(%q) = false, want true", path)
			}
		})
	}
}
