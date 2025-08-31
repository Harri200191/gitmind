package splitter

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
 
	"github.com/Harri200191/gitmind/internal/config"
)

// MultiCommitManager handles the process of creating multiple commits
type MultiCommitManager struct {
	config   config.Config
	splitter *Splitter
}

// CommitProposal represents a proposed commit
type CommitProposal struct {
	Files   []string `json:"files"`
	Message string   `json:"message"`
	Changes []Change `json:"changes"`
}

// NewMultiCommitManager creates a new multi-commit manager
func NewMultiCommitManager(cfg config.Config) *MultiCommitManager {
	return &MultiCommitManager{
		config:   cfg,
		splitter: New(cfg),
	}
}

// ProcessStagedChanges analyzes staged changes and proposes multiple commits
func (mcm *MultiCommitManager) ProcessStagedChanges() ([]CommitProposal, error) {
	if !mcm.config.MultiCommit.Enabled {
		return nil, nil
	}

	// Get the staged diff
	diff, err := mcm.getStagedDiff()
	if err != nil {
		return nil, fmt.Errorf("failed to get staged diff: %v", err)
	}

	if strings.TrimSpace(diff) == "" {
		return nil, nil
	}

	// Analyze the diff for logical changes
	changes, err := mcm.splitter.AnalyzeDiff(diff)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze diff: %v", err)
	}

	// Cluster related changes
	clusters, err := mcm.splitter.ClusterChanges(changes)
	if err != nil {
		return nil, fmt.Errorf("failed to cluster changes: %v", err)
	}

	// Generate commit proposals
	var proposals []CommitProposal
	for i, cluster := range clusters {
		proposal := CommitProposal{
			Files:   mcm.extractFilesFromCluster(cluster),
			Message: mcm.generateCommitMessage(cluster, i+1, len(clusters)),
			Changes: cluster.Changes,
		}
		proposals = append(proposals, proposal)
	}

	return proposals, nil
}

// ExecuteMultiCommit creates multiple commits based on proposals
func (mcm *MultiCommitManager) ExecuteMultiCommit(proposals []CommitProposal) error {
	if len(proposals) <= 1 {
		// If only one proposal, let normal commit process handle it
		return nil
	}

	// Prompt user for confirmation if enabled
	if mcm.config.MultiCommit.PromptUser {
		confirmed, err := mcm.promptUserForConfirmation(proposals)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Multi-commit cancelled by user")
			return nil
		}
	}

	// Store the current staging area
	if err := mcm.stashCurrentChanges(); err != nil {
		return fmt.Errorf("failed to stash changes: %v", err)
	}

	// Create each commit
	for i, proposal := range proposals {
		if err := mcm.createCommit(proposal, i+1, len(proposals)); err != nil {
			// If any commit fails, try to restore the staging area
			mcm.restoreChanges()
			return fmt.Errorf("failed to create commit %d: %v", i+1, err)
		}
	}

	fmt.Printf("Successfully created %d commits\n", len(proposals))
	return nil
}

// getStagedDiff retrieves the current staged diff
func (mcm *MultiCommitManager) getStagedDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--cached")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// extractFilesFromCluster gets all unique files from a cluster
func (mcm *MultiCommitManager) extractFilesFromCluster(cluster Cluster) []string {
	fileMap := make(map[string]bool)

	for _, change := range cluster.Changes {
		for _, file := range change.Files {
			fileMap[file] = true
		}
	}

	var files []string
	for file := range fileMap {
		files = append(files, file)
	}

	return files
}

// generateCommitMessage creates a commit message for a cluster
func (mcm *MultiCommitManager) generateCommitMessage(cluster Cluster, index, total int) string {
	baseMessage := cluster.Description

	if total > 1 {
		// Add context about this being part of a multi-commit series
		baseMessage = fmt.Sprintf("%s (%d/%d)", baseMessage, index, total)
	}

	// Add details about the changes
	if len(cluster.Changes) == 1 {
		change := cluster.Changes[0]
		if len(change.Functions) > 0 {
			baseMessage += fmt.Sprintf("\n\nModified functions: %s", strings.Join(change.Functions, ", "))
		}
	} else {
		// Multiple changes in this commit
		var allFunctions []string
		for _, change := range cluster.Changes {
			allFunctions = append(allFunctions, change.Functions...)
		}
		if len(allFunctions) > 0 {
			baseMessage += fmt.Sprintf("\n\nModified functions: %s", strings.Join(unique(allFunctions), ", "))
		}
	}

	return baseMessage
}

// promptUserForConfirmation asks user to confirm the multi-commit proposal
func (mcm *MultiCommitManager) promptUserForConfirmation(proposals []CommitProposal) (bool, error) {
	fmt.Printf("\n🔍 Multi-commit proposal detected %d logical changes:\n\n", len(proposals))

	for i, proposal := range proposals {
		fmt.Printf("Commit %d: %s\n", i+1, proposal.Message)
		fmt.Printf("  Files: %s\n", strings.Join(proposal.Files, ", "))
		fmt.Println()
	}

	fmt.Print("Do you want to proceed with multi-commit? [Y/n]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "" || response == "y" || response == "yes", nil
}

// stashCurrentChanges temporarily stores the current staging area
func (mcm *MultiCommitManager) stashCurrentChanges() error {
	// Create a temporary stash
	cmd := exec.Command("git", "stash", "push", "--staged", "--message", "gitmind-multi-commit-temp")
	return cmd.Run()
}

// restoreChanges restores the staging area from stash
func (mcm *MultiCommitManager) restoreChanges() error {
	// Pop the temporary stash
	cmd := exec.Command("git", "stash", "pop")
	return cmd.Run()
}

// createCommit creates a single commit for the given proposal
func (mcm *MultiCommitManager) createCommit(proposal CommitProposal, index, total int) error {
	// Stage only the files for this commit
	for _, file := range proposal.Files {
		if err := mcm.stageFile(file); err != nil {
			return fmt.Errorf("failed to stage file %s: %v", file, err)
		}
	}

	// Create the commit
	cmd := exec.Command("git", "commit", "-m", proposal.Message)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit failed: %v", err)
	}

	fmt.Printf("✓ Created commit %d/%d: %s\n", index, total, proposal.Message)
	return nil
}

// stageFile stages a specific file
func (mcm *MultiCommitManager) stageFile(file string) error {
	cmd := exec.Command("git", "add", file)
	return cmd.Run()
}

// stageFilePartial stages only specific hunks of a file
// This is a simplified version - real implementation would need more sophisticated hunk selection
func (mcm *MultiCommitManager) stageFilePartial(file string, hunks []Hunk) error {
	// For now, stage the entire file
	// TODO: Implement selective staging of hunks using git add --patch or similar
	return mcm.stageFile(file)
}

// InteractiveMultiCommit provides an interactive mode for multi-commit creation
func (mcm *MultiCommitManager) InteractiveMultiCommit() error {
	proposals, err := mcm.ProcessStagedChanges()
	if err != nil {
		return err
	}

	if len(proposals) <= 1 {
		fmt.Println("No multi-commit opportunities detected")
		return nil
	}

	fmt.Printf("\n🎯 Detected %d logical changes that can be split into separate commits\n", len(proposals))

	// Show proposals with options to modify
	for {
		fmt.Println("\nCommit proposals:")
		for i, proposal := range proposals {
			fmt.Printf("  %d. %s\n", i+1, proposal.Message)
			fmt.Printf("     Files: %s\n", strings.Join(proposal.Files, ", "))
		}

		fmt.Println("\nOptions:")
		fmt.Println("  1. Accept all proposals")
		fmt.Println("  2. Edit proposals")
		fmt.Println("  3. Cancel")
		fmt.Print("\nChoice [1]: ")

		reader := bufio.NewReader(os.Stdin)
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		if choice == "" || choice == "1" {
			return mcm.ExecuteMultiCommit(proposals)
		} else if choice == "2" {
			// TODO: Implement proposal editing
			fmt.Println("Proposal editing not yet implemented")
			continue
		} else if choice == "3" {
			fmt.Println("Multi-commit cancelled")
			return nil
		} else {
			fmt.Println("Invalid choice, please try again")
		}
	}
}

// Helper function (already exists in splitter.go, but added here for completeness)
func unique(items []string) []string {
	keys := make(map[string]bool)
	var result []string
	for _, item := range items {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	return result
}
