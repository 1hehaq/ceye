package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

type update struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
}

type getUpdatesResponse struct {
	OK     bool     `json:"ok"`
	Result []update `json:"result"`
}

var logger = log.NewWithOptions(os.Stdout, log.Options{ReportTimestamp: true, TimeFormat: "15:04:05"})

func main() {
	botToken := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if botToken == "" {
		logger.Fatal("TELEGRAM_BOT_TOKEN is not set")
	}

	logger.Info("starting ceye telegram host bot")

	offset := 0
	for {
		updates, err := fetchUpdates(botToken, offset)
		if err != nil {
			logger.Error("failed to fetch updates", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, u := range updates {
			if u.UpdateID >= offset {
				offset = u.UpdateID + 1
			}
			handleUpdate(botToken, u)
		}
	}
}

func fetchUpdates(token string, offset int) ([]update, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?timeout=60&offset=%d", token, offset)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res getUpdatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	if !res.OK {
		return nil, fmt.Errorf("telegram getUpdates not ok")
	}

	return res.Result, nil
}

func handleUpdate(token string, u update) {
	chatID := u.Message.Chat.ID
	text := strings.TrimSpace(u.Message.Text)
	if text == "" {
		return
	}

	switch text {
	case "/start", "start", "Start":
		message := "*Yo buddy, let's do it!*" +
			"\n\n" +
			"● Get your _ChatID_ first."
		sendWithKeyboard(token, chatID, message, [][]string{{"chat ID"}, {"credit"}, {"flags"}, {"help"}})
	case "chat ID":
		message := fmt.Sprintf("Your _ChatID_ is: `%d`\n\n● Add this to your ceye config as `telegram_chat_id`.", chatID)
		sendWithKeyboard(token, chatID, message, [][]string{{"chat ID"}, {"credit"}, {"flags"}, {"help"}})
	case "credit":
		message := fmt.Sprintf("© ceye is made by %s", "[haq](https://github.com/1hehaq)") +
			"\n\n[⭐](https://github.com/1hehaq/ceye) the repo if you like it!" +
			"\n\n_made with <3 kindly for hackers_"
		sendWithKeyboard(token, chatID, message, [][]string{{"chat ID"}, {"credit"}, {"flags"}, {"help"}})
	case "help":
		message := "*not getting updates from ceye?*" +
			"\n\n1. copy your chat id from the *chat ID* button." +
			"\n2. in your ceye config YAML, set `telegram_bot_token` to this bot's token and `telegram_chat_id` to your id." +
			"\n3. run ceye with `-notify=telegram` or `-notify=both`."
		sendWithKeyboard(token, chatID, message, [][]string{{"chat ID"}, {"credit"}, {"flags"}, {"help"}})
	case "flags":
		sendPhoto(token, chatID, "https://private-user-images.githubusercontent.com/162917546/520497770-a2302e5a-9024-48f7-935c-2e8465dd6aa0.png?jwt=eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJnaXRodWIuY29tIiwiYXVkIjoicmF3LmdpdGh1YnVzZXJjb250ZW50LmNvbSIsImtleSI6ImtleTUiLCJleHAiOjE3NjU3NDY5ODIsIm5iZiI6MTc2NTc0NjY4MiwicGF0aCI6Ii8xNjI5MTc1NDYvNTIwNDk3NzcwLWEyMzAyZTVhLTkwMjQtNDhmNy05MzVjLTJlODQ2NWRkNmFhMC5wbmc_WC1BbXotQWxnb3JpdGhtPUFXUzQtSE1BQy1TSEEyNTYmWC1BbXotQ3JlZGVudGlhbD1BS0lBVkNPRFlMU0E1M1BRSzRaQSUyRjIwMjUxMjE0JTJGdXMtZWFzdC0xJTJGczMlMkZhd3M0X3JlcXVlc3QmWC1BbXotRGF0ZT0yMDI1MTIxNFQyMTExMjJaJlgtQW16LUV4cGlyZXM9MzAwJlgtQW16LVNpZ25hdHVyZT00OTk2MGViYzcyMDI1MTYwMDM2NTM0NjNhMTQ2YmM5ZjQ3ZTRkYjAxMWRhNzA4M2M1YjQ1MDU2NzIzODcyMzNmJlgtQW16LVNpZ25lZEhlYWRlcnM9aG9zdCJ9.e0fVga0xBkU6D2r-zpOvHKo_5aw0DRlMOe2N-5xj8tk")
	default:
		message := "*request not recognized.*"
		sendWithKeyboard(token, chatID, message, [][]string{{"chat ID"}, {"credit"}, {"flags"}, {"help"}})
	}
}

func sendWithKeyboard(token string, chatID int64, text string, keyboard [][]string) {
	var rows [][]map[string]string
	for _, row := range keyboard {
		var btnRow []map[string]string
		for _, label := range row {
			btnRow = append(btnRow, map[string]string{"text": label})
		}
		rows = append(rows, btnRow)
	}

	replyMarkup := map[string]interface{}{
		"keyboard":          rows,
		"resize_keyboard":   true,
		"one_time_keyboard": false,
	}

	payload := map[string]interface{}{
		"chat_id":      chatID,
		"text":         text,
		"parse_mode":   "Markdown",
		"reply_markup": replyMarkup,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		logger.Error("failed to marshal sendMessage payload", "error", err)
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("failed to send message", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warn("telegram sendMessage returned non-200", "status", resp.StatusCode)
	}
}

func sendPhoto(token string, chatID int64, photoURL string) {
	payload := map[string]interface{}{
		"chat_id": chatID,
		"photo":   photoURL,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		logger.Error("failed to marshal sendPhoto payload", "error", err)
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", token)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("failed to send photo", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warn("telegram sendPhoto returned non-200", "status", resp.StatusCode)
	}
}
