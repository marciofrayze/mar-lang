package runtime

import (
	goruntime "runtime"
	"strings"
)

// VersionInfo carries build metadata injected by the generated Mar app binary.
type VersionInfo struct {
	MarVersion   string
	MarCommit    string
	MarBuildTime string
	AppBuildTime string
	ManifestHash string
}

// SetVersionInfo updates runtime metadata exposed by version endpoints.
func (r *Runtime) SetVersionInfo(info VersionInfo) {
	if r == nil {
		return
	}
	r.versionInfo = info
}

func (r *Runtime) publicVersionPayload() map[string]any {
	return map[string]any{
		"app": map[string]any{
			"name":         r.App.AppName,
			"buildTime":    strings.TrimSpace(r.versionInfo.AppBuildTime),
			"manifestHash": strings.TrimSpace(r.versionInfo.ManifestHash),
		},
	}
}

func (r *Runtime) protectedVersionPayload() map[string]any {
	return map[string]any{
		"app": map[string]any{
			"name":         r.App.AppName,
			"buildTime":    strings.TrimSpace(r.versionInfo.AppBuildTime),
			"manifestHash": strings.TrimSpace(r.versionInfo.ManifestHash),
		},
		"mar": map[string]any{
			"version":   strings.TrimSpace(r.versionInfo.MarVersion),
			"commit":    strings.TrimSpace(r.versionInfo.MarCommit),
			"buildTime": strings.TrimSpace(r.versionInfo.MarBuildTime),
		},
		"runtime": map[string]any{
			"goVersion": goruntime.Version(),
			"platform":  goruntime.GOOS + "/" + goruntime.GOARCH,
		},
	}
}
