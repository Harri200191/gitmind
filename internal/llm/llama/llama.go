package llama

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Harri200191/gitmind/internal/config"
	// NOTE: Replace with actual llama.cpp Go bindings import
	// "github.com/go-skynet/go-llama.cpp"
)

func Generate(cfg config.Config, diff string) (string, error) {
	if _, err := os.Stat(cfg.Model.ModelPath); err != nil {
		return "", fmt.Errorf("model file not found: %s", cfg.Model.ModelPath)
	}

	// TODO: Implement actual llama.cpp integration
	// For now, return a placeholder to demonstrate the structure
	return generatePlaceholder(cfg, diff), nil
}

// Placeholder implementation until actual llama.cpp bindings are integrated
func generatePlaceholder(cfg config.Config, diff string) string {
	// Simple heuristic-based generation as fallback
	lines := strings.Split(diff, "\n")
	addedLines := 0
	removedLines := 0
	files := make(map[string]bool)

	for _, line := range lines {
		if strings.HasPrefix(line, "+++ b/") {
			filename := strings.TrimPrefix(line, "+++ b/")
			files[filename] = true
		} else if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			addedLines++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			removedLines++
		}
	}

	fileList := make([]string, 0, len(files))
	for file := range files {
		fileList = append(fileList, file)
	}

	var subject string
	if len(fileList) == 1 {
		subject = fmt.Sprintf("feat: update %s", fileList[0])
	} else if len(fileList) > 1 {
		subject = fmt.Sprintf("feat: update %d files", len(fileList))
	} else {
		subject = "feat: update changes"
	}

	if cfg.Style == "conventional" && len(subject) > 72 {
		subject = subject[:72]
	}

	return subject
}

// TODO: Implement actual llama.cpp integration
func generateWithLlama(cfg config.Config, diff string) (string, error) {
	// This is where the actual llama.cpp integration would go
	// Example structure:

	/*
		l, err := llama.New(cfg.Model.ModelPath, llama.SetContext(cfg.Model.NCtx))
		if err != nil {
			return "", err
		}
		defer l.Free()

		prompt := buildPrompt(cfg, diff)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := l.Predict(ctx, prompt, llama.SetTemperature(cfg.Model.Temp))
		if err != nil {
			return "", err
		}

		return cleanupResponse(result), nil
	*/

	return "", errors.New("llama.cpp integration not implemented yet")
}

func buildPrompt(cfg config.Config, diff string) string {
	return fmt.Sprintf(`%s

%s

Here is the git diff of staged changes:
%s

Generate a commit message:`, cfg.Prompt.Preface, cfg.Prompt.Rules, diff)
}

func cleanupResponse(response string) string {
	// Clean up the LLM response to extract just the commit message
	lines := strings.Split(strings.TrimSpace(response), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return response
}
