package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

var (
	telegramBotToken string
	redisURL         string
)

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Printf("%+v", err)
		panic("unable to load env variables")
	}

	telegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")

	if telegramBotToken == "" {
		panic("telegram bot token is not provided")
	}

	redisURL = os.Getenv("REDIS_URL")

	if redisURL == "" {
		panic("redis url is not provided")
	}
}

func main() {
	var bot Bot
	var ps PubSub
	ps, err := NewRedisPubSub(redisURL)
	if err != nil {
		log.Fatal(err)
	}
	bot, err = NewTelegramBot(telegramBotToken, &ps)
	if err != nil {
		log.Fatal(err)
	}
	updates := bot.Updates()
	ctx := context.Background()
	msgSendS := ps.Subscribe(ctx, "telegram-send")
	for {
		select {
		case update := <-*updates:
			go bot.HandleUpdate(&update)
		case msg := <-msgSendS:
			var message MessageSend
			err := message.PopulateFromJson(msg)
			if err != nil {
				fmt.Println("Unable to get message from 'telegram-send' topic", err)
				continue
			}
			if message.ReplyToMessageId != 0 {
				bot.ReplyToMessage(message.ChatId, message.ReplyToMessageId, message.Text)
				continue
			}
			bot.Send(message.ChatId, message.Text)
		}
	}
}
