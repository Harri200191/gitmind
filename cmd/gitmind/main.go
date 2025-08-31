package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/yourname/commitgen/internal/config"
	"github.com/yourname/commitgen/internal/diff"
	"github.com/yourname/commitgen/internal/hook"
	"github.com/yourname/commitgen/internal/llm"
	"github.com/yourname/commitgen/internal/security"
	"github.com/yourname/commitgen/internal/splitter"
	"github.com/yourname/commitgen/internal/testgen"
)

const version = "0.1.0"

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	sub := os.Args[1]
	switch sub {
	case "generate":
		cmdGenerate(os.Args[2:])
	case "install-hook":
		cmdInstallHook(os.Args[2:])
	case "uninstall-hook":
		cmdUninstallHook(os.Args[2:])
	case "doctor":
		cmdDoctor()
	case "multi-commit":
		cmdMultiCommit(os.Args[2:])
	case "suggest-tests":
		cmdSuggestTests(os.Args[2:])
	case "security-check":
		cmdSecurityCheck(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Println(version)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `gitmind %s

Usage:
  gitmind install-hook              Install prepare-commit-msg hook in current repo
  gitmind uninstall-hook            Remove hook from current repo
  gitmind generate -f <path>        Generate a message into commit-msg file (hook calls this)
  gitmind multi-commit              Analyze and split staged changes into multiple commits
  gitmind suggest-tests             Generate unit tests for changed functions
  gitmind security-check            Run security analysis on staged changes
  gitmind doctor                    Check model/config availability
  gitmind version                   Print version
`, version)
}

func cmdInstallHook(args []string) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		log.Fatal(err)
	}
	if err := hook.Install(repoRoot); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Installed prepare-commit-msg hook.")
}

func cmdUninstallHook(args []string) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		log.Fatal(err)
	}
	if err := hook.Uninstall(repoRoot); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Removed prepare-commit-msg hook.")
}

func cmdDoctor() {
	cfg := config.Load()
	ok, info := llm.Doctor(cfg)
	if ok {
		fmt.Println("LLM ready:", info)
	} else {
		fmt.Println("LLM not ready:", info)
	}
}

func cmdGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	file := fs.String("f", "", "path to commit-msg file (provided by git)")
	suggestTests := fs.Bool("suggest-tests", false, "generate unit tests for changed functions")
	_ = fs.Parse(args)
	if *file == "" {
		log.Fatal("-f commit message file is required")
	}

	// Read staged diff
	d, err := diff.Staged()
	if err != nil {
		log.Fatal(err)
	}
	if d == "" {
		// nothing staged; don't clobber existing
		os.Exit(0)
	}

	cfg := config.Load()

	// Run security analysis
	if cfg.Security.Enabled {
		secAnalyzer := security.New(cfg)
		secReport, err := secAnalyzer.AnalyzeDiff(d)
		if err == nil {
			if secAnalyzer.ShouldBlockCommit(secReport) {
				fmt.Fprintf(os.Stderr, "❌ Commit blocked due to high-severity security issues:\n")
				for _, finding := range secReport.Findings {
					if finding.Severity == "high" {
						fmt.Fprintf(os.Stderr, "  %s:%d - %s\n", finding.File, finding.Line, finding.Message)
					}
				}
				os.Exit(1)
			}
		}
	}

	// Check for multi-commit opportunities
	if cfg.MultiCommit.Enabled {
		mcm := splitter.NewMultiCommitManager(cfg)
		proposals, err := mcm.ProcessStagedChanges()
		if err == nil && len(proposals) > 1 {
			fmt.Printf("💡 Detected %d logical changes. Use 'gitmind multi-commit' to split into separate commits\n", len(proposals))
		}
	}

	// Generate commit message
	message, err := llm.Generate(cfg, d)
	if err != nil {
		// fall back to heuristic
		message = diff.HeuristicMessage(d, cfg)
	}

	// Enhance message with security notes if enabled
	if cfg.Security.Enabled {
		secAnalyzer := security.New(cfg)
		secReport, err := secAnalyzer.AnalyzeDiff(d)
		if err == nil {
			message = secAnalyzer.GenerateCommitMessage(secReport, message)
		}
	}

	// Generate tests if requested
	if *suggestTests && cfg.TestGeneration.Enabled {
		testGen := testgen.New(cfg)
		functions, err := testGen.AnalyzeChangedFunctions(d)
		if err == nil && len(functions) > 0 {
			testFiles, err := testGen.GenerateTests(functions)
			if err == nil {
				testGen.WriteTestFiles(testFiles)
				message += "\n\n🧪 Generated unit tests for modified functions"
			}
		}
	}

	if err := os.WriteFile(*file, []byte(message+"\n"), 0644); err != nil {
		log.Fatal(err)
	}
}

func findRepoRoot() (string, error) {
	cwd, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(cwd, ".git")); err == nil {
			return cwd, nil
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			return "", errors.New(".git not found; run inside a repo")
		}
		cwd = parent
	}
}

func cmdMultiCommit(args []string) {
	fs := flag.NewFlagSet("multi-commit", flag.ExitOnError)
	interactive := fs.Bool("interactive", false, "interactive mode for editing proposals")
	_ = fs.Parse(args)

	cfg := config.Load()
	if !cfg.MultiCommit.Enabled {
		fmt.Println("Multi-commit is disabled in configuration")
		os.Exit(1)
	}

	mcm := splitter.NewMultiCommitManager(cfg)

	if *interactive {
		if err := mcm.InteractiveMultiCommit(); err != nil {
			log.Fatal(err)
		}
	} else {
		proposals, err := mcm.ProcessStagedChanges()
		if err != nil {
			log.Fatal(err)
		}

		if len(proposals) <= 1 {
			fmt.Println("No multi-commit opportunities detected")
			return
		}

		if err := mcm.ExecuteMultiCommit(proposals); err != nil {
			log.Fatal(err)
		}
	}
}

func cmdSuggestTests(args []string) {
	fs := flag.NewFlagSet("suggest-tests", flag.ExitOnError)
	outputDir := fs.String("output", ".", "output directory for test files")
	autoStage := fs.Bool("stage", false, "automatically stage generated test files")
	_ = fs.Parse(args)

	cfg := config.Load()
	if !cfg.TestGeneration.Enabled {
		fmt.Println("Test generation is disabled in configuration")
		os.Exit(1)
	}

	// Override config with command line options
	if *outputDir != "." {
		cfg.TestGeneration.OutputDir = *outputDir
	}
	if *autoStage {
		cfg.TestGeneration.AutoStage = true
	}

	// Get staged diff
	d, err := diff.Staged()
	if err != nil {
		log.Fatal(err)
	}
	if d == "" {
		fmt.Println("No staged changes found")
		return
	}

	testGen := testgen.New(cfg)
	functions, err := testGen.AnalyzeChangedFunctions(d)
	if err != nil {
		log.Fatal(err)
	}

	if len(functions) == 0 {
		fmt.Println("No testable functions found in staged changes")
		return
	}

	fmt.Printf("🔍 Found %d functions that can be tested:\n", len(functions))
	for _, fn := range functions {
		fmt.Printf("  - %s.%s\n", fn.Package, fn.Name)
	}

	testFiles, err := testGen.GenerateTests(functions)
	if err != nil {
		log.Fatal(err)
	}

	if err := testGen.WriteTestFiles(testFiles); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✅ Generated test files for %d packages\n", len(testFiles))
}

func cmdSecurityCheck(args []string) {
	fs := flag.NewFlagSet("security-check", flag.ExitOnError)
	blockOnHigh := fs.Bool("block", false, "block if high-severity issues found")
	verbose := fs.Bool("verbose", false, "show detailed findings")
	_ = fs.Parse(args)

	cfg := config.Load()
	if !cfg.Security.Enabled {
		fmt.Println("Security analysis is disabled in configuration")
		os.Exit(1)
	}

	// Override config with command line options
	if *blockOnHigh {
		cfg.Security.BlockOnHigh = true
	}

	// Get staged diff
	d, err := diff.Staged()
	if err != nil {
		log.Fatal(err)
	}
	if d == "" {
		fmt.Println("No staged changes found")
		return
	}

	secAnalyzer := security.New(cfg)
	report, err := secAnalyzer.AnalyzeDiff(d)
	if err != nil {
		log.Fatal(err)
	}

	// Display summary
	fmt.Printf("🔒 Security Analysis Results:\n")
	fmt.Printf("  Total findings: %d\n", report.Summary.TotalFindings)
	fmt.Printf("  High severity: %d\n", report.Summary.HighSeverity)
	fmt.Printf("  Medium severity: %d\n", report.Summary.MediumSeverity)
	fmt.Printf("  Low severity: %d\n", report.Summary.LowSeverity)

	if *verbose && len(report.Findings) > 0 {
		fmt.Println("\nDetailed Findings:")
		for _, finding := range report.Findings {
			fmt.Printf("  %s [%s] %s:%d - %s\n",
				getSeverityEmoji(finding.Severity),
				finding.Severity,
				finding.File,
				finding.Line,
				finding.Message)
			if finding.Suggestion != "" {
				fmt.Printf("    💡 %s\n", finding.Suggestion)
			}
		}
	}

	if len(report.Suggestions) > 0 {
		fmt.Println("\nGeneral Suggestions:")
		for _, suggestion := range report.Suggestions {
			fmt.Printf("  • %s\n", suggestion)
		}
	}

	if secAnalyzer.ShouldBlockCommit(report) {
		fmt.Fprintf(os.Stderr, "\n❌ Commit should be blocked due to high-severity security issues\n")
		os.Exit(1)
	}

	if report.Summary.TotalFindings == 0 {
		fmt.Println("\n✅ No security issues detected")
	}
}

func getSeverityEmoji(severity string) string {
	switch severity {
	case "high":
		return "🔴"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "❔"
	}
}
