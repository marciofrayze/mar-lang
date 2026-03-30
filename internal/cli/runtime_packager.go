package cli

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"mar/internal/appbundle"
	"mar/internal/generator"
	"mar/internal/model"
)

//go:embed compiler_assets/admin/index.html compiler_assets/admin/favicon.svg compiler_assets/admin/dist/app.js runtime_stubs/*
var compilerAssets embed.FS

type runtimeTarget struct {
	OS   string
	Arch string
}

type targetBuildOutput struct {
	Target     runtimeTarget
	OutputPath string
}

type compileBuildResult struct {
	BaseDir     string
	BinaryName  string
	Executables []targetBuildOutput
	ElmPath     string
	TSPath      string
}

var supportedRuntimeTargets = []runtimeTarget{
	{OS: "darwin", Arch: "arm64"},
	{OS: "darwin", Arch: "amd64"},
	{OS: "linux", Arch: "amd64"},
	{OS: "linux", Arch: "arm64"},
	{OS: "windows", Arch: "amd64"},
}

func buildExecutableWithOptions(app *model.App, outputPath string, options buildOptions) error {
	payload, err := buildAppPayload(app, options)
	if err != nil {
		return err
	}
	target, err := hostRuntimeTarget()
	if err != nil {
		return err
	}
	if err := packageExecutable(target, outputPath, payload); err != nil {
		return err
	}
	if options.PrintSummary {
		fmt.Println()
		fmt.Printf("%s\n", colorizeCLI(cliSupportsANSIStream(os.Stdout), "\033[1;32m", "Executable updated:"))
		fmt.Printf("  %s\n", outputPath)
		fmt.Println()
	}
	return nil
}

func compileExecutablesWithOptions(app *model.App, buildRoot, binaryName string, options buildOptions) error {
	payload, err := buildAppPayload(app, options)
	if err != nil {
		return err
	}

	elmPath, tsPath, err := generateClients(app, buildRoot)
	if err != nil {
		return err
	}

	result := compileBuildResult{
		BaseDir:    buildRoot,
		BinaryName: binaryName,
		ElmPath:    elmPath,
		TSPath:     tsPath,
	}
	for _, target := range supportedRuntimeTargets {
		outputPath := targetOutputPath(buildRoot, binaryName, target)
		if err := packageExecutable(target, outputPath, payload); err != nil {
			return fmt.Errorf("%s: %w", target.ID(), err)
		}
		result.Executables = append(result.Executables, targetBuildOutput{
			Target:     target,
			OutputPath: outputPath,
		})
	}

	if options.PrintSummary {
		printCompileSummary(result)
	}
	return nil
}

func buildAppPayload(app *model.App, options buildOptions) ([]byte, error) {
	if app == nil {
		return nil, errors.New("nil app")
	}

	manifestJSON, err := json.Marshal(app)
	if err != nil {
		return nil, err
	}
	manifestDigest := sha256.Sum256(manifestJSON)
	compilerInfo := readVersionInfo("mar")

	appUIFiles, err := embeddedAppUIFiles()
	if err != nil {
		return nil, err
	}

	publicDir, err := validatePublicSourceDir(app, options.SourcePath)
	if err != nil {
		return nil, err
	}
	if app != nil && app.Public != nil && strings.TrimSpace(app.Public.SPAFallback) != "" {
		fallbackPath := filepath.Join(publicDir, filepath.FromSlash(app.Public.SPAFallback))
		if _, err := os.Stat(fallbackPath); err != nil {
			return nil, fmt.Errorf("public.spa_fallback not found in public.dir: %s", app.Public.SPAFallback)
		}
	}

	return appbundle.BuildPayload(appbundle.BuildInput{
		ManifestJSON: manifestJSON,
		Metadata: appbundle.Metadata{
			MarVersion:   compilerInfo.Version,
			MarCommit:    compilerInfo.Commit,
			MarBuildTime: compilerInfo.BuildTime,
			AppBuildTime: time.Now().UTC().Format(time.RFC3339),
			ManifestHash: "sha256:" + hex.EncodeToString(manifestDigest[:]),
		},
		AppUIFiles: appUIFiles,
		PublicDir:  publicDir,
	})
}

func embeddedAppUIFiles() (map[string][]byte, error) {
	files := []string{
		"compiler_assets/admin/index.html",
		"compiler_assets/admin/favicon.svg",
		"compiler_assets/admin/dist/app.js",
	}
	result := make(map[string][]byte, len(files))
	for _, path := range files {
		data, err := compilerAssets.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read embedded App UI asset %s: %w", path, err)
		}
		rel := strings.TrimPrefix(path, "compiler_assets/admin/")
		result[rel] = data
	}
	return result, nil
}

func validatePublicSourceDir(app *model.App, sourcePath string) (string, error) {
	if app == nil || app.Public == nil {
		return "", nil
	}
	publicSourceDir := strings.TrimSpace(app.Public.Dir)
	if publicSourceDir == "" {
		return "", errors.New("public.dir cannot be empty")
	}
	if filepath.IsAbs(publicSourceDir) {
		publicSourceDir = filepath.Clean(publicSourceDir)
	} else {
		sourceBaseDir := "."
		if trimmed := strings.TrimSpace(sourcePath); trimmed != "" {
			if absSourcePath, err := filepath.Abs(trimmed); err == nil {
				sourceBaseDir = filepath.Dir(absSourcePath)
			}
		}
		publicSourceDir = filepath.Clean(filepath.Join(sourceBaseDir, publicSourceDir))
	}

	info, err := os.Stat(publicSourceDir)
	if err != nil {
		return "", fmt.Errorf("public.dir not found: %s", publicSourceDir)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("public.dir must be a directory: %s", publicSourceDir)
	}
	return publicSourceDir, nil
}

func packageExecutable(target runtimeTarget, outputPath string, payload []byte) error {
	stubPath := runtimeStubPath(target)
	stub, err := compilerAssets.ReadFile(stubPath)
	if err != nil {
		return fmt.Errorf("read runtime stub: %w", err)
	}
	return appbundle.WriteExecutable(stub, payload, outputPath, target.OS != "windows")
}

func generateClients(app *model.App, buildRoot string) (string, string, error) {
	manifestPath := filepath.Join(buildRoot, "manifest")

	elmClient, err := generator.GenerateElmClient(app)
	if err != nil {
		return "", "", err
	}
	elmPath := generator.ClientOutputPath(manifestPath, elmClient.FileName)
	if err := os.MkdirAll(filepath.Dir(elmPath), 0o755); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(elmPath, elmClient.Source, 0o644); err != nil {
		return "", "", err
	}

	tsClient, err := generator.GenerateTSClient(app)
	if err != nil {
		return "", "", err
	}
	tsPath := generator.ClientOutputPath(manifestPath, tsClient.FileName)
	if err := os.MkdirAll(filepath.Dir(tsPath), 0o755); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(tsPath, tsClient.Source, 0o644); err != nil {
		return "", "", err
	}

	return elmPath, tsPath, nil
}

func hostRuntimeTarget() (runtimeTarget, error) {
	host := runtimeTarget{OS: runtime.GOOS, Arch: runtime.GOARCH}
	for _, target := range supportedRuntimeTargets {
		if target == host {
			return host, nil
		}
	}
	return runtimeTarget{}, fmt.Errorf("unsupported host platform %s/%s", runtime.GOOS, runtime.GOARCH)
}

func runtimeStubPath(target runtimeTarget) string {
	name := "mar-app"
	if target.OS == "windows" {
		name += ".exe"
	}
	return filepath.ToSlash(filepath.Join("runtime_stubs", target.ID(), name))
}

func (target runtimeTarget) ID() string {
	return target.OS + "-" + target.Arch
}

func targetOutputPath(buildRoot, binaryName string, target runtimeTarget) string {
	return filepath.Join(buildRoot, target.ID(), targetBinaryName(binaryName, target))
}

func targetBinaryName(name string, target runtimeTarget) string {
	binaryName := strings.TrimSpace(name)
	if target.OS == "windows" && !strings.HasSuffix(strings.ToLower(binaryName), ".exe") {
		return binaryName + ".exe"
	}
	return binaryName
}

func printCompileSummary(result compileBuildResult) {
	useColor := cliSupportsANSIStream(os.Stdout)

	fmt.Println()
	fmt.Printf("%s\n", colorizeCLI(useColor, "\033[1m", "Build output"))
	fmt.Printf("  %s\n", colorizeCLI(useColor, "\033[1;32m", "Executables:"))
	sort.Slice(result.Executables, func(i, j int) bool {
		return result.Executables[i].Target.ID() < result.Executables[j].Target.ID()
	})
	for _, output := range result.Executables {
		fmt.Printf("    %s  %s\n", colorizeCLI(useColor, "\033[1;36m", output.Target.ID()+":"), output.OutputPath)
	}
	fmt.Printf("  %s\n", colorizeCLI(useColor, "\033[1;36m", "Clients:"))
	fmt.Printf("    %s\n", result.ElmPath)
	fmt.Printf("    %s\n", result.TSPath)

	if host, err := hostRuntimeTarget(); err == nil {
		hostPath := targetOutputPath(result.BaseDir, result.BinaryName, host)
		hostBinary := targetBinaryName(result.BinaryName, host)
		fmt.Printf("\n  %s\n", colorizeCLI(useColor, "\033[1;33m", "Hint:"))
		fmt.Printf("    %s\n", "To run the host executable and open the Mar App UI:")
		fmt.Printf("    %s\n", colorizeCLI(useColor, "\033[1;32m", "cd "+filepath.Dir(hostPath)))
		fmt.Printf("    %s\n", colorizeCLI(useColor, "\033[1;32m", "./"+hostBinary+" serve"))
	}
	fmt.Println()
}
