package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	storageMutex sync.RWMutex
	storagePath  = filepath.Join(os.Getenv("HOME"), ".hui", "cache", "fs-share", "files.json")
)

// ensureStoragePath ensures the storage directory exists
func ensureStoragePath() error {
	dir := filepath.Dir(storagePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %v", err)
	}
	return nil
}

// SetStorageItem sets a key-value pair in the storage
func SetStorageItem(key string, value interface{}) error {
	storageMutex.Lock()
	defer storageMutex.Unlock()

	// Ensure storage directory exists
	if err := ensureStoragePath(); err != nil {
		return err
	}

	// Read existing data
	data := make(map[string]interface{})
	if _, err := os.Stat(storagePath); err == nil {
		file, err := os.ReadFile(storagePath)
		if err != nil {
			return fmt.Errorf("failed to read storage file: %v", err)
		}
		if len(file) > 0 {
			if err := json.Unmarshal(file, &data); err != nil {
				return fmt.Errorf("failed to parse storage file: %v", err)
			}
		}
	}

	// Update data
	data[key] = value

	// Write back to file
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal storage data: %v", err)
	}

	if err := os.WriteFile(storagePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write storage file: %v", err)
	}

	return nil
}

// GetStorageItem retrieves a value from storage by key
func GetStorageItem(key string, defaultValue interface{}) (interface{}, error) {
	storageMutex.RLock()
	defer storageMutex.RUnlock()

	// Ensure storage directory exists
	if err := ensureStoragePath(); err != nil {
		return defaultValue, err
	}

	// Read storage file
	if _, err := os.Stat(storagePath); err != nil {
		return defaultValue, nil
	}

	file, err := os.ReadFile(storagePath)
	if err != nil {
		return defaultValue, fmt.Errorf("failed to read storage file: %v", err)
	}

	if len(file) == 0 {
		return defaultValue, nil
	}

	// Parse JSON data
	var data map[string]interface{}
	if err := json.Unmarshal(file, &data); err != nil {
		return defaultValue, fmt.Errorf("failed to parse storage file: %v", err)
	}

	// Get value
	value, exists := data[key]
	if !exists {
		return defaultValue, nil
	}

	return value, nil
}
