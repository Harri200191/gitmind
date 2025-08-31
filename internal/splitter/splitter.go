package splitter

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Harri200191/gitmind/internal/config"
)

// Change represents a logical change in the codebase
type Change struct {
	Files     []string               `json:"files"`
	Functions []string               `json:"functions"`
	Hunks     []Hunk                 `json:"hunks"`
	Message   string                 `json:"message"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// Hunk represents a diff hunk
type Hunk struct {
	File      string `json:"file"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Content   string `json:"content"`
	Type      string `json:"type"` // "add", "remove", "modify"
}

// Cluster represents a group of related changes
type Cluster struct {
	Changes     []Change `json:"changes"`
	Score       float64  `json:"score"`
	Description string   `json:"description"`
}

// Splitter handles multi-commit splitting logic
type Splitter struct {
	config config.Config
}

// New creates a new splitter instance
func New(cfg config.Config) *Splitter {
	return &Splitter{config: cfg}
}

// AnalyzeDiff parses a git diff and extracts semantic information
func (s *Splitter) AnalyzeDiff(diff string) ([]Change, error) {
	var changes []Change

	// Parse the diff into hunks
	hunks := s.parseDiffHunks(diff)

	// Group hunks by files
	fileGroups := s.groupHunksByFile(hunks)

	// Analyze each file group for semantic changes
	for file, fileHunks := range fileGroups {
		change, err := s.analyzeFileChanges(file, fileHunks)
		if err != nil {
			// If analysis fails, treat as a simple file change
			change = Change{
				Files: []string{file},
				Hunks: fileHunks,
				Metadata: map[string]interface{}{
					"analysis_failed": true,
				},
			}
		}
		changes = append(changes, change)
	}

	return changes, nil
}

// ClusterChanges groups related changes into logical commits
func (s *Splitter) ClusterChanges(changes []Change) ([]Cluster, error) {
	if !s.config.MultiCommit.Enabled || len(changes) <= 1 {
		return []Cluster{{Changes: changes, Score: 1.0}}, nil
	}

	// Calculate similarity matrix
	similarities := s.calculateSimilarities(changes)

	// Perform clustering based on similarity scores
	clusters := s.performClustering(changes, similarities)

	// Limit number of clusters
	if len(clusters) > s.config.MultiCommit.MaxClusters {
		clusters = s.mergeClusters(clusters, s.config.MultiCommit.MaxClusters)
	}

	return clusters, nil
}

// parseDiffHunks extracts individual hunks from a git diff
func (s *Splitter) parseDiffHunks(diff string) []Hunk {
	var hunks []Hunk
	lines := strings.Split(diff, "\n")

	var currentFile string
	var currentHunk *Hunk

	for _, line := range lines {
		// File header
		if strings.HasPrefix(line, "+++ b/") {
			currentFile = strings.TrimPrefix(line, "+++ b/")
			continue
		}

		// Hunk header
		if strings.HasPrefix(line, "@@") {
			if currentHunk != nil {
				hunks = append(hunks, *currentHunk)
			}

			// Parse hunk location
			re := regexp.MustCompile(`@@ -(\d+),?\d* \+(\d+),?\d* @@`)
			matches := re.FindStringSubmatch(line)
			if len(matches) >= 3 {
				currentHunk = &Hunk{
					File:      currentFile,
					StartLine: parseInt(matches[2]),
					Content:   "",
				}
			}
			continue
		}

		// Content lines
		if currentHunk != nil && (strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ")) {
			currentHunk.Content += line + "\n"

			// Determine hunk type
			if strings.HasPrefix(line, "+") {
				if currentHunk.Type == "" {
					currentHunk.Type = "add"
				} else if currentHunk.Type == "remove" {
					currentHunk.Type = "modify"
				}
			} else if strings.HasPrefix(line, "-") {
				if currentHunk.Type == "" {
					currentHunk.Type = "remove"
				} else if currentHunk.Type == "add" {
					currentHunk.Type = "modify"
				}
			}
		}
	}

	// Don't forget the last hunk
	if currentHunk != nil {
		hunks = append(hunks, *currentHunk)
	}

	return hunks
}

// groupHunksByFile groups hunks by their file path
func (s *Splitter) groupHunksByFile(hunks []Hunk) map[string][]Hunk {
	groups := make(map[string][]Hunk)
	for _, hunk := range hunks {
		groups[hunk.File] = append(groups[hunk.File], hunk)
	}
	return groups
}

// analyzeFileChanges performs AST analysis for Go files
func (s *Splitter) analyzeFileChanges(file string, hunks []Hunk) (Change, error) {
	change := Change{
		Files:    []string{file},
		Hunks:    hunks,
		Metadata: make(map[string]interface{}),
	}

	// Only analyze Go files for now
	if !strings.HasSuffix(file, ".go") {
		return change, nil
	}

	// Try to parse the file and extract function information
	functions, err := s.extractFunctionsFromHunks(file, hunks)
	if err != nil {
		return change, err
	}

	change.Functions = functions
	change.Metadata["language"] = "go"

	return change, nil
}

// extractFunctionsFromHunks analyzes hunks to identify modified functions
func (s *Splitter) extractFunctionsFromHunks(file string, hunks []Hunk) ([]string, error) {
	var functions []string

	for _, hunk := range hunks {
		// Simple heuristic: look for function signatures in the diff
		lines := strings.Split(hunk.Content, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
				content := strings.TrimPrefix(strings.TrimPrefix(line, "+"), "-")
				content = strings.TrimSpace(content)

				// Look for function declarations
				if matched, _ := regexp.MatchString(`^func\s+(\w+)`, content); matched {
					re := regexp.MustCompile(`^func\s+(\w+)`)
					matches := re.FindStringSubmatch(content)
					if len(matches) > 1 {
						functions = append(functions, matches[1])
					}
				}
			}
		}
	}

	return unique(functions), nil
}

// calculateSimilarities computes semantic similarity between changes
func (s *Splitter) calculateSimilarities(changes []Change) [][]float64 {
	n := len(changes)
	similarities := make([][]float64, n)
	for i := range similarities {
		similarities[i] = make([]float64, n)
	}

	for i := 0; i < n; i++ {
		for j := i; j < n; j++ {
			if i == j {
				similarities[i][j] = 1.0
			} else {
				score := s.calculateSimilarity(changes[i], changes[j])
				similarities[i][j] = score
				similarities[j][i] = score
			}
		}
	}

	return similarities
}

// calculateSimilarity computes similarity score between two changes
func (s *Splitter) calculateSimilarity(a, b Change) float64 {
	var score float64

	// File path similarity
	fileScore := s.calculateFilePathSimilarity(a.Files, b.Files)
	score += fileScore * 0.3

	// Function similarity
	funcScore := s.calculateFunctionSimilarity(a.Functions, b.Functions)
	score += funcScore * 0.4

	// Content similarity (basic keyword matching)
	contentScore := s.calculateContentSimilarity(a.Hunks, b.Hunks)
	score += contentScore * 0.3

	return score
}

// performClustering groups changes based on similarity scores
func (s *Splitter) performClustering(changes []Change, similarities [][]float64) []Cluster {
	n := len(changes)
	clusters := make([]Cluster, 0)
	used := make([]bool, n)

	threshold := s.config.MultiCommit.SimilarityThreshold

	for i := 0; i < n; i++ {
		if used[i] {
			continue
		}

		cluster := Cluster{
			Changes: []Change{changes[i]},
			Score:   1.0,
		}
		used[i] = true

		// Find similar changes to group together
		for j := i + 1; j < n; j++ {
			if !used[j] && similarities[i][j] >= threshold {
				cluster.Changes = append(cluster.Changes, changes[j])
				cluster.Score = (cluster.Score + similarities[i][j]) / 2
				used[j] = true
			}
		}

		cluster.Description = s.generateClusterDescription(cluster)
		clusters = append(clusters, cluster)
	}

	return clusters
}

// Helper functions
func parseInt(s string) int {
	// Simple integer parsing, ignoring errors for brevity
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
} 

func (s *Splitter) calculateFilePathSimilarity(files1, files2 []string) float64 {
	if len(files1) == 0 || len(files2) == 0 {
		return 0.0
	}

	// Check for overlapping files or similar paths
	for _, f1 := range files1 {
		for _, f2 := range files2 {
			if f1 == f2 {
				return 1.0
			}
			// Check if files are in the same directory
			if filepath.Dir(f1) == filepath.Dir(f2) {
				return 0.7
			}
			// Check if files have similar names
			if strings.Contains(f1, strings.TrimSuffix(filepath.Base(f2), filepath.Ext(f2))) ||
				strings.Contains(f2, strings.TrimSuffix(filepath.Base(f1), filepath.Ext(f1))) {
				return 0.5
			}
		}
	}

	return 0.0
}

func (s *Splitter) calculateFunctionSimilarity(funcs1, funcs2 []string) float64 {
	if len(funcs1) == 0 || len(funcs2) == 0 {
		return 0.0
	}

	common := 0
	for _, f1 := range funcs1 {
		for _, f2 := range funcs2 {
			if f1 == f2 {
				common++
				break
			}
		}
	}

	return float64(common) / float64(max(len(funcs1), len(funcs2)))
}

func (s *Splitter) calculateContentSimilarity(hunks1, hunks2 []Hunk) float64 {
	// Simple keyword-based similarity
	keywords1 := s.extractKeywords(hunks1)
	keywords2 := s.extractKeywords(hunks2)

	if len(keywords1) == 0 || len(keywords2) == 0 {
		return 0.0
	}

	common := 0
	for kw1 := range keywords1 {
		if keywords2[kw1] {
			common++
		}
	}

	return float64(common) / float64(max(len(keywords1), len(keywords2)))
}

func (s *Splitter) extractKeywords(hunks []Hunk) map[string]bool {
	keywords := make(map[string]bool)

	for _, hunk := range hunks {
		lines := strings.Split(hunk.Content, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
				content := strings.TrimPrefix(strings.TrimPrefix(line, "+"), "-")
				words := strings.Fields(content)
				for _, word := range words {
					// Simple keyword extraction
					word = strings.Trim(word, "(){}[].,;:")
					if len(word) > 3 && !isCommonWord(word) {
						keywords[strings.ToLower(word)] = true
					}
				}
			}
		}
	}

	return keywords
}

func (s *Splitter) mergeClusters(clusters []Cluster, maxClusters int) []Cluster {
	if len(clusters) <= maxClusters {
		return clusters
	}

	// Simple strategy: merge smallest clusters first
	for len(clusters) > maxClusters {
		// Find two smallest clusters
		minIdx1, minIdx2 := 0, 1
		minSize := len(clusters[0].Changes) + len(clusters[1].Changes)

		for i := 0; i < len(clusters); i++ {
			for j := i + 1; j < len(clusters); j++ {
				size := len(clusters[i].Changes) + len(clusters[j].Changes)
				if size < minSize {
					minIdx1, minIdx2 = i, j
					minSize = size
				}
			}
		}

		// Merge clusters
		merged := Cluster{
			Changes: append(clusters[minIdx1].Changes, clusters[minIdx2].Changes...),
			Score:   (clusters[minIdx1].Score + clusters[minIdx2].Score) / 2,
		}
		merged.Description = s.generateClusterDescription(merged)

		// Remove old clusters and add merged one
		newClusters := make([]Cluster, 0, len(clusters)-1)
		for i, cluster := range clusters {
			if i != minIdx1 && i != minIdx2 {
				newClusters = append(newClusters, cluster)
			}
		}
		newClusters = append(newClusters, merged)
		clusters = newClusters
	}

	return clusters
}

func (s *Splitter) generateClusterDescription(cluster Cluster) string {
	if len(cluster.Changes) == 1 {
		change := cluster.Changes[0]
		if len(change.Functions) > 0 {
			return fmt.Sprintf("Modify %s", strings.Join(change.Functions, ", "))
		}
		return fmt.Sprintf("Update %s", strings.Join(change.Files, ", "))
	}

	// Multiple changes
	allFiles := make(map[string]bool)
	allFunctions := make(map[string]bool)

	for _, change := range cluster.Changes {
		for _, file := range change.Files {
			allFiles[file] = true
		}
		for _, fn := range change.Functions {
			allFunctions[fn] = true
		}
	}

	if len(allFunctions) > 0 {
		functions := make([]string, 0, len(allFunctions))
		for fn := range allFunctions {
			functions = append(functions, fn)
		}
		return fmt.Sprintf("Update %s functions", strings.Join(functions, ", "))
	}

	return fmt.Sprintf("Update %d files", len(allFiles))
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"the": true, "and": true, "for": true, "are": true, "but": true,
		"not": true, "you": true, "all": true, "can": true, "had": true,
		"her": true, "was": true, "one": true, "our": true, "out": true,
		"day": true, "get": true, "has": true, "him": true, "his": true,
		"how": true, "its": true, "new": true, "now": true, "old": true,
		"see": true, "two": true, "who": true, "boy": true, "did": true,
		"may": true, "put": true, "say": true, "she": true, "too": true,
		"use": true, "var": true, "nil": true, "err": true, "int": true,
	}
	return commonWords[strings.ToLower(word)]
}
