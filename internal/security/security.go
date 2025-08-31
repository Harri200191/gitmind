package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
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

// AnalyzeDiff performs security analysis on git diff
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
	default:
		return nil, fmt.Errorf("unknown analyzer: %s", analyzer)
	}
}

// runGosec runs gosec security analyzer for Go files
func (sa *SecurityAnalyzer) runGosec(files []string) ([]Finding, error) {
	// Filter for Go files only
	goFiles := sa.filterFilesByExtension(files, ".go")
	if len(goFiles) == 0 {
		return nil, nil
	}

	// Check if gosec is available
	if !sa.isCommandAvailable("gosec") {
		return nil, fmt.Errorf("gosec not found in PATH")
	}

	// Run gosec with JSON output
	args := []string{"-fmt", "json", "-quiet"}
	args = append(args, goFiles...)

	cmd := exec.Command("gosec", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// gosec returns non-zero exit code when findings are present
		// Only treat it as error if stdout is empty
		if stdout.Len() == 0 {
			return nil, fmt.Errorf("gosec failed: %v, stderr: %s", err, stderr.String())
		}
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
