//go:build !windows

package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type clipboardCommand struct {
	name string
	args []string
}

type clipboardTool struct {
	copy  clipboardCommand
	paste clipboardCommand
}

var (
	clipboardLookPath = exec.LookPath
	clipboardExec     = func(name string, args ...string) *exec.Cmd { return exec.Command(name, args...) }
	clipboardGetenv   = os.Getenv
)

func writeSystemClipboard(value string) error {
	tool, err := detectSystemClipboardTool()
	if err != nil {
		return err
	}

	cmd := clipboardExec(tool.copy.name, tool.copy.args...)
	cmd.Stdin = strings.NewReader(value)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("write system clipboard: %w%s", err, clipboardStderrSuffix(stderr.String()))
	}
	return nil
}

func readSystemClipboard() (string, error) {
	tool, err := detectSystemClipboardTool()
	if err != nil {
		return "", err
	}

	cmd := clipboardExec(tool.paste.name, tool.paste.args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("read system clipboard: %w%s", err, clipboardStderrSuffix(stderr.String()))
	}
	return string(output), nil
}

func detectSystemClipboardTool() (clipboardTool, error) {
	switch runtime.GOOS {
	case "darwin":
		return clipboardToolByNames(
			clipboardCommand{name: "pbcopy"},
			clipboardCommand{name: "pbpaste"},
		)

	case "linux":
		waylandCopy := clipboardCommand{name: "wl-copy"}
		waylandPaste := clipboardCommand{name: "wl-paste"}
		xclipCopy := clipboardCommand{name: "xclip", args: []string{"-selection", "clipboard"}}
		xclipPaste := clipboardCommand{name: "xclip", args: []string{"-selection", "clipboard", "-o"}}
		xselCopy := clipboardCommand{name: "xsel", args: []string{"--clipboard", "--input"}}
		xselPaste := clipboardCommand{name: "xsel", args: []string{"--clipboard", "--output"}}

		if strings.TrimSpace(clipboardGetenv("WAYLAND_DISPLAY")) != "" {
			if tool, err := clipboardToolByNames(waylandCopy, waylandPaste); err == nil {
				return tool, nil
			}
		}
		if tool, err := clipboardToolByNames(xclipCopy, xclipPaste); err == nil {
			return tool, nil
		}
		if tool, err := clipboardToolByNames(xselCopy, xselPaste); err == nil {
			return tool, nil
		}
		if tool, err := clipboardToolByNames(waylandCopy, waylandPaste); err == nil {
			return tool, nil
		}
	}

	return clipboardTool{}, fmt.Errorf("system clipboard is unavailable on %s", runtime.GOOS)
}

func clipboardToolByNames(copyCmd clipboardCommand, pasteCmd clipboardCommand) (clipboardTool, error) {
	copyPath, err := clipboardLookPath(copyCmd.name)
	if err != nil {
		return clipboardTool{}, err
	}
	pastePath, err := clipboardLookPath(pasteCmd.name)
	if err != nil {
		return clipboardTool{}, err
	}

	return clipboardTool{
		copy: clipboardCommand{
			name: copyPath,
			args: copyCmd.args,
		},
		paste: clipboardCommand{
			name: pastePath,
			args: pasteCmd.args,
		},
	}, nil
}

func clipboardStderrSuffix(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return ": " + trimmed
}
