package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Model struct {
	Enabled   bool    `yaml:"enabled"`
	Provider  string  `yaml:"provider"`
	ModelPath string  `yaml:"model_path"`
	NCtx      int     `yaml:"n_ctx"`
	NThreads  int     `yaml:"n_threads"`
	Temp      float32 `yaml:"temperature"`
	TopP      float32 `yaml:"top_p"`
	MaxTokens int     `yaml:"max_tokens"`
}

type Prompt struct {
	Preface string `yaml:"preface"`
	Rules   string `yaml:"rules"`
}

type Config struct {
	Style           string `yaml:"style"`
	MaxSummaryLines int    `yaml:"max_summary_lines"`
	Model           Model  `yaml:"model"`
	Prompt          Prompt `yaml:"prompt"`
}

func defaultConfig() Config {
	return Config{
		Style:           "conventional",
		MaxSummaryLines: 15,
		Model:           Model{Enabled: false, Provider: "llama.cpp", NCtx: 4096, NThreads: 4, Temp: 0.2, TopP: 0.9, MaxTokens: 256},
		Prompt:          Prompt{Preface: "You are an assistant that writes precise Git commit messages.", Rules: "- Prefer imperative mood\n- Keep subject ≤ 72 chars"},
	}
}

func Load() Config {
	cfg := defaultConfig()
	// repo-level overrides
	if loadYAML(".commitgen.yaml", &cfg) == nil {
		return cfg
	}
	// home-level
	if home, err := os.UserHomeDir(); err == nil {
		_ = loadYAML(filepath.Join(home, ".commitgen.yaml"), &cfg)
	}
	return cfg
}

func loadYAML(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, out)
}
