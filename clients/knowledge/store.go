package knowledge

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// ============================================
// KNOWLEDGE STORE - лёгкий поиск по индексу
// Работает на Pi без Ollama
// ============================================

// Document - документ с эмбеддингом
type Document struct {
	ID        string            `json:"id"`
	Content   string            `json:"content"`
	Embedding []float64         `json:"embedding"`
	Metadata  map[string]string `json:"metadata"`
}

// KnowledgeIndex - индекс знаний
type KnowledgeIndex struct {
	Version   string     `json:"version"`
	CreatedAt time.Time  `json:"created_at"`
	Documents []Document `json:"documents"`
}

// SearchResult - результат поиска
type SearchResult struct {
	Document   Document
	Similarity float64
}

// Store - хранилище знаний
type Store struct {
	mu        sync.RWMutex
	documents []Document
	loaded    bool
}

// NewStore создаёт новое хранилище
func NewStore() *Store {
	return &Store{
		documents: make([]Document, 0),
	}
}

// Load загружает индекс из файла
func (s *Store) Load(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("не удалось прочитать файл: %w", err)
	}

	var index KnowledgeIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return fmt.Errorf("ошибка парсинга индекса: %w", err)
	}

	s.documents = index.Documents
	s.loaded = true

	return nil
}

// IsLoaded возвращает true если индекс загружен
func (s *Store) IsLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loaded
}

// Count возвращает количество документов
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.documents)
}

// SearchByKeywords ищет по ключевым словам (без эмбеддингов)
// Используем TF-IDF подобный подход
func (s *Store) SearchByKeywords(query string, topK int) []SearchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.documents) == 0 {
		return nil
	}

	query = strings.ToLower(query)
	words := strings.Fields(query)

	var results []SearchResult

	for _, doc := range s.documents {
		content := strings.ToLower(doc.Content)
		score := 0.0

		for _, word := range words {
			if strings.Contains(content, word) {
				// Считаем количество вхождений
				count := float64(strings.Count(content, word))
				// Нормализуем по длине документа
				score += count / float64(len(content)+1) * 1000
			}
		}

		if score > 0 {
			results = append(results, SearchResult{
				Document:   doc,
				Similarity: score,
			})
		}
	}

	// Сортируем по релевантности
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if len(results) > topK {
		results = results[:topK]
	}

	return results
}

// SearchByEmbedding ищет по эмбеддингу (если есть query embedding)
func (s *Store) SearchByEmbedding(queryEmbedding []float64, topK int, threshold float64) []SearchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.documents) == 0 || len(queryEmbedding) == 0 {
		return nil
	}

	var results []SearchResult

	for _, doc := range s.documents {
		if len(doc.Embedding) == 0 {
			continue
		}

		similarity := cosineSimilarity(queryEmbedding, doc.Embedding)
		if similarity >= threshold {
			results = append(results, SearchResult{
				Document:   doc,
				Similarity: similarity,
			})
		}
	}

	// Сортируем по релевантности
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if len(results) > topK {
		results = results[:topK]
	}

	return results
}

// GetContext получает контекст для LLM
func (s *Store) GetContext(query string, maxChunks int) string {
	results := s.SearchByKeywords(query, maxChunks)

	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Релевантная информация из базы знаний:\n\n")

	for i, result := range results {
		sb.WriteString(fmt.Sprintf("--- Источник %d (релевантность: %.2f) ---\n", i+1, result.Similarity))
		if source, ok := result.Document.Metadata["file"]; ok {
			sb.WriteString(fmt.Sprintf("Файл: %s\n", source))
		}
		sb.WriteString(result.Document.Content)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// cosineSimilarity вычисляет косинусное сходство
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
