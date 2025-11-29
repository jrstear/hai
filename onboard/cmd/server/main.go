package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"hai/onboard/internal/server"
)

var (
	port      = flag.String("port", "3000", "Server port")
	noOpen    = flag.Bool("no-open", false, "Don't automatically open browser")
	logFile   = flag.String("log-file", "", "Log file path (default: stderr)")
	outputDir = flag.String("output-dir", "data", "Base directory for all data (audio, lifelogs, diarization)")
)

func main() {
	flag.Parse()

	// Setup logging
	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	// Check for required environment variables
	// All secrets must be provided via environment variables
	if os.Getenv("HF_TOKEN") == "" {
		log.Println("Warning: HF_TOKEN not set. Diarization will fail.")
		log.Println("Set it with: export HF_TOKEN='your-token-here'")
	}

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Create server
	srv := server.NewServer(*outputDir)

	// Setup routes
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/api/submit", srv.HandleSubmit)
	http.HandleFunc("/api/status", srv.HandleStatus)
	http.HandleFunc("/api/cancel", srv.HandleCancel)

	// Start server
	addr := fmt.Sprintf(":%s", *port)
	log.Printf("Starting onboarding server on http://localhost%s", addr)
	log.Printf("Output directory: %s", *outputDir)

	// Open browser if requested
	if !*noOpen {
		go openBrowser(fmt.Sprintf("http://localhost%s", addr))
	}

	// Start server
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read template file - try multiple possible paths
	possiblePaths := []string{
		filepath.Join("templates", "index.html"),
		filepath.Join("..", "..", "templates", "index.html"),
		filepath.Join("onboard", "templates", "index.html"),
	}
	
	var data []byte
	var err error
	for _, templatePath := range possiblePaths {
		data, err = os.ReadFile(templatePath)
		if err == nil {
			break
		}
	}
	
	if err != nil {
		http.Error(w, fmt.Sprintf("Template not found. Tried: %v", possiblePaths), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(data)
}

func openBrowser(url string) {
	// Try to open browser
	var err error
	
	// macOS
	if _, err = os.Stat("/usr/bin/open"); err == nil {
		cmd := exec.Command("open", url)
		err = cmd.Start()
	} else if _, err = os.Stat("/usr/bin/xdg-open"); err == nil {
		// Linux
		cmd := exec.Command("xdg-open", url)
		err = cmd.Start()
	} else {
		// Windows
		cmd := exec.Command("cmd", "/c", "start", url)
		err = cmd.Start()
	}
	
	if err == nil {
		log.Printf("Opened browser to: %s", url)
	} else {
		log.Printf("Please open your browser to: %s", url)
	}
}
