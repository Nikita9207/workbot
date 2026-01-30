package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"workbot/internal/models"
)

// PlanGenerationState состояние инкрементальной генерации плана
type PlanGenerationState struct {
	Request       ProgramRequestV3        `json:"request"`
	GeneratedWeeks []models.TrainingWeek  `json:"generated_weeks"`
	LastWeekNum   int                     `json:"last_week_num"`
	TotalWeeks    int                     `json:"total_weeks"`
	BatchSize     int                     `json:"batch_size"`
	Status        string                  `json:"status"` // "in_progress", "completed", "error"
	LastError     string                  `json:"last_error,omitempty"`
	CreatedAt     time.Time               `json:"created_at"`
	UpdatedAt     time.Time               `json:"updated_at"`
}

// NewPlanGenerationState создаёт новое состояние генерации
func NewPlanGenerationState(req ProgramRequestV3, batchSize int) *PlanGenerationState {
	return &PlanGenerationState{
		Request:       req,
		GeneratedWeeks: []models.TrainingWeek{},
		LastWeekNum:   0,
		TotalWeeks:    req.TotalWeeks,
		BatchSize:     batchSize,
		Status:        "in_progress",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// GetNextBatchRange возвращает диапазон недель для следующего батча
func (s *PlanGenerationState) GetNextBatchRange() (start, end int, err error) {
	if s.IsComplete() {
		return 0, 0, fmt.Errorf("генерация уже завершена")
	}

	start = s.LastWeekNum + 1
	end = start + s.BatchSize - 1
	if end > s.TotalWeeks {
		end = s.TotalWeeks
	}

	return start, end, nil
}

// AddWeeks добавляет сгенерированные недели
func (s *PlanGenerationState) AddWeeks(weeks []models.TrainingWeek) {
	s.GeneratedWeeks = append(s.GeneratedWeeks, weeks...)
	if len(weeks) > 0 {
		s.LastWeekNum = weeks[len(weeks)-1].WeekNum
	}
	s.UpdatedAt = time.Now()

	if s.LastWeekNum >= s.TotalWeeks {
		s.Status = "completed"
	}
}

// IsComplete проверяет, завершена ли генерация
func (s *PlanGenerationState) IsComplete() bool {
	return s.LastWeekNum >= s.TotalWeeks || s.Status == "completed"
}

// GetProgress возвращает процент выполнения
func (s *PlanGenerationState) GetProgress() float64 {
	if s.TotalWeeks == 0 {
		return 0
	}
	return float64(s.LastWeekNum) / float64(s.TotalWeeks) * 100
}

// SaveState сохраняет состояние в файл
func (s *PlanGenerationState) SaveState() error {
	s.UpdatedAt = time.Now()

	dir := getStatesDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	filename := filepath.Join(dir, sanitizeClientName(s.Request.ClientName)+".json")
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// LoadState загружает состояние из файла
func LoadState(clientName string) (*PlanGenerationState, error) {
	filename := filepath.Join(getStatesDir(), sanitizeClientName(clientName)+".json")
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var state PlanGenerationState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// ListSavedStates возвращает список сохранённых состояний
func ListSavedStates() ([]string, error) {
	dir := getStatesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			name := strings.TrimSuffix(entry.Name(), ".json")
			names = append(names, name)
		}
	}

	return names, nil
}

// DeleteState удаляет состояние
func DeleteState(clientName string) error {
	filename := filepath.Join(getStatesDir(), sanitizeClientName(clientName)+".json")
	return os.Remove(filename)
}

func getStatesDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".plancli", "states")
}

func sanitizeClientName(name string) string {
	replacer := strings.NewReplacer(
		" ", "_",
		"/", "_",
		"\\", "_",
		":", "_",
	)
	return replacer.Replace(name)
}
