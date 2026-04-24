//go:build windows

package worker

import "syscall"

func procAttrForChild() *syscall.SysProcAttr {
	// Windows doesn't support Setpgid. Keep defaults; output already streams to gateway logs.
	return &syscall.SysProcAttr{}
}

