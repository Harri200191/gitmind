package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Harri200191/gitmind/internal/config"
)

// SecurityAnalyzer handles security analysis of code changes
type SecurityAnalyzer struct {
	config config.Config
}

// Finding represents a security finding
type Finding struct {
	Severity   string                 `json:"severity"`
	Type       string                 `json:"type"`
	File       string                 `json:"file"`
	Line       int                    `json:"line"`
	Column     int                    `json:"column"`
	Message    string                 `json:"message"`
	Rule       string                 `json:"rule"`
	Suggestion string                 `json:"suggestion"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// SecurityReport contains all security findings
type SecurityReport struct {
	Findings    []Finding `json:"findings"`
	Summary     Summary   `json:"summary"`
	Suggestions []string  `json:"suggestions"`
}

// Summary provides overview of findings
type Summary struct {
	TotalFindings  int `json:"total_findings"`
	HighSeverity   int `json:"high_severity"`
	MediumSeverity int `json:"medium_severity"`
	LowSeverity    int `json:"low_severity"`
}

// New creates a new security analyzer
func New(cfg config.Config) *SecurityAnalyzer {
	return &SecurityAnalyzer{config: cfg}
}

func (sa *SecurityAnalyzer) AnalyzeDiff(diff string) (*SecurityReport, error) {
	if !sa.config.Security.Enabled {
		return &SecurityReport{}, nil
	}

	var allFindings []Finding

	// Extract changed files from diff
	changedFiles := sa.extractChangedFiles(diff)

	// Run enabled analyzers
	for _, analyzer := range sa.config.Security.Analyzers {
		findings, err := sa.runAnalyzer(analyzer, changedFiles)
		if err != nil {
			fmt.Printf("Warning: analyzer %s failed: %v\n", analyzer, err)
			continue
		}
		allFindings = append(allFindings, findings...)
	}

	// Add pattern-based analysis for diff content
	patternFindings := sa.analyzePatterns(diff, changedFiles)
	allFindings = append(allFindings, patternFindings...)

	// Generate report
	report := &SecurityReport{
		Findings:    allFindings,
		Summary:     sa.generateSummary(allFindings),
		Suggestions: sa.generateSuggestions(allFindings),
	}

	return report, nil
}

// ShouldBlockCommit determines if commit should be blocked based on findings
func (sa *SecurityAnalyzer) ShouldBlockCommit(report *SecurityReport) bool {
	if !sa.config.Security.BlockOnHigh {
		return false
	}

	return report.Summary.HighSeverity > 0
}

// GenerateCommitMessage creates security-aware commit message additions
func (sa *SecurityAnalyzer) GenerateCommitMessage(report *SecurityReport, baseMessage string) string {
	if !sa.config.Security.IncludeInMsg || len(report.Findings) == 0 {
		return baseMessage
	}

	var securityNotes []string

	if report.Summary.HighSeverity > 0 {
		securityNotes = append(securityNotes, fmt.Sprintf("⚠️  %d high-severity security issues", report.Summary.HighSeverity))
	}

	if report.Summary.MediumSeverity > 0 {
		securityNotes = append(securityNotes, fmt.Sprintf("⚡ %d medium-severity security issues", report.Summary.MediumSeverity))
	}

	if len(securityNotes) > 0 {
		return baseMessage + "\n\nSecurity Notes:\n" + strings.Join(securityNotes, "\n")
	}

	return baseMessage
}

// extractChangedFiles gets list of changed files from diff
func (sa *SecurityAnalyzer) extractChangedFiles(diff string) []string {
	var files []string
	lines := strings.Split(diff, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "+++ b/") {
			file := strings.TrimPrefix(line, "+++ b/")
			if file != "/dev/null" {
				files = append(files, file)
			}
		}
	}

	return files
}

// runAnalyzer executes a specific security analyzer
func (sa *SecurityAnalyzer) runAnalyzer(analyzer string, files []string) ([]Finding, error) {
	switch analyzer {
	case "gosec":
		return sa.runGosec(files)
	case "bandit":
		return sa.runBandit(files)
	case "eslint-security":
		return sa.runESLintSecurity(files)
	case "semgrep":
		return sa.runSemgrep(files)
	case "safety":
		return sa.runSafety(files)
	case "brakeman":
		return sa.runBrakeman(files)
	case "spotbugs":
		return sa.runSpotBugs(files)
	case "psalm":
		return sa.runPsalm(files)
	case "phpstan":
		return sa.runPHPStan(files)
	case "cppcheck":
		return sa.runCppCheck(files)
	case "flawfinder":
		return sa.runFlawfinder(files)
	case "cargo-audit":
		return sa.runCargoAudit(files)
	case "clippy":
		return sa.runClippy(files)
	case "securecodewarrior":
		return sa.runSecureCodeWarrior(files)
	default:
		return nil, fmt.Errorf("unknown analyzer: %s", analyzer)
	}
}

func (sa *SecurityAnalyzer) runGosec(files []string) ([]Finding, error) {
	goFiles := sa.filterFilesByExtension(files, ".go")

	if len(goFiles) == 0 {
		return nil, nil
	}

	if !sa.isCommandAvailable("gosec") {
		return nil, fmt.Errorf("gosec not found in PATH")
	}

	args := []string{"-fmt", "json", "-quiet"}
	args = append(args, goFiles...)

	cmd := exec.Command("gosec", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	fmt.Println("Raw gosec output:\n", stdout.String())

	if err := cmd.Run(); err != nil {
		if stdout.Len() == 0 {
			return nil, fmt.Errorf("gosec failed: %v, stderr: %s", err, stderr.String())
		}
	}

	if stdout.Len() == 0 {
		return nil, nil
	}

	return sa.parseGosecOutput(stdout.Bytes())
}

// runBandit runs bandit security analyzer for Python files
func (sa *SecurityAnalyzer) runBandit(files []string) ([]Finding, error) {
	// Filter for Python files only
	pythonFiles := sa.filterFilesByExtension(files, ".py")
	if len(pythonFiles) == 0 {
		return nil, nil
	}

	if !sa.isCommandAvailable("bandit") {
		return nil, fmt.Errorf("bandit not found in PATH")
	}

	args := []string{"-f", "json", "-q"}
	args = append(args, pythonFiles...)

	cmd := exec.Command("bandit", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		return nil, fmt.Errorf("bandit failed: %v, stderr: %s", err, stderr.String())
	}

	return sa.parseBanditOutput(stdout.Bytes())
}

// runESLintSecurity runs ESLint with security plugins for JavaScript/TypeScript files
func (sa *SecurityAnalyzer) runESLintSecurity(files []string) ([]Finding, error) {
	// Filter for JS/TS files
	jsFiles := sa.filterFilesByExtensions(files, []string{".js", ".ts", ".jsx", ".tsx"})
	if len(jsFiles) == 0 {
		return nil, nil
	}

	if !sa.isCommandAvailable("eslint") {
		return nil, fmt.Errorf("eslint not found in PATH")
	}

	args := []string{"--format", "json", "--no-eslintrc", "--config", sa.getESLintSecurityConfig()}
	args = append(args, jsFiles...)

	cmd := exec.Command("eslint", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		return nil, fmt.Errorf("eslint failed: %v, stderr: %s", err, stderr.String())
	}

	return sa.parseESLintOutput(stdout.Bytes())
}

// runSemgrep runs Semgrep with security rules
func (sa *SecurityAnalyzer) runSemgrep(files []string) ([]Finding, error) {
	if !sa.isCommandAvailable("semgrep") {
		return nil, fmt.Errorf("semgrep not found in PATH")
	}

	args := []string{
		"--config", "auto",
		"--json",
		"--quiet",
		"--severity", "ERROR",
		"--severity", "WARNING",
	}

	// Add files or use current directory if no specific files
	if len(files) > 0 {
		args = append(args, files...)
	} else {
		args = append(args, ".")
	}

	cmd := exec.Command("semgrep", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		return nil, fmt.Errorf("semgrep failed: %v, stderr: %s", err, stderr.String())
	}

	return sa.parseSemgrepOutput(stdout.Bytes())
}

// runSafety runs Safety for Python dependency vulnerabilities
func (sa *SecurityAnalyzer) runSafety(files []string) ([]Finding, error) {
	pythonFiles := sa.filterFilesByExtensions(files, []string{".py", "requirements.txt", "Pipfile", "pyproject.toml"})
	if len(pythonFiles) == 0 {
		return nil, nil
	}

	if !sa.isCommandAvailable("safety") {
		return nil, fmt.Errorf("safety not found in PATH")
	}

	args := []string{"check", "--json"}
	cmd := exec.Command("safety", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		return nil, fmt.Errorf("safety failed: %v, stderr: %s", err, stderr.String())
	}

	return sa.parseSafetyOutput(stdout.Bytes())
}

// runBrakeman runs Brakeman for Ruby on Rails security
func (sa *SecurityAnalyzer) runBrakeman(files []string) ([]Finding, error) {
	rubyFiles := sa.filterFilesByExtensions(files, []string{".rb", ".erb", "Gemfile"})
	if len(rubyFiles) == 0 {
		return nil, nil
	}

	if !sa.isCommandAvailable("brakeman") {
		return nil, fmt.Errorf("brakeman not found in PATH")
	}

	args := []string{"-f", "json", "-q"}
	cmd := exec.Command("brakeman", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		return nil, fmt.Errorf("brakeman failed: %v, stderr: %s", err, stderr.String())
	}

	return sa.parseBrakemanOutput(stdout.Bytes())
}

// runSpotBugs runs SpotBugs for Java security analysis
func (sa *SecurityAnalyzer) runSpotBugs(files []string) ([]Finding, error) {
	javaFiles := sa.filterFilesByExtensions(files, []string{".java", ".class", ".jar"})
	if len(javaFiles) == 0 {
		return nil, nil
	}

	if !sa.isCommandAvailable("spotbugs") {
		return nil, fmt.Errorf("spotbugs not found in PATH")
	}

	args := []string{"-textui", "-xml", "-effort:max"}
	args = append(args, javaFiles...)

	cmd := exec.Command("spotbugs", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		return nil, fmt.Errorf("spotbugs failed: %v, stderr: %s", err, stderr.String())
	}

	return sa.parseSpotBugsOutput(stdout.Bytes())
}

// runPsalm runs Psalm for PHP static analysis
func (sa *SecurityAnalyzer) runPsalm(files []string) ([]Finding, error) {
	phpFiles := sa.filterFilesByExtension(files, ".php")
	if len(phpFiles) == 0 {
		return nil, nil
	}

	if !sa.isCommandAvailable("psalm") {
		return nil, fmt.Errorf("psalm not found in PATH")
	}

	args := []string{"--output-format=json", "--find-unused-code"}
	args = append(args, phpFiles...)

	cmd := exec.Command("psalm", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		return nil, fmt.Errorf("psalm failed: %v, stderr: %s", err, stderr.String())
	}

	return sa.parsePsalmOutput(stdout.Bytes())
}

// runPHPStan runs PHPStan for PHP static analysis
func (sa *SecurityAnalyzer) runPHPStan(files []string) ([]Finding, error) {
	phpFiles := sa.filterFilesByExtension(files, ".php")
	if len(phpFiles) == 0 {
		return nil, nil
	}

	if !sa.isCommandAvailable("phpstan") {
		return nil, fmt.Errorf("phpstan not found in PATH")
	}

	args := []string{"analyze", "--error-format=json", "--level=max"}
	args = append(args, phpFiles...)

	cmd := exec.Command("phpstan", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		return nil, fmt.Errorf("phpstan failed: %v, stderr: %s", err, stderr.String())
	}

	return sa.parsePHPStanOutput(stdout.Bytes())
}

// runCppCheck runs CppCheck for C/C++ static analysis
func (sa *SecurityAnalyzer) runCppCheck(files []string) ([]Finding, error) {
	cppFiles := sa.filterFilesByExtensions(files, []string{".c", ".cpp", ".cxx", ".cc", ".h", ".hpp"})
	if len(cppFiles) == 0 {
		return nil, nil
	}

	if !sa.isCommandAvailable("cppcheck") {
		return nil, fmt.Errorf("cppcheck not found in PATH")
	}

	args := []string{"--xml", "--enable=all", "--inconclusive", "--std=c++17"}
	args = append(args, cppFiles...)

	cmd := exec.Command("cppcheck", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil && stderr.Len() == 0 {
		return nil, fmt.Errorf("cppcheck failed: %v", err)
	}

	// CppCheck outputs to stderr by default
	return sa.parseCppCheckOutput(stderr.Bytes())
}

// runFlawfinder runs Flawfinder for C/C++ security analysis
func (sa *SecurityAnalyzer) runFlawfinder(files []string) ([]Finding, error) {
	cppFiles := sa.filterFilesByExtensions(files, []string{".c", ".cpp", ".cxx", ".cc", ".h", ".hpp"})
	if len(cppFiles) == 0 {
		return nil, nil
	}

	if !sa.isCommandAvailable("flawfinder") {
		return nil, fmt.Errorf("flawfinder not found in PATH")
	}

	args := []string{"--sarif"}
	args = append(args, cppFiles...)

	cmd := exec.Command("flawfinder", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		return nil, fmt.Errorf("flawfinder failed: %v, stderr: %s", err, stderr.String())
	}

	return sa.parseFlawfinderOutput(stdout.Bytes())
}

// runCargoAudit runs cargo-audit for Rust dependency vulnerabilities
func (sa *SecurityAnalyzer) runCargoAudit(files []string) ([]Finding, error) {
	rustFiles := sa.filterFilesByExtensions(files, []string{".rs", "Cargo.toml", "Cargo.lock"})
	if len(rustFiles) == 0 {
		return nil, nil
	}

	if !sa.isCommandAvailable("cargo") {
		return nil, fmt.Errorf("cargo not found in PATH")
	}

	args := []string{"audit", "--format", "json"}
	cmd := exec.Command("cargo", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		return nil, fmt.Errorf("cargo-audit failed: %v, stderr: %s", err, stderr.String())
	}

	return sa.parseCargoAuditOutput(stdout.Bytes())
}

// runClippy runs Clippy for Rust static analysis
func (sa *SecurityAnalyzer) runClippy(files []string) ([]Finding, error) {
	rustFiles := sa.filterFilesByExtension(files, ".rs")
	if len(rustFiles) == 0 {
		return nil, nil
	}

	if !sa.isCommandAvailable("cargo") {
		return nil, fmt.Errorf("cargo not found in PATH")
	}

	args := []string{"clippy", "--message-format=json", "--", "-W", "clippy::all"}
	cmd := exec.Command("cargo", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		return nil, fmt.Errorf("clippy failed: %v, stderr: %s", err, stderr.String())
	}

	return sa.parseClippyOutput(stdout.Bytes())
}

// runSecureCodeWarrior runs a comprehensive multi-language scanner
func (sa *SecurityAnalyzer) runSecureCodeWarrior(files []string) ([]Finding, error) {
	if len(files) == 0 {
		return nil, nil
	}

	// This is a placeholder for a comprehensive security scanner
	// In practice, this could integrate with commercial tools like Veracode, Checkmarx, etc.
	var findings []Finding

	// Enhanced pattern-based analysis for multi-language support
	languagePatterns := sa.getLanguageSpecificPatterns()

	for _, file := range files {
		lang := sa.detectLanguage(file)
		if patterns, exists := languagePatterns[lang]; exists {
			fileFindings := sa.analyzeFileWithPatterns(file, patterns)
			findings = append(findings, fileFindings...)
		}
	}

	return findings, nil
}

// analyzePatterns performs pattern-based security analysis on diff content
func (sa *SecurityAnalyzer) analyzePatterns(diff string, files []string) []Finding {
	var findings []Finding

	// Common security patterns to look for
	patterns := []struct {
		Pattern    *regexp.Regexp
		Severity   string
		Type       string
		Message    string
		Suggestion string
	}{
		{
			Pattern:    regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*["\'][^"\']*["\']`),
			Severity:   "high",
			Type:       "hardcoded-password",
			Message:    "Hardcoded password detected",
			Suggestion: "Use environment variables or secure configuration",
		},
		{
			Pattern:    regexp.MustCompile(`(?i)(api[_-]?key|secret[_-]?key|access[_-]?token)\s*[:=]\s*["\'][^"\']*["\']`),
			Severity:   "high",
			Type:       "hardcoded-secret",
			Message:    "Hardcoded API key or secret detected",
			Suggestion: "Use environment variables or secure vault",
		},
		{
			Pattern:    regexp.MustCompile(`(?i)eval\s*\(`),
			Severity:   "high",
			Type:       "code-injection",
			Message:    "Use of eval() function detected",
			Suggestion: "Avoid eval(), use safer alternatives",
		},
		{
			Pattern:    regexp.MustCompile(`(?i)exec\s*\(.*\$`),
			Severity:   "high",
			Type:       "command-injection",
			Message:    "Potential command injection detected",
			Suggestion: "Validate and sanitize input before exec",
		},
		{
			Pattern:    regexp.MustCompile(`(?i)sql.*\+.*\$`),
			Severity:   "medium",
			Type:       "sql-injection",
			Message:    "Potential SQL injection detected",
			Suggestion: "Use parameterized queries",
		},
		{
			Pattern:    regexp.MustCompile(`(?i)http://`),
			Severity:   "low",
			Type:       "insecure-protocol",
			Message:    "Insecure HTTP protocol detected",
			Suggestion: "Use HTTPS instead",
		},
	}

	lines := strings.Split(diff, "\n")
	currentFile := ""
	lineNumber := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "+++ b/") {
			currentFile = strings.TrimPrefix(line, "+++ b/")
			lineNumber = 0
			continue
		}

		if strings.HasPrefix(line, "@@") {
			// Extract line number from hunk header
			re := regexp.MustCompile(`\+(\d+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				fmt.Sscanf(matches[1], "%d", &lineNumber)
			}
			continue
		}

		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			lineNumber++
			content := strings.TrimPrefix(line, "+")

			// Check against security patterns
			for _, pattern := range patterns {
				if pattern.Pattern.MatchString(content) {
					finding := Finding{
						Severity:   pattern.Severity,
						Type:       pattern.Type,
						File:       currentFile,
						Line:       lineNumber,
						Message:    pattern.Message,
						Rule:       "pattern-analysis",
						Suggestion: pattern.Suggestion,
						Metadata: map[string]interface{}{
							"matched_content": strings.TrimSpace(content),
						},
					}
					findings = append(findings, finding)
				}
			}
		}
	}

	return findings
}

// Helper functions for parsing analyzer outputs
func (sa *SecurityAnalyzer) parseGosecOutput(output []byte) ([]Finding, error) {
	var result struct {
		Issues []struct {
			Severity   string `json:"severity"`
			Confidence string `json:"confidence"`
			RuleID     string `json:"rule_id"`
			Details    string `json:"details"`
			File       string `json:"file"`
			Code       string `json:"code"`
			Line       string `json:"line"`
			Column     string `json:"column"`
		} `json:"Issues"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse gosec output: %v", err)
	}

	var findings []Finding
	for _, issue := range result.Issues {
		line, _ := sa.parseInt(issue.Line)
		column, _ := sa.parseInt(issue.Column)

		finding := Finding{
			Severity:   strings.ToLower(issue.Severity),
			Type:       "gosec-" + issue.RuleID,
			File:       issue.File,
			Line:       line,
			Column:     column,
			Message:    issue.Details,
			Rule:       issue.RuleID,
			Suggestion: sa.getGosecSuggestion(issue.RuleID),
			Metadata: map[string]interface{}{
				"confidence": issue.Confidence,
				"code":       issue.Code,
			},
		}
		findings = append(findings, finding)
	}

	return findings, nil
}

func (sa *SecurityAnalyzer) parseBanditOutput(output []byte) ([]Finding, error) {
	var result struct {
		Results []struct {
			TestName   string `json:"test_name"`
			TestID     string `json:"test_id"`
			Severity   string `json:"issue_severity"`
			Confidence string `json:"issue_confidence"`
			Text       string `json:"issue_text"`
			Filename   string `json:"filename"`
			LineNumber int    `json:"line_number"`
			LineRange  []int  `json:"line_range"`
			Code       string `json:"code"`
		} `json:"results"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse bandit output: %v", err)
	}

	var findings []Finding
	for _, issue := range result.Results {
		finding := Finding{
			Severity:   strings.ToLower(issue.Severity),
			Type:       "bandit-" + issue.TestID,
			File:       issue.Filename,
			Line:       issue.LineNumber,
			Message:    issue.Text,
			Rule:       issue.TestID,
			Suggestion: sa.getBanditSuggestion(issue.TestID),
			Metadata: map[string]interface{}{
				"confidence": issue.Confidence,
				"test_name":  issue.TestName,
				"code":       issue.Code,
			},
		}
		findings = append(findings, finding)
	}

	return findings, nil
}

func (sa *SecurityAnalyzer) parseESLintOutput(output []byte) ([]Finding, error) {
	var results []struct {
		FilePath string `json:"filePath"`
		Messages []struct {
			RuleID   string `json:"ruleId"`
			Severity int    `json:"severity"`
			Message  string `json:"message"`
			Line     int    `json:"line"`
			Column   int    `json:"column"`
		} `json:"messages"`
	}

	if err := json.Unmarshal(output, &results); err != nil {
		return nil, fmt.Errorf("failed to parse eslint output: %v", err)
	}

	var findings []Finding
	for _, file := range results {
		for _, msg := range file.Messages {
			severity := "low"
			if msg.Severity == 2 {
				severity = "medium"
			}

			finding := Finding{
				Severity:   severity,
				Type:       "eslint-" + msg.RuleID,
				File:       file.FilePath,
				Line:       msg.Line,
				Column:     msg.Column,
				Message:    msg.Message,
				Rule:       msg.RuleID,
				Suggestion: sa.getESLintSuggestion(msg.RuleID),
			}
			findings = append(findings, finding)
		}
	}

	return findings, nil
}

func (sa *SecurityAnalyzer) parseSemgrepOutput(output []byte) ([]Finding, error) {
	var result struct {
		Results []struct {
			CheckID string `json:"check_id"`
			Path    string `json:"path"`
			Start   struct {
				Line int `json:"line"`
				Col  int `json:"col"`
			} `json:"start"`
			End struct {
				Line int `json:"line"`
				Col  int `json:"col"`
			} `json:"end"`
			Extra struct {
				Message  string `json:"message"`
				Severity string `json:"severity"`
			} `json:"extra"`
		} `json:"results"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse semgrep output: %v", err)
	}

	var findings []Finding
	for _, issue := range result.Results {
		finding := Finding{
			Severity:   strings.ToLower(issue.Extra.Severity),
			Type:       "semgrep-" + issue.CheckID,
			File:       issue.Path,
			Line:       issue.Start.Line,
			Column:     issue.Start.Col,
			Message:    issue.Extra.Message,
			Rule:       issue.CheckID,
			Suggestion: sa.getSemgrepSuggestion(issue.CheckID),
		}
		findings = append(findings, finding)
	}

	return findings, nil
}

// Parser functions for new analyzers
func (sa *SecurityAnalyzer) parseSafetyOutput(output []byte) ([]Finding, error) {
	var issues []struct {
		ID            string `json:"id"`
		Vulnerability string `json:"vulnerability"`
		PackageName   string `json:"package_name"`
		Version       string `json:"version"`
		Severity      string `json:"severity"`
	}

	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse safety output: %v", err)
	}

	var findings []Finding
	for _, issue := range issues {
		finding := Finding{
			Severity:   strings.ToLower(issue.Severity),
			Type:       "safety-" + issue.ID,
			File:       "requirements.txt", // Default to requirements file
			Line:       1,
			Message:    fmt.Sprintf("Vulnerability in %s %s: %s", issue.PackageName, issue.Version, issue.Vulnerability),
			Rule:       issue.ID,
			Suggestion: "Update to a secure version of the package",
			Metadata: map[string]interface{}{
				"package": issue.PackageName,
				"version": issue.Version,
			},
		}
		findings = append(findings, finding)
	}

	return findings, nil
}

func (sa *SecurityAnalyzer) parseBrakemanOutput(output []byte) ([]Finding, error) {
	var result struct {
		Warnings []struct {
			Type       string `json:"warning_type"`
			Code       string `json:"warning_code"`
			Message    string `json:"message"`
			File       string `json:"file"`
			Line       int    `json:"line"`
			Confidence string `json:"confidence"`
		} `json:"warnings"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse brakeman output: %v", err)
	}

	var findings []Finding
	for _, warning := range result.Warnings {
		severity := "medium"
		if warning.Confidence == "High" {
			severity = "high"
		} else if warning.Confidence == "Low" {
			severity = "low"
		}

		finding := Finding{
			Severity:   severity,
			Type:       "brakeman-" + warning.Type,
			File:       warning.File,
			Line:       warning.Line,
			Message:    warning.Message,
			Rule:       warning.Code,
			Suggestion: sa.getBrakemanSuggestion(warning.Type),
			Metadata: map[string]interface{}{
				"confidence": warning.Confidence,
			},
		}
		findings = append(findings, finding)
	}

	return findings, nil
}

func (sa *SecurityAnalyzer) parseSpotBugsOutput(output []byte) ([]Finding, error) {
	// SpotBugs XML parsing would be more complex
	// For simplicity, this is a basic implementation
	var findings []Finding

	// In a real implementation, you'd parse the XML output
	// This is a placeholder that looks for basic patterns
	outputStr := string(output)
	if strings.Contains(outputStr, "SECURITY") {
		finding := Finding{
			Severity:   "medium",
			Type:       "spotbugs-security",
			File:       "unknown",
			Line:       1,
			Message:    "Security issue detected by SpotBugs",
			Rule:       "SECURITY",
			Suggestion: "Review SpotBugs report for details",
		}
		findings = append(findings, finding)
	}

	return findings, nil
}

func (sa *SecurityAnalyzer) parsePsalmOutput(output []byte) ([]Finding, error) {
	var issues []struct {
		Type     string `json:"type"`
		Message  string `json:"message"`
		FilePath string `json:"file_path"`
		Line     int    `json:"line_from"`
		Severity string `json:"severity"`
	}

	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse psalm output: %v", err)
	}

	var findings []Finding
	for _, issue := range issues {
		finding := Finding{
			Severity:   strings.ToLower(issue.Severity),
			Type:       "psalm-" + issue.Type,
			File:       issue.FilePath,
			Line:       issue.Line,
			Message:    issue.Message,
			Rule:       issue.Type,
			Suggestion: sa.getPHPSuggestion(issue.Type),
		}
		findings = append(findings, finding)
	}

	return findings, nil
}

func (sa *SecurityAnalyzer) parsePHPStanOutput(output []byte) ([]Finding, error) {
	var result struct {
		Files map[string]struct {
			Messages []struct {
				Message string `json:"message"`
				Line    int    `json:"line"`
			} `json:"messages"`
		} `json:"files"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse phpstan output: %v", err)
	}

	var findings []Finding
	for filePath, fileData := range result.Files {
		for _, msg := range fileData.Messages {
			finding := Finding{
				Severity:   "medium",
				Type:       "phpstan-error",
				File:       filePath,
				Line:       msg.Line,
				Message:    msg.Message,
				Rule:       "phpstan",
				Suggestion: "Fix type errors and static analysis issues",
			}
			findings = append(findings, finding)
		}
	}

	return findings, nil
}

func (sa *SecurityAnalyzer) parseCppCheckOutput(output []byte) ([]Finding, error) {
	// CppCheck XML parsing would be more complex
	// For simplicity, this is a basic implementation
	var findings []Finding

	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")

	for _, line := range lines {
		if strings.Contains(line, "error") || strings.Contains(line, "warning") {
			// Basic parsing of cppcheck output
			finding := Finding{
				Severity:   "medium",
				Type:       "cppcheck-issue",
				File:       "unknown",
				Line:       1,
				Message:    line,
				Rule:       "cppcheck",
				Suggestion: "Review and fix C/C++ issues",
			}
			findings = append(findings, finding)
		}
	}

	return findings, nil
}

func (sa *SecurityAnalyzer) parseFlawfinderOutput(output []byte) ([]Finding, error) {
	// SARIF format parsing would be complex
	// For simplicity, basic implementation
	var findings []Finding

	outputStr := string(output)
	if strings.Contains(outputStr, "CWE") {
		finding := Finding{
			Severity:   "medium",
			Type:       "flawfinder-cwe",
			File:       "unknown",
			Line:       1,
			Message:    "Security vulnerability detected",
			Rule:       "flawfinder",
			Suggestion: "Review flawfinder report for CWE details",
		}
		findings = append(findings, finding)
	}

	return findings, nil
}

func (sa *SecurityAnalyzer) parseCargoAuditOutput(output []byte) ([]Finding, error) {
	var result struct {
		Vulnerabilities struct {
			List []struct {
				Advisory struct {
					ID          string `json:"id"`
					Title       string `json:"title"`
					Description string `json:"description"`
				} `json:"advisory"`
				Package struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				} `json:"package"`
			} `json:"list"`
		} `json:"vulnerabilities"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse cargo-audit output: %v", err)
	}

	var findings []Finding
	for _, vuln := range result.Vulnerabilities.List {
		finding := Finding{
			Severity:   "high",
			Type:       "cargo-audit-" + vuln.Advisory.ID,
			File:       "Cargo.toml",
			Line:       1,
			Message:    fmt.Sprintf("%s in %s %s: %s", vuln.Advisory.Title, vuln.Package.Name, vuln.Package.Version, vuln.Advisory.Description),
			Rule:       vuln.Advisory.ID,
			Suggestion: "Update to a patched version of the crate",
			Metadata: map[string]interface{}{
				"package": vuln.Package.Name,
				"version": vuln.Package.Version,
			},
		}
		findings = append(findings, finding)
	}

	return findings, nil
}

func (sa *SecurityAnalyzer) parseClippyOutput(output []byte) ([]Finding, error) {
	var findings []Finding
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.Contains(line, "warning") && strings.Contains(line, "clippy") {
			finding := Finding{
				Severity:   "low",
				Type:       "clippy-warning",
				File:       "unknown",
				Line:       1,
				Message:    line,
				Rule:       "clippy",
				Suggestion: "Follow Rust best practices",
			}
			findings = append(findings, finding)
		}
	}

	return findings, nil
}

// Helper utility functions
func (sa *SecurityAnalyzer) filterFilesByExtension(files []string, ext string) []string {
	var filtered []string
	for _, file := range files {
		if strings.HasSuffix(file, ext) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

func (sa *SecurityAnalyzer) filterFilesByExtensions(files []string, exts []string) []string {
	var filtered []string
	for _, file := range files {
		for _, ext := range exts {
			if strings.HasSuffix(file, ext) {
				filtered = append(filtered, file)
				break
			}
		}
	}
	return filtered
}

func (sa *SecurityAnalyzer) isCommandAvailable(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func (sa *SecurityAnalyzer) parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

func (sa *SecurityAnalyzer) generateSummary(findings []Finding) Summary {
	summary := Summary{}

	for _, finding := range findings {
		summary.TotalFindings++
		switch finding.Severity {
		case "high":
			summary.HighSeverity++
		case "medium":
			summary.MediumSeverity++
		case "low":
			summary.LowSeverity++
		}
	}

	return summary
}

func (sa *SecurityAnalyzer) generateSuggestions(findings []Finding) []string {
	suggestionMap := make(map[string]bool)

	for _, finding := range findings {
		if finding.Suggestion != "" && !suggestionMap[finding.Suggestion] {
			suggestionMap[finding.Suggestion] = true
		}
	}

	var suggestions []string
	for suggestion := range suggestionMap {
		suggestions = append(suggestions, suggestion)
	}

	return suggestions
}

func (sa *SecurityAnalyzer) getESLintSecurityConfig() string {
	// Return path to a minimal ESLint security configuration
	// In a real implementation, this would be a proper config file
	return `{
		"plugins": ["security"],
		"rules": {
			"security/detect-object-injection": "error",
			"security/detect-non-literal-regexp": "error",
			"security/detect-eval-with-expression": "error",
			"security/detect-pseudoRandomBytes": "error"
		}
	}`
}

func (sa *SecurityAnalyzer) getGosecSuggestion(ruleID string) string {
	suggestions := map[string]string{
		"G101": "Remove hardcoded credentials, use environment variables",
		"G102": "Avoid binding to all interfaces, specify specific addresses",
		"G103": "Audit use of unsafe package",
		"G104": "Check error return values",
		"G201": "Use parameterized queries to prevent SQL injection",
		"G301": "Set appropriate file permissions",
		"G302": "Set appropriate file permissions for sensitive files",
		"G401": "Use stronger cryptographic algorithms",
		"G501": "Use strong cryptographic hash functions",
	}

	if suggestion, exists := suggestions[ruleID]; exists {
		return suggestion
	}
	return "Review and fix the security issue"
}

func (sa *SecurityAnalyzer) getBanditSuggestion(testID string) string {
	suggestions := map[string]string{
		"B101": "Remove hardcoded passwords",
		"B102": "Use subprocess with shell=False",
		"B103": "Set file permissions explicitly",
		"B104": "Avoid binding to all interfaces",
		"B105": "Remove hardcoded passwords",
		"B201": "Use parameterized queries",
		"B301": "Use safe pickle alternatives",
		"B401": "Use secure random generators",
		"B501": "Don't use weak SSL/TLS protocols",
	}

	if suggestion, exists := suggestions[testID]; exists {
		return suggestion
	}
	return "Review and fix the security issue"
}

func (sa *SecurityAnalyzer) getESLintSuggestion(ruleID string) string {
	suggestions := map[string]string{
		"security/detect-object-injection":     "Validate object keys before access",
		"security/detect-non-literal-regexp":   "Use literal regex patterns",
		"security/detect-eval-with-expression": "Avoid eval(), use safer alternatives",
		"security/detect-pseudoRandomBytes":    "Use cryptographically secure random functions",
	}

	if suggestion, exists := suggestions[ruleID]; exists {
		return suggestion
	}
	return "Review and fix the security issue"
}

func (sa *SecurityAnalyzer) getSemgrepSuggestion(checkID string) string {
	// Generic suggestion based on common patterns
	if strings.Contains(checkID, "injection") {
		return "Validate and sanitize all inputs"
	}
	if strings.Contains(checkID, "crypto") {
		return "Use secure cryptographic practices"
	}
	if strings.Contains(checkID, "auth") {
		return "Implement proper authentication"
	}
	return "Review and fix the security issue"
}

// Additional suggestion functions for new analyzers
func (sa *SecurityAnalyzer) getBrakemanSuggestion(warningType string) string {
	suggestions := map[string]string{
		"SQL Injection":              "Use parameterized queries or ORM methods",
		"Cross-Site Scripting":       "Sanitize user input and use proper escaping",
		"Command Injection":          "Avoid system calls with user input",
		"File Access":                "Validate file paths and restrict access",
		"Mass Assignment":            "Use strong parameters in Rails",
		"Redirect":                   "Validate redirect URLs",
		"Session Setting":            "Use secure session configuration",
		"Cross-Site Request Forgery": "Implement CSRF protection",
	}

	if suggestion, exists := suggestions[warningType]; exists {
		return suggestion
	}
	return "Review Ruby on Rails security best practices"
}

func (sa *SecurityAnalyzer) getPHPSuggestion(issueType string) string {
	suggestions := map[string]string{
		"PossiblyUndefinedVariable": "Initialize variables before use",
		"UndefinedMethod":           "Check method names and class inheritance",
		"InvalidArgument":           "Validate function arguments and types",
		"TypeDoesNotContainType":    "Review type declarations and usage",
		"PossiblyNullReference":     "Add null checks before accessing properties",
		"UnusedVariable":            "Remove unused variables or mark as used",
	}

	if suggestion, exists := suggestions[issueType]; exists {
		return suggestion
	}
	return "Follow PHP best practices and fix type errors"
}

// Language detection and pattern analysis functions
func (sa *SecurityAnalyzer) detectLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js", ".ts", ".jsx", ".tsx":
		return "javascript"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".java":
		return "java"
	case ".c", ".cpp", ".cxx", ".cc", ".h", ".hpp":
		return "cpp"
	case ".rs":
		return "rust"
	case ".cs":
		return "csharp"
	case ".kt":
		return "kotlin"
	case ".swift":
		return "swift"
	case ".scala":
		return "scala"
	default:
		return "unknown"
	}
}

func (sa *SecurityAnalyzer) getLanguageSpecificPatterns() map[string][]SecurityPattern {
	return map[string][]SecurityPattern{
		"python": {
			{Pattern: regexp.MustCompile(`eval\s*\(`), Severity: "high", Type: "code-injection", Message: "Dangerous eval() usage", Suggestion: "Use ast.literal_eval() or safer alternatives"},
			{Pattern: regexp.MustCompile(`exec\s*\(`), Severity: "high", Type: "code-injection", Message: "Dangerous exec() usage", Suggestion: "Avoid exec() or validate input thoroughly"},
			{Pattern: regexp.MustCompile(`pickle\.loads\s*\(`), Severity: "high", Type: "deserialization", Message: "Unsafe pickle deserialization", Suggestion: "Use JSON or validate pickle data"},
			{Pattern: regexp.MustCompile(`subprocess\s*\(`), Severity: "medium", Type: "command-injection", Message: "Potential command injection", Suggestion: "Use shell=False and validate arguments"},
		},
		"javascript": {
			{Pattern: regexp.MustCompile(`eval\s*\(`), Severity: "high", Type: "code-injection", Message: "Dangerous eval() usage", Suggestion: "Use JSON.parse() or safer alternatives"},
			{Pattern: regexp.MustCompile(`innerHTML\s*=`), Severity: "medium", Type: "xss", Message: "Potential XSS via innerHTML", Suggestion: "Use textContent or sanitize HTML"},
			{Pattern: regexp.MustCompile(`document\.write\s*\(`), Severity: "medium", Type: "xss", Message: "Potential XSS via document.write", Suggestion: "Use safer DOM manipulation methods"},
			{Pattern: regexp.MustCompile(`localStorage\.setItem`), Severity: "low", Type: "data-exposure", Message: "Sensitive data in localStorage", Suggestion: "Avoid storing sensitive data in localStorage"},
		},
		"php": {
			{Pattern: regexp.MustCompile(`eval\s*\(`), Severity: "high", Type: "code-injection", Message: "Dangerous eval() usage", Suggestion: "Remove eval() usage"},
			{Pattern: regexp.MustCompile(`\$_GET\[.*\]`), Severity: "medium", Type: "injection", Message: "Unvalidated GET parameter", Suggestion: "Validate and sanitize input"},
			{Pattern: regexp.MustCompile(`\$_POST\[.*\]`), Severity: "medium", Type: "injection", Message: "Unvalidated POST parameter", Suggestion: "Validate and sanitize input"},
			{Pattern: regexp.MustCompile(`mysql_query\s*\(`), Severity: "high", Type: "sql-injection", Message: "Deprecated MySQL function", Suggestion: "Use PDO or mysqli with prepared statements"},
		},
		"java": {
			{Pattern: regexp.MustCompile(`Runtime\.getRuntime\(\)\.exec`), Severity: "high", Type: "command-injection", Message: "Dangerous Runtime.exec usage", Suggestion: "Use ProcessBuilder and validate input"},
			{Pattern: regexp.MustCompile(`Class\.forName\s*\(`), Severity: "medium", Type: "reflection", Message: "Dynamic class loading", Suggestion: "Validate class names and use allowlists"},
			{Pattern: regexp.MustCompile(`ObjectInputStream\.readObject`), Severity: "high", Type: "deserialization", Message: "Unsafe deserialization", Suggestion: "Validate serialized data or use safer formats"},
		},
		"cpp": {
			{Pattern: regexp.MustCompile(`strcpy\s*\(`), Severity: "high", Type: "buffer-overflow", Message: "Unsafe strcpy usage", Suggestion: "Use strncpy or safer string functions"},
			{Pattern: regexp.MustCompile(`gets\s*\(`), Severity: "high", Type: "buffer-overflow", Message: "Dangerous gets() function", Suggestion: "Use fgets() instead"},
			{Pattern: regexp.MustCompile(`sprintf\s*\(`), Severity: "medium", Type: "buffer-overflow", Message: "Potentially unsafe sprintf", Suggestion: "Use snprintf for safer formatting"},
		},
		"rust": {
			{Pattern: regexp.MustCompile(`unsafe\s*\{`), Severity: "medium", Type: "unsafe-code", Message: "Unsafe code block", Suggestion: "Review unsafe code for memory safety"},
			{Pattern: regexp.MustCompile(`\.unwrap\(\)`), Severity: "low", Type: "panic", Message: "Potential panic with unwrap", Suggestion: "Use proper error handling"},
		},
	}
}

type SecurityPattern struct {
	Pattern    *regexp.Regexp
	Severity   string
	Type       string
	Message    string
	Suggestion string
}

func (sa *SecurityAnalyzer) analyzeFileWithPatterns(filename string, patterns []SecurityPattern) []Finding {
	var findings []Finding

	// This would read the file and analyze it with the patterns
	// For this implementation, we'll skip the file reading part
	// In a real implementation, you'd read the file content and analyze it

	return findings
}
