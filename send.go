package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
	if webhookURL == "" {
		return
	}

	domainList := strings.Join(domains, "\n")

	payload := map[string]interface{}{
		"tts": false,
		"embeds": []map[string]interface{}{
			{
				"title":       fmt.Sprintf("%s  [%d]", target, len(domains)),
				"description": fmt.Sprintf("```\n%s\n```", domainList),
				"color":       2829617,
				"author": map[string]string{
					"name": "1hehaq/ceye",
					"url":  "https://github.com/1hehaq/ceye",
				},
				"timestamp": time.Now().Format(time.RFC3339),
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		logger.Error("failed to marshal payload", "error", err)
		return
	}

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
