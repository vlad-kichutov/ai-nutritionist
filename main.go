package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	openai "github.com/sashabaranov/go-openai"
)

// Global variables for bot and OpenAI client
var (
	tgBot        *tgbotapi.BotAPI
	openaiClient *openai.Client
)

func main() {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable not set")
	}

	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	if openaiAPIKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	// Initialize Telegram Bot API
	var err error
	tgBot, err = tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	tgBot.Debug = true
	log.Printf("Authorized on account %s", tgBot.Self.UserName)

	// Initialize OpenAI client
	openaiClient = openai.NewClient(openaiAPIKey)

	// Get the port from the environment variable (Cloud Run provides this)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port if not specified
	}

	// Register webhook handler
	http.HandleFunc("/", handleWebhook)

	log.Printf("Listening on port %s", port)
	// Start the HTTP server
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// handleWebhook processes incoming Telegram updates via HTTP POST requests
func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are accepted", http.StatusMethodNotAllowed)
		return
	}

	// Decode the incoming Telegram update
	var update tgbotapi.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		log.Printf("Failed to decode update: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Process the message
	if update.Message == nil {
		log.Println("Received non-message update, ignoring.")
		w.WriteHeader(http.StatusOK) // Always respond OK to Telegram
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "start":
			msg.Text = "Hello! I'm your AI nutritionist bot. How can I help you today?"
		case "help":
			msg.Text = "You can ask me questions about nutrition, meal plans, or anything related to healthy eating."
		default:
			msg.Text = "I don't know that command. Please ask me a question about nutrition or use /start or /help."
		}
	} else {
		// Send non-command messages to ChatGPT
		res, err := openaiClient.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model: openai.GPT3Dot5Turbo,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleSystem,
						Content: "You are an AI nutritionist. Provide helpful and accurate nutritional advice, meal ideas, and healthy eating tips. Be encouraging and supportive.",
					},
					{
						Role:    openai.ChatMessageRoleUser,
						Content: update.Message.Text,
					},
				},
			},
		)

		if err != nil {
			log.Printf("ChatCompletion error: %v\n", err)
			msg.Text = "Sorry, I couldn't get a response from ChatGPT." // Provide a user-friendly error message
		} else if len(res.Choices) > 0 {
			msg.Text = res.Choices[0].Message.Content
		} else {
			msg.Text = "No response from ChatGPT."
		}
	}

	// Send the response back to Telegram
	if _, err := tgBot.Send(msg); err != nil {
		log.Printf("Failed to send message to Telegram: %v", err)
	}

	// Respond to Telegram that the update was received successfully
	w.WriteHeader(http.StatusOK)
}
