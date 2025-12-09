package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/crisp-im/go-crisp-api/crisp/v3"
)

type CrispWebhookEvent struct {
	Event string `json:"event"`
	Data  struct {
		WebsiteID string `json:"website_id"`
		SessionID string `json:"session_id"`
		Content   string `json:"content"`
		From      string `json:"from"`
		Type      string `json:"type"`
	} `json:"data"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SupabaseAuthResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	User         struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

var client *crisp.Client
var supabaseURL string
var supabaseAnonKey string
var supabaseServiceKey string

func main() {
	// Initialize Crisp client
	client = crisp.New()
	client.AuthenticateTier("plugin", os.Getenv("CRISP_IDENTIFIER"), os.Getenv("CRISP_KEY"))

	// Initialize Supabase config
	supabaseURL = os.Getenv("SUPABASE_URL")
	supabaseAnonKey = os.Getenv("SUPABASE_ANON_KEY")
	supabaseServiceKey = os.Getenv("SUPABASE_SERVICE_ROLE_KEY")

	if supabaseURL == "" || supabaseServiceKey == "" {
		log.Println("Warning: SUPABASE_URL or SUPABASE_SERVICE_ROLE_KEY not set")
	}

	// Static file serving
	http.HandleFunc("/", serveIndex)
	http.HandleFunc("/login", serveLogin)
	http.HandleFunc("/knowledge", serveKnowledge)

	// API routes
	http.HandleFunc("/api/auth/login", corsMiddleware(handleLogin))
	http.HandleFunc("/api/knowledge", corsMiddleware(handleKnowledgeList))
	http.HandleFunc("/api/knowledge/upload", corsMiddleware(authMiddleware(handleKnowledgeUpload)))
	http.HandleFunc("/api/knowledge/", corsMiddleware(authMiddleware(handleKnowledgeDetail)))

	// Crisp webhook
	http.HandleFunc("/crisp/message", handleCrispWebhook)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// CORS middleware
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// Auth middleware - validates JWT token from Authorization header
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error": "Missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error": "Invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}

		// Pass the request to the next handler with the token in context
		// In a real implementation, you would validate the JWT here
		next(w, r)
	}
}

// Static file handlers
func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "index.html")
}

func serveLogin(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "login.html")
}

func serveKnowledge(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "knowledge.html")
}

// API Handlers

// Login handler - proxies to Supabase Auth
func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var loginReq LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Call Supabase Auth API
	authURL := fmt.Sprintf("%s/auth/v1/token?grant_type=password", supabaseURL)
	payload := map[string]string{
		"email":    loginReq.Email,
		"password": loginReq.Password,
	}

	jsonData, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(jsonData))
	if err != nil {
		http.Error(w, `{"error": "Failed to create request"}`, http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", supabaseAnonKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, `{"error": "Failed to connect to auth service"}`, http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

// Get knowledge list - proxies to Supabase
func handleKnowledgeList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get auth token from header
	authHeader := r.Header.Get("Authorization")
	token := ""
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 {
			token = parts[1]
		}
	}

	// Call Supabase REST API to get knowledge base entries
	apiURL := fmt.Sprintf("%s/rest/v1/knowledge_base?select=*&order=created_at.desc", supabaseURL)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		http.Error(w, `{"error": "Failed to create request"}`, http.StatusInternalServerError)
		return
	}

	req.Header.Set("apikey", supabaseServiceKey)
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Error calling Supabase: %v", err)
		http.Error(w, `{"error": "Failed to fetch knowledge base"}`, http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

// Upload knowledge - proxies to Supabase Edge Function
func handleKnowledgeUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get auth token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, `{"error": "Missing authorization"}`, http.StatusUnauthorized)
		return
	}

	// Read the entire request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error": "Failed to read request body"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Get the original Content-Type header (includes boundary)
	contentType := r.Header.Get("Content-Type")

	// Forward to Supabase
	apiURL := fmt.Sprintf("%s/functions/v1/knowledge-upload", supabaseURL)
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		http.Error(w, `{"error": "Failed to create request"}`, http.StatusInternalServerError)
		return
	}

	// Copy headers
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", contentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Error uploading to Supabase: %v", err)
		http.Error(w, `{"error": "Failed to upload"}`, http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

// Get/Delete single knowledge entry
func handleKnowledgeDetail(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/knowledge/")
	if path == "" {
		http.Error(w, `{"error": "Missing knowledge ID"}`, http.StatusBadRequest)
		return
	}

	authHeader := r.Header.Get("Authorization")

	var apiURL string
	var method string

	switch r.Method {
	case http.MethodGet:
		// Get single knowledge entry
		apiURL = fmt.Sprintf("%s/functions/v1/knowledge-base/%s", supabaseURL, path)
		method = "GET"
	case http.MethodDelete:
		// Delete knowledge entry
		apiURL = fmt.Sprintf("%s/functions/v1/knowledge-base/%s", supabaseURL, path)
		method = "DELETE"
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req, err := http.NewRequest(method, apiURL, nil)
	if err != nil {
		http.Error(w, `{"error": "Failed to create request"}`, http.StatusInternalServerError)
		return
	}

	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Error calling Supabase: %v", err)
		http.Error(w, `{"error": "Failed to process request"}`, http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

// Crisp webhook handler
func handleCrispWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading webhook body: %v", err)
		http.Error(w, "Error reading body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("Received webhook: %s", string(body))

	var event CrispWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf("Error parsing webhook JSON: %v", err)
		http.Error(w, "Error parsing JSON", http.StatusBadRequest)
		return
	}

	if event.Data.From == "user" && event.Data.Type == "text" {
		log.Printf("User message: %s (session: %s)", event.Data.Content, event.Data.SessionID)

		reply := generateAIReply(event.Data.Content)

		if err := sendCrispMessage(event.Data.WebsiteID, event.Data.SessionID, reply); err != nil {
			log.Printf("Error sending reply to Crisp: %v", err)
		} else {
			log.Printf("Reply sent successfully: %s", reply)
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func generateAIReply(userMessage string) string {
	return fmt.Sprintf("您说：%s\n\n这是AI自动回复。请替换此函数以集成您的AI模型。", userMessage)
}

func sendCrispMessage(websiteID, sessionID, message string) error {
	_, _, err := client.Website.SendTextMessageInConversation(websiteID, sessionID, crisp.ConversationTextMessageNew{
		Type:    "text",
		From:    "operator",
		Origin:  "chat",
		Content: message,
	})
	if err != nil {
		return fmt.Errorf("error sending message: %w", err)
	}

	return nil
}
