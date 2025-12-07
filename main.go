package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

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

var client *crisp.Client

func main() {
	client = crisp.New()
	client.AuthenticateTier("plugin", os.Getenv("CRISP_IDENTIFIER"), os.Getenv("CRISP_KEY"))

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
