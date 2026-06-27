//go:build !windows

package profile

import "fmt"

func decryptMasterKey(encodedKey string) ([]byte, error) {
	return nil, fmt.Errorf("%w: DPAPI decryption requires Windows", ErrNotSupported)
}

func decryptCookieValue(key, encrypted []byte) ([]byte, error) {
	return nil, fmt.Errorf("%w: AES-GCM cookie decryption requires Windows", ErrNotSupported)
}
