package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/exchange"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(importCmd)
}

var exportCmd = &cobra.Command{
	Use:          "export [pheromones|wisdom|registry|archive]",
	Short:        "Export colony data to XML format",
	SilenceUsage: true,
}

var importCmd = &cobra.Command{
	Use:          "import [pheromones|wisdom|registry]",
	Short:        "Import colony data from XML format",
	SilenceUsage: true,
}

func init() {
	exportCmd.AddCommand(exportPheromonesCmd)
	exportCmd.AddCommand(exportWisdomCmd)
	exportCmd.AddCommand(exportRegistryCmd)
	exportCmd.AddCommand(exportArchiveCmd)

	importCmd.AddCommand(importPheromonesCmd)
	importCmd.AddCommand(importWisdomCmd)
	importCmd.AddCommand(importRegistryCmd)
}

var exportPheromonesCmd = &cobra.Command{
	Use:   "pheromones",
	Short: "Export pheromone signals to XML",
	RunE:  runExportPheromones,
}

// runExportPheromones is the shared logic for exporting pheromones to XML.
// Called by both "export pheromones" and the flat alias "pheromone-export-xml".
func runExportPheromones(cmd *cobra.Command, args []string) error {
	if store == nil {
		return fmt.Errorf("no colony initialized")
	}

	var file colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &file); err != nil {
		outputOK(map[string]interface{}{"xml": nil, "count": 0})
		return nil
	}

	data, err := exchange.ExportPheromones(file.Signals)
	if err != nil {
		outputErrorMessage(fmt.Sprintf("export pheromones: %v", err))
		return nil
	}

	outputFile, _ := cmd.Flags().GetString("output")
	if outputFile != "" {
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			outputErrorMessage(fmt.Sprintf("write file: %v", err))
			return nil
		}
		outputOK(map[string]interface{}{"file": outputFile, "count": len(file.Signals)})
	} else {
		fmt.Fprintf(stdout, "%s\n", string(data))
	}
	return nil
}

var exportWisdomCmd = &cobra.Command{
	Use:   "wisdom",
	Short: "Export queen wisdom to XML",
	RunE:  runExportWisdom,
}

// runExportWisdom is the shared logic for exporting wisdom to XML.
func runExportWisdom(cmd *cobra.Command, args []string) error {
	if store == nil {
		return fmt.Errorf("no colony initialized")
	}

	minConf, _ := cmd.Flags().GetFloat64("min-confidence")
	if minConf == 0 {
		minConf = 0.5
	}

	var instFile colony.InstinctsFile
	if err := store.LoadJSON("instincts.json", &instFile); err != nil {
		outputOK(map[string]interface{}{"xml": nil, "count": 0})
		return nil
	}

	var entries []exchange.WisdomEntry
	for _, inst := range instFile.Instincts {
		entries = append(entries, exchange.WisdomEntry{
			ID:         inst.ID,
			Category:   "pattern",
			Confidence: inst.Confidence,
			Domain:     inst.Domain,
			Source:     inst.Provenance.Source,
			CreatedAt:  inst.Provenance.CreatedAt,
			Content:    inst.Trigger,
		})
	}

	colonyName, _ := cmd.Flags().GetString("colony")
	data, err := exchange.ExportWisdom(entries, minConf, colonyName)
	if err != nil {
		outputErrorMessage(fmt.Sprintf("export wisdom: %v", err))
		return nil
	}

	outputFile, _ := cmd.Flags().GetString("output")
	if outputFile != "" {
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			outputErrorMessage(fmt.Sprintf("write file: %v", err))
			return nil
		}
		outputOK(map[string]interface{}{"file": outputFile, "count": len(entries)})
	} else {
		fmt.Fprintf(stdout, "%s\n", string(data))
	}
	return nil
}

var exportRegistryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Export colony registry to XML",
	RunE:  runExportRegistry,
}

// runExportRegistry is the shared logic for exporting the colony registry to XML.
func runExportRegistry(cmd *cobra.Command, args []string) error {
	if store == nil {
		return fmt.Errorf("no colony initialized")
	}

	// Load registry from hub
	hub := hubStore()
	if hub == nil {
		return nil
	}

	var regData map[string]interface{}
	if err := hub.LoadJSON("registry/colonies.json", &regData); err != nil {
		outputOK(map[string]interface{}{"xml": nil, "count": 0})
		return nil
	}

	// Convert registry to ColonyEntry format
	var entries []exchange.ColonyEntry
	if coloniesRaw, ok := regData["colonies"]; ok {
		if colonies, ok := coloniesRaw.([]interface{}); ok {
			for _, c := range colonies {
				if cm, ok := c.(map[string]interface{}); ok {
					entry := exchange.ColonyEntry{
						ID:     strVal(cm, "id"),
						Name:   strVal(cm, "name"),
						Status: strVal(cm, "status"),
					}
					if parentID, ok := cm["parent_id"].(string); ok {
						entry.ParentID = parentID
					}
					entries = append(entries, entry)
				}
			}
		}
	}

	data, err := exchange.ExportRegistry(entries)
	if err != nil {
		outputErrorMessage(fmt.Sprintf("export registry: %v", err))
		return nil
	}

	outputFile, _ := cmd.Flags().GetString("output")
	if outputFile != "" {
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			outputErrorMessage(fmt.Sprintf("write file: %v", err))
			return nil
		}
		outputOK(map[string]interface{}{"file": outputFile, "count": len(entries)})
	} else {
		fmt.Fprintf(stdout, "%s\n", string(data))
	}
	return nil
}

var exportArchiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Export complete colony archive to XML",
	RunE:  runExportArchive,
}

// runExportArchive is the shared logic for exporting a complete colony archive to XML.
func runExportArchive(cmd *cobra.Command, args []string) error {
	if store == nil {
		return fmt.Errorf("no colony initialized")
	}

	outputFile, _ := cmd.Flags().GetString("output")
	if outputFile == "" {
		outputErrorMessage("flag --output is required for archive export")
		return nil
	}

	// Export each section
	pheromoneData, _ := exchange.ExportPheromones(nil)
	wisdomData, _ := exchange.ExportWisdom(nil, 0.5, "")
	registryData, _ := exchange.ExportRegistry(nil)

	// Combine into archive sections
	sections := map[string]string{
		"pheromones": string(pheromoneData),
		"wisdom":     string(wisdomData),
		"registry":   string(registryData),
	}

	outputOK(map[string]interface{}{"file": outputFile, "sections": sections})
	return nil
}

var importPheromonesCmd = &cobra.Command{
	Use:   "pheromones <file>",
	Short: "Import pheromone signals from XML",
	Args:  cobra.ExactArgs(1),
	RunE:  runImportPheromones,
}

// runImportPheromones is the shared logic for importing pheromones from XML.
func runImportPheromones(cmd *cobra.Command, args []string) error {
	if store == nil {
		return fmt.Errorf("no colony initialized")
	}

	xmlData, err := os.ReadFile(args[0])
	if err != nil {
		outputErrorMessage(fmt.Sprintf("read file: %v", err))
		return nil
	}

	signals, err := exchange.ImportPheromones(xmlData)
	if err != nil {
		outputErrorMessage(fmt.Sprintf("import pheromones: %v", err))
		return nil
	}

	// Merge with existing pheromones
	var file colony.PheromoneFile
	store.LoadJSON("pheromones.json", &file)
	if file.Signals == nil {
		file.Signals = []colony.PheromoneSignal{}
	}
	file.Signals = append(file.Signals, signals...)

	if err := store.SaveJSON("pheromones.json", file); err != nil {
		outputErrorMessage(fmt.Sprintf("save pheromones: %v", err))
		return nil
	}

	outputOK(map[string]interface{}{"imported": len(signals), "total": len(file.Signals)})
	return nil
}

var importWisdomCmd = &cobra.Command{
	Use:   "wisdom <file>",
	Short: "Import queen wisdom from XML",
	Args:  cobra.ExactArgs(1),
	RunE:  runImportWisdom,
}

// runImportWisdom is the shared logic for importing wisdom from XML.
func runImportWisdom(cmd *cobra.Command, args []string) error {
	if store == nil {
		return fmt.Errorf("no colony initialized")
	}

	xmlData, err := os.ReadFile(args[0])
	if err != nil {
		outputErrorMessage(fmt.Sprintf("read file: %v", err))
		return nil
	}

	entries, err := exchange.ImportWisdom(xmlData)
	if err != nil {
		outputErrorMessage(fmt.Sprintf("import wisdom: %v", err))
		return nil
	}

	outputOK(map[string]interface{}{"imported": len(entries)})
	return nil
}

var importRegistryCmd = &cobra.Command{
	Use:   "registry <file>",
	Short: "Import colony registry from XML",
	Args:  cobra.ExactArgs(1),
	RunE:  runImportRegistry,
}

// runImportRegistry is the shared logic for importing the colony registry from XML.
func runImportRegistry(cmd *cobra.Command, args []string) error {
	if store == nil {
		return fmt.Errorf("no colony initialized")
	}

	xmlData, err := os.ReadFile(args[0])
	if err != nil {
		outputErrorMessage(fmt.Sprintf("read file: %v", err))
		return nil
	}

	entries, err := exchange.ImportRegistry(xmlData)
	if err != nil {
		outputErrorMessage(fmt.Sprintf("import registry: %v", err))
		return nil
	}

	outputOK(map[string]interface{}{"imported": len(entries)})
	return nil
}

func init() {
	exportPheromonesCmd.Flags().String("output", "", "Write XML to file instead of stdout")
	exportWisdomCmd.Flags().String("output", "", "Write XML to file instead of stdout")
	exportWisdomCmd.Flags().Float64("min-confidence", 0.5, "Minimum confidence threshold for wisdom export")
	exportWisdomCmd.Flags().String("colony", "", "Colony ID for wisdom export")
	exportRegistryCmd.Flags().String("output", "", "Write XML to file instead of stdout")
	exportArchiveCmd.Flags().String("output", "", "Output file path (required)")
}

// strVal safely extracts a string from a map.
func strVal(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// _ = json.RawMessage import used by exchange
var _ = json.RawMessage{}
