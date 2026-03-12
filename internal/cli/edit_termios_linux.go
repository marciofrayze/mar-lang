//go:build linux

package cli

import "golang.org/x/sys/unix"

const (
	editorTermiosGetReq = unix.TCGETS
	editorTermiosSetReq = unix.TCSETS
)
