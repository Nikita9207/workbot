package ai

import (
	"fmt"
	"os"
)

// Provider тип AI провайдера
type Provider string

const (
	ProviderAuto      Provider = "auto"
	ProviderOllama    Provider = "ollama"
	ProviderOpenRouter Provider = "openrouter"
)

// Модели
const (
	ModelGLM4Flash = "glm-4-flash"
)

// ProviderConfig конфигурация провайдера
type ProviderConfig struct {
	Provider    Provider
	OllamaURL   string
	OllamaModel string
}

// NewAIClient создаёт AI клиент на основе конфигурации
func NewAIClient(cfg ProviderConfig) (*Client, error) {
	switch cfg.Provider {
	case ProviderOllama, "":
		return NewClientWithURL(cfg.OllamaURL, cfg.OllamaModel), nil
	case ProviderOpenRouter:
		apiKey := os.Getenv("OPENROUTER_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OPENROUTER_API_KEY не задан")
		}
		// Для OpenRouter используем тот же клиент с другим URL
		return NewClientWithURL("https://openrouter.ai/api/v1", cfg.OllamaModel), nil
	case ProviderAuto:
		// Пробуем OpenRouter, потом Ollama
		apiKey := os.Getenv("OPENROUTER_API_KEY")
		if apiKey != "" {
			return NewClientWithURL("https://openrouter.ai/api/v1", cfg.OllamaModel), nil
		}
		return NewClientWithURL(cfg.OllamaURL, cfg.OllamaModel), nil
	default:
		return NewClientWithURL(cfg.OllamaURL, cfg.OllamaModel), nil
	}
}

// GetDefaultProvider возвращает провайдер по умолчанию
func GetDefaultProvider() Provider {
	if os.Getenv("OPENROUTER_API_KEY") != "" {
		return ProviderOpenRouter
	}
	return ProviderOllama
}

// GetProviderName возвращает название провайдера
func GetProviderName(p Provider) string {
	switch p {
	case ProviderOllama:
		return "Ollama (локальный)"
	case ProviderOpenRouter:
		return "OpenRouter"
	default:
		return "Auto"
	}
}
