# Phase 40: Stable Publish Hardening - Pattern Map

**Mapped:** 2026-04-23
**Files analyzed:** 6
**Analogs found:** 6 / 6

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `cmd/publish_cmd.go` | command | request-response | `cmd/install_cmd.go` | exact |
| `cmd/publish_cmd_test.go` | test | request-response | `cmd/install_cmd_test.go` | exact |
| `cmd/version.go` (modify) | command | request-response | `cmd/version.go` (self) | exact |
| `AETHER-OPERATIONS-GUIDE.md` (modify) | documentation | N/A | `AETHER-OPERATIONS-GUIDE.md` (self) | exact |
| `.aether/version.json` (verify) | config | file-I/O | `.aether/version.json` (self) | exact |
| `cmd/install_cmd.go` (modify) | command | request-response | `cmd/install_cmd.go` (self) | exact |

## Pattern Assignments

### `cmd/publish_cmd.go` (command, request-response)

**Analog:** `cmd/install_cmd.go`

**Command definition pattern** (lines 22-43):
```go
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install platform assets and refresh the shared Aether hub",
	Long: "Install Aether globally by copying platform assets to their respective\n" +
		"directories and setting up the distribution hub. ...",
	Args: cobra.NoArgs,
	RunE: runInstall,
}
```

**Flag registration pattern** (lines 56-65):
```go
func init() {
	installCmd.Flags().String("package-dir", "", "Override the embedded install assets with a local Aether checkout or package directory")
	installCmd.Flags().String("home-dir", "", "User home directory (default: $HOME)")
	installCmd.Flags().String("channel", "", "Runtime channel to install (stable or dev; default: infer from binary/env)")
	installCmd.Flags().Bool("download-binary", false, "Also download the Go binary from GitHub Releases")
	installCmd.Flags().String("binary-dest", "", "Destination directory for binary ...")
	installCmd.Flags().String("binary-version", "", "Binary version to download (default: current version)")
	installCmd.Flags().Bool("skip-build-binary", false, "Skip auto-building the Go binary when installing from an Aether source checkout")

	rootCmd.AddCommand(installCmd)
}
```

**Channel resolution pattern** (lines 69-70):
```go
func runInstall(cmd *cobra.Command, args []string) error {
	channel := runtimeChannelFromFlag(cmd.Flags())
```

**Home directory resolution pattern** (lines 77-91):
```go
	homeDir, err := cmd.Flags().GetString("home-dir")
	if err != nil {
		return fmt.Errorf("failed to read --home-dir: %w", err)
	}
	if homeDir == "" {
		homeDir = os.Getenv("HOME")
		if homeDir == "" {
			homeDir = os.Getenv("USERPROFILE")
		}
		if homeDir == "" {
			return fmt.Errorf("cannot determine home directory: set HOME or use --home-dir")
		}
	}
```

**Package directory resolution pattern** (lines 93-100):
```go
	resolvedPackageDir, cleanupPackageDir, err := resolveInstallPackageDir(packageDir)
	if err != nil {
		return err
	}
	if cleanupPackageDir != nil {
		defer cleanupPackageDir()
	}
	packageDir = resolvedPackageDir
```

**Hub setup + version.json writing pattern** (lines 556-655):
```go
func setupInstallHub(hubDir, packageDir string) map[string]interface{} {
	result := map[string]interface{}{
		"label": "Hub",
		"src":   ".aether/",
		"dest":  hubDir,
	}
	// ... sync companion files ...
	// Write version.json using git tags or ldflags
	versionPath := filepath.Join(hubDir, "version.json")
	resolved := resolveVersion(packageDir)
	versionContent := fmt.Sprintf(`{"version":"%s","updated_at":"now"}`, resolved)
	if err := os.WriteFile(versionPath, []byte(versionContent), 0644); err != nil {
		result["version_error"] = fmt.Sprintf("failed to write version: %v", err)
	} else {
		result["version"] = resolved
	}
	return result
}
```

**Binary build pattern** (lines 843-919):
```go
func runLocalBinaryBuildFromInstall(cmd *cobra.Command, homeDir, packageDir string, channel runtimeChannel) error {
	sourceRoot := findAetherModuleRoot(packageDir)
	destDir, _ := cmd.Flags().GetString("binary-dest")
	if destDir == "" {
		destDir = defaultLocalBinaryDest(homeDir, channel)
	}
	version := resolveVersion(sourceRoot)
	result, err := buildLocalBinary(sourceRoot, destDir, version, channel)
	// ... output workflow ...
}

func buildLocalBinary(sourceRoot, destDir, version string, channel runtimeChannel) (*downloader.DownloadResult, error) {
	// ... go build with ldflags ...
	ldflags := fmt.Sprintf("-X github.com/calcosmic/Aether/cmd.Version=%s", version)
	buildCmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", tmpPath, "./cmd/aether")
	buildCmd.Dir = sourceRoot
	// ...
}
```

**Output workflow pattern** (lines 165-172):
```go
	result := map[string]interface{}{
		"message":             fmt.Sprintf("Install complete: %d files copied, %d unchanged", totalCopied, totalSkipped),
		"details":             results,
		"channel":             string(channel),
		"binary_refresh_mode": installBinaryRefreshMode(cmd, packageDir),
		"binary_refresh_note": installBinaryRefreshNote(installBinaryRefreshMode(cmd, packageDir), channel),
	}
	outputWorkflow(result, renderInstallVisual(homeDir, results, totalCopied, totalSkipped, installBinaryRefreshMode(cmd, packageDir)))
```

**Error handling pattern** (lines 160-163):
```go
	if len(syncErrors) > 0 {
		outputError(2, fmt.Sprintf("install failed with %d sync error(s)", len(syncErrors)), map[string]interface{}{"details": results})
		return nil
	}
```

---

### `cmd/publish_cmd_test.go` (test, request-response)

**Analog:** `cmd/install_cmd_test.go`

**Test setup pattern** (lines 12-27):
```go
func TestInstallCommandExists(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	cmd, _, err := rootCmd.Find([]string{"install"})
	if err != nil {
		t.Fatalf("install command not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("install command is nil")
	}
	if cmd.Use != "install" {
		t.Errorf("install command Use = %q, want %q", cmd.Use, "install")
	}
}
```

**Flag verification pattern** (lines 29-44):
```go
func TestInstallCommandFlags(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	cmd, _, err := rootCmd.Find([]string{"install"})
	if err != nil {
		t.Fatalf("install command not found: %v", err)
	}

	expectedFlags := []string{"package-dir", "home-dir", "channel", "download-binary", "binary-dest", "binary-version", "skip-build-binary"}
	for _, name := range expectedFlags {
		if f := cmd.Flags().Lookup(name); f == nil {
			t.Errorf("install command missing flag --%s", name)
		}
	}
}
```

**End-to-end execution pattern** (lines 86-120):
```go
func TestInstallUsesEmbeddedAssetsWithoutPackageDir(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	homeDir := t.TempDir()
	workDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer os.Chdir(oldDir)

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"install", "--home-dir", homeDir})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	// Verify expected files exist
	hubWorkers := filepath.Join(homeDir, ".aether", "system", "workers.md")
	if _, err := os.Stat(hubWorkers); os.IsNotExist(err) {
		t.Fatalf("expected embedded hub file %s to exist after install", hubWorkers)
	}
}
```

**JSON output verification pattern** (lines 488-521):
```go
func TestInstallOutputJSON(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	// ... setup ...
	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"install", "--package-dir", tmpDir, "--home-dir", homeDir})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("install command failed: %v", err)
	}

	output := buf.String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("expected valid JSON output, got parse error: %v, output: %s", err, output)
	}
	if ok, exists := result["ok"]; !exists || ok != true {
		t.Errorf("expected JSON output with ok:true, got: %v", result)
	}
}
```

**Version verification pattern** (from `cmd/e2e_install_setup_update_test.go`, lines 121-127):
```go
		// Verify version.json was created
		versionFile := filepath.Join(hubDir, "version.json")
		if _, err := os.Stat(versionFile); os.IsNotExist(err) {
			t.Fatal("hub version.json not created")
		}
```

---

### `cmd/version.go` (modify - add `--check` flag)

**Analog:** `cmd/version.go` (self)

**Current version command pattern** (lines 11-18):
```go
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print aether version",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputOK(resolveVersion())
		return nil
	},
}
```

**Flag addition pattern** (from `cmd/install_cmd.go`, lines 56-65):
```go
func init() {
	installCmd.Flags().Bool("skip-build-binary", false, "Skip auto-building the Go binary when installing from an Aether source checkout")
	rootCmd.AddCommand(installCmd)
}
```

**Version resolution pattern** (from `cmd/root.go`, lines 28-69):
```go
func resolveVersion(dir ...string) string {
	if Version != "0.0.0-dev" {
		return normalizeVersion(Version)
	}
	// ... git tags, repo version, hub version fallback
}

func readInstalledHubVersion() string {
	hubDir := resolveHubPath()
	if hubDir == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(hubDir, "version.json"))
	// ...
}
```

---

### `cmd/install_cmd.go` (modify - backward compat)

**Analog:** `cmd/install_cmd.go` (self)

**No changes needed to core logic** - the existing `install --package-dir` flow remains intact. The publish command should delegate to or replicate the relevant parts of `runInstall` and `setupInstallHub`.

**Key reusable functions** (already in `cmd/install_cmd.go`):
- `resolveInstallPackageDir()` - lines 220-244
- `setupInstallHub()` - lines 556-655
- `syncDirToHub()` - lines 660-764
- `runLocalBinaryBuildFromInstall()` - lines 843-870
- `buildLocalBinary()` - lines 886-919

---

### `AETHER-OPERATIONS-GUIDE.md` (modify)

**Analog:** `AETHER-OPERATIONS-GUIDE.md` (self)

**Section to update** (lines 149-153):
```markdown
### **Step C — Publish the dev channel from source**

```bash
go run ./cmd/aether install --channel dev --package-dir "$PWD" --binary-dest "/Users/callumcowie/repos/Aether-dev/bin"
```
```

Should become:
```bash
aether publish --channel dev --binary-dest "/Users/callumcowie/repos/Aether-dev/bin"
```

Or from source checkout:
```bash
go run ./cmd/aether publish --channel dev --binary-dest "/Users/callumcowie/repos/Aether-dev/bin"
```

---

## Shared Patterns

### Channel Resolution
**Source:** `cmd/runtime_channel.go`
**Apply to:** `cmd/publish_cmd.go`, `cmd/version.go`
```go
func runtimeChannelFromFlag(cmd flagGetter) runtimeChannel {
	if cmd != nil {
		if value, err := cmd.GetString("channel"); err == nil {
			if channel := normalizeRuntimeChannel(value); channel == channelDev || strings.TrimSpace(value) == "" {
				if strings.TrimSpace(value) != "" {
					return channel
				}
			}
		}
	}
	return resolveRuntimeChannel()
}
```

### Hub Path Resolution
**Source:** `cmd/runtime_channel.go` lines 83-88
**Apply to:** `cmd/publish_cmd.go`, `cmd/version.go`
```go
func resolveHubPathForHome(homeDir string, channel runtimeChannel) string {
	if dir := strings.TrimSpace(os.Getenv("AETHER_HUB_DIR")); dir != "" {
		return dir
	}
	return filepath.Join(homeDir, defaultHubDirName(channel))
}
```

### Version Resolution
**Source:** `cmd/root.go` lines 28-69
**Apply to:** `cmd/publish_cmd.go`, `cmd/version.go`
```go
func resolveVersion(dir ...string) string {
	if Version != "0.0.0-dev" {
		return normalizeVersion(Version)
	}
	// ... git tags, repo version, hub version fallback
}

func normalizeVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}
```

### Output Workflow
**Source:** `cmd/codex_visuals.go` lines 213-222
**Apply to:** All command files
```go
func outputWorkflow(result interface{}, visual string) {
	if shouldRenderVisualOutput(stdout) {
		if !strings.HasSuffix(visual, "\n") {
			visual += "\n"
		}
		writeVisualOutput(stdout, visual)
		return
	}
	outputOK(result)
}
```

### JSON Output
**Source:** `cmd/helpers.go` lines 18-25
**Apply to:** All command files
```go
func outputOK(result interface{}) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		outputError(2, fmt.Sprintf("failed to marshal command result: %v", err), nil)
		return
	}
	fmt.Fprintf(stdout, "{\"ok\":true,\"result\":%s}\n", string(resultJSON))
}
```

### Error Output
**Source:** `cmd/helpers.go` lines 32-57
**Apply to:** All command files
```go
func outputError(code int, message string, details interface{}) {
	if shouldRenderVisualOutput(stderr) {
		fmt.Fprint(stderr, renderVisualError(message, details))
		return
	}
	envelope := struct {
		OK      bool        `json:"ok"`
		Error   string      `json:"error"`
		Code    int         `json:"code"`
		Details interface{} `json:"details,omitempty"`
	}{
		OK:    false,
		Error: message,
		Code:  code,
	}
	// ... marshal and write to stderr
}
```

### Test Globals Management
**Source:** `cmd/testing_main_test.go` lines 64-98
**Apply to:** `cmd/publish_cmd_test.go`
```go
func saveGlobals(t *testing.T) {
	t.Helper()
	origStore := store
	origStdout := stdout
	origStderr := stderr
	// ... save all globals ...
	t.Cleanup(func() {
		store = origStore
		stdout = origStdout
		stderr = origStderr
		// ... restore all globals ...
	})
}
```

### Test Root Command Reset
**Source:** `cmd/testing_main_test.go` lines 105-114
**Apply to:** `cmd/publish_cmd_test.go`
```go
func resetRootCmd(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		rootCmd.SetArgs([]string{})
		rootCmd.SetOut(os.Stdout)
		resetFlags(rootCmd)
	})
}
```

## No Analog Found

No files with no close match - all files have strong analogs in the codebase.

## Metadata

**Analog search scope:** `cmd/`, `.aether/`, root directory
**Files scanned:** 12
**Pattern extraction date:** 2026-04-23

### Key Patterns Summary

1. **All commands use Cobra** with `RunE`, flag registration in `init()`, and `rootCmd.AddCommand()`
2. **All commands use `outputWorkflow()`** for dual-mode output (visual vs JSON)
3. **All commands use `runtimeChannelFromFlag()`** for channel resolution
4. **All tests use `saveGlobals(t)` + `resetRootCmd(t)`** as first actions
5. **All tests override `stdout` with `bytes.Buffer`** to capture JSON output
6. **Version resolution follows priority chain:** ldflags > git tags > repo version.json > hub version.json > "0.0.0-dev"
7. **Hub version.json format:** `{"version":"X.Y.Z","updated_at":"now"}`
8. **Binary build uses ldflags:** `-X github.com/calcosmic/Aether/cmd.Version=<version>`
9. **Error handling returns `nil` after `outputError()`** (Cobra RunE convention)
