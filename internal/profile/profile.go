// Package profile provides access to Chrome browser profile data,
// including cookie decryption using Windows DPAPI and AES-256-GCM.
package profile

import "errors"

// Common errors.
var (
	ErrNoCookies    = errors.New("no chatgpt.com cookies found")
	ErrNotSupported = errors.New("chrome profile cookie extraction is only supported on Windows")
	ErrNoMasterKey  = errors.New("no encrypted master key found in Local State")
)

// Cookie represents a decrypted Chrome cookie.
type Cookie struct {
	Name  string
	Value string
	Host  string
	Path  string
}

// Summary holds the result of a successful profile read.
type Summary struct {
	ProfileName  string
	CookieCount  int
	SessionToken string // __Secure-next-auth.session-token value, if found
}
