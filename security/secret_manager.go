package security

import (
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
		return nil, false, errors.New("APP_SECRET environment variable is not set")
	} else {

		// Try to load curent secret from file
		currentData, err := os.ReadFile(secretPath)
		if err == nil {
			current = string(currentData)
			if current != envSecret {
				changeDetected = true
				logger.Log.Infof("Secret rotation detected. Previous secret will be kept to manage secret rotation.")
			}
			// Update current secret with env variable value
			current = envSecret
			if err := os.WriteFile(secretPath, []byte(current), 0600); err != nil {
				return nil, false, err
			}
			// Keep previous secret if rotation is detected
			if changeDetected {
				previous = string(currentData)
				if err := os.WriteFile(prevPath, []byte(previous), 0600); err != nil {
					return nil, false, err
				}
			}
		} else {
			// First sime APP_SECRET is set, persist it and set previous = current
			current, previous = envSecret, envSecret
			if err := os.WriteFile(secretPath, []byte(current), 0600); err != nil {
				return nil, false, err
			}
			if err := os.WriteFile(prevPath, []byte(previous), 0600); err != nil {
				return nil, false, err
			}
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

	logger.Log.Infof("Rotate the secret")

	return nil
}
