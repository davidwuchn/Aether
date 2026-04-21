package cmd

import (
	"context"
	"encoding/json"

	"github.com/calcosmic/Aether/pkg/colony"
)

func emitPromptIntegrityEvents(command string, records []colony.PromptIntegrityRecord) {
	if store == nil || len(records) == 0 {
		return
	}
	bus, err := newEventBus()
	if err != nil {
		return
	}

	for _, record := range records {
		if record.Action == colony.PromptIntegrityActionAllow && len(record.Findings) == 0 {
			continue
		}
		payload, err := json.Marshal(map[string]interface{}{
			"command":          command,
			"name":             record.Name,
			"title":            record.Title,
			"source":           record.Source,
			"base_trust_class": record.BaseTrustClass,
			"trust_class":      record.TrustClass,
			"action":           record.Action,
			"blocked":          record.Blocked,
			"findings":         record.Findings,
		})
		if err != nil {
			continue
		}

		topic := "prompt.integrity"
		switch record.Action {
		case colony.PromptIntegrityActionBlock:
			topic = "prompt.integrity.block"
		case colony.PromptIntegrityActionWarn:
			topic = "prompt.integrity.warn"
		}
		_, _ = bus.Publish(context.Background(), topic, payload, command)
	}
}

func colonyPrimeIntegrityRecords(output colonyPrimeOutput) []colony.PromptIntegrityRecord {
	records := make([]colony.PromptIntegrityRecord, 0, len(output.Ledger.Included)+len(output.Ledger.Trimmed)+len(output.Ledger.Blocked))
	for _, group := range [][]colonyPrimeLedgerItem{output.Ledger.Included, output.Ledger.Trimmed, output.Ledger.Blocked} {
		for _, item := range group {
			records = append(records, colony.PromptIntegrityRecord{
				Name:           item.Name,
				Title:          item.Title,
				Source:         item.Source,
				BaseTrustClass: item.BaseTrustClass,
				TrustClass:     item.TrustClass,
				Action:         item.Action,
				Blocked:        item.Blocked,
				Findings:       append([]colony.PromptIntegrityFinding(nil), item.Findings...),
			})
		}
	}
	return records
}
