package core

import (
	"errors"
	"os"

	"github.com/anyproto/anytype-cli/core/config"
	"github.com/zalando/go-keyring"
)

const (
	keyringService        = "anytype-cli"
	keyringTokenUser      = "session-token"
	keyringAccountKeyUser = "account-key"
)

var (
	keyringUnavailable = false
	ErrNotFound        = errors.New("credentials not found")
)

// isKeyringAvailable checks if the keyring is accessible
func isKeyringAvailable() bool {
	if keyringUnavailable {
		return false
	}
	// A second instance can opt out of the shared system keyring (single creds
	// slot) and keep its creds in its own DATA_PATH config.json instead.
	if os.Getenv("ANYTYPE_NO_KEYRING") != "" {
		return false
	}

	err := keyring.Set(keyringService, "test", "test")
	if err != nil {
		keyringUnavailable = true
		return false
	}
	_ = keyring.Delete(keyringService, "test")
	return true
}

// SaveSessionToken saves the session token to the keyring if available, otherwise to the config file.
// Returns true if saved to keyring, false if saved to config file.
func SaveSessionToken(token string) (bool, error) {
	if isKeyringAvailable() {
		return true, keyring.Set(keyringService, keyringTokenUser, token)
	}

	return false, config.SetSessionTokenToConfig(token)
}

// GetStoredSessionToken retrieves the session token from keyring if available, otherwise from config file.
// Returns the token, true if retrieved from keyring (false if from config file), and any error.
func GetStoredSessionToken() (string, bool, error) {
	if isKeyringAvailable() {
		token, err := keyring.Get(keyringService, keyringTokenUser)
		if err == nil {
			return token, true, nil
		}
		if !errors.Is(err, keyring.ErrNotFound) {
			keyringUnavailable = true
		}
	}

	token, _ := config.GetSessionTokenFromConfig()
	if token == "" {
		return "", false, ErrNotFound
	}
	return token, false, nil
}

func DeleteStoredSessionToken() error {
	var keyringErr error
	if isKeyringAvailable() {
		keyringErr = keyring.Delete(keyringService, keyringTokenUser)
	}

	configErr := config.SetSessionTokenToConfig("")

	if keyringErr != nil && configErr != nil {
		return configErr
	}
	return nil
}

// SaveAccountKey saves the account key to the keyring if available, otherwise to the config file.
// Returns true if saved to keyring, false if saved to config file.
func SaveAccountKey(accountKey string) (bool, error) {
	if isKeyringAvailable() {
		return true, keyring.Set(keyringService, keyringAccountKeyUser, accountKey)
	}

	return false, config.SetAccountKeyToConfig(accountKey)
}

// GetStoredAccountKey retrieves the account key from keyring if available, otherwise from config file.
// Returns the account key, true if retrieved from keyring (false if from config file), and any error.
func GetStoredAccountKey() (string, bool, error) {
	if isKeyringAvailable() {
		key, err := keyring.Get(keyringService, keyringAccountKeyUser)
		if err == nil {
			return key, true, nil
		}
		if !errors.Is(err, keyring.ErrNotFound) {
			keyringUnavailable = true
		}
	}

	key, _ := config.GetAccountKeyFromConfig()
	if key == "" {
		return "", false, ErrNotFound
	}
	return key, false, nil
}

func DeleteStoredAccountKey() error {
	var keyringErr error
	if isKeyringAvailable() {
		keyringErr = keyring.Delete(keyringService, keyringAccountKeyUser)
	}

	configErr := config.SetAccountKeyToConfig("")

	if keyringErr != nil && configErr != nil {
		return configErr
	}
	return nil
}
