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
```

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
Commit messages with intelligence and Security!
