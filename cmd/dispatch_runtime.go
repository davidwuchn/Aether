package cmd

import (
	"strings"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
)

func spawnTreeDispatchObserver(spawnTree *agent.SpawnTree, activePrefix string) codex.DispatchObserver {
	if spawnTree == nil {
		return nil
	}
	return func(event codex.DispatchLifecycleEvent) {
		status := strings.TrimSpace(event.Status)
		if status == "" {
			return
		}

		summary := dispatchLifecycleSummary(event, activePrefix)
		_ = spawnTree.UpdateStatus(event.Dispatch.WorkerName, status, summary)
	}
}

func dispatchLifecycleSummary(event codex.DispatchLifecycleEvent, activePrefix string) string {
	status := strings.TrimSpace(event.Status)
	switch status {
	case "starting":
		if task := strings.TrimSpace(event.Dispatch.TaskBrief); task != "" {
			return firstContentLine(task)
		}
		if task := strings.TrimSpace(event.Dispatch.TaskID); task != "" {
			return task
		}
		return "Starting worker"
	case "active", "running":
		task := strings.TrimSpace(firstContentLine(event.Dispatch.TaskBrief))
		if task == "" {
			task = strings.TrimSpace(event.Dispatch.TaskID)
		}
		if task == "" {
			task = strings.TrimSpace(event.Dispatch.WorkerName)
		}
		if strings.TrimSpace(activePrefix) == "" {
			return task
		}
		return strings.TrimSpace(activePrefix + ": " + task)
	default:
		if event.WorkerResult != nil {
			if summary := strings.TrimSpace(event.WorkerResult.Summary); summary != "" {
				return summary
			}
			if len(event.WorkerResult.Blockers) > 0 {
				return strings.Join(event.WorkerResult.Blockers, "; ")
			}
		}
		if event.Error != nil {
			return strings.TrimSpace(event.Error.Error())
		}
		return strings.TrimSpace(event.Dispatch.WorkerName)
	}
}

func firstContentLine(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimPrefix(line, "- "))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		return line
	}
	return ""
}
