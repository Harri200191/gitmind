# commitgen-go

Offline, privacy-first Git commit message generator written in Go. Installs a `prepare-commit-msg` hook that reads the staged diff and produces a high-quality commit message using a local LLM (llama.cpp via Go bindings). Falls back to heuristic summaries when no model is configured.

---

## Repository Structure

```
commitgen-go/
├─ README.md
├─ LICENSE
├─ go.mod
├─ .gitignore
├─ Makefile
├─ .goreleaser.yaml
├─ cmd/
│  └─ commitgen/
│     └─ main.go
├─ internal/
│  ├─ config/
│  │  └─ config.go
│  ├─ diff/
│  │  └─ diff.go
│  ├─ hook/
│  │  └─ hook.go
│  └─ llm/
│     ├─ llm.go
│     └─ llama/
│        └─ llama.go
├─ hooks/
│  └─ prepare-commit-msg.tmpl
├─ configs/
│  └─ .commitgen.yaml.example
├─ models/
│  └─ README.md
└─ scripts/
   └─ install.sh
```

---

## README.md

````markdown
# commitgen-go

Offline Git commit message generator with a `prepare-commit-msg` hook.

- **Local LLM**: Uses llama.cpp via Go bindings (no network/API).
- **Portable**: Single binary written in Go.
- **Safe fallback**: Heuristic summaries if no model configured.
- **Styles**: Conventional Commits or custom templates.

## Quick Start

```bash
# 1) Build
make build

# 2) Install globally (adds to PATH as ./dist/commitgen)
make install

# 3) In a repo, install the hook
commitgen install-hook

# 4) Configure model (optional, for offline LLM)
# Copy example config to your repo root or $HOME
cp configs/.commitgen.yaml.example .commitgen.yaml
# edit MODEL_PATH and parameters

# 5) Stage changes and commit
git add -A
git commit  # hook will generate a message; edit/accept as usual
````

## Configuration

Create `.commitgen.yaml` in the repo root or `$HOME` (repo overrides home):

```yaml
style: conventional  # conventional | plain
max_summary_lines: 15
model:
  enabled: true
  provider: llama.cpp
  model_path: /path/to/your/model/Meta-Llama-3.1-8B.Q4_K_M.gguf
  n_ctx: 8192
  n_threads: 6
  temperature: 0.2
  top_p: 0.9
  max_tokens: 256
prompt:
  preface: |
    You are an assistant that writes precise Git commit messages.
    Analyze the staged diff and produce a single-line subject and an optional wrapped body.
  rules: |
    - Prefer imperative mood ("fix", "add", "update").
    - Keep subject ≤ 72 chars; wrap body at 72.
    - Summarize intent, not code mechanics.
    - If multiple logical changes, summarize concisely.
```

## Heuristic Fallback

If model isn't available, `commitgen` will:

* Detect change type (feat, fix, docs, chore, test, refactor) using simple heuristics.
* Generate a concise subject (≤72 chars) from changed files & hunks.

## Uninstall

```bash
commitgen uninstall-hook
```

## Build Notes

* Requires a C toolchain for llama.cpp bindings.
* macOS: `brew install cmake`.
* Linux: `apt-get install build-essential cmake`.

## License

MIT

```
```

---

## LICENSE

```text
MIT License

Copyright (c) 2025 Your Name

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

---

## go.mod

```go
module github.com/yourname/commitgen

go 1.22

require (
	github.com/go-skynet/go-llama.cpp v0.0.0-20240701000000-abcdef123456 // indirect version example
	gopkg.in/yaml.v3 v3.0.1
)
```

> NOTE: Replace the llama.cpp version with a valid tag/commit when you set up the repo.

---

## .gitignore

```gitignore
# build artifacts
/dist/
/bin/
*.exe
*.dll
*.so
*.dylib

# editors
.vscode/
.idea/

# models (kept out of repo)
/models/*
!models/README.md

# local config
.commitgen.yaml
```

---

## Makefile

```makefile
BINARY_NAME=commitgen
DIST_DIR=dist

.PHONY: build install clean

build:
	GO111MODULE=on CGO_ENABLED=1 go build -o $(DIST_DIR)/$(BINARY_NAME) ./cmd/commitgen

install: build
	install -m 0755 $(DIST_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

clean:
	rm -rf $(DIST_DIR)
```

---

## .goreleaser.yaml

```yaml
project_name: commitgen
before:
  hooks:
    - go mod tidy
builds:
  - id: commitgen
    main: ./cmd/commitgen
    env:
      - CGO_ENABLED=1
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    binary: commitgen
archives:
  - format: tar.gz
    builds: [commitgen]
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"
```

---

## cmd/commitgen/main.go

```go
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
	case "version", "-v", "--version":
		fmt.Println(version)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `commitgen %s

Usage:
  commitgen install-hook              Install prepare-commit-msg hook in current repo
  commitgen uninstall-hook            Remove hook from current repo
  commitgen generate -f <path>        Generate a message into commit-msg file (hook calls this)
  commitgen doctor                    Check model/config availability
  commitgen version                   Print version
`, version)
}

func cmdInstallHook(args []string) {
	repoRoot, err := findRepoRoot()
	if err != nil { log.Fatal(err) }
	if err := hook.Install(repoRoot); err != nil { log.Fatal(err) }
	fmt.Println("Installed prepare-commit-msg hook.")
}

func cmdUninstallHook(args []string) {
	repoRoot, err := findRepoRoot()
	if err != nil { log.Fatal(err) }
	if err := hook.Uninstall(repoRoot); err != nil { log.Fatal(err) }
	fmt.Println("Removed prepare-commit-msg hook.")
}

func cmdDoctor() {
	cfg := config.Load()
	ok, info := llm.Doctor(cfg)
	if ok { fmt.Println("LLM ready:", info) } else { fmt.Println("LLM not ready:", info) }
}

func cmdGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	file := fs.String("f", "", "path to commit-msg file (provided by git)")
	_ = fs.Parse(args)
	if *file == "" { log.Fatal("-f commit message file is required") }

	// Read staged diff
	d, err := diff.Staged()
	if err != nil { log.Fatal(err) }
	if d == "" {
		// nothing staged; don't clobber existing
		os.Exit(0)
	}

	cfg := config.Load()
	message, err := llm.Generate(cfg, d)
	if err != nil {
		// fall back to heuristic
		message = diff.HeuristicMessage(d, cfg)
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
		if parent == cwd { return "", errors.New(".git not found; run inside a repo") }
		cwd = parent
	}
}
```

---

## internal/config/config.go

```go
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Model struct {
	Enabled   bool   `yaml:"enabled"`
	Provider  string `yaml:"provider"`
	ModelPath string `yaml:"model_path"`
	NCtx      int    `yaml:"n_ctx"`
	NThreads  int    `yaml:"n_threads"`
	Temp      float32 `yaml:"temperature"`
	TopP      float32 `yaml:"top_p"`
	MaxTokens int    `yaml:"max_tokens"`
}

type Prompt struct {
	Preface string `yaml:"preface"`
	Rules   string `yaml:"rules"`
}

type Config struct {
	Style            string `yaml:"style"`
	MaxSummaryLines  int    `yaml:"max_summary_lines"`
	Model            Model  `yaml:"model"`
	Prompt           Prompt `yaml:"prompt"`
}

func defaultConfig() Config {
	return Config{
		Style:           "conventional",
		MaxSummaryLines: 15,
		Model: Model{Enabled: false, Provider: "llama.cpp", NCtx: 4096, NThreads: 4, Temp: 0.2, TopP: 0.9, MaxTokens: 256},
		Prompt: Prompt{Preface: "You are an assistant that writes precise Git commit messages.", Rules: "- Prefer imperative mood\n- Keep subject ≤ 72 chars"},
	}
}

func Load() Config {
	cfg := defaultConfig()
	// repo-level overrides
	if loadYAML(".commitgen.yaml", &cfg) == nil { return cfg }
	// home-level
	if home, err := os.UserHomeDir(); err == nil {
		_ = loadYAML(filepath.Join(home, ".commitgen.yaml"), &cfg)
	}
	return cfg
}

func loadYAML(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil { return err }
	return yaml.Unmarshal(b, out)
}
```

---

## internal/diff/diff.go

```go
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
	if err := cmd.Run(); err != nil { return "", err }
	return buf.String(), nil
}

func HeuristicMessage(d string, cfg config.Config) string {
	// crude mapping by filenames/extensions
	files := changedFiles(d)
	kind := kindFromFiles(files)
	subject := strings.TrimSpace(kind + ": " + summarizeFiles(files))
	if len(subject) > 72 { subject = subject[:72] }
	body := summarizeHunks(d, cfg.MaxSummaryLines)
	if body == "" { return subject }
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
		reDocs = regexp.MustCompile(`(?i)\.(md|rst|adoc)$`)
		reTests = regexp.MustCompile(`(?i)(^|/)test(s)?/|(_test)\.go$`)
		reConfig = regexp.MustCompile(`(?i)\.(ya?ml|json|toml|ini)$`)
	)
	if anyMatch(files, reDocs) { return "docs" }
	if anyMatch(files, reTests) { return "test" }
	if anyMatch(files, reConfig) { return "chore" }
	return "feat"
}

func summarizeFiles(files []string) string {
	if len(files) == 0 { return "update changes" }
	if len(files) == 1 { return "update " + files[0] }
	if len(files) == 2 { return "update " + files[0] + ", " + files[1] }
	return "update " + files[0] + " and " +  string(len(files)-1+'0') + " more files"
}

func summarizeHunks(d string, maxLines int) string {
	lines := []string{}
	for _, l := range strings.Split(d, "\n") {
		if strings.HasPrefix(l, "+") && !strings.HasPrefix(l, "+++") {
			if trimmed := strings.TrimSpace(strings.TrimPrefix(l, "+")); trimmed != "" {
				lines = append(lines, "+ " + trimWidth(trimmed, 72))
			}
		}
		if strings.HasPrefix(l, "-") && !strings.HasPrefix(l, "---") {
			if trimmed := strings.TrimSpace(strings.TrimPrefix(l, "-")); trimmed != "" {
				lines = append(lines, "- " + trimWidth(trimmed, 72))
			}
		}
		if len(lines) >= maxLines { break }
	}
	return strings.Join(lines, "\n")
}

func trimWidth(s string, n int) string {
	if len(s) <= n { return s }
	return s[:n]
}

func anyMatch(files []string, re *regexp.Regexp) bool {
	for _, f := range files { if re.MatchString(f) { return true } }
	return false
}

func unique(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in { if _, ok := seen[s]; !ok { seen[s]=struct{}{}; out=append(out,s) } }
	return out
}
```

---

## internal/hook/hook.go

```go
package hook

import (
	"fmt"
	"os"
	"path/filepath"
)

const hookName = "prepare-commit-msg"

func Install(repoRoot string) error {
	hooksDir := filepath.Join(repoRoot, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil { return err }
	hookPath := filepath.Join(hooksDir, hookName)
	content := script()
	if err := os.WriteFile(hookPath, []byte(content), 0755); err != nil { return err }
	return nil
}

func Uninstall(repoRoot string) error {
	hookPath := filepath.Join(repoRoot, ".git", "hooks", hookName)
	if _, err := os.Stat(hookPath); err == nil {
		return os.Remove(hookPath)
	}
	return nil
}

func script() string {
	return fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
MSG_FILE="$1"
# Source, SHA may be present as $2 $3 but we don't use them.
if ! command -v commitgen >/dev/null 2>&1; then
  exit 0 # do not block commit if not installed
fi
commitgen generate -f "$MSG_FILE"
`)
}
```

---

## internal/llm/llm.go

```go
package llm

import (
	"errors"
	"fmt"
	"strings"

	"github.com/yourname/commitgen/internal/config"
)

func Doctor(cfg config.Config) (bool, string) {
	if !cfg.Model.Enabled { return false, "model disabled in config" }
	if cfg.Model.ModelPath == "" { return false, "model_path not set" }
	return true, fmt.Sprintf("%s: %s", cfg.Model.Provider, cfg.Model.ModelPath)
}

func Generate(cfg config.Config, diff string) (string, error) {
	if !cfg.Model.Enabled {
		return "", errors.New("model disabled")
	}
	switch strings.ToLower(cfg.Model.Provider) {
```
