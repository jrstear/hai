package server

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"
)

// WebServer handles HTTP requests for the web frontend
type WebServer struct {
	apiURL         string
	apiClient      *http.Client
	templates      *template.Template
	limitlessAPIKey string
}

// NewWebServer creates a new web server instance
func NewWebServer(apiURL string) (*WebServer, error) {
	// Initialize templates
	// Look for templates in multiple possible locations
	templateDirs := []string{
		"templates",
		"web/templates",
		"../web/templates",
	}

	var templates *template.Template
	var err error
	// Define template functions
	funcMap := template.FuncMap{
		"firstChar": func(s string) string {
			if len(s) == 0 {
				return "?"
			}
			return string(s[0])
		},
		"substr": func(s string, start, length int) string {
			if start >= len(s) {
				return ""
			}
			end := start + length
			if end > len(s) {
				end = len(s)
			}
			return s[start:end]
		},
		"formatDate": func(dateStr interface{}) string {
			// Handle both string and time.Time
			var t time.Time
			switch v := dateStr.(type) {
			case string:
				parsed, err := time.Parse(time.RFC3339, v)
				if err != nil {
					return "Unknown"
				}
				t = parsed
			case time.Time:
				t = v
			default:
				return "Unknown"
			}
			
			now := time.Now()
			diff := now.Sub(t)
			
			// Relative time formatting
			if diff < time.Minute {
				return "Just now"
			} else if diff < time.Hour {
				minutes := int(diff.Minutes())
				return fmt.Sprintf("%dm ago", minutes)
			} else if diff < 24*time.Hour {
				hours := int(diff.Hours())
				return fmt.Sprintf("%dh ago", hours)
			} else if diff < 7*24*time.Hour {
				days := int(diff.Hours() / 24)
				return fmt.Sprintf("%dd ago", days)
			} else {
				// Format as "12/2" or "Dec 2"
				return t.Format("1/2")
			}
		},
		"formatDuration": func(seconds interface{}) string {
			// Convert seconds (int64 or float64) to HH:MM format
			var sec int64
			switch v := seconds.(type) {
			case int64:
				sec = v
			case int:
				sec = int64(v)
			case float64:
				sec = int64(v)
			case float32:
				sec = int64(v)
			default:
				return "0:00"
			}
			
			hours := sec / 3600
			minutes := (sec % 3600) / 60
			
			return fmt.Sprintf("%d:%02d", hours, minutes)
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"formatInt": func(value interface{}) string {
			// Format integer values to avoid scientific notation in templates
			switch v := value.(type) {
			case int:
				return fmt.Sprintf("%d", v)
			case int64:
				return fmt.Sprintf("%d", v)
			case float64:
				return fmt.Sprintf("%.0f", v)
			case float32:
				return fmt.Sprintf("%.0f", v)
			default:
				return fmt.Sprintf("%v", value)
			}
		},
	}

	for _, dir := range templateDirs {
		// Parse base.html first, then other templates
		templates = template.New("base.html").Funcs(funcMap)
		_, err = templates.ParseGlob(dir + "/*.html")
		if err == nil && templates != nil {
			break
		}
	}

	if templates == nil || err != nil {
		// If no templates found, create empty template set
		// This allows the server to start even without templates (fallback HTML will be used)
		templates = template.New("base.html").Funcs(funcMap)
	}

	// Get Limitless API key from environment (single-user mode)
	limitlessAPIKey := os.Getenv("LIMITLESS_API_KEY")

	return &WebServer{
		apiURL:         apiURL,
		apiClient:      &http.Client{},
		templates:      templates,
		limitlessAPIKey: limitlessAPIKey,
	}, nil
}

// renderTemplate renders a template with the given data
func (s *WebServer) renderTemplate(w http.ResponseWriter, name string, data interface{}) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	
	// Execute the specific template directly
	// Each page template is now self-contained with full HTML structure
	err := s.templates.ExecuteTemplate(w, name, data)
	return err
}

