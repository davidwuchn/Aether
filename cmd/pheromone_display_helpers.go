package cmd

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func signalDecayWindowDays(signalType string) float64 {
	switch signalType {
	case "FOCUS":
		return 30
	case "REDIRECT":
		return 60
	case "FEEDBACK":
		return 90
	default:
		return 30
	}
}

func signalLifetimeSummary(sig colony.PheromoneSignal, now time.Time) string {
	parts := make([]string, 0, 2)

	if sig.ExpiresAt != nil && strings.TrimSpace(*sig.ExpiresAt) != "" {
		if expiresAt, err := time.Parse(time.RFC3339, *sig.ExpiresAt); err == nil {
			if expiresAt.After(now) {
				parts = append(parts, fmt.Sprintf("ttl %s left", humanizePheromoneDuration(expiresAt.Sub(now))))
			} else {
				parts = append(parts, "expired")
			}
		}
	} else if sig.Type == "FOCUS" {
		parts = append(parts, "phase-scoped")
	}

	remainingDecay := remainingSignalDecay(sig, now)
	parts = append(parts, fmt.Sprintf("%s decay", humanizePheromoneDuration(remainingDecay)))

	return strings.Join(parts, " | ")
}

func remainingSignalDecay(sig colony.PheromoneSignal, now time.Time) time.Duration {
	window := time.Duration(signalDecayWindowDays(sig.Type)*24) * time.Hour

	createdAt, err := time.Parse(time.RFC3339, sig.CreatedAt)
	if err != nil {
		return window
	}

	elapsed := now.Sub(createdAt)
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed >= window {
		return 0
	}
	return window - elapsed
}

func humanizePheromoneDuration(d time.Duration) string {
	if d <= 0 {
		return "0h"
	}

	if d >= 24*time.Hour {
		days := int(math.Ceil(d.Hours() / 24))
		return fmt.Sprintf("%dd", days)
	}

	if d >= time.Hour {
		hours := int(math.Ceil(d.Hours()))
		return fmt.Sprintf("%dh", hours)
	}

	minutes := int(math.Ceil(d.Minutes()))
	if minutes < 1 {
		minutes = 1
	}
	return fmt.Sprintf("%dm", minutes)
}
