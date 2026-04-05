package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/calcosmic/Aether/pkg/downloader"
)

// binaryDownloadCmd implements "aether binary-download" which downloads the
// Go binary from GitHub Releases for the current platform.
var binaryDownloadCmd = &cobra.Command{
	Use:   "binary-download",
	Short: "Download the aether Go binary from GitHub Releases",
	Long: `Download the aether Go binary from GitHub Releases for the current platform.

Detects your OS and architecture, downloads the correct archive from
https://github.com/calcosmic/Aether/releases, verifies the SHA-256
checksum, and installs the binary atomically to ~/.aether/bin/aether.

Use this to update the binary without reinstalling everything.`,
	Args: cobra.NoArgs,
	RunE: runBinaryDownload,
}

func init() {
	binaryDownloadCmd.Flags().String("version", "", "Version to download (default: current aether version)")
	binaryDownloadCmd.Flags().String("dest", "", "Destination directory (default: ~/.aether/bin)")
	rootCmd.AddCommand(binaryDownloadCmd)
}

func runBinaryDownload(cmd *cobra.Command, args []string) error {
	version, _ := cmd.Flags().GetString("version")
	if version == "" {
		version = Version
	}

	destDir, _ := cmd.Flags().GetString("dest")
	if destDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		destDir = filepath.Join(home, downloader.DefaultDestSubdir())
	}

	outputOK(map[string]interface{}{
		"message": fmt.Sprintf("Downloading aether v%s binary for %s/%s...", version, getGOOS(), getGOArch()),
		"version": version,
		"dest":    destDir,
	})

	result, err := downloader.DownloadBinary(version, destDir)
	if err != nil {
		if downloader.IsVersionNotFoundErr(err) {
			return fmt.Errorf("version v%s not found. Run 'aether version' to check the latest version: %w", version, err)
		}
		return fmt.Errorf("download failed: %w", err)
	}

	outputOK(map[string]interface{}{
		"message": fmt.Sprintf("Binary installed successfully to %s", result.Path),
		"path":    result.Path,
		"version": result.Version,
	})

	return nil
}

// getGOOS returns runtime.GOOS for use in output messages.
func getGOOS() string { return runtime.GOOS }

// getGOArch returns runtime.GOARCH for use in output messages.
func getGOArch() string { return runtime.GOARCH }
