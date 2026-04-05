package cmd

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Suggestion data types
// ---------------------------------------------------------------------------

// Suggestion represents a single pheromone suggestion produced by analysis.
type Suggestion struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	Reason   string `json:"reason"`
	Priority string `json:"priority"`
	Hash     string `json:"hash,omitempty"`
}

// SuggestionsFile represents the top-level suggestions.json structure.
type SuggestionsFile struct {
	Suggestions []Suggestion `json:"suggestions"`
}

// ---------------------------------------------------------------------------
// suggest-analyze
// ---------------------------------------------------------------------------

var suggestAnalyzeCmd = &cobra.Command{
	Use:   "suggest-analyze",
	Short: "Analyze codebase for patterns worth capturing as pheromones",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		contextText, _ := cmd.Flags().GetString("context")
		maxSuggestions, _ := cmd.Flags().GetInt("max")

		if maxSuggestions <= 0 {
			maxSuggestions = 5
		}

		// Collect suggestions by scanning data directory and context
		var suggestions []Suggestion
		analyzedFiles := 0
		patternsFound := 0

		// Read existing pheromones for deduplication
		existingContentHashes := make(map[string]bool)
		pheroData, err := store.ReadFile("pheromones.json")
		if err == nil {
			var pheroFile map[string]interface{}
			if json.Unmarshal(pheroData, &pheroFile) == nil {
				if signals, ok := pheroFile["signals"].([]interface{}); ok {
					for _, sig := range signals {
						if sigMap, ok := sig.(map[string]interface{}); ok {
							if ch, ok := sigMap["content_hash"].(string); ok {
								existingContentHashes[ch] = true
							}
							// Also index content text for dedup
							if contentRaw, ok := sigMap["content"]; ok {
								switch c := contentRaw.(type) {
								case map[string]interface{}:
									if text, ok := c["text"].(string); ok {
										h := sha256.Sum256([]byte(text))
										existingContentHashes[fmt.Sprintf("%x", h[:])] = true
									}
								case string:
									h := sha256.Sum256([]byte(c))
									existingContentHashes[fmt.Sprintf("%x", h[:])] = true
								}
							}
						}
					}
				}
			}
		}

		// Read session suggestions for deduplication
		sessionHashes := make(map[string]bool)
		sessionData, err := store.ReadFile("session.json")
		if err == nil {
			var sessionFile map[string]interface{}
			if json.Unmarshal(sessionData, &sessionFile) == nil {
				if suggested, ok := sessionFile["suggested_pheromones"].([]interface{}); ok {
					for _, entry := range suggested {
						if entryMap, ok := entry.(map[string]interface{}); ok {
							if h, ok := entryMap["hash"].(string); ok {
								sessionHashes[h] = true
							}
						}
					}
				}
			}
		}

		// Scan source directory for patterns
		sourceDir := store.BasePath()
		// Walk up to find project root
		for dir := sourceDir; dir != "/" && dir != ""; dir = filepath.Dir(dir) {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				sourceDir = dir
				break
			}
			if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
				sourceDir = dir
				break
			}
		}

		// Analyze source files for patterns
		filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			// Skip excluded directories
			excludes := []string{"node_modules", ".aether", ".git", "dist", "build", "coverage", "vendor"}
			for _, part := range strings.Split(path, string(filepath.Separator)) {
				for _, exc := range excludes {
					if part == exc {
						return nil
					}
				}
			}

			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".go" && ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" && ext != ".py" && ext != ".sh" && ext != ".md" {
				return nil
			}

			analyzedFiles++

			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			content := string(data)
			lines := strings.Count(content, "\n") + 1

			relPath, _ := filepath.Rel(sourceDir, path)

			// Pattern: Large files (> 300 lines)
			if lines > 300 {
				patternsFound++
				sugContent := fmt.Sprintf("Large file: consider refactoring (%d lines)", lines)
				reason := "File exceeds 300 lines, consider breaking into smaller modules"
				hash := contentHash(relPath, "FOCUS", sugContent)
				if !existingContentHashes[hash] && !sessionHashes[hash] {
					suggestions = append(suggestions, Suggestion{
						ID:       fmt.Sprintf("sug_%d", len(suggestions)+1),
						Type:     "FOCUS",
						Content:  sugContent,
						Reason:   reason,
						Priority: "normal",
						Hash:     hash,
					})
				}
			}

			// Pattern: TODO/FIXME comments
			todoCount := strings.Count(content, "TODO") + strings.Count(content, "FIXME") + strings.Count(content, "XXX")
			if todoCount > 0 {
				patternsFound++
				sugContent := fmt.Sprintf("%d pending TODO/FIXME comments", todoCount)
				reason := "Unresolved markers indicate technical debt"
				hash := contentHash(relPath, "FEEDBACK", sugContent)
				if !existingContentHashes[hash] && !sessionHashes[hash] {
					suggestions = append(suggestions, Suggestion{
						ID:       fmt.Sprintf("sug_%d", len(suggestions)+1),
						Type:     "FEEDBACK",
						Content:  sugContent,
						Reason:   reason,
						Priority: "low",
						Hash:     hash,
					})
				}
			}

			// Pattern: Debug artifacts
			if ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" {
				debugCount := strings.Count(content, "console.log") + strings.Count(content, "debugger")
				if debugCount > 0 {
					patternsFound++
					sugContent := fmt.Sprintf("Remove debug artifacts before commit (%d found)", debugCount)
					reason := "Debug statements should not be committed to production code"
					hash := contentHash(relPath, "REDIRECT", sugContent)
					if !existingContentHashes[hash] && !sessionHashes[hash] {
						suggestions = append(suggestions, Suggestion{
							ID:       fmt.Sprintf("sug_%d", len(suggestions)+1),
							Type:     "REDIRECT",
							Content:  sugContent,
							Reason:   reason,
							Priority: "high",
							Hash:     hash,
						})
					}
				}
			}

			return nil
		})

		// Apply context hint if provided
		if contextText != "" {
			hash := contentHash("context", "FOCUS", contextText)
			if !existingContentHashes[hash] && !sessionHashes[hash] {
				suggestions = append(suggestions, Suggestion{
					ID:       fmt.Sprintf("sug_ctx_%d", time.Now().Unix()),
					Type:     "FOCUS",
					Content:  contextText,
					Reason:   "User-specified context area",
					Priority: "normal",
					Hash:     hash,
				})
			}
		}

		// Sort by priority (high > normal > low) and limit to max
		sort.Slice(suggestions, func(i, j int) bool {
			return priorityOrder(suggestions[i].Priority) > priorityOrder(suggestions[j].Priority)
		})
		if len(suggestions) > maxSuggestions {
			suggestions = suggestions[:maxSuggestions]
		}

		// Count deduplicated
		deduplicated := patternsFound - len(suggestions)
		if deduplicated < 0 {
			deduplicated = 0
		}

		outputOK(map[string]interface{}{
			"suggestions":    suggestions,
			"count":          len(suggestions),
			"analyzed_files": analyzedFiles,
			"patterns_found": patternsFound,
			"deduplicated":   deduplicated,
		})
		return nil
	},
}

// contentHash generates a SHA-256 hash for suggestion deduplication.
func contentHash(file, sugType, content string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s", file, sugType, content)))
	return fmt.Sprintf("%x", h[:])
}

// priorityOrder returns a numeric value for sorting by priority.
func priorityOrder(p string) int {
	switch p {
	case "high":
		return 3
	case "normal":
		return 2
	case "low":
		return 1
	default:
		return 2
	}
}

// ---------------------------------------------------------------------------
// suggest-record
// ---------------------------------------------------------------------------

var suggestRecordCmd = &cobra.Command{
	Use:   "suggest-record",
	Short: "Record a new suggestion to suggestions.json",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		content := mustGetString(cmd, "content")
		if content == "" {
			return nil // mustGetString already output error
		}

		sigType, _ := cmd.Flags().GetString("type")
		if sigType == "" {
			sigType = "FOCUS"
		}
		reason, _ := cmd.Flags().GetString("reason")
		priority, _ := cmd.Flags().GetString("priority")
		if priority == "" {
			priority = "normal"
		}

		// Auto-generate ID
		id := fmt.Sprintf("sug_%d", time.Now().UnixNano())

		// Compute content hash for dedup
		hash := contentHash("manual", sigType, content)

		// Read existing suggestions
		suggestionsFile := SuggestionsFile{Suggestions: []Suggestion{}}
		data, err := store.ReadFile("suggestions.json")
		if err == nil {
			json.Unmarshal(data, &suggestionsFile)
		}

		// Check for duplicate by content hash
		duplicate := false
		for _, existing := range suggestionsFile.Suggestions {
			if existing.Hash == hash {
				duplicate = true
				break
			}
		}

		if duplicate {
			outputOK(map[string]interface{}{
				"recorded":  false,
				"id":        "",
				"type":      sigType,
				"content":   content,
				"duplicate": true,
			})
			return nil
		}

		// Append new suggestion
		sug := Suggestion{
			ID:       id,
			Type:     sigType,
			Content:  content,
			Reason:   reason,
			Priority: priority,
			Hash:     hash,
		}
		suggestionsFile.Suggestions = append(suggestionsFile.Suggestions, sug)

		// Save back
		if err := store.SaveJSON("suggestions.json", suggestionsFile); err != nil {
			outputError(1, fmt.Sprintf("failed to save suggestion: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"recorded":  true,
			"id":        id,
			"type":      sigType,
			"content":   content,
			"duplicate": false,
		})
		return nil
	},
}

// ---------------------------------------------------------------------------
// suggest-check
// ---------------------------------------------------------------------------

var suggestCheckCmd = &cobra.Command{
	Use:   "suggest-check",
	Short: "Read pending suggestions with dedup against active signals",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		limit, _ := cmd.Flags().GetInt("limit")
		if limit <= 0 {
			limit = 20
		}

		// Read suggestions
		suggestionsFile := SuggestionsFile{Suggestions: []Suggestion{}}
		data, err := store.ReadFile("suggestions.json")
		if err == nil {
			json.Unmarshal(data, &suggestionsFile)
		}

		// Read active pheromones for dedup
		existingHashes := make(map[string]bool)
		pheroData, err := store.ReadFile("pheromones.json")
		if err == nil {
			var pheroFile map[string]interface{}
			if json.Unmarshal(pheroData, &pheroFile) == nil {
				if signals, ok := pheroFile["signals"].([]interface{}); ok {
					for _, sig := range signals {
						if sigMap, ok := sig.(map[string]interface{}); ok {
							if ch, ok := sigMap["content_hash"].(string); ok {
								existingHashes[ch] = true
							}
						}
					}
				}
			}
		}

		// Filter out duplicates
		var filtered []Suggestion
		deduplicatedAgainst := 0
		for _, sug := range suggestionsFile.Suggestions {
			if existingHashes[sug.Hash] {
				deduplicatedAgainst++
				continue
			}
			filtered = append(filtered, sug)
		}

		// Apply limit
		if len(filtered) > limit {
			filtered = filtered[:limit]
		}

		outputOK(map[string]interface{}{
			"suggestions":          filtered,
			"count":                len(filtered),
			"deduplicated_against": deduplicatedAgainst,
		})
		return nil
	},
}

// ---------------------------------------------------------------------------
// suggest-approve (existing)
// ---------------------------------------------------------------------------

var suggestApproveCmd = &cobra.Command{
	Use:   "suggest-approve",
	Short: "Approve pending suggestions as pheromone signals",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		approveID, _ := cmd.Flags().GetString("id")
		sigType, _ := cmd.Flags().GetString("type")
		if sigType == "" {
			sigType = "FOCUS"
		}

		// Read suggestions
		data, err := store.ReadFile("suggestions.json")
		if err != nil {
			outputOK(map[string]interface{}{
				"approved": 0,
				"ids":      []string{},
			})
			return nil
		}

		var suggestionsFile map[string]interface{}
		if err := json.Unmarshal(data, &suggestionsFile); err != nil {
			outputError(1, fmt.Sprintf("failed to parse suggestions.json: %v", err), nil)
			return nil
		}

		rawSuggestions, _ := suggestionsFile["suggestions"].([]interface{})
		if rawSuggestions == nil {
			outputOK(map[string]interface{}{
				"approved": 0,
				"ids":      []string{},
			})
			return nil
		}

		var approvedIDs []string
		var remaining []interface{}

		for _, raw := range rawSuggestions {
			sug, ok := raw.(map[string]interface{})
			if !ok {
				remaining = append(remaining, raw)
				continue
			}

			sugID, _ := sug["id"].(string)
			content, _ := sug["content"].(string)

			// If --id specified, only approve matching; otherwise approve all
			if approveID != "" && sugID != approveID {
				remaining = append(remaining, raw)
				continue
			}

			// Create pheromone signal from suggestion
			now := time.Now().UTC().Format(time.RFC3339)
			signal := map[string]interface{}{
				"id":         fmt.Sprintf("sig_%s_%d", sugID, time.Now().Unix()),
				"type":       sigType,
				"active":     true,
				"created_at": now,
				"content":    map[string]string{"text": content},
				"source":     "suggestion",
			}
			switch sigType {
			case "REDIRECT":
				signal["priority"] = "high"
			case "FEEDBACK":
				signal["priority"] = "low"
			default:
				signal["priority"] = "normal"
			}

			// Append to pheromones JSONL
			store.AppendJSONL("pheromones.jsonl", signal)

			approvedIDs = append(approvedIDs, sugID)
		}

		// Save remaining suggestions
		suggestionsFile["suggestions"] = remaining
		if len(remaining) == 0 {
			os.Remove(store.BasePath() + "/suggestions.json")
		} else {
			store.SaveJSON("suggestions.json", suggestionsFile)
		}

		outputOK(map[string]interface{}{
			"approved": len(approvedIDs),
			"ids":      approvedIDs,
		})
		return nil
	},
}

// ---------------------------------------------------------------------------
// suggest-quick-dismiss (existing)
// ---------------------------------------------------------------------------

var suggestQuickDismissCmd = &cobra.Command{
	Use:   "suggest-quick-dismiss",
	Short: "Dismiss all pending suggestions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		data, err := store.ReadFile("suggestions.json")
		if err != nil {
			outputOK(map[string]interface{}{
				"dismissed": 0,
			})
			return nil
		}

		var suggestionsFile map[string]interface{}
		if err := json.Unmarshal(data, &suggestionsFile); err != nil {
			outputError(1, fmt.Sprintf("failed to parse suggestions.json: %v", err), nil)
			return nil
		}

		rawSuggestions, _ := suggestionsFile["suggestions"].([]interface{})
		count := 0
		if rawSuggestions != nil {
			count = len(rawSuggestions)
		}

		// Remove the suggestions file
		if err := os.Remove(store.BasePath() + "/suggestions.json"); err != nil {
			outputError(2, fmt.Sprintf("failed to remove suggestions: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"dismissed": count,
		})
		return nil
	},
}

// isTestArtifact checks if a pheromone signal matches test artifact patterns.
func isTestArtifact(signal map[string]interface{}) bool {
	id, _ := signal["id"].(string)
	contentRaw := signal["content"]
	content := ""
	if contentMap, ok := contentRaw.(map[string]interface{}); ok {
		content, _ = contentMap["text"].(string)
	} else if contentStr, ok := contentRaw.(string); ok {
		content = contentStr
	}

	// Check id patterns
	if strings.HasPrefix(id, "test_") || strings.HasPrefix(id, "demo_") {
		return true
	}

	// Check content patterns
	lower := strings.ToLower(content)
	if strings.Contains(lower, "test signal") || strings.Contains(lower, "demo pattern") {
		return true
	}

	return false
}

func init() {
	// suggest-analyze flags
	suggestAnalyzeCmd.Flags().String("context", "", "Optional context text to include as suggestion")
	suggestAnalyzeCmd.Flags().Int("max", 5, "Maximum suggestions to return")

	// suggest-record flags
	suggestRecordCmd.Flags().String("content", "", "Suggestion content (required)")
	suggestRecordCmd.Flags().String("type", "FOCUS", "Pheromone type (FOCUS, REDIRECT, FEEDBACK)")
	suggestRecordCmd.Flags().String("reason", "", "Reason for the suggestion")
	suggestRecordCmd.Flags().String("priority", "normal", "Priority (high, normal, low)")

	// suggest-check flags
	suggestCheckCmd.Flags().Int("limit", 20, "Maximum suggestions to return")

	// suggest-approve flags
	suggestApproveCmd.Flags().String("id", "", "Suggestion ID to approve (omit to approve all)")
	suggestApproveCmd.Flags().String("type", "FOCUS", "Pheromone type (FOCUS, REDIRECT, FEEDBACK)")

	// Register all commands
	rootCmd.AddCommand(suggestAnalyzeCmd)
	rootCmd.AddCommand(suggestRecordCmd)
	rootCmd.AddCommand(suggestCheckCmd)
	rootCmd.AddCommand(suggestApproveCmd)
	rootCmd.AddCommand(suggestQuickDismissCmd)
}
