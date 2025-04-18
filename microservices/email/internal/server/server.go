package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/microservices/email/internal/email"
	"github.com/emoss08/trenova/microservices/email/internal/server/templates"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// WebSocketClients tracks active WebSocket connections
type WebSocketClients struct {
	clients map[*websocket.Conn]bool
	mutex   sync.Mutex
}

// NewWebSocketClients creates a new WebSocketClients instance
func NewWebSocketClients() *WebSocketClients {
	return &WebSocketClients{
		clients: make(map[*websocket.Conn]bool),
	}
}

// Add adds a new WebSocket client
func (wsc *WebSocketClients) Add(conn *websocket.Conn) {
	wsc.mutex.Lock()
	defer wsc.mutex.Unlock()
	wsc.clients[conn] = true
}

// Remove removes a WebSocket client
func (wsc *WebSocketClients) Remove(conn *websocket.Conn) {
	wsc.mutex.Lock()
	defer wsc.mutex.Unlock()
	delete(wsc.clients, conn)
	conn.Close()
}

// Broadcast sends a message to all connected clients
func (wsc *WebSocketClients) Broadcast(message []byte) {
	wsc.mutex.Lock()
	defer wsc.mutex.Unlock()

	for client := range wsc.clients {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Error().Err(err).Msg("Error broadcasting to WebSocket client")
			client.Close()
			delete(wsc.clients, client)
		}
	}
}

// Server represents the HTTP server for template management
type Server struct {
	router          *chi.Mux
	addr            string
	templateService *email.TemplateService
	templatesDir    string
	samplesDir      string
	wsClients       *WebSocketClients
	wsUpgrader      websocket.Upgrader
}

// NewServer creates a new HTTP server for template management
func NewServer(addr string, templateService *email.TemplateService, templatesDir string) *Server {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Create the samples directory if it doesn't exist
	samplesDir := "data/samples"
	if _, err := os.Stat(samplesDir); os.IsNotExist(err) {
		if err := os.MkdirAll(samplesDir, 0755); err != nil {
			log.Error().Err(err).Str("path", samplesDir).Msg("Failed to create samples directory")
		}
	}

	server := &Server{
		router:          r,
		addr:            addr,
		templateService: templateService,
		templatesDir:    templatesDir,
		samplesDir:      samplesDir,
		wsClients:       NewWebSocketClients(),
		wsUpgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Allow all origins for development
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}

	// Register routes
	server.registerRoutes()

	return server
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Info().Str("addr", s.addr).Msg("Starting template management server (DEV MODE ONLY)")
	return http.ListenAndServe(s.addr, s.router)
}

// registerRoutes registers all routes for the server
func (s *Server) registerRoutes() {
	// Serve static assets
	assetsDir := http.Dir("internal/server/assets")
	s.router.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(assetsDir)))

	// API routes
	s.router.Get("/", s.handleIndex)
	s.router.Route("/api/templates", func(r chi.Router) {
		r.Get("/", s.handleListTemplates)
		r.Get("/{name}", s.handleGetTemplate)
		r.Put("/{name}", s.handleUpdateTemplate)
		r.Post("/preview/{name}", s.handlePreviewTemplate)
	})

	// Sample data management API
	s.router.Route("/api/samples", func(r chi.Router) {
		r.Get("/", s.handleListSamples)
		r.Get("/{name}", s.handleGetSample)
		r.Put("/{name}", s.handleUpdateSample)
	})

	// WebSocket endpoint for live updates
	s.router.Get("/ws", s.handleWebSocket)
}

// handleIndex handles the root path
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Render the template manager using templates.html
	component := templates.TemplateManager()
	err := component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render template: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleListTemplates handles the template listing endpoint
func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := listTemplateFiles(s.templatesDir)
	if err != nil {
		http.Error(w, "Failed to list templates: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

// handleGetTemplate handles getting a template's content
func (s *Server) handleGetTemplate(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		http.Error(w, "Template name is required", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(s.templatesDir, name+".html")
	content, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "Failed to read template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write(content)
}

// handleUpdateTemplate handles updating a template's content
func (s *Server) handleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		http.Error(w, "Template name is required", http.StatusBadRequest)
		return
	}

	content, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filePath := filepath.Join(s.templatesDir, name+".html")
	err = os.WriteFile(filePath, content, 0600)
	if err != nil {
		http.Error(w, "Failed to write template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Clear template cache to reload the updated template
	s.templateService.ClearTemplateCache(name)

	// Notify all connected clients about the update
	updateMsg := struct {
		Type         string `json:"type"`
		TemplateName string `json:"templateName"`
	}{
		Type:         "template_updated",
		TemplateName: name,
	}

	msgBytes, _ := json.Marshal(updateMsg)
	s.wsClients.Broadcast(msgBytes)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Template %s updated successfully", name)
}

// handlePreviewTemplate handles the template preview endpoint
func (s *Server) handlePreviewTemplate(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		http.Error(w, "Template name is required", http.StatusBadRequest)
		return
	}

	// Read the template content from the request body
	content, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create sample data for the preview based on template name
	data := s.loadSampleData(name)

	// Render the template with sample data
	renderedHTML, err := s.templateService.RenderInlineTemplate(string(content), data)
	if err != nil {
		http.Error(w, "Failed to render template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(renderedHTML))
}

// handleListSamples handles listing all sample data files
func (s *Server) handleListSamples(w http.ResponseWriter, r *http.Request) {
	samples, err := s.listSampleFiles()
	if err != nil {
		http.Error(w, "Failed to list samples: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(samples)
}

// handleGetSample handles retrieving a sample data file
func (s *Server) handleGetSample(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		http.Error(w, "Sample name is required", http.StatusBadRequest)
		return
	}

	data, err := s.loadSampleDataRaw(name)
	if err != nil {
		http.Error(w, "Failed to read sample: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// handleUpdateSample handles updating a sample data file
func (s *Server) handleUpdateSample(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		http.Error(w, "Sample name is required", http.StatusBadRequest)
		return
	}

	// Read the sample data from the request body
	content, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Validate JSON
	var testData map[string]any
	if err := json.Unmarshal(content, &testData); err != nil {
		http.Error(w, "Invalid JSON format: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Ensure the sample name has .json extension
	if !strings.HasSuffix(name, ".json") {
		name += ".json"
	}

	// Save the sample data
	filePath := filepath.Join(s.samplesDir, name)
	err = os.WriteFile(filePath, content, 0600)
	if err != nil {
		http.Error(w, "Failed to write sample: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Notify all connected clients about the update
	sampleName := strings.TrimSuffix(name, ".json")
	updateMsg := struct {
		Type       string `json:"type"`
		SampleName string `json:"sampleName"`
	}{
		Type:       "sample_updated",
		SampleName: sampleName,
	}

	msgBytes, _ := json.Marshal(updateMsg)
	s.wsClients.Broadcast(msgBytes)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Sample %s updated successfully", name)
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade connection to WebSocket")
		return
	}

	// Add client to our connections list
	s.wsClients.Add(conn)

	// Handle disconnect
	go func() {
		defer s.wsClients.Remove(conn)

		for {
			// Read message (only to detect disconnection)
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

// Helper functions

// listTemplateFiles returns a list of template names (without extension)
func listTemplateFiles(templatesDir string) ([]string, error) {
	files, err := os.ReadDir(templatesDir)
	if err != nil {
		return nil, err
	}

	var templates []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".html") {
			name := strings.TrimSuffix(file.Name(), ".html")
			templates = append(templates, name)
		}
	}

	return templates, nil
}

// loadSampleData loads sample data for a template
func (s *Server) loadSampleData(templateName string) map[string]any {
	// First try to load template-specific sample
	data, err := s.loadSampleDataMap(templateName)
	if err == nil {
		return data
	}

	// Fall back to default sample
	defaultData, err := s.loadSampleDataMap("default")
	if err == nil {
		return defaultData
	}

	// Last resort: return basic data
	return map[string]any{
		"Year":     time.Now().Year(),
		"Name":     "John Doe",
		"Username": "johndoe",
		"Email":    "john.doe@example.com",
	}
}

// loadSampleDataMap loads and unmarshals a sample data file
func (s *Server) loadSampleDataMap(name string) (map[string]any, error) {
	data, err := s.loadSampleDataRaw(name)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// loadSampleDataRaw loads a sample data file as raw bytes
func (s *Server) loadSampleDataRaw(name string) ([]byte, error) {
	// Ensure the sample name has .json extension
	if !strings.HasSuffix(name, ".json") {
		name += ".json"
	}

	filePath := filepath.Join(s.samplesDir, name)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// listSampleFiles returns a list of sample data file names (without extension)
func (s *Server) listSampleFiles() ([]string, error) {
	files, err := os.ReadDir(s.samplesDir)
	if err != nil {
		return nil, err
	}

	var samples []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			name := strings.TrimSuffix(file.Name(), ".json")
			samples = append(samples, name)
		}
	}

	return samples, nil
}
