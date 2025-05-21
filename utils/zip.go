package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ZipDirectory creates a zip file from a directory
func ZipDirectory(sourceDir, outPath string) error {
	// Create the output file
	zipFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("error creating zip file: %v", err)
	}
	defer zipFile.Close()

	// Create a new zip writer
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Walk through the source directory
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory
		if path == sourceDir {
			return nil
		}

		// Create a new file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("error creating zip header: %v", err)
		}

		// Set the relative path in the zip file
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("error getting relative path: %v", err)
		}
		header.Name = relPath

		// If it's a directory, just create the header
		if info.IsDir() {
			header.Name += "/"
			_, err = zipWriter.CreateHeader(header)
			return err
		}

		// For files, create a writer and copy the file contents
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("error creating zip writer: %v", err)
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("error opening source file: %v", err)
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	if err != nil {
		return fmt.Errorf("error walking directory: %v", err)
	}

	return nil
}

// ParseFileName extracts the filename from a path
func ParseFileName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}
