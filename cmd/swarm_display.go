package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// loadColonyState reads and parses COLONY_STATE.json from the store.
func loadColonyState() (*colony.ColonyState, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}
	data, err := store.ReadFile("COLONY_STATE.json")
	if err != nil {
		return nil, nil // no file = no colony, not an error
	}
	var state colony.ColonyState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse colony state: %w", err)
	}
	return &state, nil
}

// goalStr safely dereferences a goal pointer.
func goalStr(g *string) string {
	if g == nil || *g == "" {
		return "(no goal)"
	}
	return *g
}

// ---------------------------------------------------------------------------
// swarm-display-render
// ---------------------------------------------------------------------------

var swarmDisplayRenderCmd = &cobra.Command{
	Use:   "swarm-display-render",
	Short: "Render ASCII tree from colony state with configurable format",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		maxDepth, _ := cmd.Flags().GetInt("max-depth")
		if format == "" {
			format = "tree"
		}
		if maxDepth <= 0 {
			maxDepth = 3
		}

		state, err := loadColonyState()
		if err != nil {
			outputErrorMessage(err.Error())
			return nil
		}
		if state == nil {
			outputOK(map[string]interface{}{
				"format": format, "lines": []string{"No colony state found"}, "total_lines": 1,
			})
			return nil
		}

		var lines []string
		switch format {
		case "json":
			raw, _ := store.ReadFile("COLONY_STATE.json")
			lines = append(lines, string(raw))
		case "flat":
			lines = append(lines, fmt.Sprintf("Goal: %s", goalStr(state.Goal)))
			lines = append(lines, fmt.Sprintf("Milestone: %s", state.Milestone))
			lines = append(lines, fmt.Sprintf("State: %s", string(state.State)))
			for _, p := range state.Plan.Phases {
				if maxDepth > 0 && p.ID > maxDepth {
					continue
				}
				lines = append(lines, fmt.Sprintf("Phase %d: %s [%s]", p.ID, p.Name, p.Status))
			}
		default: // tree
			lines = append(lines, "Colony")
			lines = append(lines, fmt.Sprintf("├── Goal: %s", goalStr(state.Goal)))
			lines = append(lines, fmt.Sprintf("├── Milestone: %s", state.Milestone))
			lines = append(lines, fmt.Sprintf("├── State: %s", string(state.State)))
			for i, p := range state.Plan.Phases {
				if maxDepth > 0 && p.ID > maxDepth {
					continue
				}
				prefix := "└──"
				if i < len(state.Plan.Phases)-1 {
					prefix = "├──"
				}
				lines = append(lines, fmt.Sprintf("%s Phase %d: %s [%s]", prefix, p.ID, p.Name, p.Status))
			}
		}

		outputOK(map[string]interface{}{
			"format": format, "lines": lines, "total_lines": len(lines),
		})
		return nil
	},
}

// ---------------------------------------------------------------------------
// swarm-display-inline
// ---------------------------------------------------------------------------

var swarmDisplayInlineCmd = &cobra.Command{
	Use:   "swarm-display-inline",
	Short: "Render a single-line status summary",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		section, _ := cmd.Flags().GetString("section")

		state, err := loadColonyState()
		if err != nil {
			outputErrorMessage(err.Error())
			return nil
		}
		if state == nil {
			outputOK(map[string]interface{}{
				"inline": "[No colony]", "sections": map[string]interface{}{},
			})
			return nil
		}

		totalPhases := len(state.Plan.Phases)
		completedPhases := 0
		for _, p := range state.Plan.Phases {
			if p.Status == "complete" {
				completedPhases++
			}
		}

		instinctCount := activeInstinctCount(store, state)

		sections := map[string]interface{}{
			"progress": fmt.Sprintf("Phase %d/%d (%d complete)", state.CurrentPhase, totalPhases, completedPhases),
			"memory": fmt.Sprintf("%d learnings | %d instincts",
				len(state.Memory.PhaseLearnings), instinctCount),
		}

		if section != "" {
			parts := strings.Split(section, ",")
			filtered := make(map[string]interface{})
			for _, s := range parts {
				s = strings.TrimSpace(s)
				if v, ok := sections[s]; ok {
					filtered[s] = v
				}
			}
			sections = filtered
		}

		parts := make([]string, 0, len(sections))
		if v, ok := sections["progress"]; ok {
			parts = append(parts, fmt.Sprintf("%v", v))
		}
		if v, ok := sections["memory"]; ok {
			parts = append(parts, fmt.Sprintf("%v", v))
		}

		outputOK(map[string]interface{}{
			"inline": strings.Join(parts, " | "), "sections": sections,
		})
		return nil
	},
}

// ---------------------------------------------------------------------------
// swarm-display-text
// ---------------------------------------------------------------------------

var swarmDisplayTextCmd = &cobra.Command{
	Use:   "swarm-display-text",
	Short: "Render a multi-line text block for terminal display",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		section, _ := cmd.Flags().GetString("section")
		maxWidth, _ := cmd.Flags().GetInt("max-width")
		if maxWidth <= 0 {
			maxWidth = 80
		}

		state, err := loadColonyState()
		if err != nil {
			outputErrorMessage(err.Error())
			return nil
		}
		if state == nil {
			outputOK(map[string]interface{}{
				"text": "No colony state found", "lines": []string{"No colony state found"}, "width": maxWidth,
			})
			return nil
		}

		var lines []string
		lines = append(lines, fmt.Sprintf("Colony: %s", goalStr(state.Goal)))
		lines = append(lines, fmt.Sprintf("Milestone: %s | State: %s", state.Milestone, string(state.State)))
		instinctCount := activeInstinctCount(store, state)
		lines = append(lines, fmt.Sprintf("Memory: %d phase learnings, %d decisions, %d instincts",
			len(state.Memory.PhaseLearnings), len(state.Memory.Decisions), instinctCount))
		lines = append(lines, "")

		showSection := section == "" || strings.Contains(section, "phases")
		if showSection {
			for _, p := range state.Plan.Phases {
				status := p.Status
				if status == "" {
					status = "pending"
				}
				line := fmt.Sprintf("  Phase %d: %s [%s]", p.ID, p.Name, status)
				if len(line) > maxWidth {
					line = line[:maxWidth-3] + "..."
				}
				lines = append(lines, line)
			}
		}

		outputOK(map[string]interface{}{
			"text": strings.Join(lines, "\n"), "lines": lines, "width": maxWidth,
		})
		return nil
	},
}

func init() {
	swarmDisplayRenderCmd.Flags().String("format", "tree", "Output format: tree, json, flat")
	swarmDisplayRenderCmd.Flags().Int("max-depth", 3, "Maximum depth for tree rendering")
	swarmDisplayInlineCmd.Flags().String("section", "", "Sections to show (progress,memory)")
	swarmDisplayTextCmd.Flags().String("section", "", "Section to display")
	swarmDisplayTextCmd.Flags().Int("max-width", 80, "Maximum line width")

	rootCmd.AddCommand(swarmDisplayRenderCmd)
	rootCmd.AddCommand(swarmDisplayInlineCmd)
	rootCmd.AddCommand(swarmDisplayTextCmd)
}
