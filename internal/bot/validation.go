package bot

import (
	"regexp"
	"strings"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

// validateName и validatePhone определены в registration.go

// validateWeight validates weight in kg
func validateWeight(weight float64) error {
	if weight <= 0 {
		return ValidationError{Field: "weight", Message: "Вес должен быть положительным числом"}
	}
	if weight > 500 {
		return ValidationError{Field: "weight", Message: "Вес слишком большой (максимум 500 кг)"}
	}
	return nil
}

// validateReps validates repetitions count
func validateReps(reps int) error {
	if reps <= 0 {
		return ValidationError{Field: "reps", Message: "Количество повторений должно быть положительным"}
	}
	if reps > 100 {
		return ValidationError{Field: "reps", Message: "Слишком много повторений (максимум 100)"}
	}
	return nil
}

// validateSets validates sets count
func validateSets(sets int) error {
	if sets <= 0 {
		return ValidationError{Field: "sets", Message: "Количество подходов должно быть положительным"}
	}
	if sets > 20 {
		return ValidationError{Field: "sets", Message: "Слишком много подходов (максимум 20)"}
	}
	return nil
}

// validateExerciseName validates exercise name
func validateExerciseName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ValidationError{Field: "exercise_name", Message: "Название упражнения не может быть пустым"}
	}
	if len(name) < 3 {
		return ValidationError{Field: "exercise_name", Message: "Название слишком короткое (минимум 3 символа)"}
	}
	if len(name) > 100 {
		return ValidationError{Field: "exercise_name", Message: "Название слишком длинное (максимум 100 символов)"}
	}
	return nil
}

// validateDate validates date string in format DD.MM.YYYY or DD.MM
var datePatterns = []*regexp.Regexp{
	regexp.MustCompile(`^\d{1,2}\.\d{1,2}\.\d{4}$`),
	regexp.MustCompile(`^\d{1,2}\.\d{1,2}$`),
}

func validateDateString(date string) error {
	date = strings.TrimSpace(date)
	if date == "" {
		return ValidationError{Field: "date", Message: "Дата не может быть пустой"}
	}

	for _, pattern := range datePatterns {
		if pattern.MatchString(date) {
			return nil
		}
	}

	return ValidationError{Field: "date", Message: "Неверный формат даты. Используйте ДД.ММ.ГГГГ или ДД.ММ"}
}

// validateWeeks validates training plan weeks
func validateWeeks(weeks int) error {
	if weeks < 1 {
		return ValidationError{Field: "weeks", Message: "Минимум 1 неделя"}
	}
	if weeks > 52 {
		return ValidationError{Field: "weeks", Message: "Максимум 52 недели"}
	}
	return nil
}

// validateDaysPerWeek validates training days per week
func validateDaysPerWeek(days int) error {
	if days < 1 {
		return ValidationError{Field: "days_per_week", Message: "Минимум 1 день в неделю"}
	}
	if days > 7 {
		return ValidationError{Field: "days_per_week", Message: "Максимум 7 дней в неделю"}
	}
	return nil
}

// validate1PM validates 1PM value
func validate1PM(weight float64) error {
	if weight <= 0 {
		return ValidationError{Field: "1pm", Message: "1ПМ должен быть положительным числом"}
	}
	if weight > 600 {
		return ValidationError{Field: "1pm", Message: "1ПМ слишком большой (максимум 600 кг)"}
	}
	return nil
}

// validateIntensityPercent validates intensity percentage
func validateIntensityPercent(percent float64) error {
	if percent < 0 || percent > 100 {
		return ValidationError{Field: "intensity", Message: "Интенсивность должна быть от 0 до 100%"}
	}
	return nil
}
