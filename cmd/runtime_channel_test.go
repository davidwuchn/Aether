package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/calcosmic/Aether/pkg/downloader"
)

func TestNormalizeRuntimeChannel(t *testing.T) {
	tests := []struct {
		in   string
		want runtimeChannel
	}{
		{"", channelStable},
		{"stable", channelStable},
		{"dev", channelDev},
		{"DEV", channelDev},
		{"weird", channelStable},
	}

	for _, tt := range tests {
		if got := normalizeRuntimeChannel(tt.in); got != tt.want {
			t.Errorf("normalizeRuntimeChannel(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRuntimeChannelFromInvocation(t *testing.T) {
	tests := []struct {
		arg0 string
		want runtimeChannel
	}{
		{"/usr/local/bin/aether", channelStable},
		{"/usr/local/bin/aether-dev", channelDev},
		{`C:\bin\aether-dev.exe`, channelDev},
		{"aether.exe", channelStable},
	}

	for _, tt := range tests {
		if got := runtimeChannelFromInvocation(tt.arg0); got != tt.want {
			t.Errorf("runtimeChannelFromInvocation(%q) = %q, want %q", tt.arg0, got, tt.want)
		}
	}
}

func TestDefaultChannelPathsAndBinaryNames(t *testing.T) {
	if got := defaultHubDirName(channelStable); got != ".aether" {
		t.Fatalf("defaultHubDirName(stable) = %q, want .aether", got)
	}
	if got := defaultHubDirName(channelDev); got != ".aether-dev" {
		t.Fatalf("defaultHubDirName(dev) = %q, want .aether-dev", got)
	}
	if got := defaultBinaryName(channelStable); got != "aether" {
		t.Fatalf("defaultBinaryName(stable) = %q, want aether", got)
	}
	if got := defaultBinaryName(channelDev); got != "aether-dev" {
		t.Fatalf("defaultBinaryName(dev) = %q, want aether-dev", got)
	}
	if got := defaultBinaryDestSubdirForChannel(channelDev); got != ".aether-dev/bin" {
		t.Fatalf("defaultBinaryDestSubdirForChannel(dev) = %q, want .aether-dev/bin", got)
	}
}

func TestAlignDownloadedBinaryToChannelRenamesDevBinary(t *testing.T) {
	destDir := t.TempDir()
	stablePath := filepath.Join(destDir, "aether")
	if err := os.WriteFile(stablePath, []byte("binary"), 0755); err != nil {
		t.Fatalf("failed to seed downloaded binary: %v", err)
	}

	result, err := alignDownloadedBinaryToChannel(&downloader.DownloadResult{Path: stablePath, Version: "1.0.19"}, destDir, channelDev)
	if err != nil {
		t.Fatalf("alignDownloadedBinaryToChannel returned error: %v", err)
	}
	if result.Path != filepath.Join(destDir, "aether-dev") {
		t.Fatalf("renamed path = %q, want %q", result.Path, filepath.Join(destDir, "aether-dev"))
	}
	if _, err := os.Stat(result.Path); err != nil {
		t.Fatalf("expected renamed dev binary to exist: %v", err)
	}
	if _, err := os.Stat(stablePath); !os.IsNotExist(err) {
		t.Fatalf("expected original stable path to be gone, got err=%v", err)
	}
}
