package llm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Harri200191/gitmind/internal/config"
)

func Doctor(cfg config.Config) (bool, string) {
	if !cfg.Model.Enabled {
		return false, "model disabled in config"
	}

	switch strings.ToLower(cfg.Model.Provider) {
	case "ollama":
		// Check if Ollama is running and model is available
		if err := checkOllamaHealth(); err != nil {
			return false, fmt.Sprintf("Ollama not accessible: %v", err)
		}
		if cfg.Model.ModelPath == "" {
			return false, "model_path not set for Ollama"
		}
		return true, fmt.Sprintf("Ollama: %s", cfg.Model.ModelPath)
	default:
		return false, fmt.Sprintf("unsupported provider: %s", cfg.Model.Provider)
	}
}

func Generate(cfg config.Config, diff string) (string, error) {
	if !cfg.Model.Enabled {
		return "", errors.New("model disabled")
	}
	switch strings.ToLower(cfg.Model.Provider) {
	case "ollama":
		return generateWithOllama(cfg, diff) 
	default:
		return "", errors.New("unsupported provider: " + cfg.Model.Provider)
	}
}

// OllamaRequest represents the request structure for Ollama API
type OllamaRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// OllamaResponse represents the response structure from Ollama API
type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
	Error    string `json:"error,omitempty"`
}

func checkOllamaHealth() error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		return fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	return nil
}

func generateWithOllama(cfg config.Config, diff string) (string, error) {
	prompt := buildPrompt(cfg, diff)

	req := OllamaRequest{
		Model:  cfg.Model.ModelPath,
		Prompt: prompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature": cfg.Model.Temp,
			"top_p":       cfg.Model.TopP,
			"num_predict": cfg.Model.MaxTokens,
		},
	}

	reqBody, err := json.Marshal(req)

	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{Timeout: 5 * 60 * time.Second}
	resp, err := client.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(reqBody))

	if err != nil {
		return "", fmt.Errorf("failed to call Ollama API: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama api returned status %d", resp.StatusCode)
	}

	var ollamaResp OllamaResponse

	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if ollamaResp.Error != "" {
		return "", fmt.Errorf("ollama error: %s", ollamaResp.Error)
	}

	return strings.TrimSpace(ollamaResp.Response), nil
}

func buildPrompt(cfg config.Config, diff string) string {
	var prompt strings.Builder

	// Add preface
	if cfg.Prompt.Preface != "" {
		prompt.WriteString(cfg.Prompt.Preface)
		prompt.WriteString("\n\n")
	}

	// Add rules
	if cfg.Prompt.Rules != "" {
		prompt.WriteString("Rules:\n")
		prompt.WriteString(cfg.Prompt.Rules)
		prompt.WriteString("\n\n")
	}

	// Add the task
	prompt.WriteString("Generate a commit message for the following git diff:\n\n")
	prompt.WriteString(diff)
	prompt.WriteString("\n\nCommit message:")

	return prompt.String()
}
