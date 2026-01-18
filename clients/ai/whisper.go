package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"
)

// ============================================
// Groq Whisper - только для транскрипции голоса
// ============================================

const (
	GroqWhisperAPIURL = "https://api.groq.com/openai/v1/audio/transcriptions"
	WhisperModel      = "whisper-large-v3-turbo"
)

// WhisperClient - клиент для транскрипции через Groq
type WhisperClient struct {
	apiKey     string
	httpClient *http.Client
}

// TranscribeResponse - ответ от Whisper API
type TranscribeResponse struct {
	Text  string `json:"text"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// NewWhisperClient создаёт клиент для транскрипции
func NewWhisperClient(apiKey string) *WhisperClient {
	return &WhisperClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// IsAvailable проверяет доступность (есть ли API ключ)
func (c *WhisperClient) IsAvailable() bool {
	return c.apiKey != ""
}

// TranscribeAudio транскрибирует аудио файл по URL
func (c *WhisperClient) TranscribeAudio(audioURL string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("GROQ_API_KEY не настроен")
	}

	// Скачиваем аудио файл
	resp, err := c.httpClient.Get(audioURL)
	if err != nil {
		return "", fmt.Errorf("ошибка скачивания аудио: %w", err)
	}
	defer resp.Body.Close()

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения аудио: %w", err)
	}

	// Создаём multipart form
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Добавляем файл
	filename := filepath.Base(audioURL)
	if filename == "" || filename == "." {
		filename = "audio.ogg"
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("ошибка создания form file: %w", err)
	}
	if _, err := part.Write(audioData); err != nil {
		return "", fmt.Errorf("ошибка записи аудио данных: %w", err)
	}

	// Добавляем модель
	if err := writer.WriteField("model", WhisperModel); err != nil {
		return "", fmt.Errorf("ошибка добавления модели: %w", err)
	}

	// Добавляем язык (русский)
	if err := writer.WriteField("language", "ru"); err != nil {
		return "", fmt.Errorf("ошибка добавления языка: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("ошибка закрытия writer: %w", err)
	}

	// Создаём запрос
	req, err := http.NewRequest("POST", GroqWhisperAPIURL, &requestBody)
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Выполняем запрос
	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка запроса транскрипции: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	var transcribeResp TranscribeResponse
	if err := json.Unmarshal(body, &transcribeResp); err != nil {
		return "", fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	if transcribeResp.Error != nil {
		return "", fmt.Errorf("ошибка Whisper API: %s", transcribeResp.Error.Message)
	}

	return transcribeResp.Text, nil
}
