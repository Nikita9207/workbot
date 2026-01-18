package ai

import (
	"fmt"
	"strconv"
	"strings"
)

// TrainerAI - AI-ассистент тренера
type TrainerAI struct {
	client *Client
}

// ClientProfile - профиль клиента для AI
type ClientProfile struct {
	Name         string
	Surname      string
	Age          int
	Level        string // новичок, средний, продвинутый
	Goal         string // похудение, масса, сила, выносливость, здоровье
	Restrictions string // травмы и ограничения
}

// TrainingParams - параметры для генерации тренировки
type TrainingParams struct {
	Type       string // силовая, кардио, функциональная
	Direction  string // верх, низ, фуллбоди, push, pull
	Duration   int    // минуты
	Equipment  string // зал, дом, минимум
	Notes      string // дополнительные пожелания
}

// GeneratedTraining - сгенерированная тренировка
type GeneratedTraining struct {
	Warmup    string
	Exercises []GeneratedExercise
	Cooldown  string
	Notes     string
}

// GeneratedExercise - упражнение
type GeneratedExercise struct {
	Name    string
	Sets    int
	Reps    int
	Weight  string
	Rest    string
	Notes   string
}

// NewTrainerAI создаёт нового AI-ассистента
func NewTrainerAI(ollamaURL, model string) *TrainerAI {
	return &TrainerAI{
		client: NewClientWithURL(ollamaURL, model),
	}
}

// GenerateTraining генерирует одну тренировку
func (t *TrainerAI) GenerateTraining(profile ClientProfile, params TrainingParams) (string, error) {
	// Формируем контекст клиента
	clientCtx := t.buildClientContext(profile)

	// Формируем запрос
	request := t.buildTrainingRequest(params)

	// Полный запрос
	userMessage := clientCtx + "\n" + request

	response, err := t.client.SimpleChat(SystemPromptTrainer, userMessage)
	if err != nil {
		return "", fmt.Errorf("ошибка генерации тренировки: %w", err)
	}

	return response, nil
}

// GenerateWeekPlan генерирует план на неделю
func (t *TrainerAI) GenerateWeekPlan(profile ClientProfile, daysPerWeek int, goal string) (string, error) {
	clientCtx := t.buildClientContext(profile)

	request := fmt.Sprintf(`Составь план тренировок на неделю:
- Тренировок в неделю: %d
- Главная цель: %s

Для каждого дня укажи:
1. День и тип тренировки
2. Основные группы мышц
3. Примерный список из 5-6 упражнений
4. Общий объём (подходы x повторы)`, daysPerWeek, goal)

	userMessage := clientCtx + "\n" + request

	response, err := t.client.SimpleChat(SystemPromptTrainer, userMessage)
	if err != nil {
		return "", fmt.Errorf("ошибка генерации недельного плана: %w", err)
	}

	return response, nil
}

// GenerateYearPlan генерирует годовой план
func (t *TrainerAI) GenerateYearPlan(profile ClientProfile, mainGoal string) (string, error) {
	clientCtx := t.buildClientContext(profile)

	request := fmt.Sprintf(`Составь годовой план тренировок (12 месяцев):
- Главная цель: %s

Для каждого месяца укажи:
1. Название фазы/периода
2. Конкретную цель месяца
3. Количество тренировок в неделю
4. Типы тренировок
5. На что обратить внимание

Учитывай периодизацию: подготовка -> основная работа -> пик -> восстановление`, mainGoal)

	userMessage := clientCtx + "\n" + request

	response, err := t.client.SimpleChat(SystemPromptPlan, userMessage)
	if err != nil {
		return "", fmt.Errorf("ошибка генерации годового плана: %w", err)
	}

	return response, nil
}

// ModifyTraining модифицирует тренировку по указаниям
func (t *TrainerAI) ModifyTraining(originalTraining, instructions string) (string, error) {
	request := fmt.Sprintf(`Исходная тренировка:
%s

Инструкции по изменению:
%s

Выведи изменённую тренировку в том же формате.`, originalTraining, instructions)

	response, err := t.client.SimpleChat(SystemPromptTrainer, request)
	if err != nil {
		return "", fmt.Errorf("ошибка модификации тренировки: %w", err)
	}

	return response, nil
}

// AskQuestion задаёт вопрос AI по тренировкам
func (t *TrainerAI) AskQuestion(question string) (string, error) {
	systemPrompt := `Ты — опытный персональный тренер. Отвечай на вопросы о тренировках, питании, восстановлении.
Будь конкретен и давай практические советы. Отвечай на русском языке.`

	response, err := t.client.SimpleChat(systemPrompt, question)
	if err != nil {
		return "", fmt.Errorf("ошибка ответа на вопрос: %w", err)
	}

	return response, nil
}

// ProgressionParams - параметры для плана с прогрессией
type ProgressionParams struct {
	Weeks        int    // количество недель (4, 6, 8, 12)
	DaysPerWeek  int    // тренировок в неделю (2-6)
	Goal         string // цель: сила, масса, похудение, рельеф
	StartWeights string // текущие рабочие веса клиента (если известны)
}

// GenerateProgressionPlan генерирует план с детальной прогрессией по неделям
func (t *TrainerAI) GenerateProgressionPlan(profile ClientProfile, params ProgressionParams) (string, error) {
	clientCtx := t.buildClientContext(profile)

	request := fmt.Sprintf(`Составь программу тренировок с ДЕТАЛЬНОЙ ПРОГРЕССИЕЙ:

ПАРАМЕТРЫ:
- Длительность: %d недель
- Тренировок в неделю: %d
- Цель: %s
`, params.Weeks, params.DaysPerWeek, params.Goal)

	if params.StartWeights != "" {
		request += fmt.Sprintf("- Текущие рабочие веса клиента: %s\n", params.StartWeights)
	}

	request += `
ОБЯЗАТЕЛЬНО ВКЛЮЧИ:
1. Структуру тренировочной недели (какой день что качаем)
2. Список упражнений для каждого дня (6-8 упражнений)
3. ТАБЛИЦУ ПРОГРЕССИИ с конкретными весами на каждую неделю
4. Разгрузочные недели (deload) каждую 4-ю неделю
5. Правила повышения весов

Формат таблицы прогрессии:
| Упражнение | Нед 1 | Нед 2 | Нед 3 | Нед 4 (deload) | ...
| Жим лёжа   | 4x8 60кг | 4x8 62.5кг | 4x9 62.5кг | 3x6 45кг | ...`

	userMessage := clientCtx + "\n" + request

	response, err := t.client.SimpleChat(SystemPromptProgression, userMessage)
	if err != nil {
		return "", fmt.Errorf("ошибка генерации плана с прогрессией: %w", err)
	}

	return response, nil
}

// GetMethodology получает информацию о методике тренировок
func (t *TrainerAI) GetMethodology(methodName string) (string, error) {
	request := fmt.Sprintf(`Расскажи подробно о методике тренировок: %s

Включи:
1. Описание методики и её автора
2. Для кого подходит (уровень, цели)
3. Принцип работы
4. Схему прогрессии
5. Плюсы и минусы
6. Конкретный пример программы на неделю
7. Источники (книги, исследования)`, methodName)

	response, err := t.client.SimpleChat(SystemPromptMethodology, request)
	if err != nil {
		return "", fmt.Errorf("ошибка получения информации о методике: %w", err)
	}

	return response, nil
}

// CompareMethodologies сравнивает несколько методик
func (t *TrainerAI) CompareMethodologies(methods []string, goal string) (string, error) {
	methodList := strings.Join(methods, ", ")

	request := fmt.Sprintf(`Сравни методики тренировок: %s

Цель клиента: %s

Для каждой методики укажи:
1. Краткое описание
2. Кому подходит лучше всего
3. Преимущества для данной цели
4. Недостатки

В конце дай рекомендацию: какую методику выбрать для указанной цели и почему.`, methodList, goal)

	response, err := t.client.SimpleChat(SystemPromptMethodology, request)
	if err != nil {
		return "", fmt.Errorf("ошибка сравнения методик: %w", err)
	}

	return response, nil
}

// CompetitionParams - параметры для подготовки к соревнованиям
type CompetitionParams struct {
	Sport       string // пауэрлифтинг, бодибилдинг, кроссфит, strongman
	Date        string // дата соревнований
	WeeksLeft   int    // недель до соревнований
	CurrentWeight float64 // текущий вес
	TargetWeight  float64 // целевая весовая категория
	CurrentLevel  string  // уровень атлета
}

// GenerateCompetitionPlan генерирует план подготовки к соревнованиям
func (t *TrainerAI) GenerateCompetitionPlan(profile ClientProfile, params CompetitionParams) (string, error) {
	clientCtx := t.buildClientContext(profile)

	request := fmt.Sprintf(`Составь план подготовки к соревнованиям:

СОРЕВНОВАНИЕ:
- Вид спорта: %s
- Дата: %s
- Недель до старта: %d
- Уровень атлета: %s
`, params.Sport, params.Date, params.WeeksLeft, params.CurrentLevel)

	if params.CurrentWeight > 0 && params.TargetWeight > 0 {
		request += fmt.Sprintf(`
ВЕСОВАЯ КАТЕГОРИЯ:
- Текущий вес: %.1f кг
- Целевая категория: %.1f кг
`, params.CurrentWeight, params.TargetWeight)
	}

	request += `
ВКЛЮЧИ В ПЛАН:
1. Периодизацию по неделям (общеподготовительный, специальный, предсоревновательный)
2. Объём и интенсивность по периодам
3. Ключевые тренировки каждого периода
4. Пиковую неделю
5. Стратегию выхода на вес (если нужно)
6. Рекомендации по питанию
7. План на день соревнований`

	userMessage := clientCtx + "\n" + request

	response, err := t.client.SimpleChat(SystemPromptCompetition, userMessage)
	if err != nil {
		return "", fmt.Errorf("ошибка генерации плана к соревнованиям: %w", err)
	}

	return response, nil
}

// buildClientContext формирует контекст клиента
func (t *TrainerAI) buildClientContext(profile ClientProfile) string {
	var parts []string

	parts = append(parts, "КЛИЕНТ:")
	parts = append(parts, fmt.Sprintf("Имя: %s %s", profile.Name, profile.Surname))

	if profile.Age > 0 {
		parts = append(parts, fmt.Sprintf("Возраст: %d лет", profile.Age))
	}

	if profile.Level != "" {
		parts = append(parts, fmt.Sprintf("Уровень: %s", profile.Level))
	}

	if profile.Goal != "" {
		parts = append(parts, fmt.Sprintf("Цель: %s", profile.Goal))
	}

	if profile.Restrictions != "" {
		parts = append(parts, fmt.Sprintf("Ограничения: %s", profile.Restrictions))
	}

	return strings.Join(parts, "\n")
}

// buildTrainingRequest формирует запрос на тренировку
func (t *TrainerAI) buildTrainingRequest(params TrainingParams) string {
	var parts []string

	parts = append(parts, "ЗАДАНИЕ: Составь тренировку")

	if params.Type != "" {
		parts = append(parts, fmt.Sprintf("Тип: %s", params.Type))
	}

	if params.Direction != "" {
		parts = append(parts, fmt.Sprintf("Направленность: %s", params.Direction))
	}

	if params.Duration > 0 {
		parts = append(parts, fmt.Sprintf("Длительность: %d минут", params.Duration))
	}

	if params.Equipment != "" {
		parts = append(parts, fmt.Sprintf("Оборудование: %s", params.Equipment))
	}

	if params.Notes != "" {
		parts = append(parts, fmt.Sprintf("Дополнительно: %s", params.Notes))
	}

	return strings.Join(parts, "\n")
}

// ParseExercises пытается распарсить упражнения из ответа AI
func ParseExercises(response string) []GeneratedExercise {
	var exercises []GeneratedExercise

	lines := strings.Split(response, "\n")
	var currentExercise *GeneratedExercise

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Ищем номер упражнения (1. Название)
		if len(line) > 2 && line[0] >= '1' && line[0] <= '9' && line[1] == '.' {
			if currentExercise != nil {
				exercises = append(exercises, *currentExercise)
			}
			currentExercise = &GeneratedExercise{
				Name: strings.TrimSpace(line[2:]),
			}
			continue
		}

		// Парсим параметры упражнения
		if currentExercise != nil {
			lineLower := strings.ToLower(line)

			if strings.Contains(lineLower, "подход") {
				// Пытаемся извлечь число подходов
				parts := strings.Split(line, "|")
				for _, part := range parts {
					part = strings.TrimSpace(part)
					partLower := strings.ToLower(part)

					if strings.Contains(partLower, "подход") {
						currentExercise.Sets = extractNumber(part)
					} else if strings.Contains(partLower, "повтор") {
						currentExercise.Reps = extractNumber(part)
					} else if strings.Contains(partLower, "вес") {
						currentExercise.Weight = extractValue(part)
					} else if strings.Contains(partLower, "отдых") {
						currentExercise.Rest = extractValue(part)
					}
				}
			} else if strings.Contains(lineLower, "примечание") || strings.Contains(lineLower, "заметк") {
				currentExercise.Notes = extractValue(line)
			}
		}
	}

	// Добавляем последнее упражнение
	if currentExercise != nil {
		exercises = append(exercises, *currentExercise)
	}

	return exercises
}

// extractNumber извлекает число из строки
func extractNumber(s string) int {
	var numStr string
	for _, c := range s {
		if c >= '0' && c <= '9' {
			numStr += string(c)
		} else if len(numStr) > 0 {
			break
		}
	}
	n, _ := strconv.Atoi(numStr)
	return n
}

// extractValue извлекает значение после двоеточия
func extractValue(s string) string {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(s)
}
