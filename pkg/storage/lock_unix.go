//go:build !windows

package storage

import (
	"os"
	"syscall"
)

func platformLockFile(f *os.File, shared bool) error {
	mode := syscall.LOCK_EX
	if shared {
		mode = syscall.LOCK_SH
	}
	return syscall.Flock(int(f.Fd()), mode)
}

func platformUnlockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
