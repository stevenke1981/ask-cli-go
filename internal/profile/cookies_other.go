//go:build !windows

package profile

import (
	"database/sql"
	"fmt"
)

func init() {
	openCookiesDBCopy = openCookiesDBCopyOther
}

// openCookiesDBCopyOther is a stub for non-Windows platforms.
func openCookiesDBCopyOther(src string) (*sql.DB, error) {
	return nil, fmt.Errorf("cannot open cookies db: Chrome may be running; close Chrome and retry. On non-Windows, try copying the file manually.")
}
