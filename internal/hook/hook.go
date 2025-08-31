package hook

import (
	"fmt"
	"os"
	"path/filepath"
)

const hookName = "prepare-commit-msg"

func Install(repoRoot string) error {
	hooksDir := filepath.Join(repoRoot, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return err
	}
	hookPath := filepath.Join(hooksDir, hookName)
	content := script()
	if err := os.WriteFile(hookPath, []byte(content), 0755); err != nil {
		return err
	}
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
if ! command -v gitmind >/dev/null 2>&1; then
  exit 0 # do not block commit if not installed
fi
gitmind generate -f "$MSG_FILE"
`)
}
