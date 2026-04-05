package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
	"github.com/spf13/cobra"
)

var (
	eventTopic   string
	eventPayload string
	eventSource  string
	eventPattern string
	eventSince   string
	eventLimit   int
	eventDryRun  bool
)

// newEventBus creates an event bus backed by the current store.
func newEventBus() (*events.Bus, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}
	return events.NewBus(store, events.DefaultConfig()), nil
}

var eventBusPublishCmd = &cobra.Command{
	Use:   "event-bus-publish",
	Short: "Publish a typed event to the event bus",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		topic := mustGetString(cmd, "topic")
		payload := mustGetString(cmd, "payload")
		source := mustGetString(cmd, "source")
		if topic == "" || payload == "" {
			return nil
		}

		bus, err := newEventBus()
		if err != nil {
			outputErrorMessage(err.Error())
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		evt, err := bus.Publish(ctx, topic, json.RawMessage(payload), source)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"id":         evt.ID,
			"topic":      evt.Topic,
			"source":     evt.Source,
			"timestamp":  evt.Timestamp,
			"ttl_days":   evt.TTLDays,
			"expires_at": evt.ExpiresAt,
		})
		return nil
	},
}

var eventBusQueryCmd = &cobra.Command{
	Use:   "event-bus-query",
	Short: "Query events from the event bus matching a topic pattern",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		pattern := mustGetString(cmd, "pattern")
		if pattern == "" {
			return nil
		}

		bus, err := newEventBus()
		if err != nil {
			outputErrorMessage(err.Error())
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var since time.Time
		if eventSince != "" {
			var parseErr error
			since, parseErr = time.Parse(time.RFC3339, eventSince)
			if parseErr != nil {
				outputError(1, "invalid --since format, use RFC3339", nil)
				return nil
			}
		}

		evts, err := bus.Query(ctx, pattern, since, eventLimit)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"events": evts,
			"count":  len(evts),
		})
		return nil
	},
}

var eventBusReplayCmd = &cobra.Command{
	Use:   "event-bus-replay",
	Short: "Replay persisted events for a specific topic in chronological order",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		topic := mustGetString(cmd, "topic")
		if topic == "" {
			return nil
		}

		bus, err := newEventBus()
		if err != nil {
			outputErrorMessage(err.Error())
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var since time.Time
		if eventSince != "" {
			var parseErr error
			since, parseErr = time.Parse(time.RFC3339, eventSince)
			if parseErr != nil {
				outputError(1, "invalid --since format, use RFC3339", nil)
				return nil
			}
		}

		evts, err := bus.Replay(ctx, topic, since, eventLimit)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"events": evts,
			"count":  len(evts),
		})
		return nil
	},
}

var eventBusCleanupCmd = &cobra.Command{
	Use:   "event-bus-cleanup",
	Short: "Remove expired events from the event bus JSONL file",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		bus, err := newEventBus()
		if err != nil {
			outputErrorMessage(err.Error())
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		removed, remaining, err := bus.Cleanup(ctx, eventDryRun)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"removed":   removed,
			"remaining": remaining,
			"dry_run":   eventDryRun,
		})
		return nil
	},
}

func init() {
	// event-bus-publish
	rootCmd.AddCommand(eventBusPublishCmd)
	eventBusPublishCmd.Flags().StringVar(&eventTopic, "topic", "", "Event topic (required)")
	eventBusPublishCmd.Flags().StringVar(&eventPayload, "payload", "", "JSON payload (required)")
	eventBusPublishCmd.Flags().StringVar(&eventSource, "source", "", "Event source")

	// event-bus-query
	rootCmd.AddCommand(eventBusQueryCmd)
	eventBusQueryCmd.Flags().StringVar(&eventPattern, "pattern", "", "Topic pattern (supports trailing * wildcard)")
	eventBusQueryCmd.Flags().StringVar(&eventSince, "since", "", "Only return events after this RFC3339 timestamp")
	eventBusQueryCmd.Flags().IntVar(&eventLimit, "limit", 0, "Maximum events to return (0 for default)")

	// event-bus-replay
	rootCmd.AddCommand(eventBusReplayCmd)
	eventBusReplayCmd.Flags().StringVar(&eventTopic, "topic", "", "Exact topic to replay")
	eventBusReplayCmd.Flags().StringVar(&eventSince, "since", "", "Only return events after this RFC3339 timestamp")
	eventBusReplayCmd.Flags().IntVar(&eventLimit, "limit", 0, "Maximum events to return (0 for default)")

	// event-bus-cleanup
	rootCmd.AddCommand(eventBusCleanupCmd)
	eventBusCleanupCmd.Flags().BoolVar(&eventDryRun, "dry-run", false, "Report counts without modifying the file")
}
