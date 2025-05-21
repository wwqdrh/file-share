package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/wwqdrh/file-share/api"
	"github.com/wwqdrh/file-share/utils"
)

var (
	port *int = flag.Int("port", 5421, "端口号")
)

//go:embed dist/*
var embeddedFiles embed.FS

func main() {
	flag.Parse()

	mux := http.NewServeMux()

	// Create a sub filesystem from the embedded files, stripping the "dist" prefix
	fsys, err := fs.Sub(embeddedFiles, "dist")
	if err != nil {
		panic(fmt.Sprintf("failed to create sub filesystem: %v", err))
	}

	// Static file server with the embedded files
	// The files will be served from root "/" without the "dist" prefix in URL
	mux.Handle("/", http.FileServer(http.FS(fsys)))

	// API routes
	mux.HandleFunc("/api/files", api.HandleFiles)
	mux.HandleFunc("/api/download", api.HandleDownload)
	mux.HandleFunc("/api/login", api.HandleLogin)
	mux.HandleFunc("/api/addFile", api.HandleAddFile)
	mux.HandleFunc("/api/addText", api.HandleAddText)
	mux.HandleFunc("/api/registrySSE", api.RegistrySSE)

	// Wrap all API routes with auth filter
	handler := api.AuthFilter(mux)

	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", *port), // 监听所有网络接口
		Handler: handler,
	}
	fmt.Printf("servers is start on %s:%d\n", utils.GetIPAddress(0, "ipv4"), *port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("Server error: %v\n", err)
	}
}
