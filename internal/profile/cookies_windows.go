//go:build windows

package profile

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	_ "modernc.org/sqlite"
)

var (
	kernel32      = syscall.NewLazyDLL("kernel32.dll")
	procCopyFileW = kernel32.NewProc("CopyFileW")
)

func init() {
	openCookiesDBCopy = openCookiesDBCopyWindows
}

// openCookiesDBCopyWindows copies the cookies DB using Windows CopyFileW API
// and opens the copy. CopyFileW can sometimes bypass user-mode file locks.
func openCookiesDBCopyWindows(src string) (*sql.DB, error) {
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("ask-cli-cookies-%d.db", time.Now().UnixNano()))

	if err := copyFileWinAPI(src, tmpFile); err != nil {
		return nil, fmt.Errorf("cannot copy cookies db: %w", err)
	}

	db, err := sql.Open("sqlite", tmpFile)
	if err != nil {
		os.Remove(tmpFile)
		return nil, fmt.Errorf("open temp copy: %w", err)
	}
	return db, nil
}

// copyFileWinAPI copies a file using the Windows CopyFileW API.
func copyFileWinAPI(src, dst string) error {
	srcPtr, err := syscall.UTF16PtrFromString(src)
	if err != nil {
		return fmt.Errorf("convert src path: %w", err)
	}
	dstPtr, err := syscall.UTF16PtrFromString(dst)
	if err != nil {
		return fmt.Errorf("convert dst path: %w", err)
	}

	ret, _, err := procCopyFileW.Call(
		uintptr(unsafe.Pointer(srcPtr)),
		uintptr(unsafe.Pointer(dstPtr)),
		0,
	)
	if ret == 0 {
		return fmt.Errorf("CopyFileW: %w", err)
	}
	return nil
}
