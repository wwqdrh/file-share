package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	UploadPathKey = "uploadPath"
	PortKey       = "port"
	IPKey         = "ip"
	AuthEnableKey = "authEnable"
	PasswordKey   = "password"
	TusEnableKey  = "tusEnable"
	ChunkSizeKey  = "chunkSize"
)

type Settings struct {
	UploadPath string `json:"uploadPath"`
	Port       int    `json:"port"`
	IP         string `json:"ip"`
	AuthEnable bool   `json:"authEnable"`
	Password   string `json:"password"`
	TusEnable  bool   `json:"tusEnable"`
	ChunkSize  int    `json:"chunkSize"`
}

var (
	settings     Settings
	settingsLock sync.RWMutex
	configFile   string
)

// InitSettings initializes the settings with default values
func InitSettings(configPath string) error {
	configFile = configPath
	settings = Settings{
		UploadPath: getDefaultUploadPath(),
		Port:       5421,
		IP:         GetIPAddress(0, "ipv4"),
		AuthEnable: false,
		Password:   "password",
		TusEnable:  false,
		ChunkSize:  20,
	}

	// Load settings from file if it exists
	if _, err := os.Stat(configFile); err == nil {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config file: %v", err)
		}

		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("error parsing config file: %v", err)
		}
	}

	return saveSettings()
}

// GetSettings returns the current settings
func GetSettings() Settings {
	settingsLock.RLock()
	defer settingsLock.RUnlock()
	return settings
}

// UpdateSettings updates the settings with new values
func UpdateSettings(newSettings Settings) error {
	settingsLock.Lock()
	defer settingsLock.Unlock()

	// Validate upload path
	if newSettings.UploadPath != settings.UploadPath {
		if err := validateUploadPath(newSettings.UploadPath); err != nil {
			return err
		}
	}

	// Validate port
	if newSettings.Port <= 0 || newSettings.Port > 65535 {
		return fmt.Errorf("invalid port number")
	}

	// Validate chunk size
	if newSettings.ChunkSize <= 0 {
		return fmt.Errorf("chunk size must be greater than 0")
	}

	settings = newSettings
	return saveSettings()
}

// GetUploadPath returns the current upload path
func GetUploadPath() string {
	settingsLock.RLock()
	defer settingsLock.RUnlock()
	return settings.UploadPath
}

// GetPort returns the current port
func GetPort() int {
	settingsLock.RLock()
	defer settingsLock.RUnlock()
	return settings.Port
}

// GetIP returns the current IP
func GetIP() string {
	settingsLock.RLock()
	defer settingsLock.RUnlock()
	return settings.IP
}

// GetAuthEnable returns whether authentication is enabled
func GetAuthEnable() bool {
	settingsLock.RLock()
	defer settingsLock.RUnlock()
	return settings.AuthEnable
}

// GetPassword returns the current password
func GetPassword() string {
	settingsLock.RLock()
	defer settingsLock.RUnlock()
	return settings.Password
}

// GetTusEnable returns whether TUS upload is enabled
func GetTusEnable() bool {
	settingsLock.RLock()
	defer settingsLock.RUnlock()
	return settings.TusEnable
}

// GetChunkSize returns the current chunk size
func GetChunkSize() int {
	settingsLock.RLock()
	defer settingsLock.RUnlock()
	return settings.ChunkSize
}

// GetURL returns the current server URL
func GetURL() string {
	settingsLock.RLock()
	defer settingsLock.RUnlock()
	return fmt.Sprintf("http://%s:%d", settings.IP, settings.Port)
}

// Helper functions

func getDefaultUploadPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(homeDir, "Downloads")
}

func validateUploadPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("upload path does not exist: %v", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("upload path must be a directory")
	}
	return nil
}

func saveSettings() error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling settings: %v", err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("error writing config file: %v", err)
	}

	return nil
}
