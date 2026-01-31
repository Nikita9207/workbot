package formatter

import (
	"fmt"
	"strings"

	"workbot/internal/models"
)

// TelegramFormatter - форматтер для Telegram
type TelegramFormatter struct{}

// NewTelegramFormatter создаёт новый форматтер
func NewTelegramFormatter() *TelegramFormatter {
	return &TelegramFormatter{}
}

// FormatProgram форматирует всю программу
func (f *TelegramFormatter) FormatProgram(program *models.GeneratedProgram) string {
	var sb strings.Builder

	// Заголовок
	sb.WriteString(f.formatHeader(program))
	sb.WriteString("\n")

	// Фазы
	sb.WriteString(f.formatPhases(program.Phases))
	sb.WriteString("\n")

	// Недели
	for _, week := range program.Weeks {
		sb.WriteString(f.formatWeek(week))
		sb.WriteString("\n")
	}

	// Статистика
	sb.WriteString(f.formatStatistics(program.Statistics))

	// Замены
	if len(program.Substitutions) > 0 {
		sb.WriteString("\n")
		sb.WriteString(f.formatSubstitutions(program.Substitutions))
	}

	return sb.String()
}

// formatHeader форматирует заголовок программы
func (f *TelegramFormatter) formatHeader(program *models.GeneratedProgram) string {
	goalNames := map[models.TrainingGoal]string{
		models.GoalStrength:    "Сила",
		models.GoalHypertrophy: "Набор массы",
		models.GoalFatLoss:     "Жиросжигание",
		models.GoalHyrox:       "Hyrox",
		models.GoalEndurance:   "Выносливость",
		models.GoalGeneral:     "ОФП",
	}

	periodNames := map[models.PeriodizationType]string{
		models.PeriodLinear:     "линейная",
		models.PeriodUndulating: "волнообразная",
		models.PeriodBlock:      "блочная",
		models.PeriodReverse:    "обратная",
	}

	goal := goalNames[program.Goal]
	if goal == "" {
		goal = string(program.Goal)
	}

	period := periodNames[program.Periodization]
	if period == "" {
		period = string(program.Periodization)
	}

	return fmt.Sprintf(`ПРОГРАММА: %s
Клиент: %s

Длительность: %d недель
Тренировок в неделю: %d
Периодизация: %s
`,
		goal,
		program.ClientName,
		program.TotalWeeks,
		program.DaysPerWeek,
		period,
	)
}

// formatPhases форматирует фазы программы
func (f *TelegramFormatter) formatPhases(phases []models.ProgramPhase) string {
	if len(phases) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("══════════════════════════════════\n")
	sb.WriteString("ФАЗЫ ПРОГРАММЫ\n")
	sb.WriteString("══════════════════════════════════\n\n")

	for _, phase := range phases {
		sb.WriteString(fmt.Sprintf("▸ %s (недели %d-%d)\n", phase.Name, phase.WeekStart, phase.WeekEnd))
		sb.WriteString(fmt.Sprintf("  %s\n", phase.Focus))
		sb.WriteString(fmt.Sprintf("  Интенсивность: %.0f-%.0f%%\n", phase.IntensityMin, phase.IntensityMax))
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatWeek форматирует неделю
func (f *TelegramFormatter) formatWeek(week models.GeneratedWeek) string {
	var sb strings.Builder

	// Заголовок недели
	weekHeader := fmt.Sprintf("НЕДЕЛЯ %d", week.WeekNum)
	if week.IsDeload {
		weekHeader += " (РАЗГРУЗКА)"
	}
	if week.PhaseName != "" {
		weekHeader += fmt.Sprintf(" | %s", week.PhaseName)
	}

	sb.WriteString("┌─────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("│ %s\n", weekHeader))
	if !week.IsDeload {
		sb.WriteString(fmt.Sprintf("│ Интенсивность: %.0f%% | RPE: %.1f\n", week.IntensityPercent, week.RPETarget))
	}
	sb.WriteString("└─────────────────────────────────\n\n")

	// Дни
	for _, day := range week.Days {
		sb.WriteString(f.formatDay(day))
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatDay форматирует тренировочный день
func (f *TelegramFormatter) formatDay(day models.GeneratedDay) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s\n", day.Name))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━\n\n")

	for _, ex := range day.Exercises {
		sb.WriteString(f.formatExercise(ex))
		sb.WriteString("\n")
	}

	if day.EstimatedDuration > 0 {
		sb.WriteString(fmt.Sprintf("⏱ ~%d мин\n", day.EstimatedDuration))
	}

	return sb.String()
}

// formatExercise форматирует упражнение
func (f *TelegramFormatter) formatExercise(ex models.GeneratedExercise) string {
	var sb strings.Builder

	// Номер и название
	sb.WriteString(fmt.Sprintf("%d. %s\n", ex.OrderNum, ex.ExerciseName))

	// Параметры
	params := fmt.Sprintf("   %dx%s", ex.Sets, ex.Reps)

	// Вес или уровень TRX
	if ex.Weight > 0 {
		params += fmt.Sprintf(" @%.1f кг", ex.Weight)
		if ex.WeightPercent > 0 {
			params += fmt.Sprintf(" (%.0f%%)", ex.WeightPercent)
		}
	} else if ex.TRXLevel > 0 {
		params += fmt.Sprintf(" @уровень %d", ex.TRXLevel)
	}

	sb.WriteString(params)

	// Темп (если есть)
	if ex.Tempo != "" {
		sb.WriteString(fmt.Sprintf(" | Темп %s", ex.Tempo))
	}

	// Отдых
	if ex.RestSeconds > 0 {
		sb.WriteString(fmt.Sprintf(" | Отдых %s", formatRest(ex.RestSeconds)))
	}

	sb.WriteString("\n")

	// RPE (если есть)
	if ex.RPE > 0 {
		sb.WriteString(fmt.Sprintf("   RPE: %.1f\n", ex.RPE))
	}

	// Альтернатива
	if ex.Alternative != nil {
		sb.WriteString(fmt.Sprintf("   ↳ %s\n", ex.Alternative.ExerciseName))
	}

	// Заметки
	if ex.Notes != "" {
		sb.WriteString(fmt.Sprintf("   📝 %s\n", ex.Notes))
	}

	return sb.String()
}

// formatStatistics форматирует статистику
func (f *TelegramFormatter) formatStatistics(stats models.ProgramStats) string {
	var sb strings.Builder

	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("СТАТИСТИКА ПРОГРАММЫ\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━\n")

	sb.WriteString(fmt.Sprintf("Всего тренировок: %d\n", stats.TotalWorkouts))
	sb.WriteString(fmt.Sprintf("Всего подходов: %d\n", stats.TotalSets))

	if stats.TotalVolume > 0 {
		sb.WriteString(fmt.Sprintf("Общий тоннаж: %.0f кг\n", stats.TotalVolume))
	}

	if stats.AvgWorkoutDur > 0 {
		sb.WriteString(fmt.Sprintf("Средняя тренировка: ~%d мин\n", stats.AvgWorkoutDur))
	}

	// Объём по мышцам
	if len(stats.SetsPerMuscle) > 0 {
		sb.WriteString("\nОбъём по мышцам (подходов/неделю):\n")
		for muscle, sets := range stats.SetsPerMuscle {
			muscleName := getMuscleNameRu(muscle)
			sb.WriteString(fmt.Sprintf("  %s: %d\n", muscleName, sets))
		}
	}

	// Баланс паттернов движения
	if stats.MovementBalance != nil {
		sb.WriteString(f.formatMovementBalance(stats.MovementBalance))
	}

	return sb.String()
}

// formatMovementBalance форматирует баланс паттернов движения
func (f *TelegramFormatter) formatMovementBalance(b *models.MovementBalance) string {
	var sb strings.Builder

	sb.WriteString("\n━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("БАЛАНС ПАТТЕРНОВ\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━\n")

	// Push/Pull
	if b.PushSets > 0 || b.PullSets > 0 {
		sb.WriteString(fmt.Sprintf("Push/Pull: %d/%d ", b.PushSets, b.PullSets))
		if b.PullSets > 0 {
			sb.WriteString(fmt.Sprintf("(%.2f:1) %s\n", b.PushPullRatio, getStatusEmoji(b.PushPullStatus)))
		} else {
			sb.WriteString("(∞) ⚠️\n")
		}
	}

	// Quad/Hip
	if b.QuadSets > 0 || b.HipSets > 0 {
		sb.WriteString(fmt.Sprintf("Quad/Hip: %d/%d ", b.QuadSets, b.HipSets))
		if b.HipSets > 0 {
			sb.WriteString(fmt.Sprintf("(%.2f:1) %s\n", b.QuadHipRatio, getStatusEmoji(b.QuadHipStatus)))
		} else {
			sb.WriteString("(∞) ⚠️\n")
		}
	}

	// Horiz/Vert Push
	if b.HorizontalPushSets > 0 || b.VerticalPushSets > 0 {
		sb.WriteString(fmt.Sprintf("H/V Push: %d/%d\n", b.HorizontalPushSets, b.VerticalPushSets))
	}

	// Horiz/Vert Pull
	if b.HorizontalPullSets > 0 || b.VerticalPullSets > 0 {
		sb.WriteString(fmt.Sprintf("H/V Pull: %d/%d\n", b.HorizontalPullSets, b.VerticalPullSets))
	}

	// Bi/Uni
	if b.BilateralLegSets > 0 || b.UnilateralLegSets > 0 {
		sb.WriteString(fmt.Sprintf("Bi/Uni (ноги): %d/%d\n", b.BilateralLegSets, b.UnilateralLegSets))
	}

	// Core
	if b.CoreSets > 0 {
		sb.WriteString(fmt.Sprintf("Core: %d сетов\n", b.CoreSets))
	}

	// Оценка
	sb.WriteString(fmt.Sprintf("\nОценка: %d/100 %s\n", b.OverallScore, getAssessmentEmoji(b.Assessment)))

	// Рекомендации
	if len(b.Recommendations) > 0 {
		sb.WriteString("\nРекомендации:\n")
		for _, rec := range b.Recommendations {
			sb.WriteString(fmt.Sprintf("• %s\n", rec))
		}
	}

	return sb.String()
}

// getStatusEmoji возвращает эмодзи для статуса баланса
func getStatusEmoji(status string) string {
	switch status {
	case "balanced":
		return "✅"
	case "slightly_push_heavy", "slightly_pull_heavy", "slightly_quad_heavy", "slightly_hip_heavy":
		return "⚠️"
	case "push_heavy", "pull_heavy", "quad_heavy", "hip_heavy":
		return "❌"
	default:
		return ""
	}
}

// getAssessmentEmoji возвращает эмодзи для общей оценки
func getAssessmentEmoji(assessment string) string {
	switch assessment {
	case "excellent":
		return "🏆"
	case "good":
		return "👍"
	case "needs_attention":
		return "⚠️"
	case "imbalanced":
		return "❌"
	default:
		return ""
	}
}

// formatSubstitutions форматирует замены
func (f *TelegramFormatter) formatSubstitutions(subs []models.Substitution) string {
	var sb strings.Builder

	sb.WriteString("ЗАМЕНЫ ИЗ-ЗА ОГРАНИЧЕНИЙ:\n")

	for _, sub := range subs {
		sb.WriteString(fmt.Sprintf("- %s → %s (%s)\n", sub.OriginalName, sub.ReplacedName, sub.Reason))
	}

	return sb.String()
}

// FormatWeekOnly форматирует только одну неделю (для отправки по частям)
func (f *TelegramFormatter) FormatWeekOnly(week models.GeneratedWeek) string {
	return f.formatWeek(week)
}

// FormatDayOnly форматирует только один день
func (f *TelegramFormatter) FormatDayOnly(day models.GeneratedDay, weekNum int) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📅 НЕДЕЛЯ %d\n\n", weekNum))
	sb.WriteString(f.formatDay(day))

	return sb.String()
}

// === Утилиты ===

func formatRest(seconds int) string {
	if seconds >= 60 {
		mins := seconds / 60
		secs := seconds % 60
		if secs == 0 {
			return fmt.Sprintf("%d мин", mins)
		}
		return fmt.Sprintf("%d:%02d", mins, secs)
	}
	return fmt.Sprintf("%d сек", seconds)
}

func getMuscleNameRu(muscle models.MuscleGroupExt) string {
	names := map[models.MuscleGroupExt]string{
		models.MuscleChest:        "Грудь",
		models.MuscleBack:         "Спина",
		models.MuscleShoulders:    "Плечи",
		models.MuscleBiceps:       "Бицепс",
		models.MuscleTriceps:      "Трицепс",
		models.MuscleQuads:        "Квадрицепс",
		models.MuscleHamstrings:   "Бицепс бедра",
		models.MuscleGlutes:       "Ягодицы",
		models.MuscleCore:         "Кор",
		models.MuscleCalves:       "Икры",
		models.MuscleTraps:        "Трапеции",
		models.MuscleRearDelts:    "Задние дельты",
		models.MuscleForearms:     "Предплечья",
		models.MuscleLowerBack:    "Поясница",
		models.MuscleFullBody:     "Всё тело",
		models.MuscleCardioSystem: "Кардио",
	}

	if name, ok := names[muscle]; ok {
		return name
	}
	return string(muscle)
}
