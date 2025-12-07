package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

type CrispMessageRequest struct {
	Type    string `json:"type"`
	From    string `json:"from"`
	Origin  string `json:"origin"`
	Content string `json:"content"`
}

var (
	crispIdentifier = os.Getenv("CRISP_IDENTIFIER")
	crispKey        = os.Getenv("CRISP_KEY")
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/crisp/message", handleCrispWebhook)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

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
	if crispIdentifier == "" || crispKey == "" {
		return fmt.Errorf("Crisp API credentials not configured")
	}

	url := fmt.Sprintf("https://api.crisp.chat/v1/website/%s/conversation/%s/message", websiteID, sessionID)

	payload := CrispMessageRequest{
		Type:    "text",
		From:    "operator",
		Origin:  "chat",
		Content: message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.SetBasicAuth(crispIdentifier, crispKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Crisp-Tier", "plugin")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Crisp API error: %d - %s", resp.StatusCode, string(respBody))
	}

	return nil
}
