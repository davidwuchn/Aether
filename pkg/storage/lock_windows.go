//go:build windows

package storage

import (
	"os"

	"golang.org/x/sys/windows"
)

func platformLockFile(f *os.File, shared bool) error {
	flags := uint32(0)
	if !shared {
		flags = windows.LOCKFILE_EXCLUSIVE_LOCK
	}
	var overlapped windows.Overlapped
	return windows.LockFileEx(windows.Handle(f.Fd()), flags, 0, 1, 0, &overlapped)
}

func platformUnlockFile(f *os.File) error {
	var overlapped windows.Overlapped
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &overlapped)
}
