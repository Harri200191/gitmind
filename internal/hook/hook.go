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

	fmt.Printf("✅ Installed %s hook\n", hookName)
 
	if err := os.Chmod(hookPath, 0755); err != nil {
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
    return `#!/usr/bin/env bash
set -euo pipefail
MSG_FILE="$1"

if ! command -v gitmind >/dev/null 2>&1; then
    echo "⚠️  gitmind not found, skipping commit message generation"
    exit 0
fi

if ! gitmind generate -f "$MSG_FILE"; then
    echo "❌ gitmind failed to generate commit message" >&2
    exit 1
fi
`
}
