package llm

import (
	"errors"
	"fmt"
	"strings"

	"github.com/yourname/commitgen/internal/config"
	"github.com/yourname/commitgen/internal/llm/llama"
)

func Doctor(cfg config.Config) (bool, string) {
	if !cfg.Model.Enabled {
		return false, "model disabled in config"
	}
	if cfg.Model.ModelPath == "" {
		return false, "model_path not set"
	}
	return true, fmt.Sprintf("%s: %s", cfg.Model.Provider, cfg.Model.ModelPath)
}

func Generate(cfg config.Config, diff string) (string, error) {
	if !cfg.Model.Enabled {
		return "", errors.New("model disabled")
	}
	switch strings.ToLower(cfg.Model.Provider) {
	case "llama.cpp":
		return llama.Generate(cfg, diff)
	default:
		return "", errors.New("unsupported provider: " + cfg.Model.Provider)
	}
}
