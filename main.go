package main

import (
	"bytes"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/icza/mjpeg"
)

//go:embed static/*
var staticFiles embed.FS

type CaptureRequest struct {
	Image string `json:"image"`
}

func main() {
	// Create captures directory if it doesn't exist
	capturesDir := "captures"
	if err := os.MkdirAll(capturesDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create captures directory: %v", err)
	}

	// Serve static files from embedded FS
	subFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("Failed to create sub filesystem: %v", err)
	}
	http.Handle("/", http.FileServer(http.FS(subFS)))

	// API endpoint to handle uploaded frames
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CaptureRequest
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		err = json.Unmarshal(body, &req)
		if err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Remove base64 header if present (e.g., data:image/jpeg;base64,)
		b64data := req.Image
		if idx := strings.Index(b64data, ","); idx != -1 {
			b64data = b64data[idx+1:]
		}

		// Decode base64
		imgData, err := base64.StdEncoding.DecodeString(b64data)
		if err != nil {
			http.Error(w, "Failed to decode image", http.StatusBadRequest)
			return
		}

		// Save the file with a timestamp
		timestamp := time.Now().Format("2006-01-02_15-04-05-000")
		filename := filepath.Join(capturesDir, fmt.Sprintf("frame_%s.jpg", timestamp))

		err = os.WriteFile(filename, imgData, 0644)
		if err != nil {
			http.Error(w, "Failed to save image", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	})

	// API endpoint to compile images into AVI video
	http.HandleFunc("/compile", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read all JPEGs from captures directory
		files, err := os.ReadDir(capturesDir)
		if err != nil {
			http.Error(w, "Failed to read captures directory", http.StatusInternalServerError)
			return
		}

		var jpgFiles []string
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".jpg") {
				jpgFiles = append(jpgFiles, filepath.Join(capturesDir, f.Name()))
			}
		}

		if len(jpgFiles) == 0 {
			http.Error(w, "No frames to compile", http.StatusBadRequest)
			return
		}

		// Parse request to get framerate
		type CompileReq struct {
			FPS int32 `json:"fps"`
		}
		var req CompileReq
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &req)
		if req.FPS <= 0 {
			req.FPS = 10 // default 10 fps
		}

		// Output filename
		videosDir := "videos"
		os.MkdirAll(videosDir, os.ModePerm)
		outFilename := filepath.Join(videosDir, fmt.Sprintf("timelapse_%s.avi", time.Now().Format("2006-01-02_15-04-05")))

		// We need to know width and height. Let's decode the first image.
		firstImgBytes, err := os.ReadFile(jpgFiles[0])
		if err != nil {
			http.Error(w, "Failed to read first image", http.StatusInternalServerError)
			return
		}
		
		imgConfig, err := jpeg.DecodeConfig(bytes.NewReader(firstImgBytes))
		if err != nil {
			http.Error(w, "Failed to get image dimensions", http.StatusInternalServerError)
			return
		}

		aw, err := mjpeg.New(outFilename, int32(imgConfig.Width), int32(imgConfig.Height), req.FPS)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create video: %v", err), http.StatusInternalServerError)
			return
		}

		// Add all frames
		for _, file := range jpgFiles {
			data, err := os.ReadFile(file)
			if err == nil {
				aw.AddFrame(data)
			}
		}
		
		aw.Close()

		// Optional: Clean up images after compile
		for _, file := range jpgFiles {
			os.Remove(file)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`{"status": "success", "file": "%s"}`, outFilename)))
	})

	// Start server in background
	port := "8080"
	serverUrl := "http://localhost:" + port
	fmt.Printf("Starting server at %s...\n", serverUrl)

	go func() {
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Give the server a tiny fraction to start
	time.Sleep(100 * time.Millisecond)

	// Automatically open the browser
	openBrowser(serverUrl)

	// Block main goroutine
	select {}
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default: // "linux", "freebsd", "openbsd", "netbsd"
		err = exec.Command("xdg-open", url).Start()
	}
	if err != nil {
		log.Printf("Could not open browser automatically, please go to %s manually.\n", url)
	}
}
