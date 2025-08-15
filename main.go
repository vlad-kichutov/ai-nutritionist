package main

import (
	"context"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	openai "github.com/sashabaranov/go-openai"
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

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	openaiClient := openai.NewClient(openaiAPIKey)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
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

		if _, err := bot.Send(msg); err != nil {
			log.Println(err)
		}
	}
}
