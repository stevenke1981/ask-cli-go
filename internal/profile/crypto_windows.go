//go:build windows

package profile

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	crypt32                = syscall.NewLazyDLL("crypt32.dll")
	procCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

// decryptMasterKey base64-decodes and DPAPI-decrypts Chrome's master key
// from the "encrypted_key" field in Local State.
func decryptMasterKey(encodedKey string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, fmt.Errorf("base64 decode master key: %w", err)
	}
	if len(raw) < 5 || string(raw[:5]) != "DPAPI" {
		return nil, errors.New("invalid encrypted key: missing DPAPI prefix")
	}
	return decryptDPAPI(raw[5:])
}

// decryptCookieValue decrypts a Chrome cookie encrypted_value using
// AES-256-GCM.  Expected format: "v10" (3 bytes) + nonce (12 bytes) +
// AES-GCM output (ciphertext + 16-byte tag).
func decryptCookieValue(key, encrypted []byte) ([]byte, error) {
	if len(encrypted) < 15 {
		return nil, errors.New("encrypted value too short")
	}
	if string(encrypted[:3]) != "v10" {
		return nil, fmt.Errorf("unknown encryption version: %q", string(encrypted[:3]))
	}

	nonce := encrypted[3:15]
	ciphertext := encrypted[15:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("AES init: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("GCM init: %w", err)
	}

	plain, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("AES-GCM decrypt: %w", err)
	}
	return plain, nil
}

// decryptDPAPI calls Windows CryptUnprotectData to decrypt a DPAPI blob.
func decryptDPAPI(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("empty DPAPI blob")
	}

	var in, out dataBlob
	in.cbData = uint32(len(data))
	in.pbData = &data[0]

	ret, _, err := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&in)),  // pDataIn
		0,                             // ppszDataDescr
		0,                             // pOptionalEntropy
		0,                             // pvReserved
		0,                             // pPromptStruct
		0,                             // dwFlags
		uintptr(unsafe.Pointer(&out)), // pDataOut
	)
	if ret == 0 {
		return nil, fmt.Errorf("CryptUnprotectData failed: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.pbData)))

	result := make([]byte, out.cbData)
	copy(result, unsafe.Slice(out.pbData, int(out.cbData)))
	return result, nil
}
