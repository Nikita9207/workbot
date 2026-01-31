package models

import "fmt"

// ===============================================
// БАЛАНС ПАТТЕРНОВ ДВИЖЕНИЯ
// ===============================================

// MovementCategory - категория движения для расчёта баланса
type MovementCategory string

const (
	// Верхняя часть тела
	CategoryPush MovementCategory = "push" // Толкающие (жим, отжимания)
	CategoryPull MovementCategory = "pull" // Тянущие (тяга, подтягивания)

	// Нижняя часть тела
	CategoryQuadDominant MovementCategory = "quad_dominant" // Квад-доминантные (приседания, выпады)
	CategoryHipDominant  MovementCategory = "hip_dominant"  // Хип-доминантные (RDL, hip thrust, тяга)

	// Другие категории для полного анализа
	CategoryCore     MovementCategory = "core"     // Кор/пресс
	CategoryCarry    MovementCategory = "carry"    // Переноски
	CategoryCardio   MovementCategory = "cardio"   // Кардио
	CategoryPlyo     MovementCategory = "plyo"     // Плиометрика
	CategoryRotation MovementCategory = "rotation" // Ротационные
)

// MovementBalance - баланс паттернов в программе/тренировке
type MovementBalance struct {
	// Push/Pull баланс (верх тела)
	PushSets       int     `json:"push_sets"`
	PullSets       int     `json:"pull_sets"`
	PushPullRatio  float64 `json:"push_pull_ratio"`  // Идеал: 1.0 (1:1)
	PushPullStatus string  `json:"push_pull_status"` // "balanced", "push_heavy", "pull_heavy"

	// Quad/Hip баланс (низ тела)
	QuadSets       int     `json:"quad_sets"`
	HipSets        int     `json:"hip_sets"`
	QuadHipRatio   float64 `json:"quad_hip_ratio"`   // Идеал: 1.0 (1:1)
	QuadHipStatus  string  `json:"quad_hip_status"`  // "balanced", "quad_heavy", "hip_heavy"

	// Горизонтальные vs Вертикальные (верх тела)
	HorizontalPushSets int     `json:"horizontal_push_sets"` // Жим лёжа, отжимания
	VerticalPushSets   int     `json:"vertical_push_sets"`   // Жим стоя, жим над головой
	HorizontalPullSets int     `json:"horizontal_pull_sets"` // Тяга в наклоне
	VerticalPullSets   int     `json:"vertical_pull_sets"`   // Подтягивания, тяга сверху
	HVPushRatio        float64 `json:"hv_push_ratio"`        // Горизонтальный/Вертикальный Push
	HVPullRatio        float64 `json:"hv_pull_ratio"`        // Горизонтальный/Вертикальный Pull

	// Bilateral vs Unilateral (низ тела)
	BilateralLegSets  int     `json:"bilateral_leg_sets"`  // Приседания, RDL
	UnilateralLegSets int     `json:"unilateral_leg_sets"` // Выпады, болгарские сплиты
	BiUniRatio        float64 `json:"bi_uni_ratio"`        // Идеал: 1:1 или 2:1

	// Дополнительные метрики
	CoreSets   int `json:"core_sets"`
	CarrySets  int `json:"carry_sets"`
	CardioSets int `json:"cardio_sets"`

	// Общий счёт баланса (0-100)
	OverallScore int    `json:"overall_score"`
	Assessment   string `json:"assessment"` // "excellent", "good", "needs_attention", "imbalanced"

	// Рекомендации
	Recommendations []string `json:"recommendations,omitempty"`
}

// BalanceThresholds - пороги для оценки баланса
var BalanceThresholds = struct {
	// Push/Pull
	PushPullIdealMin   float64 // Нижняя граница идеального баланса
	PushPullIdealMax   float64 // Верхняя граница идеального баланса
	PushPullAcceptMin  float64 // Допустимый минимум
	PushPullAcceptMax  float64 // Допустимый максимум

	// Quad/Hip
	QuadHipIdealMin  float64
	QuadHipIdealMax  float64
	QuadHipAcceptMin float64
	QuadHipAcceptMax float64

	// H/V (Push/Pull)
	HVIdealMin  float64
	HVIdealMax  float64
	HVAcceptMin float64
	HVAcceptMax float64

	// Bi/Uni
	BiUniIdealMin  float64
	BiUniIdealMax  float64
	BiUniAcceptMin float64
	BiUniAcceptMax float64
}{
	// Push:Pull — идеал 1:1, допустимо 0.8:1 - 1.2:1
	PushPullIdealMin:  0.9,
	PushPullIdealMax:  1.1,
	PushPullAcceptMin: 0.75,
	PushPullAcceptMax: 1.33,

	// Quad:Hip — идеал 1:1, допустимо 0.8:1 - 1.5:1 (квады обычно чуть больше)
	QuadHipIdealMin:  0.9,
	QuadHipIdealMax:  1.2,
	QuadHipAcceptMin: 0.7,
	QuadHipAcceptMax: 1.5,

	// Horizontal:Vertical — идеал 1:1, допустимо 0.5:1 - 2:1
	HVIdealMin:  0.8,
	HVIdealMax:  1.25,
	HVAcceptMin: 0.5,
	HVAcceptMax: 2.0,

	// Bilateral:Unilateral — идеал 1.5:1 - 2:1 (билатеральных обычно больше)
	BiUniIdealMin:  1.0,
	BiUniIdealMax:  2.0,
	BiUniAcceptMin: 0.5,
	BiUniAcceptMax: 3.0,
}

// MovementCategoryMap - маппинг MovementType -> MovementCategory
var MovementCategoryMap = map[MovementType]MovementCategory{
	MovementPush:     CategoryPush,
	MovementPull:     CategoryPull,
	MovementHinge:    CategoryHipDominant,
	MovementSquat:    CategoryQuadDominant,
	MovementLunge:    CategoryQuadDominant, // Выпады = квад-доминантные
	MovementCarry:    CategoryCarry,
	MovementRotation: CategoryRotation,
	MovementCardio:   CategoryCardio,
	MovementPlyo:     CategoryPlyo,
	MovementCore:     CategoryCore,
}

// MuscleToCategory - маппинг MuscleGroupExt -> MovementCategory (для дополнительной классификации)
var MuscleToCategory = map[MuscleGroupExt]MovementCategory{
	// Push
	MuscleChest:     CategoryPush,
	MuscleShoulders: CategoryPush, // Передняя дельта
	MuscleTriceps:   CategoryPush,

	// Pull
	MuscleBack:      CategoryPull,
	MuscleUpperBack: CategoryPull,
	MuscleRearDelts: CategoryPull,
	MuscleBiceps:    CategoryPull,
	MuscleTraps:     CategoryPull,
	MuscleForearms:  CategoryPull,

	// Quad-dominant
	MuscleQuads:    CategoryQuadDominant,
	MuscleCalves:   CategoryQuadDominant, // Условно
	MuscleAdductors: CategoryQuadDominant,

	// Hip-dominant
	MuscleHamstrings: CategoryHipDominant,
	MuscleGlutes:     CategoryHipDominant,
	MuscleLowerBack:  CategoryHipDominant,
	MuscleHipFlexors: CategoryHipDominant,

	// Core
	MuscleCore: CategoryCore,
}

// SubCategory для детализации Push/Pull
type SubCategory string

const (
	SubCategoryHorizontalPush SubCategory = "horizontal_push" // Жим лёжа, отжимания
	SubCategoryVerticalPush   SubCategory = "vertical_push"   // Жим над головой
	SubCategoryHorizontalPull SubCategory = "horizontal_pull" // Тяга в наклоне
	SubCategoryVerticalPull   SubCategory = "vertical_pull"   // Подтягивания
)

// ExerciseSubCategory - подкатегория упражнения (для H/V баланса)
type ExerciseSubCategory struct {
	MovementCategory MovementCategory
	SubCategory      SubCategory
	IsUnilateral     bool // Односторонние упражнения
}

// CalculateBalance вычисляет баланс паттернов для набора упражнений
func CalculateBalance(exercises []GeneratedExercise) *MovementBalance {
	b := &MovementBalance{}

	for _, ex := range exercises {
		category := GetMovementCategory(ex.MovementType, ex.MuscleGroup)
		sets := ex.Sets

		switch category {
		case CategoryPush:
			b.PushSets += sets
			// Детализация H/V (по умолчанию считаем горизонтальным)
			if isVerticalPush(ex.ExerciseName) {
				b.VerticalPushSets += sets
			} else {
				b.HorizontalPushSets += sets
			}

		case CategoryPull:
			b.PullSets += sets
			if isVerticalPull(ex.ExerciseName) {
				b.VerticalPullSets += sets
			} else {
				b.HorizontalPullSets += sets
			}

		case CategoryQuadDominant:
			b.QuadSets += sets
			if isUnilateral(ex.ExerciseName) {
				b.UnilateralLegSets += sets
			} else {
				b.BilateralLegSets += sets
			}

		case CategoryHipDominant:
			b.HipSets += sets
			if isUnilateral(ex.ExerciseName) {
				b.UnilateralLegSets += sets
			} else {
				b.BilateralLegSets += sets
			}

		case CategoryCore:
			b.CoreSets += sets

		case CategoryCarry:
			b.CarrySets += sets

		case CategoryCardio:
			b.CardioSets += sets
		}
	}

	// Рассчитываем ratios
	b.calculateRatios()

	// Оценка и рекомендации
	b.assess()

	return b
}

// GetMovementCategory определяет категорию движения
func GetMovementCategory(movementType MovementType, muscleGroup MuscleGroupExt) MovementCategory {
	// Сначала пробуем по MovementType
	if cat, ok := MovementCategoryMap[movementType]; ok {
		return cat
	}

	// Fallback на MuscleGroup
	if cat, ok := MuscleToCategory[muscleGroup]; ok {
		return cat
	}

	return CategoryCore // По умолчанию
}

// calculateRatios вычисляет все соотношения
func (b *MovementBalance) calculateRatios() {
	// Push/Pull ratio
	if b.PullSets > 0 {
		b.PushPullRatio = float64(b.PushSets) / float64(b.PullSets)
	} else if b.PushSets > 0 {
		b.PushPullRatio = 999 // Нет Pull упражнений
	}

	// Quad/Hip ratio
	if b.HipSets > 0 {
		b.QuadHipRatio = float64(b.QuadSets) / float64(b.HipSets)
	} else if b.QuadSets > 0 {
		b.QuadHipRatio = 999
	}

	// H/V Push ratio
	if b.VerticalPushSets > 0 {
		b.HVPushRatio = float64(b.HorizontalPushSets) / float64(b.VerticalPushSets)
	} else if b.HorizontalPushSets > 0 {
		b.HVPushRatio = 999
	}

	// H/V Pull ratio
	if b.VerticalPullSets > 0 {
		b.HVPullRatio = float64(b.HorizontalPullSets) / float64(b.VerticalPullSets)
	} else if b.HorizontalPullSets > 0 {
		b.HVPullRatio = 999
	}

	// Bi/Uni ratio
	if b.UnilateralLegSets > 0 {
		b.BiUniRatio = float64(b.BilateralLegSets) / float64(b.UnilateralLegSets)
	} else if b.BilateralLegSets > 0 {
		b.BiUniRatio = 999
	}
}

// assess оценивает баланс и генерирует рекомендации
func (b *MovementBalance) assess() {
	score := 100
	t := BalanceThresholds

	// Push/Pull оценка
	if b.PushSets > 0 || b.PullSets > 0 {
		if b.PushPullRatio >= t.PushPullIdealMin && b.PushPullRatio <= t.PushPullIdealMax {
			b.PushPullStatus = "balanced"
		} else if b.PushPullRatio < t.PushPullAcceptMin {
			b.PushPullStatus = "pull_heavy"
			score -= 15
			b.Recommendations = append(b.Recommendations,
				fmt.Sprintf("Добавьте Push упражнения (жим, отжимания). Push:Pull = %.2f:1", b.PushPullRatio))
		} else if b.PushPullRatio > t.PushPullAcceptMax {
			b.PushPullStatus = "push_heavy"
			score -= 15
			b.Recommendations = append(b.Recommendations,
				fmt.Sprintf("Добавьте Pull упражнения (тяги, подтягивания). Push:Pull = %.2f:1", b.PushPullRatio))
		} else if b.PushPullRatio < t.PushPullIdealMin {
			b.PushPullStatus = "slightly_pull_heavy"
			score -= 5
		} else {
			b.PushPullStatus = "slightly_push_heavy"
			score -= 5
		}
	}

	// Quad/Hip оценка
	if b.QuadSets > 0 || b.HipSets > 0 {
		if b.QuadHipRatio >= t.QuadHipIdealMin && b.QuadHipRatio <= t.QuadHipIdealMax {
			b.QuadHipStatus = "balanced"
		} else if b.QuadHipRatio < t.QuadHipAcceptMin {
			b.QuadHipStatus = "hip_heavy"
			score -= 15
			b.Recommendations = append(b.Recommendations,
				fmt.Sprintf("Добавьте Quad упражнения (приседания, выпады). Quad:Hip = %.2f:1", b.QuadHipRatio))
		} else if b.QuadHipRatio > t.QuadHipAcceptMax {
			b.QuadHipStatus = "quad_heavy"
			score -= 15
			b.Recommendations = append(b.Recommendations,
				fmt.Sprintf("Добавьте Hip упражнения (RDL, hip thrust). Quad:Hip = %.2f:1", b.QuadHipRatio))
		} else if b.QuadHipRatio < t.QuadHipIdealMin {
			b.QuadHipStatus = "slightly_hip_heavy"
			score -= 5
		} else {
			b.QuadHipStatus = "slightly_quad_heavy"
			score -= 5
		}
	}

	// H/V баланс (мелкие поправки)
	if b.HVPushRatio > t.HVAcceptMax {
		score -= 5
		b.Recommendations = append(b.Recommendations,
			"Добавьте вертикальные Push (жим над головой)")
	}
	if b.HVPullRatio > t.HVAcceptMax {
		score -= 5
		b.Recommendations = append(b.Recommendations,
			"Добавьте вертикальные Pull (подтягивания)")
	}

	// Bi/Uni баланс
	if b.UnilateralLegSets == 0 && (b.QuadSets > 0 || b.HipSets > 0) {
		score -= 10
		b.Recommendations = append(b.Recommendations,
			"Добавьте унилатеральные упражнения (выпады, болгарские сплиты)")
	}

	// Core
	totalUpperLower := b.PushSets + b.PullSets + b.QuadSets + b.HipSets
	if totalUpperLower > 0 && b.CoreSets == 0 {
		score -= 5
		b.Recommendations = append(b.Recommendations,
			"Добавьте упражнения на кор")
	}

	// Финальный score
	if score < 0 {
		score = 0
	}
	b.OverallScore = score

	// Assessment
	switch {
	case score >= 90:
		b.Assessment = "excellent"
	case score >= 75:
		b.Assessment = "good"
	case score >= 60:
		b.Assessment = "needs_attention"
	default:
		b.Assessment = "imbalanced"
	}
}

// isVerticalPush проверяет, является ли упражнение вертикальным Push
func isVerticalPush(name string) bool {
	verticalPushKeywords := []string{
		"жим стоя", "жим над головой", "армейский жим", "жим гантелей стоя",
		"push press", "overhead press", "military press",
		"жим арнольда", "жим сидя",
	}
	return containsAny(name, verticalPushKeywords)
}

// isVerticalPull проверяет, является ли упражнение вертикальным Pull
func isVerticalPull(name string) bool {
	verticalPullKeywords := []string{
		"подтягивания", "подтягивание", "тяга верхнего блока", "тяга сверху",
		"lat pulldown", "pull-up", "pullup", "chin-up", "chinup",
		"тяга вертикального блока",
	}
	return containsAny(name, verticalPullKeywords)
}

// isUnilateral проверяет, является ли упражнение односторонним
func isUnilateral(name string) bool {
	unilateralKeywords := []string{
		"выпад", "болгарский", "сплит", "на одной ноге", "одной ногой",
		"одной рукой", "одной гантелей", "попеременно",
		"lunge", "split squat", "single leg", "single arm", "one arm", "one leg",
		"step-up", "степ-ап",
	}
	return containsAny(name, unilateralKeywords)
}

// containsAny проверяет, содержит ли строка любое из ключевых слов (case-insensitive)
func containsAny(s string, keywords []string) bool {
	sLower := toLower(s)
	for _, kw := range keywords {
		if contains(sLower, toLower(kw)) {
			return true
		}
	}
	return false
}

// toLower - простой lowercase (для русских и английских букв)
func toLower(s string) string {
	result := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			r = r + 32
		} else if r >= 'А' && r <= 'Я' {
			r = r + 32
		} else if r == 'Ё' {
			r = 'ё'
		}
		result = append(result, r)
	}
	return string(result)
}

// contains - проверка подстроки
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

// findSubstring - поиск подстроки
func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// BalanceReport - отчёт по балансу для экспорта
type BalanceReport struct {
	ProgramName string           `json:"program_name"`
	ClientName  string           `json:"client_name"`
	TotalWeeks  int              `json:"total_weeks"`
	Weekly      []MovementBalance `json:"weekly_balance"`  // Баланс по неделям
	Overall     MovementBalance   `json:"overall_balance"` // Общий баланс программы
}

// CalculateProgramBalance вычисляет баланс для всей программы
func CalculateProgramBalance(program *GeneratedProgram) *MovementBalance {
	var allExercises []GeneratedExercise

	for _, week := range program.Weeks {
		for _, day := range week.Days {
			allExercises = append(allExercises, day.Exercises...)
		}
	}

	return CalculateBalance(allExercises)
}

// CalculateWeekBalance вычисляет баланс для одной недели
func CalculateWeekBalance(week *GeneratedWeek) *MovementBalance {
	var exercises []GeneratedExercise

	for _, day := range week.Days {
		exercises = append(exercises, day.Exercises...)
	}

	return CalculateBalance(exercises)
}

// CalculateDayBalance вычисляет баланс для одного дня
func CalculateDayBalance(day *GeneratedDay) *MovementBalance {
	return CalculateBalance(day.Exercises)
}

// GetBalanceReport создаёт полный отчёт по балансу программы
func GetBalanceReport(program *GeneratedProgram) *BalanceReport {
	report := &BalanceReport{
		ProgramName: string(program.Goal),
		ClientName:  program.ClientName,
		TotalWeeks:  program.TotalWeeks,
		Weekly:      make([]MovementBalance, 0, len(program.Weeks)),
	}

	for i := range program.Weeks {
		weekBalance := CalculateWeekBalance(&program.Weeks[i])
		report.Weekly = append(report.Weekly, *weekBalance)
	}

	overallBalance := CalculateProgramBalance(program)
	report.Overall = *overallBalance

	return report
}

// FormatBalanceReport форматирует отчёт о балансе для Telegram
func FormatBalanceReport(b *MovementBalance) string {
	var result string

	result = "📊 *Баланс паттернов движения*\n\n"

	// Push/Pull
	result += fmt.Sprintf("*Push/Pull* (верх тела)\n")
	result += fmt.Sprintf("  Push: %d сетов | Pull: %d сетов\n", b.PushSets, b.PullSets)
	result += fmt.Sprintf("  Ratio: %.2f:1 %s\n\n", b.PushPullRatio, getStatusEmoji(b.PushPullStatus))

	// Quad/Hip
	result += fmt.Sprintf("*Quad/Hip* (низ тела)\n")
	result += fmt.Sprintf("  Quad: %d сетов | Hip: %d сетов\n", b.QuadSets, b.HipSets)
	result += fmt.Sprintf("  Ratio: %.2f:1 %s\n\n", b.QuadHipRatio, getStatusEmoji(b.QuadHipStatus))

	// H/V детализация
	if b.HorizontalPushSets > 0 || b.VerticalPushSets > 0 {
		result += fmt.Sprintf("*Horiz/Vert Push*: %d/%d (%.1f:1)\n",
			b.HorizontalPushSets, b.VerticalPushSets, b.HVPushRatio)
	}
	if b.HorizontalPullSets > 0 || b.VerticalPullSets > 0 {
		result += fmt.Sprintf("*Horiz/Vert Pull*: %d/%d (%.1f:1)\n",
			b.HorizontalPullSets, b.VerticalPullSets, b.HVPullRatio)
	}

	// Bi/Uni
	if b.BilateralLegSets > 0 || b.UnilateralLegSets > 0 {
		result += fmt.Sprintf("*Bi/Uni (ноги)*: %d/%d (%.1f:1)\n",
			b.BilateralLegSets, b.UnilateralLegSets, b.BiUniRatio)
	}

	result += "\n"

	// Core
	if b.CoreSets > 0 {
		result += fmt.Sprintf("Core: %d сетов\n", b.CoreSets)
	}

	// Общая оценка
	result += fmt.Sprintf("\n*Оценка: %d/100* %s\n", b.OverallScore, getAssessmentEmoji(b.Assessment))

	// Рекомендации
	if len(b.Recommendations) > 0 {
		result += "\n*Рекомендации:*\n"
		for _, rec := range b.Recommendations {
			result += fmt.Sprintf("• %s\n", rec)
		}
	}

	return result
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

// ===============================================
// ОПТИМИЗАТОР БАЛАНСА
// ===============================================

// BalanceOptimizer оптимизирует баланс паттернов при генерации программы
type BalanceOptimizer struct {
	targetRatios BalanceTargets
}

// BalanceTargets — целевые соотношения для идеального баланса
type BalanceTargets struct {
	PushPullRatio float64 // Идеал Push:Pull (1.0 = 1:1)
	QuadHipRatio  float64 // Идеал Quad:Hip (1.0 = 1:1)
	HVPushRatio   float64 // Идеал Horizontal:Vertical Push (1.0 = 1:1)
	HVPullRatio   float64 // Идеал Horizontal:Vertical Pull (1.0 = 1:1)
	BiUniRatio    float64 // Идеал Bilateral:Unilateral (1.5 = 1.5:1)
}

// DefaultBalanceTargets — оптимальные целевые соотношения
var DefaultBalanceTargets = BalanceTargets{
	PushPullRatio: 1.0,  // Идеальный Push:Pull = 1:1
	QuadHipRatio:  1.0,  // Идеальный Quad:Hip = 1:1
	HVPushRatio:   1.0,  // Горизонтальный:Вертикальный Push = 1:1
	HVPullRatio:   1.0,  // Горизонтальный:Вертикальный Pull = 1:1
	BiUniRatio:    1.5,  // Bilateral:Unilateral = 1.5:1
}

// NewBalanceOptimizer создаёт оптимизатор с заданными целями
func NewBalanceOptimizer(targets *BalanceTargets) *BalanceOptimizer {
	if targets == nil {
		targets = &DefaultBalanceTargets
	}
	return &BalanceOptimizer{targetRatios: *targets}
}

// BalanceDeficit описывает дефицит конкретного паттерна
type BalanceDeficit struct {
	Category     MovementCategory // Категория с дефицитом
	SubCategory  SubCategory      // Подкатегория (для H/V)
	NeededSets   int              // Сколько сетов нужно добавить
	IsUnilateral bool             // Нужны унилатеральные упражнения
	Priority     int              // Приоритет (1-10, где 10 — критический)
}

// AnalyzeDeficits анализирует баланс и возвращает дефициты
func (o *BalanceOptimizer) AnalyzeDeficits(balance *MovementBalance) []BalanceDeficit {
	var deficits []BalanceDeficit
	t := BalanceThresholds

	// Push/Pull дефицит
	if balance.PushSets > 0 || balance.PullSets > 0 {
		pushPullDeficit := o.calculatePushPullDeficit(balance, t)
		if pushPullDeficit != nil {
			deficits = append(deficits, *pushPullDeficit)
		}
	}

	// Quad/Hip дефицит
	if balance.QuadSets > 0 || balance.HipSets > 0 {
		quadHipDeficit := o.calculateQuadHipDeficit(balance, t)
		if quadHipDeficit != nil {
			deficits = append(deficits, *quadHipDeficit)
		}
	}

	// H/V Push дефицит
	if balance.HorizontalPushSets > 0 || balance.VerticalPushSets > 0 {
		hvPushDeficit := o.calculateHVPushDeficit(balance)
		if hvPushDeficit != nil {
			deficits = append(deficits, *hvPushDeficit)
		}
	}

	// H/V Pull дефицит
	if balance.HorizontalPullSets > 0 || balance.VerticalPullSets > 0 {
		hvPullDeficit := o.calculateHVPullDeficit(balance)
		if hvPullDeficit != nil {
			deficits = append(deficits, *hvPullDeficit)
		}
	}

	// Unilateral дефицит
	if (balance.QuadSets > 0 || balance.HipSets > 0) && balance.UnilateralLegSets == 0 {
		totalLegSets := balance.QuadSets + balance.HipSets
		neededUni := totalLegSets / 4 // Минимум 25% унилатеральных
		if neededUni < 2 {
			neededUni = 2
		}
		deficits = append(deficits, BalanceDeficit{
			Category:     CategoryQuadDominant, // По умолчанию квады (выпады)
			IsUnilateral: true,
			NeededSets:   neededUni,
			Priority:     6,
		})
	}

	// Core дефицит
	totalUpperLower := balance.PushSets + balance.PullSets + balance.QuadSets + balance.HipSets
	if totalUpperLower > 0 && balance.CoreSets < 3 {
		deficits = append(deficits, BalanceDeficit{
			Category:   CategoryCore,
			NeededSets: 3 - balance.CoreSets,
			Priority:   4,
		})
	}

	// Сортируем по приоритету (высший первый)
	sortDeficitsByPriority(deficits)

	return deficits
}

// calculatePushPullDeficit вычисляет дефицит Push или Pull
func (o *BalanceOptimizer) calculatePushPullDeficit(balance *MovementBalance, t struct {
	PushPullIdealMin  float64
	PushPullIdealMax  float64
	PushPullAcceptMin float64
	PushPullAcceptMax float64
	QuadHipIdealMin   float64
	QuadHipIdealMax   float64
	QuadHipAcceptMin  float64
	QuadHipAcceptMax  float64
	HVIdealMin        float64
	HVIdealMax        float64
	HVAcceptMin       float64
	HVAcceptMax       float64
	BiUniIdealMin     float64
	BiUniIdealMax     float64
	BiUniAcceptMin    float64
	BiUniAcceptMax    float64
}) *BalanceDeficit {
	// Вычисляем идеальное количество
	targetPush := float64(balance.PullSets) * o.targetRatios.PushPullRatio
	targetPull := float64(balance.PushSets) / o.targetRatios.PushPullRatio

	if balance.PushPullRatio < t.PushPullIdealMin {
		// Не хватает Push
		needed := int(targetPush) - balance.PushSets
		if needed < 2 {
			needed = 2
		}
		priority := 8
		if balance.PushPullRatio < t.PushPullAcceptMin {
			priority = 10 // Критический дефицит
		}
		return &BalanceDeficit{
			Category:   CategoryPush,
			NeededSets: needed,
			Priority:   priority,
		}
	}

	if balance.PushPullRatio > t.PushPullIdealMax {
		// Не хватает Pull
		needed := int(targetPull) - balance.PullSets
		if needed < 2 {
			needed = 2
		}
		priority := 8
		if balance.PushPullRatio > t.PushPullAcceptMax {
			priority = 10
		}
		return &BalanceDeficit{
			Category:   CategoryPull,
			NeededSets: needed,
			Priority:   priority,
		}
	}

	return nil
}

// calculateQuadHipDeficit вычисляет дефицит Quad или Hip
func (o *BalanceOptimizer) calculateQuadHipDeficit(balance *MovementBalance, t struct {
	PushPullIdealMin  float64
	PushPullIdealMax  float64
	PushPullAcceptMin float64
	PushPullAcceptMax float64
	QuadHipIdealMin   float64
	QuadHipIdealMax   float64
	QuadHipAcceptMin  float64
	QuadHipAcceptMax  float64
	HVIdealMin        float64
	HVIdealMax        float64
	HVAcceptMin       float64
	HVAcceptMax       float64
	BiUniIdealMin     float64
	BiUniIdealMax     float64
	BiUniAcceptMin    float64
	BiUniAcceptMax    float64
}) *BalanceDeficit {
	targetQuad := float64(balance.HipSets) * o.targetRatios.QuadHipRatio
	targetHip := float64(balance.QuadSets) / o.targetRatios.QuadHipRatio

	if balance.QuadHipRatio < t.QuadHipIdealMin {
		// Не хватает Quad
		needed := int(targetQuad) - balance.QuadSets
		if needed < 2 {
			needed = 2
		}
		priority := 8
		if balance.QuadHipRatio < t.QuadHipAcceptMin {
			priority = 10
		}
		return &BalanceDeficit{
			Category:   CategoryQuadDominant,
			NeededSets: needed,
			Priority:   priority,
		}
	}

	if balance.QuadHipRatio > t.QuadHipIdealMax {
		// Не хватает Hip
		needed := int(targetHip) - balance.HipSets
		if needed < 2 {
			needed = 2
		}
		priority := 8
		if balance.QuadHipRatio > t.QuadHipAcceptMax {
			priority = 10
		}
		return &BalanceDeficit{
			Category:   CategoryHipDominant,
			NeededSets: needed,
			Priority:   priority,
		}
	}

	return nil
}

// calculateHVPushDeficit вычисляет дефицит H/V Push
func (o *BalanceOptimizer) calculateHVPushDeficit(balance *MovementBalance) *BalanceDeficit {
	t := BalanceThresholds

	if balance.HVPushRatio > t.HVAcceptMax {
		// Не хватает Vertical Push (жим над головой)
		targetVert := int(float64(balance.HorizontalPushSets) / o.targetRatios.HVPushRatio)
		needed := targetVert - balance.VerticalPushSets
		if needed < 2 {
			needed = 2
		}
		return &BalanceDeficit{
			Category:    CategoryPush,
			SubCategory: SubCategoryVerticalPush,
			NeededSets:  needed,
			Priority:    5,
		}
	}

	if balance.HVPushRatio < t.HVAcceptMin && balance.HVPushRatio > 0 {
		// Не хватает Horizontal Push
		targetHoriz := int(float64(balance.VerticalPushSets) * o.targetRatios.HVPushRatio)
		needed := targetHoriz - balance.HorizontalPushSets
		if needed < 2 {
			needed = 2
		}
		return &BalanceDeficit{
			Category:    CategoryPush,
			SubCategory: SubCategoryHorizontalPush,
			NeededSets:  needed,
			Priority:    5,
		}
	}

	return nil
}

// calculateHVPullDeficit вычисляет дефицит H/V Pull
func (o *BalanceOptimizer) calculateHVPullDeficit(balance *MovementBalance) *BalanceDeficit {
	t := BalanceThresholds

	if balance.HVPullRatio > t.HVAcceptMax {
		// Не хватает Vertical Pull (подтягивания)
		targetVert := int(float64(balance.HorizontalPullSets) / o.targetRatios.HVPullRatio)
		needed := targetVert - balance.VerticalPullSets
		if needed < 2 {
			needed = 2
		}
		return &BalanceDeficit{
			Category:    CategoryPull,
			SubCategory: SubCategoryVerticalPull,
			NeededSets:  needed,
			Priority:    5,
		}
	}

	if balance.HVPullRatio < t.HVAcceptMin && balance.HVPullRatio > 0 {
		// Не хватает Horizontal Pull
		targetHoriz := int(float64(balance.VerticalPullSets) * o.targetRatios.HVPullRatio)
		needed := targetHoriz - balance.HorizontalPullSets
		if needed < 2 {
			needed = 2
		}
		return &BalanceDeficit{
			Category:    CategoryPull,
			SubCategory: SubCategoryHorizontalPull,
			NeededSets:  needed,
			Priority:    5,
		}
	}

	return nil
}

// sortDeficitsByPriority сортирует дефициты по приоритету (высший первый)
func sortDeficitsByPriority(deficits []BalanceDeficit) {
	for i := 0; i < len(deficits)-1; i++ {
		for j := i + 1; j < len(deficits); j++ {
			if deficits[j].Priority > deficits[i].Priority {
				deficits[i], deficits[j] = deficits[j], deficits[i]
			}
		}
	}
}

// CorrectiveExercise — корректирующее упражнение для исправления дисбаланса
type CorrectiveExercise struct {
	Name         string           // Название упражнения
	NameRu       string           // Название на русском
	Category     MovementCategory // Категория
	SubCategory  SubCategory      // Подкатегория (для H/V)
	MuscleGroup  MuscleGroupExt   // Целевая мышечная группа
	MovementType MovementType     // Тип движения
	IsUnilateral bool             // Унилатеральное упражнение
	Sets         int              // Рекомендуемое количество подходов
	Reps         string           // Диапазон повторений
	RestSeconds  int              // Отдых между подходами
}

// CorrectiveExercisesDB — база корректирующих упражнений для каждого дефицита
var CorrectiveExercisesDB = map[MovementCategory][]CorrectiveExercise{
	CategoryPush: {
		// Vertical Push
		{
			Name: "overhead_press", NameRu: "Жим штанги стоя",
			Category: CategoryPush, SubCategory: SubCategoryVerticalPush,
			MuscleGroup: MuscleShoulders, MovementType: MovementPush,
			Sets: 3, Reps: "8-10", RestSeconds: 90,
		},
		{
			Name: "db_shoulder_press", NameRu: "Жим гантелей сидя",
			Category: CategoryPush, SubCategory: SubCategoryVerticalPush,
			MuscleGroup: MuscleShoulders, MovementType: MovementPush,
			Sets: 3, Reps: "10-12", RestSeconds: 75,
		},
		{
			Name: "arnold_press", NameRu: "Жим Арнольда",
			Category: CategoryPush, SubCategory: SubCategoryVerticalPush,
			MuscleGroup: MuscleShoulders, MovementType: MovementPush,
			Sets: 3, Reps: "10-12", RestSeconds: 75,
		},
		// Horizontal Push
		{
			Name: "bench_press", NameRu: "Жим штанги лёжа",
			Category: CategoryPush, SubCategory: SubCategoryHorizontalPush,
			MuscleGroup: MuscleChest, MovementType: MovementPush,
			Sets: 3, Reps: "8-10", RestSeconds: 120,
		},
		{
			Name: "db_bench_press", NameRu: "Жим гантелей лёжа",
			Category: CategoryPush, SubCategory: SubCategoryHorizontalPush,
			MuscleGroup: MuscleChest, MovementType: MovementPush,
			Sets: 3, Reps: "10-12", RestSeconds: 90,
		},
		{
			Name: "incline_db_press", NameRu: "Жим гантелей на наклонной скамье",
			Category: CategoryPush, SubCategory: SubCategoryHorizontalPush,
			MuscleGroup: MuscleChest, MovementType: MovementPush,
			Sets: 3, Reps: "10-12", RestSeconds: 75,
		},
		{
			Name: "dips", NameRu: "Отжимания на брусьях",
			Category: CategoryPush, SubCategory: SubCategoryHorizontalPush,
			MuscleGroup: MuscleChest, MovementType: MovementPush,
			Sets: 3, Reps: "8-12", RestSeconds: 90,
		},
	},
	CategoryPull: {
		// Vertical Pull
		{
			Name: "pull_ups", NameRu: "Подтягивания",
			Category: CategoryPull, SubCategory: SubCategoryVerticalPull,
			MuscleGroup: MuscleBack, MovementType: MovementPull,
			Sets: 3, Reps: "6-10", RestSeconds: 120,
		},
		{
			Name: "lat_pulldown", NameRu: "Тяга верхнего блока",
			Category: CategoryPull, SubCategory: SubCategoryVerticalPull,
			MuscleGroup: MuscleBack, MovementType: MovementPull,
			Sets: 3, Reps: "10-12", RestSeconds: 75,
		},
		{
			Name: "chin_ups", NameRu: "Подтягивания обратным хватом",
			Category: CategoryPull, SubCategory: SubCategoryVerticalPull,
			MuscleGroup: MuscleBack, MovementType: MovementPull,
			Sets: 3, Reps: "6-10", RestSeconds: 120,
		},
		// Horizontal Pull
		{
			Name: "barbell_row", NameRu: "Тяга штанги в наклоне",
			Category: CategoryPull, SubCategory: SubCategoryHorizontalPull,
			MuscleGroup: MuscleBack, MovementType: MovementPull,
			Sets: 3, Reps: "8-10", RestSeconds: 90,
		},
		{
			Name: "db_row", NameRu: "Тяга гантели в наклоне",
			Category: CategoryPull, SubCategory: SubCategoryHorizontalPull,
			MuscleGroup: MuscleBack, MovementType: MovementPull,
			Sets: 3, Reps: "10-12", RestSeconds: 75,
		},
		{
			Name: "cable_row", NameRu: "Тяга нижнего блока",
			Category: CategoryPull, SubCategory: SubCategoryHorizontalPull,
			MuscleGroup: MuscleBack, MovementType: MovementPull,
			Sets: 3, Reps: "10-12", RestSeconds: 75,
		},
		{
			Name: "face_pull", NameRu: "Тяга к лицу",
			Category: CategoryPull, SubCategory: SubCategoryHorizontalPull,
			MuscleGroup: MuscleRearDelts, MovementType: MovementPull,
			Sets: 3, Reps: "15-20", RestSeconds: 60,
		},
	},
	CategoryQuadDominant: {
		// Bilateral
		{
			Name: "squat", NameRu: "Приседания со штангой",
			Category: CategoryQuadDominant, MuscleGroup: MuscleQuads,
			MovementType: MovementSquat,
			Sets: 3, Reps: "8-10", RestSeconds: 120,
		},
		{
			Name: "leg_press", NameRu: "Жим ногами",
			Category: CategoryQuadDominant, MuscleGroup: MuscleQuads,
			MovementType: MovementSquat,
			Sets: 3, Reps: "10-12", RestSeconds: 90,
		},
		{
			Name: "goblet_squat", NameRu: "Гоблет-приседания",
			Category: CategoryQuadDominant, MuscleGroup: MuscleQuads,
			MovementType: MovementSquat,
			Sets: 3, Reps: "12-15", RestSeconds: 75,
		},
		// Unilateral
		{
			Name: "lunge", NameRu: "Выпады",
			Category: CategoryQuadDominant, MuscleGroup: MuscleQuads,
			MovementType: MovementLunge, IsUnilateral: true,
			Sets: 3, Reps: "10-12 на ногу", RestSeconds: 75,
		},
		{
			Name: "bulgarian_split_squat", NameRu: "Болгарский сплит-присед",
			Category: CategoryQuadDominant, MuscleGroup: MuscleQuads,
			MovementType: MovementLunge, IsUnilateral: true,
			Sets: 3, Reps: "8-10 на ногу", RestSeconds: 90,
		},
		{
			Name: "step_up", NameRu: "Зашагивания",
			Category: CategoryQuadDominant, MuscleGroup: MuscleQuads,
			MovementType: MovementLunge, IsUnilateral: true,
			Sets: 3, Reps: "10-12 на ногу", RestSeconds: 60,
		},
	},
	CategoryHipDominant: {
		// Bilateral
		{
			Name: "rdl", NameRu: "Румынская тяга",
			Category: CategoryHipDominant, MuscleGroup: MuscleHamstrings,
			MovementType: MovementHinge,
			Sets: 3, Reps: "8-10", RestSeconds: 90,
		},
		{
			Name: "hip_thrust", NameRu: "Ягодичный мост",
			Category: CategoryHipDominant, MuscleGroup: MuscleGlutes,
			MovementType: MovementHinge,
			Sets: 3, Reps: "10-12", RestSeconds: 75,
		},
		{
			Name: "good_morning", NameRu: "Гуд-морнинг",
			Category: CategoryHipDominant, MuscleGroup: MuscleHamstrings,
			MovementType: MovementHinge,
			Sets: 3, Reps: "10-12", RestSeconds: 75,
		},
		{
			Name: "leg_curl", NameRu: "Сгибание ног лёжа",
			Category: CategoryHipDominant, MuscleGroup: MuscleHamstrings,
			MovementType: MovementHinge,
			Sets: 3, Reps: "12-15", RestSeconds: 60,
		},
		// Unilateral
		{
			Name: "single_leg_rdl", NameRu: "Румынская тяга на одной ноге",
			Category: CategoryHipDominant, MuscleGroup: MuscleHamstrings,
			MovementType: MovementHinge, IsUnilateral: true,
			Sets: 3, Reps: "8-10 на ногу", RestSeconds: 75,
		},
		{
			Name: "single_leg_hip_thrust", NameRu: "Ягодичный мост на одной ноге",
			Category: CategoryHipDominant, MuscleGroup: MuscleGlutes,
			MovementType: MovementHinge, IsUnilateral: true,
			Sets: 3, Reps: "10-12 на ногу", RestSeconds: 60,
		},
	},
	CategoryCore: {
		{
			Name: "plank", NameRu: "Планка",
			Category: CategoryCore, MuscleGroup: MuscleCore,
			MovementType: MovementCore,
			Sets: 3, Reps: "30-60 сек", RestSeconds: 45,
		},
		{
			Name: "dead_bug", NameRu: "Мёртвый жук",
			Category: CategoryCore, MuscleGroup: MuscleCore,
			MovementType: MovementCore,
			Sets: 3, Reps: "10-12 на сторону", RestSeconds: 45,
		},
		{
			Name: "pallof_press", NameRu: "Жим Палоффа",
			Category: CategoryCore, MuscleGroup: MuscleCore,
			MovementType: MovementCore,
			Sets: 3, Reps: "10-12 на сторону", RestSeconds: 45,
		},
		{
			Name: "hanging_leg_raise", NameRu: "Подъём ног в висе",
			Category: CategoryCore, MuscleGroup: MuscleCore,
			MovementType: MovementCore,
			Sets: 3, Reps: "10-15", RestSeconds: 60,
		},
	},
}

// GetCorrectiveExercises возвращает корректирующие упражнения для дефицита
func (o *BalanceOptimizer) GetCorrectiveExercises(deficit BalanceDeficit, excludeNames []string) []CorrectiveExercise {
	candidates := CorrectiveExercisesDB[deficit.Category]
	var result []CorrectiveExercise

	for _, ex := range candidates {
		// Проверяем подкатегорию если указана
		if deficit.SubCategory != "" && ex.SubCategory != deficit.SubCategory {
			continue
		}

		// Проверяем унилатеральность если требуется
		if deficit.IsUnilateral && !ex.IsUnilateral {
			continue
		}

		// Проверяем что не в списке исключений
		excluded := false
		for _, name := range excludeNames {
			if ex.Name == name || ex.NameRu == name {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		result = append(result, ex)
	}

	return result
}

// OptimizeWeekBalance оптимизирует баланс недели, возвращая корректирующие упражнения
func (o *BalanceOptimizer) OptimizeWeekBalance(week *GeneratedWeek, usedExerciseNames []string) []CorrectiveExercise {
	balance := CalculateWeekBalance(week)
	deficits := o.AnalyzeDeficits(balance)

	if len(deficits) == 0 || balance.OverallScore >= 90 {
		return nil // Баланс уже хороший
	}

	var corrections []CorrectiveExercise

	// Обрабатываем дефициты по приоритету
	for _, deficit := range deficits {
		if len(corrections) >= 3 {
			// Не добавляем больше 3 корректирующих упражнений за раз
			break
		}

		exercises := o.GetCorrectiveExercises(deficit, usedExerciseNames)
		if len(exercises) > 0 {
			// Выбираем первое подходящее упражнение
			corrections = append(corrections, exercises[0])
			usedExerciseNames = append(usedExerciseNames, exercises[0].Name)
		}
	}

	return corrections
}

// ConvertCorrectiveToGenerated конвертирует корректирующее упражнение в GeneratedExercise
func ConvertCorrectiveToGenerated(corrective CorrectiveExercise, orderNum int) GeneratedExercise {
	return GeneratedExercise{
		OrderNum:     orderNum,
		ExerciseID:   corrective.Name,
		ExerciseName: corrective.NameRu,
		MuscleGroup:  corrective.MuscleGroup,
		MovementType: corrective.MovementType,
		Sets:         corrective.Sets,
		Reps:         corrective.Reps,
		RestSeconds:  corrective.RestSeconds,
		Notes:        "Корректирующее упражнение для баланса",
	}
}

// IsBalanced проверяет, сбалансирован ли набор упражнений
func (o *BalanceOptimizer) IsBalanced(exercises []GeneratedExercise) bool {
	balance := CalculateBalance(exercises)
	return balance.OverallScore >= 85
}

// GetBalanceScore возвращает оценку баланса (0-100)
func (o *BalanceOptimizer) GetBalanceScore(exercises []GeneratedExercise) int {
	balance := CalculateBalance(exercises)
	return balance.OverallScore
}
