package profile

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

const (
	sessionCookieName = "__Secure-next-auth.session-token"
	queryCookies      = `SELECT name, encrypted_value, host_key, path FROM cookies`
)

// ExtractSessionToken reads chatgpt.com cookies from the given profile
// directory and returns the decrypted session token.
func ExtractSessionToken(profileDir string, masterKey []byte) (string, error) {
	cookies, err := queryCookiesForHosts(profileDir, masterKey, "chatgpt.com", "chat.openai.com")
	if err != nil {
		return "", err
	}

	// Look specifically for the session token.
	for _, c := range cookies {
		if c.Name == sessionCookieName && c.Value != "" {
			return c.Value, nil
		}
	}

	return "", ErrNoCookies
}

// ExtractCookies reads and decrypts all cookies for chatgpt.com and
// chat.openai.com from the given profile directory.
func ExtractCookies(profileDir string, masterKey []byte) ([]*Cookie, error) {
	return queryCookiesForHosts(profileDir, masterKey, "chatgpt.com", "chat.openai.com")
}

// queryCookiesForHosts reads and decrypts cookies whose host_key matches
// any of the given hosts (via LIKE).
func queryCookiesForHosts(profileDir string, masterKey []byte, hosts ...string) ([]*Cookie, error) {
	dbPath := CookiesDBPath(profileDir)

	db, err := openCookiesDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open cookies db: %w", err)
	}
	defer db.Close()

	// Build a LIKE clause for the hosts.
	var conditions []string
	var args []any
	for _, h := range hosts {
		conditions = append(conditions, `host_key LIKE ? OR host_key LIKE ?`)
		args = append(args, `%.`+h, h)
	}
	where := strings.Join(conditions, " OR ")

	query := fmt.Sprintf("%s WHERE (%s) ORDER BY host_key, name", queryCookies, where)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query cookies: %w", err)
	}
	defer rows.Close()

	var cookies []*Cookie
	for rows.Next() {
		var name, hostKey, path string
		var encryptedValue []byte
		if err := rows.Scan(&name, &encryptedValue, &hostKey, &path); err != nil {
			continue // skip malformed rows
		}

		value := decryptCookieValueOrFallback(masterKey, encryptedValue)
		cookies = append(cookies, &Cookie{
			Name:  name,
			Value: value,
			Host:  hostKey,
			Path:  path,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cookies: %w", err)
	}

	if len(cookies) == 0 {
		return nil, ErrNoCookies
	}
	return cookies, nil
}

// openCookiesDB opens the Chrome cookies SQLite database.
// On Windows, it falls back to copying the file if Chrome holds a lock.
func openCookiesDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return openCookiesDBCopy(path)
	}
	return db, nil
}

// openCookiesDBCopy is defined in platform-specific files
// (cookies_windows.go / cookies_other.go).
var openCookiesDBCopy func(string) (*sql.DB, error)

// decryptCookieValueOrFallback tries AES-GCM decryption; if that fails,
// falls back to treating the value as plaintext (for unencrypted cookies).
func decryptCookieValueOrFallback(masterKey, encrypted []byte) string {
	if len(encrypted) == 0 {
		return ""
	}

	plain, err := decryptCookieValue(masterKey, encrypted)
	if err != nil {
		// Some older cookies may be stored as plaintext.
		if len(encrypted) < 256 && isLikelyPlainText(string(encrypted)) {
			return string(encrypted)
		}
		return ""
	}
	return string(plain)
}

// isLikelyPlainText is a cheap heuristic: returns true if the string
// contains only printable ASCII characters and whitespace.
func isLikelyPlainText(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r < 0x20 || r > 0x7e {
			if r != '\t' && r != '\n' && r != '\r' {
				return false
			}
		}
	}
	return true
}
