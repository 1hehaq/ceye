package main

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jmoiron/jsonq"
)

const (
	pingInterval   = 30 * time.Second
	pongWait       = 60 * time.Second
	writeWait      = 10 * time.Second
	reconnectDelay = 5 * time.Second
	maxReconnect   = 10
)

type CertstreamClient struct {
	url        string
	conn       *websocket.Conn
	mu         sync.Mutex
	done       chan struct{}
	eventChan  chan jsonq.JsonQuery
	errorChan  chan error
	reconnects int
}

func NewCertstreamClient(url string) *CertstreamClient {
	return &CertstreamClient{
		url:       url,
		eventChan: make(chan jsonq.JsonQuery, 5000),
		errorChan: make(chan error, 100),
		done:      make(chan struct{}),
	}
}

func (c *CertstreamClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
	}

	conn, _, err := dialer.Dial(c.url, nil)
	if err != nil {
		return err
	}

	c.conn = conn
	c.reconnects = 0
	return nil
}

func (c *CertstreamClient) Start() (chan jsonq.JsonQuery, chan error) {
	go c.run()
	return c.eventChan, c.errorChan
}

func (c *CertstreamClient) Stop() {
	close(c.done)
	c.mu.Lock()
	if c.conn != nil {
		c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()
}

func (c *CertstreamClient) run() {
	for {
		select {
		case <-c.done:
			return
		default:
		}

		if err := c.Connect(); err != nil {
			c.reconnects++
			if c.reconnects > maxReconnect {
				c.errorChan <- err
				c.reconnects = 0
			}
			time.Sleep(reconnectDelay)
			continue
		}

		c.readLoop()

		select {
		case <-c.done:
			return
		default:
			time.Sleep(reconnectDelay)
		}
	}
}

func (c *CertstreamClient) readLoop() {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return
	}

	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	pingTicker := time.NewTicker(pingInterval)
	pingDone := make(chan struct{})

	go func() {
		defer pingTicker.Stop()
		for {
			select {
			case <-pingTicker.C:
				c.mu.Lock()
				if c.conn != nil {
					c.conn.SetWriteDeadline(time.Now().Add(writeWait))
					if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
						c.mu.Unlock()
						return
					}
				}
				c.mu.Unlock()
			case <-pingDone:
				return
			case <-c.done:
				return
			}
		}
	}()

	defer func() {
		close(pingDone)
		c.mu.Lock()
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		c.mu.Unlock()
	}()

	for {
		select {
		case <-c.done:
			return
		default:
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			c.errorChan <- err
			return
		}

		var data map[string]interface{}
		if err := json.Unmarshal(message, &data); err != nil {
			continue
		}

		jq := jsonq.NewQuery(data)

		select {
		case c.eventChan <- *jq:
		default:
		}
	}
}

func CertStreamEventStream(url string) (chan jsonq.JsonQuery, chan error) {
	client := NewCertstreamClient(url)
	return client.Start()
}
