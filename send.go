package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	batchDelay    = 5 * time.Second
	maxBatchSize  = 25
	rateLimitWait = 2 * time.Second
	maxRetries    = 3
)

type notificationBuffer struct {
	mu      sync.Mutex
	pending map[string][]string
	timers  map[string]*time.Timer
}

var notifier = &notificationBuffer{
	pending: make(map[string][]string),
	timers:  make(map[string]*time.Timer),
}

func sendToDiscord(domain, target string) {
	notifier.add(target, domain)
}

func (n *notificationBuffer) add(target, domain string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.pending[target] = append(n.pending[target], domain)

	if len(n.pending[target]) >= maxBatchSize {
		domains := n.pending[target]
		delete(n.pending, target)
		if timer, exists := n.timers[target]; exists {
			timer.Stop()
			delete(n.timers, target)
		}
		go n.send(target, domains)
		return
	}

	if _, exists := n.timers[target]; !exists {
		n.timers[target] = time.AfterFunc(batchDelay, func() {
			n.flush(target)
		})
	}
}

func (n *notificationBuffer) flush(target string) {
	n.mu.Lock()
	domains, exists := n.pending[target]
	if !exists || len(domains) == 0 {
		n.mu.Unlock()
		return
	}
	delete(n.pending, target)
	delete(n.timers, target)
	n.mu.Unlock()

	n.send(target, domains)
}

func (n *notificationBuffer) send(target string, domains []string) {
	if notifyDiscord && webhookURL != "" {
		payload := buildDiscordPayload(target, domains)

		jsonData, err := json.Marshal(payload)
		if err != nil {
			logger.Error("failed to marshal payload", "error", err)
		} else {
			for attempt := 0; attempt < maxRetries; attempt++ {
				resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
				if err != nil {
					logger.Error("failed to send notification", "error", err)
					return
				}

				switch resp.StatusCode {
				case http.StatusOK, http.StatusNoContent:
					resp.Body.Close()
					return
				case http.StatusTooManyRequests:
					resp.Body.Close()
					logger.Warn("rate limited, waiting", "attempt", attempt+1)
					time.Sleep(rateLimitWait * time.Duration(attempt+1))
					continue
				default:
					resp.Body.Close()
					logger.Warn("webhook error", "status", resp.StatusCode)
					return
				}
			}

			logger.Error("failed to send after retries", "target", target)
		}
	}

	if notifyTelegram {
		sendToTelegram(target, domains)
	}
}

func sendToTelegram(target string, domains []string) {
	if telegramToken == "" || telegramChatID == "" {
		return
	}

	text := buildTelegramMessage(target, domains)

	payload := map[string]interface{}{
		"chat_id":                  telegramChatID,
		"text":                     text,
		"parse_mode":               "Markdown",
		"disable_web_page_preview": true,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		logger.Error("failed to marshal telegram payload", "error", err)
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegramToken)

	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			logger.Error("failed to send telegram notification", "error", err)
			return
		}

		if resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			logger.Warn("telegram rate limited, waiting", "attempt", attempt+1)
			time.Sleep(rateLimitWait * time.Duration(attempt+1))
			continue
		}

		resp.Body.Close()
		logger.Warn("telegram send error", "status", resp.StatusCode)
		return
	}

	logger.Error("failed to send telegram after retries", "target", target)
}
