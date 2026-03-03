package runtime

import (
	"fmt"
	"os"
	"strings"
)

const (
	ansiReset   = "\033[0m"
	ansiTitle   = "\033[1;97m"
	ansiLabel   = "\033[1;36m"
	ansiSection = "\033[1;34m"
	ansiHint    = "\033[1;33m"
	ansiCommand = "\033[1;32m"
	ansiValue   = "\033[37m"
	ansiMuted   = "\033[90m"
	ansiDim     = "\033[2m"
)

// printStartupBanner prints a human-friendly runtime summary with optional ANSI colors.
func (r *Runtime) printStartupBanner() {
	useColor := supportsANSI()
	apiURL := fmt.Sprintf("http://localhost:%d", r.App.Port)

	fmt.Printf("\n%s %q\n", colorize(useColor, ansiTitle, "Belm app"), r.App.AppName)
	fmt.Printf("  %s %s\n", colorize(useColor, ansiLabel, "API"), colorize(useColor, ansiValue, apiURL))
	fmt.Printf("  %s %s\n", colorize(useColor, ansiLabel, "SQLite"), colorize(useColor, ansiMuted, r.App.Database))

	if r.authEnabled() {
		fmt.Printf("\n%s\n", colorize(useColor, ansiSection, "Auth"))
		fmt.Printf("  %s %s\n", colorize(useColor, ansiDim, "POST"), colorize(useColor, ansiValue, "/auth/request-code"))
		fmt.Printf("  %s %s\n", colorize(useColor, ansiDim, "POST"), colorize(useColor, ansiValue, "/auth/login"))
		fmt.Printf("  %s %s\n", colorize(useColor, ansiDim, "POST"), colorize(useColor, ansiValue, "/auth/logout"))
		fmt.Printf("  %s %s\n", colorize(useColor, ansiDim, "GET "), colorize(useColor, ansiValue, "/auth/me"))
	}

	if len(r.App.Entities) > 0 {
		fmt.Printf("\n%s\n", colorize(useColor, ansiSection, "CRUD"))
		for _, entity := range r.App.Entities {
			fmt.Printf("  %s %s\n", colorize(useColor, ansiMuted, "ALL "), colorize(useColor, ansiValue, entity.Resource))
		}
	}

	if len(r.App.Actions) > 0 {
		fmt.Printf("\n%s\n", colorize(useColor, ansiSection, "Actions"))
		for _, action := range r.App.Actions {
			fmt.Printf("  %s %s\n", colorize(useColor, ansiDim, "POST"), colorize(useColor, ansiValue, "/actions/"+action.Name))
		}
	}

	fmt.Printf("\n%s run %s to open Belm Admin\n", colorize(useColor, ansiHint, "Hint:"), colorize(useColor, ansiCommand, os.Args[0]+" admin"))
}

func colorize(enabled bool, colorCode, value string) string {
	if !enabled {
		return value
	}
	return colorCode + value + ansiReset
}

func supportsANSI() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	term := strings.ToLower(strings.TrimSpace(os.Getenv("TERM")))
	if term == "" || term == "dumb" {
		return false
	}
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
