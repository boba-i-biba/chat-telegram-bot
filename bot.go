package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot todo: need to abstract types
type Bot interface {
	HandleUpdate(update *tgbotapi.Update)
	Updates() *tgbotapi.UpdatesChannel
	ReplyToMessage(chatId int64, replyToMessageId int, text string)
	Send(chatId int64, text string)
}

type TelegramBot struct {
	instance *tgbotapi.BotAPI `json:"instance"`
	ps       *PubSub          `json:"ps"`
}

type MessagePayload struct {
	UserId    int64    `json:"userId"`
	ChatId    int64    `json:"chatId"`
	MessageId int      `json:"messageId"`
	Text      string   `json:"text"`
	Types     []string `json:"types"`
	Timestamp int64    `json:"timestamp"`
}

type MessageSend struct {
	ChatId           int64  `json:"chatId"`
	ReplyToMessageId int    `json:"replyToMessageId"`
	Text             string `json:"text"`
}

func (msg *MessageSend) Validate() error {
	var validationErrors []string

	if msg.ChatId == 0 {
		validationErrors = append(validationErrors, "no chat id")
	}
	if msg.Text == "" {
		validationErrors = append(validationErrors, "no text")
	}

	if validationErrors != nil {
		return errors.New(strings.Join(validationErrors, ", "))
	}

	return nil
}

func (msg *MessageSend) PopulateFromJson(str string) error {
	err := json.Unmarshal([]byte(str), msg)
	if err != nil {
		return err
	}
	err = msg.Validate()
	if err != nil {
		return err
	}
	return nil
}

func NewTelegramBot(token string, ps *PubSub) (*TelegramBot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)
	bot.Debug = true
	return &TelegramBot{
		instance: bot,
		ps:       ps,
	}, nil
}

func (bot *TelegramBot) Send(chatId int64, text string) {
	message := tgbotapi.NewMessage(chatId, text)
	_, err := bot.instance.Send(message)
	if err != nil {
		log.Println("Error when sending a message: ", err)
	}
}

func (bot *TelegramBot) ReplyToMessage(chatId int64, replyToMessageId int, text string) {
	message := tgbotapi.NewMessage(chatId, text)
	message.ReplyToMessageID = replyToMessageId
	_, err := bot.instance.Send(message)
	if err != nil {
		log.Println("Error when sending a reply: ", err)
	}
}

func (bot *TelegramBot) Updates() *tgbotapi.UpdatesChannel {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.instance.GetUpdatesChan(u)
	return &updates
}

func (bot *TelegramBot) HandleUpdate(update *tgbotapi.Update) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	message, err := GetUpdateMessage(update)
	if err != nil {
		log.Println(err)
		return
	}

	msgTypes := MapMessageTypes(GetMessageTypes(message))
	payload, err := PrepareMessagePayload(
		message.From.ID,
		message.Chat.ID,
		message.MessageID,
		message.Text,
		msgTypes,
		time.Now().Unix(),
	)
	if err != nil {
		fmt.Printf("error when preparing text payload:\n%+v\n", err)
		return
	}
	err = (*bot.ps).Publish(
		ctx,
		"telegram-message",
		payload,
	)
	if err != nil {
		log.Println("Unable to publish to 'telegram-message' topic")
	}
}

func GetMessageTypes(message *tgbotapi.Message) *[]string {
	var types []string
	if message.EditDate != 0 {
		types = append(types, "edited")

		return &types
	}
	fieldTypes := map[bool]string{
		message.Animation != nil:       "animation",
		message.Text != "":             "text",
		message.Audio != nil:           "audio",
		message.Photo != nil:           "photo",
		message.Sticker != nil:         "sticker",
		message.Video != nil:           "video",
		message.Voice != nil:           "voice",
		message.Document != nil:        "file",
		message.VideoNote != nil:       "videoNote",
		message.ForwardFrom != nil:     "forward",
		message.ForwardFromChat != nil: "repost",
	}
	for exists, value := range fieldTypes {
		if exists == true {
			types = append(types, value)
		}
	}
	return &types
}

func MapMessageTypes(types *[]string) []string {
	typesMap := map[string]string{
		"video":     "media",
		"photo":     "media",
		"file":      "media",
		"text":      "text",
		"animation": "reaction",
		"sticker":   "reaction",
		"voice":     "social_whore",
		"videoNote": "social_whore",
		"repost":    "content",
		"edited":    "grammar_nazi",
		"forward":   "proof_checker",
	}
	uniqueMap := make(map[string]bool)
	var mappedTypes []string
	for _, typeName := range *types {
		if _, exists := uniqueMap[typeName]; !exists {
			uniqueMap[typeName] = true
			mappedTypes = append(mappedTypes, typesMap[typeName])
		}
	}
	return mappedTypes
}

func GetUpdateMessage(update *tgbotapi.Update) (*tgbotapi.Message, error) {
	if update.Message != nil {
		return update.Message, nil
	}
	if update.EditedMessage != nil {
		return update.EditedMessage, nil
	}
	return nil, errors.New("no message")
}

func PrepareMessagePayload(
	userId int64,
	chatId int64,
	messageId int,
	text string,
	types []string,
	timestamp int64,
) (string, error) {
	payload := &MessagePayload{
		UserId:    userId,
		ChatId:    chatId,
		MessageId: messageId,
		Text:      text,
		Types:     types,
		Timestamp: timestamp,
	}
	result, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("unable to convert message payload to json \n %+v \n", err)
		return "", err
	}
	return string(result), nil
}
