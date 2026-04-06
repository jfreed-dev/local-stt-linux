package stt

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"nhooyr.io/websocket"
)

// StreamEvent is sent by the mode manager to control audio streaming.
type StreamEvent struct {
	Type string // "start", "chunk", "end"
	Data []byte // PCM data for "chunk" events
}

// Client connects to the Aria STT server and manages the firmware WebSocket protocol.
type Client struct {
	url       string
	insecure  bool
	streamCh  <-chan StreamEvent
	partialCh chan<- string
	finalCh   chan<- string
	turnID    atomic.Int64
}

func NewClient(url string, insecure bool, streamCh <-chan StreamEvent, partialCh, finalCh chan<- string) *Client {
	return &Client{
		url:       url,
		insecure:  insecure,
		streamCh:  streamCh,
		partialCh: partialCh,
		finalCh:   finalCh,
	}
}

// Run connects to the server and processes events until ctx is cancelled.
// Reconnects automatically on disconnection.
func (c *Client) Run(ctx context.Context) error {
	backoff := time.Second
	for {
		err := c.session(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		log.Printf("stt: disconnected: %v, reconnecting in %v", err, backoff)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, 30*time.Second)
	}
}

func (c *Client) session(ctx context.Context) error {
	opts := &websocket.DialOptions{
		HTTPHeader: nil,
	}
	if c.insecure {
		// For insecure TLS, the URL should use ws:// not wss://
		// If wss:// is needed with self-signed certs, a custom dialer would be required
	}

	hostname, _ := os.Hostname()
	sessionID := fmt.Sprintf("linux-stt-%s", hostname)
	dialURL := fmt.Sprintf("%s?session_id=%s", c.url, sessionID)

	conn, _, err := websocket.Dial(ctx, dialURL, opts)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.CloseNow()

	conn.SetReadLimit(512 * 1024) // 512KB

	// Send hello
	hello := map[string]interface{}{
		"type":        "hello",
		"device":      "linux-stt",
		"device_name": "linux-dictation",
		"version":     "0.1.0",
	}
	helloJSON, _ := json.Marshal(hello)
	if err := conn.Write(ctx, websocket.MessageText, helloJSON); err != nil {
		return fmt.Errorf("send hello: %w", err)
	}
	log.Printf("stt: connected to %s (session=%s)", c.url, sessionID)

	// Read messages in a separate goroutine
	readCtx, readCancel := context.WithCancel(ctx)
	defer readCancel()

	go c.readLoop(readCtx, conn)

	// Process stream events
	var totalBytes int
	for {
		select {
		case <-ctx.Done():
			conn.Close(websocket.StatusNormalClosure, "shutdown")
			return ctx.Err()
		case evt, ok := <-c.streamCh:
			if !ok {
				return nil
			}
			switch evt.Type {
			case "start":
				turnID := c.turnID.Add(1)
				msg := map[string]interface{}{
					"type":    "audio_stream_start",
					"turn_id": turnID,
				}
				data, _ := json.Marshal(msg)
				if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
					return fmt.Errorf("send stream_start: %w", err)
				}
				totalBytes = 0
				log.Printf("stt: stream start (turn=%d)", turnID)

			case "chunk":
				if err := conn.Write(ctx, websocket.MessageBinary, evt.Data); err != nil {
					return fmt.Errorf("send chunk: %w", err)
				}
				totalBytes += len(evt.Data)

			case "end":
				turnID := c.turnID.Load()
				msg := map[string]interface{}{
					"type":        "audio_stream_end",
					"turn_id":     turnID,
					"total_bytes": totalBytes,
				}
				data, _ := json.Marshal(msg)
				if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
					return fmt.Errorf("send stream_end: %w", err)
				}
				log.Printf("stt: stream end (turn=%d, %d bytes)", turnID, totalBytes)
			}
		}
	}
}

func (c *Client) readLoop(ctx context.Context, conn *websocket.Conn) {
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("stt: read error: %v", err)
			return
		}

		var msg struct {
			Type       string `json:"type"`
			Text       string `json:"text"`
			Transcript string `json:"transcript"`
			IsFinal    bool   `json:"is_final"`
		}
		if err := json.Unmarshal(data, &msg); err != nil {
			continue // ignore binary frames (TTS audio) and malformed JSON
		}

		switch msg.Type {
		case "transcript_partial":
			select {
			case c.partialCh <- msg.Text:
			default:
			}
		case "transcript_final":
			if msg.Text != "" {
				select {
				case c.finalCh <- msg.Text:
				default:
				}
			}
		case "stt_result":
			// Batch STT result from server — uses "transcript" field
			if msg.Transcript != "" {
				select {
				case c.finalCh <- msg.Transcript:
				default:
				}
			}
		case "connected":
			log.Printf("stt: server confirmed connection")
		case "error":
			log.Printf("stt: server error: %s", msg.Text)
		}
		// Ignore all other message types (LLM responses, TTS, etc.)
	}
}
