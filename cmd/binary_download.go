package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/calcosmic/Aether/pkg/downloader"
	"github.com/spf13/cobra"
)

// binaryDownloadCmd implements "aether binary-download" which downloads the
// Go binary from GitHub Releases for the current platform.
var binaryDownloadCmd = &cobra.Command{
	Use:   "binary-download",
	Short: "Download the aether Go binary from GitHub Releases",
	Long: `Download the aether Go binary from GitHub Releases for the current platform.

Detects your OS and architecture, downloads the correct archive from
https://github.com/calcosmic/Aether/releases, verifies the SHA-256
checksum, and installs the binary atomically into the selected channel bin.

Use this to update the binary without reinstalling everything.`,
	Args: cobra.NoArgs,
	RunE: runBinaryDownload,
}

func init() {
	binaryDownloadCmd.Flags().String("channel", "", "Runtime channel to download for (stable or dev; default: infer from binary/env)")
	binaryDownloadCmd.Flags().String("version", "", "Version to download (default: current aether version)")
	binaryDownloadCmd.Flags().String("dest", "", "Destination directory (default: channel-specific hub bin)")
	rootCmd.AddCommand(binaryDownloadCmd)
}

func runBinaryDownload(cmd *cobra.Command, args []string) error {
	channel := runtimeChannelFromFlag(cmd.Flags())

	versionFlag, _ := cmd.Flags().GetString("version")
	version, err := resolveReleaseVersion(versionFlag)
	if err != nil {
		return err
	}

	destDir, _ := cmd.Flags().GetString("dest")
	if destDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		destDir = filepath.Join(home, defaultBinaryDestSubdirForChannel(channel))
	}

	outputOK(map[string]interface{}{
		"message": fmt.Sprintf("Downloading %s v%s binary for %s/%s...", defaultBinaryName(channel), version, getGOOS(), getGOArch()),
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
	result, err = alignDownloadedBinaryToChannel(result, destDir, channel)
	if err != nil {
		return fmt.Errorf("download succeeded but channel binary rename failed: %w", err)
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
