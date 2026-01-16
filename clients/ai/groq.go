package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	GroqAPIURL = "https://api.groq.com/openai/v1/chat/completions"
	// Доступные модели на Groq (январь 2026):
	// TEXT TO TEXT:
	// - "llama-3.3-70b-versatile" - Llama 3.3 70B
	// - "llama4-scout-17b-16e-instruct" - Llama 4 Scout
	// - "qwen-qwq-32b" - Qwen 3 32B
	// - "gpt-oss-120b" - GPT OSS 120B
	// - "kimi-k2" - Kimi K2
	DefaultModel  = "llama-3.3-70b-versatile"
	FallbackModel = "llama4-scout-17b-16e-instruct"
)

// Client - клиент для работы с Groq API
type Client struct {
	apiKey     string
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
}

// ChatResponse - ответ от API
type ChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// NewClient создаёт новый клиент Groq
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		model: DefaultModel,
	}
}

// SetModel устанавливает модель
func (c *Client) SetModel(model string) {
	c.model = model
}

// Chat отправляет сообщение и получает ответ
func (c *Client) Chat(messages []Message, temperature float64) (string, error) {
	// Пробуем основную модель, при ошибке - fallback
	models := []string{c.model}
	if c.model != FallbackModel {
		models = append(models, FallbackModel)
	}

	var lastErr error
	for _, model := range models {
		result, err := c.chatWithModel(messages, temperature, model)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	return "", lastErr
}

// chatWithModel выполняет запрос к конкретной модели
func (c *Client) chatWithModel(messages []Message, temperature float64, model string) (string, error) {
	req := ChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   4096,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("ошибка сериализации: %w", err)
	}

	httpReq, err := http.NewRequest("POST", GroqAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("ошибка запроса: %w", err)
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
		return "", fmt.Errorf("ошибка API: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("пустой ответ от API")
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
