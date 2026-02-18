package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"jtso/logger"
	"os"
	"path/filepath"
)

type SecretManager struct {
	Current  []byte
	Previous []byte
}

func NewSecretManager(dataDir string) (*SecretManager, bool, error) {
	var changeDetected bool

	envSecret := os.Getenv("APP_SECRET")

	secretPath := filepath.Join(dataDir, "secret.txt")
	prevPath := filepath.Join(dataDir, "secret.previous.txt")

	var current string
	var previous string

	// --------------------------------------------------
	// CASE 1: No APP_SECRET provided
	// --------------------------------------------------
	if envSecret == "" {

		if _, err := os.Stat(secretPath); errors.Is(err, os.ErrNotExist) {
			// First ever start → generate secret
			raw := make([]byte, 32)
			if _, err := rand.Read(raw); err != nil {
				return nil, false, err
			}
			current = hex.EncodeToString(raw)

			if err := os.WriteFile(secretPath, []byte(current), 0600); err != nil {
				return nil, false, err
			}

			// previous = current on first boot
			if err := os.WriteFile(prevPath, []byte(current), 0600); err != nil {
				return nil, false, err
			}

			previous = current

		} else {
			// Load existing secret
			data, err := os.ReadFile(secretPath)
			if err != nil {
				return nil, false, err
			}
			current = string(data)

			prevData, err := os.ReadFile(prevPath)
			if err == nil {
				previous = string(prevData)
			} else {
				previous = current
				if err := os.WriteFile(prevPath, []byte(current), 0600); err != nil {
					return nil, false, err
				}
			}
		}

	} else {

		// --------------------------------------------------
		// CASE 2: APP_SECRET is set
		// --------------------------------------------------

		current = envSecret

		// Check if previous exists
		prevData, err := os.ReadFile(prevPath)
		if errors.Is(err, os.ErrNotExist) {

			// First time encryption is enabled
			previous = current

			if err := os.WriteFile(prevPath, []byte(current), 0600); err != nil {
				return nil, false, err
			}

		} else if err != nil {
			return nil, false, err
		} else {
			previous = string(prevData)

			// Detect rotation
			if previous != current {
				changeDetected = true
				logger.Log.Infof("Secret rotation detected. Previous secret will be kept to manage secret rotation.")
			}
		}
		// Always persist current secret
		if err := os.WriteFile(secretPath, []byte(current), 0600); err != nil {
			return nil, false, err
		}
	}

	// --------------------------------------------------
	// Hash keys
	// --------------------------------------------------

	currentKey := sha256.Sum256([]byte(current))

	var previousKey []byte
	if previous != "" {
		hash := sha256.Sum256([]byte(previous))
		previousKey = hash[:]
	}

	sm := &SecretManager{
		Current:  currentKey[:],
		Previous: previousKey,
	}

	return sm, changeDetected, nil
}

func (sm *SecretManager) Rotate() error {
	if sm.Previous == nil {
		return errors.New("no previous secret to rotate to")
	}

	// Set current as previous
	sm.Previous = sm.Current

	// Update previous secret file
	if err := os.WriteFile(filepath.Join("data", "secret.previous.txt"), []byte(hex.EncodeToString(sm.Current)), 0600); err != nil {
		return err
	}

	logger.Log.Infof("Secret rotation completed. Previous secret is now active.")

	return nil
}
