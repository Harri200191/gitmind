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
	if err != nil {
		log.Fatal(err)
	}
	if err := hook.Install(repoRoot); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Installed prepare-commit-msg hook.")
}

func cmdUninstallHook(args []string) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		log.Fatal(err)
	}
	if err := hook.Uninstall(repoRoot); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Removed prepare-commit-msg hook.")
}

func cmdDoctor() {
	cfg := config.Load()
	ok, info := llm.Doctor(cfg)
	if ok {
		fmt.Println("LLM ready:", info)
	} else {
		fmt.Println("LLM not ready:", info)
	}
}

func cmdGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	file := fs.String("f", "", "path to commit-msg file (provided by git)")
	_ = fs.Parse(args)
	if *file == "" {
		log.Fatal("-f commit message file is required")
	}

	// Read staged diff
	d, err := diff.Staged()
	if err != nil {
		log.Fatal(err)
	}
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
		if parent == cwd {
			return "", errors.New(".git not found; run inside a repo")
		}
		cwd = parent
	}
}
