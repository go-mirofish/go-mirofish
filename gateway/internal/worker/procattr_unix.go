//go:build !windows

package worker

import "syscall"

func procAttrForChild() *syscall.SysProcAttr {
	// Start child in its own process group so we can terminate the whole tree.
	return &syscall.SysProcAttr{Setpgid: true}
}

