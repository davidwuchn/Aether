package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var swarmCmd = &cobra.Command{
	Use:   "swarm [problem]",
	Short: "Codex compatibility entrypoint for swarm routing and live colony activity",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		watch, _ := cmd.Flags().GetBool("watch")
		target := strings.TrimSpace(strings.Join(args, " "))
		result := buildSwarmCompatibilityResult(target, watch)
		outputWorkflow(result, renderSwarmCompatibilityVisual(result))
		return nil
	},
}

func init() {
	swarmCmd.Flags().Bool("watch", false, "Show live colony activity instead of routing a problem")
	rootCmd.AddCommand(swarmCmd)
}

func buildSwarmCompatibilityResult(target string, watch bool) map[string]interface{} {
	state, _ := loadColonyState()
	active := loadActiveSpawnEntries(store)

	next := "aether init \"describe the goal\""
	mode := "route"
	if watch || target == "" {
		mode = "watch"
		next = "aether status"
	}

	phaseName := ""
	stateName := ""
	goal := ""
	if state != nil {
		next = nextCommandFromState(*state)
		phaseName = lookupPhaseName(*state, state.CurrentPhase)
		stateName = string(state.State)
		if state.Goal != nil {
			goal = strings.TrimSpace(*state.Goal)
		}
		if mode == "watch" && strings.TrimSpace(next) == "" {
			next = "aether status"
		}
	}

	workers := make([]map[string]interface{}, 0, len(active))
	for _, entry := range active {
		workers = append(workers, map[string]interface{}{
			"name":   entry.AgentName,
			"caste":  entry.Caste,
			"task":   entry.Task,
			"status": entry.Status,
		})
	}

	return map[string]interface{}{
		"mode":                mode,
		"target":              target,
		"autopilot_available": false,
		"goal":                goal,
		"state":               stateName,
		"phase_name":          phaseName,
		"active_workers":      workers,
		"active_count":        len(workers),
		"next":                next,
		"watch":               watch || target == "",
	}
}

func renderSwarmCompatibilityVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner("🔥", "Swarm"))
	b.WriteString(visualDivider)

	mode := strings.TrimSpace(stringValue(result["mode"]))
	target := strings.TrimSpace(stringValue(result["target"]))
	goal := strings.TrimSpace(stringValue(result["goal"]))
	stateName := strings.TrimSpace(stringValue(result["state"]))
	phaseName := strings.TrimSpace(stringValue(result["phase_name"]))
	activeCount := intValue(result["active_count"])

	if mode == "watch" {
		b.WriteString("Live colony activity view.\n")
	} else {
		b.WriteString("Codex CLI does not expose a one-shot swarm autopilot.\n")
	}
	if target != "" {
		b.WriteString("Target: ")
		b.WriteString(target)
		b.WriteString("\n")
	}
	if goal != "" {
		b.WriteString("Goal: ")
		b.WriteString(goal)
		b.WriteString("\n")
	}
	if stateName != "" {
		b.WriteString("State: ")
		b.WriteString(stateName)
		if phaseName != "" {
			b.WriteString(" — ")
			b.WriteString(phaseName)
		}
		b.WriteString("\n")
	}

	activeWorkers, _ := result["active_workers"].([]map[string]interface{})
	if activeWorkers == nil {
		if raw, ok := result["active_workers"].([]interface{}); ok {
			activeWorkers = make([]map[string]interface{}, 0, len(raw))
			for _, item := range raw {
				entry, _ := item.(map[string]interface{})
				if entry != nil {
					activeWorkers = append(activeWorkers, entry)
				}
			}
		}
	}

	if activeCount > 0 {
		b.WriteString("\nActive Workers\n")
		for _, entry := range activeWorkers {
			caste := stringValue(entry["caste"])
			b.WriteString("  ")
			b.WriteString(casteEmoji(caste))
			b.WriteString(" ")
			b.WriteString(stringValue(entry["name"]))
			b.WriteString(" (")
			b.WriteString(emptyFallback(caste, "worker"))
			b.WriteString(") — ")
			b.WriteString(stringValue(entry["task"]))
			status := strings.TrimSpace(stringValue(entry["status"]))
			if status != "" {
				b.WriteString(" [")
				b.WriteString(status)
				b.WriteString("]")
			}
			b.WriteString("\n")
		}
	} else if mode == "watch" {
		b.WriteString("\nNo active swarm workers recorded.\n")
	}

	next := strings.TrimSpace(stringValue(result["next"]))
	if next == "" {
		next = "aether status"
	}
	if mode == "watch" {
		b.WriteString(renderNextUp(
			fmt.Sprintf("Run `%s` to inspect the colony in more detail.", next),
			`Run `+"`aether swarm \"describe the problem\"`"+` to get routed to the right explicit Codex workflow step.`,
		))
		return b.String()
	}

	b.WriteString(renderNextUp(
		fmt.Sprintf("Run `%s` to continue the explicit Codex lifecycle.", next),
		fmt.Sprintf("Run `aether flag %q` if you want to record this as an active issue first.", strings.TrimSpace(target)),
		"Run `aether swarm --watch` to inspect live worker activity instead of routing a new issue.",
	))
	return b.String()
}
