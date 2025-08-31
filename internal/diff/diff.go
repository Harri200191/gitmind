package diff

import (
	"bytes"
	"os/exec"
	"regexp"
	"strings"

	"github.com/yourname/commitgen/internal/config"
)

func Staged() (string, error) {
	cmd := exec.Command("git", "diff", "--cached", "-U0")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func HeuristicMessage(d string, cfg config.Config) string {
	// crude mapping by filenames/extensions
	files := changedFiles(d)
	kind := kindFromFiles(files)
	subject := strings.TrimSpace(kind + ": " + summarizeFiles(files))
	if len(subject) > 72 {
		subject = subject[:72]
	}
	body := summarizeHunks(d, cfg.MaxSummaryLines)
	if body == "" {
		return subject
	}
	return subject + "\n\n" + body
}

func changedFiles(d string) []string {
	var files []string
	for _, line := range strings.Split(d, "\n") {
		if strings.HasPrefix(line, "+++ b/") {
			files = append(files, strings.TrimPrefix(line, "+++ b/"))
		}
	}
	return unique(files)
}

func kindFromFiles(files []string) string {
	var (
		reDocs   = regexp.MustCompile(`(?i)\.(md|rst|adoc)$`)
		reTests  = regexp.MustCompile(`(?i)(^|/)test(s)?/|(_test)\.go$`)
		reConfig = regexp.MustCompile(`(?i)\.(ya?ml|json|toml|ini)$`)
	)
	if anyMatch(files, reDocs) {
		return "docs"
	}
	if anyMatch(files, reTests) {
		return "test"
	}
	if anyMatch(files, reConfig) {
		return "chore"
	}
	return "feat"
}

func summarizeFiles(files []string) string {
	if len(files) == 0 {
		return "update changes"
	}
	if len(files) == 1 {
		return "update " + files[0]
	}
	if len(files) == 2 {
		return "update " + files[0] + ", " + files[1]
	}
	return "update " + files[0] + " and " + string(len(files)-1+'0') + " more files"
}

func summarizeHunks(d string, maxLines int) string {
	lines := []string{}
	for _, l := range strings.Split(d, "\n") {
		if strings.HasPrefix(l, "+") && !strings.HasPrefix(l, "+++") {
			if trimmed := strings.TrimSpace(strings.TrimPrefix(l, "+")); trimmed != "" {
				lines = append(lines, "+ "+trimWidth(trimmed, 72))
			}
		}
		if strings.HasPrefix(l, "-") && !strings.HasPrefix(l, "---") {
			if trimmed := strings.TrimSpace(strings.TrimPrefix(l, "-")); trimmed != "" {
				lines = append(lines, "- "+trimWidth(trimmed, 72))
			}
		}
		if len(lines) >= maxLines {
			break
		}
	}
	return strings.Join(lines, "\n")
}

func trimWidth(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func anyMatch(files []string, re *regexp.Regexp) bool {
	for _, f := range files {
		if re.MatchString(f) {
			return true
		}
	}
	return false
}

func unique(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}
