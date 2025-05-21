package utils

import (
	"fmt"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// OpenFile opens the file explorer at the specified path
func OpenFile(filePath string) error {
	fileDir := filePath
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %v", err)
	}

	if !fileInfo.IsDir() {
		fileDir = filepath.Dir(filePath)
	}

	// On macOS, we can use the 'open' command to open Finder
	cmd := exec.Command("open", fileDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open file explorer: %v", err)
	}

	fmt.Printf("file explore open: %s\n", filePath)
	return nil
}

// ListFilesInDir lists all files in the specified directory
func ListFilesInDir(fileDir string) ([]FileInfo, error) {
	fmt.Println(fileDir)

	// Check if path is a directory
	fileInfo, err := os.Stat(fileDir)
	if err != nil {
		return nil, fmt.Errorf("file not exists: %v", err)
	}
	if !fileInfo.IsDir() {
		return nil, fmt.Errorf("file is not directory")
	}

	var files []FileInfo
	entries, err := os.ReadDir(fileDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %v", err)
	}

	for _, entry := range entries {
		fmt.Println(entry.Name())
		fileAbsPath := filepath.Join(fileDir, entry.Name())
		fileType := "file"
		if entry.IsDir() {
			fileType = "directory"
		}
		files = append(files, FileInfo{
			Type: fileType,
			Name: entry.Name(),
			Path: fileAbsPath,
		})
	}

	return files, nil
}

// ExtractFileName extracts the filename from a path
func ExtractFileName(filePath string) string {
	words := strings.Split(filePath, string(filepath.Separator))
	words = filterEmpty(words)
	if len(words) > 0 {
		return words[len(words)-1]
	}
	return ""
}

// filterEmpty filters out empty strings from a slice
func filterEmpty(slice []string) []string {
	var result []string
	for _, s := range slice {
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

// GetAllFiles recursively gets all files in a directory
func GetAllFiles(dirPath string) ([]string, error) {
	var files []string
	err := filepath.Walk(dirPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// GetTotalSize calculates the total size of all files in a directory
func GetTotalSize(directoryPath string) (int64, error) {
	files, err := GetAllFiles(directoryPath)
	if err != nil {
		return 0, err
	}

	var totalSize int64
	for _, filePath := range files {
		info, err := os.Stat(filePath)
		if err != nil {
			return 0, err
		}
		totalSize += info.Size()
	}
	return totalSize, nil
}

// ConvertBytes converts bytes to human-readable format
func ConvertBytes(bytes int64) string {
	if bytes == 0 {
		return "n/a"
	}

	sizes := []string{"Bytes", "KB", "MB", "GB", "TB"}
	i := int(math.Floor(math.Log(float64(bytes)) / math.Log(1024)))
	if i == 0 {
		return fmt.Sprintf("%d %s", bytes, sizes[i])
	}
	return fmt.Sprintf("%.1f %s", float64(bytes)/math.Pow(1024, float64(i)), sizes[i])
}

// GetTotalSizeReadable gets the total size in human-readable format
func GetTotalSizeReadable(directoryPath string) (string, error) {
	size, err := GetTotalSize(directoryPath)
	if err != nil {
		return "", err
	}
	return ConvertBytes(size), nil
}
