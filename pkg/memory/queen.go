package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/storage"
)

// validSections defines the allowed QUEEN.md section names for WriteEntry.
var validSections = map[string]bool{
	"User Preferences":  true,
	"Codebase Patterns": true,
	"Build Learnings":   true,
	"Instincts":         true,
}

// sectionPlaceholders maps section names to their V2 template placeholder text.
var sectionPlaceholders = map[string]string{
	"User Preferences":  "_No user preferences recorded yet._",
	"Codebase Patterns": "_No codebase patterns recorded yet._",
	"Build Learnings":   "_No build learnings recorded yet._",
	"Instincts":         "_No instincts promoted yet._",
}

// metadataPattern extracts the JSON from <!-- METADATA {json} -->.
var metadataPattern = regexp.MustCompile(`<!-- METADATA\s+(\{.*?\})\s*-->`)

// QueenPromotionResult holds the outcome of a QUEEN.md promotion.
type QueenPromotionResult struct {
	EntryID      string
	Section      string
	QueenPath    string
	EntriesAdded int
	TotalEntries int
}

// QueenService promotes instincts and patterns to QUEEN.md using the V2 template.
type QueenService struct {
	store *storage.Store
	bus   *events.Bus
}

// NewQueenService creates a new queen promotion service.
func NewQueenService(store *storage.Store, bus *events.Bus) *QueenService {
	return &QueenService{store: store, bus: bus}
}

// WriteEntry is the core write method that all other promotion methods delegate to.
// It validates the section, loads or creates the V2 template, checks for duplicates,
// replaces placeholders or appends entries, updates the Evolution Log and METADATA,
// and publishes a queen.write event.
func (s *QueenService) WriteEntry(ctx context.Context, queenPath string, section string, entry string, colonyName string) (*QueenPromotionResult, error) {
	// a. Validate section
	if !validSections[section] {
		return nil, fmt.Errorf("queen: invalid section %q", section)
	}

	// Early rejection of empty entries
	if entry == "" {
		return nil, fmt.Errorf("queen: refusing to overwrite with empty content")
	}

	now := events.FormatTimestamp(time.Now().UTC())

	// b. Load existing or create V2 template
	var content string
	data, err := s.store.ReadFile(queenPath)
	if err != nil {
		// File does not exist -- create V2 template
		content = buildV2Template(now)
	} else {
		content = string(data)
	}

	// c. Find the target section
	sectionContent, _, _ := findSection(content, section)

	// d. Check for duplicate entry
	if entry != "" && strings.Contains(sectionContent, entry) {
		// Return with EntriesAdded=0, file not modified
		meta := parseMetadata(content)
		totalEntries := 0
		if v, ok := meta["total_entries"].(float64); ok {
			totalEntries = int(v)
		}
		return &QueenPromotionResult{
			Section:      section,
			QueenPath:    queenPath,
			EntriesAdded: 0,
			TotalEntries: totalEntries,
		}, nil
	}

	// e. Replace placeholder or append entry
	placeholder := sectionPlaceholders[section]
	if strings.Contains(content, placeholder) && strings.Contains(findSectionRaw(content, section), placeholder) {
		content = replaceInSection(content, section, placeholder, entry)
	} else {
		content = appendToSection(content, section, entry)
	}

	// f. Add row to Evolution Log
	details := entry
	if len(details) > 50 {
		details = details[:50]
	}
	content = addToEvolutionLog(content, now, colonyName, section, details)

	// g. Update METADATA
	meta := parseMetadata(content)
	totalEntries := 0
	if v, ok := meta["total_entries"].(float64); ok {
		totalEntries = int(v)
	}
	totalEntries++
	meta["total_entries"] = totalEntries
	meta["last_updated"] = now
	content = updateMetadata(content, meta)

	// h. Safety guard
	if len(content) == 0 {
		return nil, fmt.Errorf("queen: refusing to overwrite with empty content")
	}

	// i. Write via AtomicWrite
	if err := s.store.AtomicWrite(queenPath, []byte(content)); err != nil {
		return nil, fmt.Errorf("queen: write QUEEN.md: %w", err)
	}

	// j. Publish queen.write event
	payload, _ := json.Marshal(map[string]interface{}{
		"section":       section,
		"queen_path":    queenPath,
		"entries_added": 1,
		"colony_name":   colonyName,
	})
	s.bus.Publish(ctx, "queen.write", payload, "queen")

	// k. Return result
	return &QueenPromotionResult{
		Section:      section,
		QueenPath:    queenPath,
		EntriesAdded: 1,
		TotalEntries: totalEntries,
	}, nil
}

// PromoteInstinct formats an instinct entry and writes it to the Instincts section.
func (s *QueenService) PromoteInstinct(ctx context.Context, queenPath string, instinct colony.InstinctEntry, colonyName string) (*QueenPromotionResult, error) {
	entry := fmt.Sprintf("- [instinct] **%s** (%.2f): When %s, then %s",
		instinct.Domain, instinct.Confidence, instinct.Trigger, instinct.Action)
	return s.WriteEntry(ctx, queenPath, "Instincts", entry, colonyName)
}

// PromotePattern writes a pattern entry to the Codebase Patterns section.
func (s *QueenService) PromotePattern(ctx context.Context, queenPath string, content string, colonyName string) (*QueenPromotionResult, error) {
	now := events.FormatTimestamp(time.Now().UTC())
	entry := fmt.Sprintf("- [pattern] **%s** (%s): %s", colonyName, now, content)
	return s.WriteEntry(ctx, queenPath, "Codebase Patterns", entry, colonyName)
}

// PromoteBuildLearning writes a build learning entry to the Build Learnings section.
func (s *QueenService) PromoteBuildLearning(ctx context.Context, queenPath string, tag string, claim string, phaseID string, phaseName string, colonyName string) (*QueenPromotionResult, error) {
	now := events.FormatTimestamp(time.Now().UTC())
	entry := fmt.Sprintf("- [%s] %s -- *Phase %s (%s)* (%s)", tag, claim, phaseID, phaseName, now)
	return s.WriteEntry(ctx, queenPath, "Build Learnings", entry, colonyName)
}

// PromotePreference writes a preference entry to the User Preferences section.
func (s *QueenService) PromotePreference(ctx context.Context, queenPath string, content string, colonyName string) (*QueenPromotionResult, error) {
	now := events.FormatTimestamp(time.Now().UTC())
	entry := fmt.Sprintf("- **%s** (%s): %s", colonyName, now, content)
	return s.WriteEntry(ctx, queenPath, "User Preferences", entry, colonyName)
}

// buildV2Template returns the full QUEEN.md V2 template string.
func buildV2Template(now string) string {
	return fmt.Sprintf(`# QUEEN.md -- Colony Wisdom
> Last evolved: %s
> Wisdom version: 2.0.0
---
## User Preferences
_No user preferences recorded yet._
---
## Codebase Patterns
_No codebase patterns recorded yet._
---
## Build Learnings
_No build learnings recorded yet._
---
## Instincts
_No instincts promoted yet._
---
## Evolution Log
| Date | Source | Type | Details |
|------|--------|------|---------|
---
<!-- METADATA {"total_entries":0,"last_updated":"%s","version":"2.0.0"} -->
`, now, now)
}

// findSection extracts the content of a specific section (between ## header and next ---).
func findSection(content, sectionName string) (string, int, int) {
	header := "## " + sectionName
	startIdx := strings.Index(content, header)
	if startIdx == -1 {
		return "", -1, -1
	}

	// Move past the header line
	afterHeader := content[startIdx:]
	newlineIdx := strings.Index(afterHeader, "\n")
	if newlineIdx == -1 {
		return "", startIdx, -1
	}

	// Find the closing ---
	sectionStart := startIdx + newlineIdx + 1
	afterSection := content[sectionStart:]
	endIdx := strings.Index(afterSection, "\n---")
	if endIdx == -1 {
		// No closing delimiter -- return rest of content
		return afterSection, sectionStart, len(content)
	}

	return afterSection[:endIdx], sectionStart, sectionStart + endIdx
}

// findSectionRaw returns the raw content of a section (between ## header and next ---).
// Used to check if placeholder is in the right section.
func findSectionRaw(content, sectionName string) string {
	sectionContent, _, _ := findSection(content, sectionName)
	return sectionContent
}

// replaceInSection replaces a specific string within a named section.
func replaceInSection(content, sectionName, old, newStr string) string {
	header := "## " + sectionName
	startIdx := strings.Index(content, header)
	if startIdx == -1 {
		return content
	}

	// Find the section end
	afterHeader := content[startIdx:]
	sectionEnd := findSectionEnd(afterHeader)
	sectionBlock := afterHeader[:sectionEnd]

	// Replace within the section block
	updated := strings.Replace(sectionBlock, old, newStr, 1)

	return content[:startIdx] + updated + afterHeader[sectionEnd:]
}

// appendToSection appends an entry after the last entry in a section (before closing ---).
func appendToSection(content, sectionName, entry string) string {
	header := "## " + sectionName
	startIdx := strings.Index(content, header)
	if startIdx == -1 {
		return content
	}

	afterHeader := content[startIdx:]
	sectionEnd := findSectionEnd(afterHeader)
	sectionBlock := afterHeader[:sectionEnd]

	// Append the entry before the closing ---
	updated := sectionBlock + entry + "\n"

	return content[:startIdx] + updated + afterHeader[sectionEnd:]
}

// findSectionEnd finds the position of the closing --- after a section header.
func findSectionEnd(afterHeader string) int {
	// Skip past the header line
	newlineIdx := strings.Index(afterHeader, "\n")
	if newlineIdx == -1 {
		return len(afterHeader)
	}

	// Find \n--- (the delimiter)
	rest := afterHeader[newlineIdx:]
	delimiterIdx := strings.Index(rest, "\n---")
	if delimiterIdx == -1 {
		return len(afterHeader)
	}
	return newlineIdx + delimiterIdx
}

// parseMetadata extracts the JSON from <!-- METADATA {json} -->.
func parseMetadata(content string) map[string]interface{} {
	matches := metadataPattern.FindStringSubmatch(content)
	if len(matches) < 2 {
		return map[string]interface{}{}
	}
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(matches[1]), &meta); err != nil {
		return map[string]interface{}{}
	}
	return meta
}

// updateMetadata replaces the METADATA HTML comment JSON.
func updateMetadata(content string, meta map[string]interface{}) string {
	jsonBytes, err := json.Marshal(meta)
	if err != nil {
		return content
	}
	replacement := fmt.Sprintf("<!-- METADATA %s -->", string(jsonBytes))
	return metadataPattern.ReplaceAllString(content, replacement)
}

// addToEvolutionLog adds a row to the Evolution Log table after the header separator line.
func addToEvolutionLog(content, date, source, sectionType, details string) string {
	separator := "|------|--------|------|---------|"
	idx := strings.Index(content, separator)
	if idx == -1 {
		return content
	}

	// Insert after the separator line
	insertPos := idx + len(separator)
	row := fmt.Sprintf("\n| %s | %s | %s | %s |", date, source, sectionType, details)
	return content[:insertPos] + row + content[insertPos:]
}

// ReadFileWithError is a helper to check if a file exists using the store.
func readFileCheck(store *storage.Store, path string) ([]byte, error) {
	data, err := store.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file does not exist: %w", err)
		}
		return nil, err
	}
	return data, nil
}
