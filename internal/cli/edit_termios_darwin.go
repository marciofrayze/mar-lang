//go:build darwin

package cli

import "golang.org/x/sys/unix"

const (
	editorTermiosGetReq = unix.TIOCGETA
	editorTermiosSetReq = unix.TIOCSETA
)
