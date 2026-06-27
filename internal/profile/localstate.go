package profile

import (
	"encoding/json"
	"fmt"
	"os"
)

// LocalState represents the relevant portion of Chrome's Local State JSON.
type LocalState struct {
	OSCrypt struct {
		EncryptedKey string `json:"encrypted_key"`
	} `json:"os_crypt"`
}

// ReadLocalState reads and parses Chrome's Local State JSON file.
func ReadLocalState(path string) (*LocalState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading Local State: %w", err)
	}

	var ls LocalState
	if err := json.Unmarshal(data, &ls); err != nil {
		return nil, fmt.Errorf("parsing Local State: %w", err)
	}

	if ls.OSCrypt.EncryptedKey == "" {
		return nil, fmt.Errorf("%w in %s", ErrNoMasterKey, path)
	}

	return &ls, nil
}

// DecryptMasterKey decrypts the Chrome master key from the Local State.
// It is platform-specific and delegates to the OS crypto layer.
func DecryptMasterKey(ls *LocalState) ([]byte, error) {
	return decryptMasterKey(ls.OSCrypt.EncryptedKey)
}
