package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ============================================
// AI Client - работает с Ollama
// Ollama имеет OpenAI-совместимый API
// ============================================

const (
	DefaultOllamaURL   = "http://localhost:11434"
	DefaultOllamaModel = "gemma2:9b-instruct-q4_K_M"
)

// Client - клиент для работы с Ollama API
type Client struct {
	baseURL    string
	httpClient *http.Client
	model      string
}

// Message - сообщение для чата
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest - запрос к API
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream"`
}

// ChatResponse - ответ от API
type ChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// NewClient создаёт новый клиент Ollama
// apiKey игнорируется (для совместимости с существующим кодом)
func NewClient(apiKey string) *Client {
	return &Client{
		baseURL: DefaultOllamaURL,
		httpClient: &http.Client{
			Timeout: 600 * time.Second, // 10 минут для больших программ
		},
		model: DefaultOllamaModel,
	}
}

// NewClientWithURL создаёт клиент с указанным URL Ollama
func NewClientWithURL(ollamaURL, model string) *Client {
	if ollamaURL == "" {
		ollamaURL = DefaultOllamaURL
	}
	if model == "" {
		model = DefaultOllamaModel
	}
	return &Client{
		baseURL: ollamaURL,
		httpClient: &http.Client{
			Timeout: 600 * time.Second, // 10 минут для больших программ
		},
		model: model,
	}
}

// SetModel устанавливает модель
func (c *Client) SetModel(model string) {
	c.model = model
}

// SetBaseURL устанавливает URL Ollama
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

// IsAvailable проверяет доступность Ollama
func (c *Client) IsAvailable() bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(c.baseURL + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// Chat отправляет сообщение и получает ответ
func (c *Client) Chat(messages []Message, temperature float64) (string, error) {
	// Ollama OpenAI-совместимый endpoint
	url := c.baseURL + "/v1/chat/completions"

	req := ChatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   16384, // Увеличено для больших программ с периодизацией
		Stream:      false,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("ошибка сериализации: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("Ollama недоступен: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("ошибка Ollama: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("пустой ответ от Ollama")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// SimpleChat - простой запрос с одним сообщением
func (c *Client) SimpleChat(systemPrompt, userMessage string) (string, error) {
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}
	return c.Chat(messages, 0.7)
}
