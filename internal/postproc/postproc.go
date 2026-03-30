package postproc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const systemPrompt = `Fix only clear homophone errors and obvious grammar mistakes in the dictated text below. Do NOT change proper nouns, technical terms, or names. Do NOT add or remove words. Do NOT wrap in quotes. Return ONLY the corrected text, nothing else.`

// Processor corrects STT output using a local LLM.
type Processor struct {
	endpoint string
	model    string
	client   *http.Client
	enabled  bool
}

func NewProcessor(endpoint, model string, enabled bool) *Processor {
	return &Processor{
		endpoint: endpoint,
		model:    model,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		enabled: enabled,
	}
}

// Process sends the raw transcription to the LLM for correction.
// Returns the original text if the LLM is unavailable or disabled.
func (p *Processor) Process(ctx context.Context, text string) string {
	if !p.enabled || text == "" {
		return text
	}

	// Use independent timeout so parent context cancellation doesn't drop text
	queryCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	corrected, err := p.query(queryCtx, text)
	if err != nil {
		log.Printf("postproc: LLM error (using raw text): %v", err)
		return text
	}

	corrected = strings.TrimSpace(corrected)
	// Strip wrapping quotes the LLM may add
	if len(corrected) >= 2 && corrected[0] == '"' && corrected[len(corrected)-1] == '"' {
		corrected = corrected[1 : len(corrected)-1]
	}
	corrected = strings.TrimSpace(corrected)
	if corrected == "" {
		return text
	}

	if corrected != text {
		log.Printf("postproc: %q -> %q", text, corrected)
	}
	return corrected
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (p *Processor) query(ctx context.Context, text string) (string, error) {
	reqBody := chatRequest{
		Model: p.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: text},
		},
		MaxTokens:   256,
		Temperature: 0.1,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := strings.TrimRight(p.endpoint, "/") + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", err
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}
