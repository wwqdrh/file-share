package api

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/wwqdrh/file-share/utils"
)

func parsePath(filename string) (map[string]interface{}, error) {
	if filename == "" {
		return map[string]interface{}{
			"finalPath": "",
			"filePaths": []string{},
			"startPath": "",
		}, nil
	}

	filePaths := strings.Split(filename, "/")
	var filteredPaths []string
	for _, p := range filePaths {
		if p != "" {
			filteredPaths = append(filteredPaths, p)
		}
	}

	if len(filteredPaths) == 0 {
		return nil, fmt.Errorf("invalid path")
	}

	startPath := filteredPaths[0]
	startFile := GetFile(startPath)
	if startFile.Name == "" {
		return nil, fmt.Errorf("分享列表未找到该文件")
	}

	return map[string]interface{}{
		"finalPath": startFile.Path,
		"filePaths": filteredPaths,
		"startPath": startPath,
	}, nil
}

func StopServer() {
	if server != nil {
		server.Close()
	}
	status = StatusStop
	TriggerEvent(map[string]interface{}{
		"type": "server.statusChange",
		"data": map[string]string{"status": StatusStop},
	})
}

// Handler functions
func HandleFiles(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")

	parseResult, err := parsePath(path)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	finalPath := parseResult["finalPath"].(string)
	if finalPath == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 200,
			"data": map[string]interface{}{
				"path":  []string{},
				"files": ListFiles(),
			},
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"path":  parseResult["filePaths"],
			"files": ListFilesInPath(finalPath),
		},
	})
}

func ListFilesInPath(path string) []interface{} {
	files, err := utils.ListFilesInDir(path)
	if err != nil {
		return nil
	}

	result := make([]interface{}, len(files))
	for i, file := range files {
		result[i] = file
	}
	return result
}

func ParseFileName(path string) string {
	return utils.ExtractFileName(path)
}

func ZipDirectory(sourcePath, destPath string) error {
	return nil
}

func HandleDownload(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if GetAuthEnable() && (token == "" || !session[token]) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	filename := r.URL.Query().Get("filename")
	parseResult, err := parsePath(filename)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	sourceFilePath := parseResult["finalPath"].(string)
	if sourceFilePath == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if file exists
	if _, err := os.Stat(sourceFilePath); os.IsNotExist(err) {
		fmt.Println("file not exist")
		// Remove file from database if it doesn't exist
		filePaths := parseResult["filePaths"].([]string)
		if len(filePaths) > 0 {
			utils.RemoveFileFromDb(utils.FileInfo{Name: filePaths[0]})
		}
		w.WriteHeader(http.StatusNotFound)
		return
	}

	fileInfo, err := os.Stat(sourceFilePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if fileInfo.IsDir() {
		// Handle directory download
		fileName := utils.ExtractFileName(sourceFilePath)
		destZipFile := filepath.Join(filepath.Dir(sourceFilePath), fileName+".zip")

		if err := utils.ZipDirectory(sourceFilePath, destZipFile); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer os.Remove(destZipFile) // Clean up zip file after sending

		downloadName := utils.ExtractFileName(destZipFile)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", url.QueryEscape(downloadName)))
		w.Header().Set("Content-Type", "application/zip")
		http.ServeFile(w, r, destZipFile)
	} else {
		// Handle file download
		downloadName := utils.ExtractFileName(sourceFilePath)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", url.QueryEscape(downloadName)))
		w.Header().Set("download-filename", url.QueryEscape(downloadName))
		http.ServeFile(w, r, sourceFilePath)
	}
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if !GetAuthEnable() {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    200,
			"message": "success",
		})
		return
	}

	var loginData struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&loginData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if loginData.Password != GetPassword() {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    403,
			"message": "密码错误",
		})
		return
	}

	hash := md5.New()
	hash.Write([]byte(loginData.Password))
	token := hex.EncodeToString(hash.Sum(nil))

	sessionMutex.Lock()
	session[token] = true
	sessionMutex.Unlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 200,
		"data": map[string]string{
			"Authorization": token,
		},
		"message": "success",
	})
}

func HandleAddFile(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    500,
			"message": "文件上传失败",
		})
		return
	}
	defer file.Close()

	// Create upload directory if it doesn't exist
	uploadDir := filepath.Join(os.Getenv("HOME"), ".hui", "cache", "fs-share", "files")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    500,
			"message": "创建上传目录失败",
		})
		return
	}

	// Create the destination file
	dstPath := filepath.Join(uploadDir, header.Filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    500,
			"message": "创建目标文件失败",
		})
		return
	}
	defer dst.Close()

	// Copy the uploaded file to the destination
	if _, err := io.Copy(dst, file); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    500,
			"message": "保存文件失败",
		})
		return
	}

	sourceip := getClientIP(r)
	utils.AddFileToDb(
		utils.FileInfo{
			Name:     header.Filename,
			Path:     dstPath,
			Username: sourceip,
		},
	)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    200,
		"message": "添加成功",
	})
}

func HandleAddText(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    500,
			"message": "消息不能为空",
		})
		return
	}

	sourceIP := getClientIP(r)
	AddText(data.Message, sourceIP)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    200,
		"message": "添加成功",
	})
}

func GetServerStatus() string {
	return status
}

func GetFile(name string) utils.FileInfo {
	file, _ := utils.GetFileFromDb(name)
	return file
}

func RemoveFile(file interface{}) {
	if f, ok := file.(utils.FileInfo); ok {
		utils.RemoveFileFromDb(f)
	}
}

func ListFiles() []interface{} {
	files, err := utils.ListFilesFromDb()
	if err != nil {
		return nil
	}

	result := make([]interface{}, len(files))
	for i, file := range files {
		result[i] = file
	}
	return result
}

func AddText(text, username string) {
	utils.AddTextToDb(text, username)
}

func getClientIP(r *http.Request) string {
	ip := r.RemoteAddr
	re := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
	matches := re.FindString(ip)

	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	return matches
}

const (
	StatusStart = "start"
	StatusStop  = "stop"
)

var (
	session      = make(map[string]bool)
	server       *http.Server
	status       = StatusStop
	sessionMutex sync.RWMutex
)

// Empty function stubs for external dependencies
func GetAuthEnable() bool {
	return false
}

func GetPassword() string {
	return ""
}

func GetUrl() string {
	return "http://localhost:8080"
}

func GetUploadPath() string {
	return "./uploads"
}

func RegistryEventListener(event string, callback func()) {
}

func TriggerEvent(event interface{}) {
}

func AuthFilter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !GetAuthEnable() {
			next.ServeHTTP(w, r)
			return
		}

		// Whitelist paths
		if r.URL.Path == "/" ||
			r.URL.Path == "/index.html" ||
			r.URL.Path == "/favicon.ico" ||
			r.URL.Path == "/api/login" ||
			strings.HasPrefix(r.URL.Path, "/api/download") ||
			strings.HasPrefix(r.URL.Path, "/static") {
			next.ServeHTTP(w, r)
			return
		}

		// Validate session
		token := r.Header.Get("Authorization")
		sessionMutex.RLock()
		_, exists := session[token]
		sessionMutex.RUnlock()

		if exists {
			next.ServeHTTP(w, r)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code":    401,
				"message": "认证失败",
			})
		}
	})
}
