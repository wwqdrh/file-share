package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	machineIDMutex sync.RWMutex
	machineID      string
)

// getMachineID returns a unique machine ID
func getMachineID() string {
	machineIDMutex.RLock()
	if machineID != "" {
		machineIDMutex.RUnlock()
		return machineID
	}
	machineIDMutex.RUnlock()

	machineIDMutex.Lock()
	defer machineIDMutex.Unlock()

	// Generate a simple machine ID based on hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	machineID = hostname
	return machineID
}

// getFileDBKey returns the storage key for the file database
func getFileDBKey() string {
	return "FileDb:" + getMachineID()
}

// FileInfo represents the structure of file information
type FileInfo struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Username string `json:"username"`
	Content  string `json:"content,omitempty"`
	Intro    string `json:"intro,omitempty"`
}

// FileDB represents the file database structure
type FileDB map[string]FileInfo

// addFileToDb adds a file to the database
func addFileToDb(fileName string, fileInfo FileInfo) error {
	fileDb, err := getFileDb()
	if err != nil {
		return err
	}

	fileDb[fileName] = fileInfo
	jsonData, err := json.Marshal(fileDb)
	if err != nil {
		return fmt.Errorf("failed to marshal file database: %v", err)
	}

	return SetStorageItem(getFileDBKey(), string(jsonData))
}

// removeFileToDb removes a file from the database
func removeFileToDb(fileName string) error {
	fileDb, err := getFileDb()
	if err != nil {
		return err
	}

	delete(fileDb, fileName)
	jsonData, err := json.Marshal(fileDb)
	if err != nil {
		return fmt.Errorf("failed to marshal file database: %v", err)
	}

	return SetStorageItem(getFileDBKey(), string(jsonData))
}

// getFileDb retrieves the file database
func getFileDb() (FileDB, error) {
	value, err := GetStorageItem(getFileDBKey(), "{}")
	if err != nil {
		return nil, err
	}

	var fileDb FileDB
	if err := json.Unmarshal([]byte(value.(string)), &fileDb); err != nil {
		return nil, fmt.Errorf("failed to parse file database: %v", err)
	}

	return fileDb, nil
}

// AddFileToDb adds a file to the database
func AddFileToDb(file FileInfo) error {
	fmt.Printf("--- addFile --- %+v\n", file)

	fileInfo := filepath.Clean(file.Path)
	fileStat, err := os.Stat(fileInfo)
	if err != nil {
		return fmt.Errorf("failed to stat file: %v", err)
	}

	if fileStat.IsDir() {
		filename := filepath.Base(fileInfo)
		finalFilename := filename
		suffix := 1

		fileDb, err := getFileDb()
		if err != nil {
			return err
		}

		for {
			if existingFile, exists := fileDb[finalFilename]; !exists || existingFile.Path == fileInfo {
				break
			}
			finalFilename = fmt.Sprintf("%s_%d", filename, suffix)
			suffix++
		}

		fmt.Printf("%s finalFilename\n", finalFilename)
		return addFileToDb(finalFilename, FileInfo{
			Type:     "directory",
			Name:     finalFilename,
			Path:     fileInfo,
			Username: file.Username,
		})
	}

	return addFileToDb(file.Name, FileInfo{
		Type:     "file",
		Name:     file.Name,
		Path:     fileInfo,
		Username: file.Username,
	})
}

// AddTextToDb adds a text entry to the database
func AddTextToDb(text, username string) error {
	fmt.Printf("--- addText --- %s\n", text)
	fmt.Printf("--- username --- %s\n", username)

	name := text
	if len(text) > 20 {
		name = text[:20] + "..."
	}

	intro := text
	if len(text) > 100 {
		intro = text[:100]
	}

	textBody := FileInfo{
		Type:     "text",
		Name:     name,
		Content:  text,
		Intro:    intro,
		Username: username,
	}

	return addFileToDb(textBody.Name, textBody)
}

// RemoveFileFromDb removes a file from the database
func RemoveFileFromDb(file FileInfo) error {
	fmt.Printf("removeFile: %s\n", file.Name)
	return removeFileToDb(file.Name)
}

// ListFilesFromDb returns all files in the database
func ListFilesFromDb() ([]FileInfo, error) {
	fileDb, err := getFileDb()
	if err != nil {
		return nil, err
	}

	files := make([]FileInfo, 0, len(fileDb))
	for _, file := range fileDb {
		files = append(files, file)
	}
	return files, nil
}

// GetFileFromDb retrieves a file from the database by name
func GetFileFromDb(fileName string) (FileInfo, error) {
	fileDb, err := getFileDb()
	if err != nil {
		return FileInfo{}, err
	}

	if file, exists := fileDb[fileName]; exists {
		return file, nil
	}
	return FileInfo{}, fmt.Errorf("file not found: %s", fileName)
}
