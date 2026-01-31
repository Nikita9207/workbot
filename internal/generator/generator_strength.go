package generator

import (
	"fmt"

	"workbot/internal/generator/progression"
	"workbot/internal/models"
)

// StrengthGenerator - генератор программ для максимальной силы
type StrengthGenerator struct {
	selector *ExerciseSelector
	client   *models.ClientProfile
}

// NewStrengthGenerator создаёт новый генератор силы
func NewStrengthGenerator(selector *ExerciseSelector, client *models.ClientProfile) *StrengthGenerator {
	return &StrengthGenerator{
		selector: selector,
		client:   client,
	}
}

// StrengthConfig - конфигурация программы силы
type StrengthConfig struct {
	TotalWeeks       int                          // Всего недель (6-12)
	DaysPerWeek      int                          // Дней в неделю (3-4)
	Focus            string                       // squat/bench/deadlift/all
	ProgressionModel progression.ProgressionModel // Модель прогрессии
	UseAdvanced      bool                         // Использовать расширенную периодизацию
	WavePattern      progression.WavePattern      // Паттерн волновой периодизации (none/three_plus_one/stepped)
}

// Generate генерирует программу силы
func (g *StrengthGenerator) Generate(config StrengthConfig) (*models.GeneratedProgram, error) {
	program := &models.GeneratedProgram{
		ClientID:      g.client.ID,
		ClientName:    g.client.Name,
		Goal:          models.GoalStrength,
		Periodization: models.PeriodBlock,
		TotalWeeks:    config.TotalWeeks,
		DaysPerWeek:   config.DaysPerWeek,
	}

	// Используем расширенную периодизацию если указано
	if config.UseAdvanced {
		return g.generateAdvanced(config)
	}

	// Определяем фазы (блочная периодизация)
	program.Phases = g.definePhasesStrength(config.TotalWeeks)

	// Генерируем недели
	for weekNum := 1; weekNum <= config.TotalWeeks; weekNum++ {
		week := g.generateWeekStrength(weekNum, config)
		// Оптимизируем баланс недели
		g.ensureWeekBalance(&week)
		program.Weeks = append(program.Weeks, week)
	}

	// Считаем статистику
	program.Statistics = g.calculateStats(program)

	return program, nil
}

// generateAdvanced генерирует программу с расширенной периодизацией
func (g *StrengthGenerator) generateAdvanced(config StrengthConfig) (*models.GeneratedProgram, error) {
	program := &models.GeneratedProgram{
		ClientID:      g.client.ID,
		ClientName:    g.client.Name,
		Goal:          models.GoalStrength,
		Periodization: models.PeriodBlock,
		TotalWeeks:    config.TotalWeeks,
		DaysPerWeek:   config.DaysPerWeek,
	}

	// Определяем волновой паттерн: из конфига или дефолтный для цели
	wavePattern := config.WavePattern
	if wavePattern == "" {
		wavePattern = progression.GetWavePattern("strength") // По умолчанию 3+1 для силы
	}

	// Автоматический расчёт блоков с выбранным волновым паттерном
	blocks := progression.CalculateBlockLengthsWithWave("strength", config.TotalWeeks, config.DaysPerWeek, wavePattern)

	// Конвертируем блоки в фазы
	program.Phases = g.blocksToPhases(blocks)

	// Создаём расширенные прогрессии для основных движений
	advancedProgs := make(map[string]*progression.WeightProgression)
	for movement, onePM := range g.client.OnePM {
		model := config.ProgressionModel
		if model == "" {
			model = progression.ProgressionWave // По умолчанию волновая для силы
		}
		advancedProgs[movement] = progression.NewAdvancedProgression(onePM, model, "intermediate_strength")
	}

	// Генерируем недели по блокам
	for _, block := range blocks {
		for weekInBlock := 1; weekInBlock <= block.Weeks; weekInBlock++ {
			weekNum := block.WeekStart + weekInBlock - 1
			week := g.generateAdvancedWeek(weekNum, weekInBlock, block, config, advancedProgs)
			// Оптимизируем баланс недели
			g.ensureWeekBalance(&week)
			program.Weeks = append(program.Weeks, week)
		}
	}

	// Считаем статистику
	program.Statistics = g.calculateStats(program)

	return program, nil
}

// blocksToPhases конвертирует блоки в фазы программы
func (g *StrengthGenerator) blocksToPhases(blocks []progression.CalculatedBlock) []models.ProgramPhase {
	phases := make([]models.ProgramPhase, 0, len(blocks))

	for _, block := range blocks {
		phase := models.ProgramPhase{
			Name:         block.Config.NameRu,
			WeekStart:    block.WeekStart,
			WeekEnd:      block.WeekEnd,
			IntensityMin: block.Config.IntensityStart,
			IntensityMax: block.Config.IntensityEnd,
		}

		switch block.Config.Type {
		case progression.BlockAccumulation:
			phase.Focus = "Объём, техника, рабочая гипертрофия"
			phase.VolumeLevel = "high"
		case progression.BlockTransmutation:
			phase.Focus = "Конверсия объёма в силу"
			phase.VolumeLevel = "medium"
		case progression.BlockRealization:
			phase.Focus = "Выход на пик, максимальные веса"
			phase.VolumeLevel = "low"
		}

		phases = append(phases, phase)
	}

	return phases
}

// generateAdvancedWeek генерирует неделю с расширенной прогрессией
func (g *StrengthGenerator) generateAdvancedWeek(
	weekNum, weekInBlock int,
	block progression.CalculatedBlock,
	config StrengthConfig,
	advancedProgs map[string]*progression.WeightProgression,
) models.GeneratedWeek {
	weekParams := block.WeeklyParams[weekInBlock-1]

	week := models.GeneratedWeek{
		WeekNum:           weekNum,
		PhaseName:         block.Config.NameRu,
		IsDeload:          weekParams.IsDeload,
		IntensityPercent:  weekParams.IntensityPercent,
		VolumePercent:     weekParams.VolumeMultiplier * 100,
		RPETarget:         g.getRPEForBlockType(block.Config.Type, weekParams.IsDeload),
	}

	// Генерируем дни с H/M/L паттерном
	dayTypes := g.getStrengthDayTypes(config.DaysPerWeek, config.Focus)
	for dayNum, dayType := range dayTypes {
		dayIntensity := progression.GetDayIntensity(dayNum+1, config.DaysPerWeek)
		day := g.generateAdvancedDay(dayNum+1, dayType, weekNum, block, weekInBlock, dayIntensity, config, advancedProgs)
		week.Days = append(week.Days, day)
	}

	return week
}

// generateAdvancedDay генерирует день с расширенной прогрессией
func (g *StrengthGenerator) generateAdvancedDay(
	dayNum int,
	dayType string,
	weekNum int,
	block progression.CalculatedBlock,
	weekInBlock int,
	dayIntensity progression.DayIntensity,
	config StrengthConfig,
	advancedProgs map[string]*progression.WeightProgression,
) models.GeneratedDay {
	weekParams := block.WeeklyParams[weekInBlock-1]
	isDeload := weekParams.IsDeload

	day := models.GeneratedDay{
		DayNum: dayNum,
		Name:   g.getDayNameWithIntensity(dayType, dayNum, dayIntensity),
		Type:   dayType,
	}

	// Упражнения для дня
	exercises := g.getExercisesForDay(dayType, string(block.Config.Type), isDeload)

	for orderNum, exDef := range exercises {
		ex := g.createAdvancedExercise(exDef, orderNum+1, block, weekInBlock, dayIntensity, isDeload, advancedProgs)
		day.Exercises = append(day.Exercises, ex)
	}

	// Оценка длительности
	totalSets := 0
	for _, ex := range day.Exercises {
		totalSets += ex.Sets
	}
	day.EstimatedDuration = totalSets * 4

	return day
}

// createAdvancedExercise создаёт упражнение с расширенной прогрессией
func (g *StrengthGenerator) createAdvancedExercise(
	exDef ExerciseDef,
	orderNum int,
	block progression.CalculatedBlock,
	weekInBlock int,
	dayIntensity progression.DayIntensity,
	isDeload bool,
	advancedProgs map[string]*progression.WeightProgression,
) models.GeneratedExercise {
	ex := models.GeneratedExercise{
		OrderNum:     orderNum,
		ExerciseName: exDef.Name,
	}

	// Для основных движений используем расширенную прогрессию
	if exDef.IsMain && exDef.Movement != "" {
		wp, ok := advancedProgs[exDef.Movement]
		if ok {
			params := wp.GetBlockParams(block, weekInBlock, dayIntensity)

			// Применяем базовый процент для вариаций
			if exDef.BasePercent < 100 && exDef.BasePercent > 0 {
				params.Intensity = params.Intensity * exDef.BasePercent / 100
				params.Weight = wp.CalculateWeight(params.Intensity)
			}

			ex.Weight = params.Weight
			ex.WeightPercent = params.Intensity
			ex.Reps = fmt.Sprintf("%d", params.Reps)
			ex.Sets = params.Sets
			ex.RestSeconds = params.RestSeconds
			ex.RPE = params.RPE
		}
	} else if exDef.IsAccessory {
		// Подсобные упражнения
		ex.Sets = g.getAccessorySetsForBlock(block.Config.Type, isDeload)
		ex.Reps = g.getAccessoryRepsForBlock(block.Config.Type, isDeload)
		ex.RestSeconds = 90
		ex.RPE = 7.5

		// Если есть связь с основным движением
		if exDef.Movement != "" && exDef.BasePercent > 0 {
			if wp, ok := advancedProgs[exDef.Movement]; ok {
				params := wp.GetBlockParams(block, weekInBlock, dayIntensity)
				adjustedIntensity := params.Intensity * exDef.BasePercent / 100
				ex.Weight = wp.CalculateWeight(adjustedIntensity)
				ex.WeightPercent = adjustedIntensity
			}
		}
	}

	return ex
}

// getDayNameWithIntensity возвращает название дня с типом нагрузки
func (g *StrengthGenerator) getDayNameWithIntensity(dayType string, dayNum int, dayIntensity progression.DayIntensity) string {
	baseName := g.getDayName(dayType, dayNum)

	intensityNames := map[progression.DayIntensity]string{
		progression.DayHeavy:  "Тяжёлый",
		progression.DayMedium: "Средний",
		progression.DayLight:  "Лёгкий",
	}

	if name, ok := intensityNames[dayIntensity]; ok {
		return fmt.Sprintf("%s (%s)", baseName, name)
	}
	return baseName
}

// getRPEForBlockType возвращает целевой RPE для типа блока
func (g *StrengthGenerator) getRPEForBlockType(blockType progression.BlockType, isDeload bool) float64 {
	if isDeload {
		return 5.0
	}
	switch blockType {
	case progression.BlockAccumulation:
		return 7.5
	case progression.BlockTransmutation:
		return 8.5
	case progression.BlockRealization:
		return 9.5
	default:
		return 8.0
	}
}

// getAccessorySetsForBlock возвращает подходы для подсобки по типу блока
func (g *StrengthGenerator) getAccessorySetsForBlock(blockType progression.BlockType, isDeload bool) int {
	if isDeload {
		return 2
	}
	switch blockType {
	case progression.BlockAccumulation:
		return 4
	case progression.BlockTransmutation:
		return 3
	case progression.BlockRealization:
		return 2
	default:
		return 3
	}
}

// getAccessoryRepsForBlock возвращает повторения для подсобки по типу блока
func (g *StrengthGenerator) getAccessoryRepsForBlock(blockType progression.BlockType, isDeload bool) string {
	if isDeload {
		return "8"
	}
	switch blockType {
	case progression.BlockAccumulation:
		return "10-12"
	case progression.BlockTransmutation:
		return "8-10"
	case progression.BlockRealization:
		return "6-8"
	default:
		return "8-10"
	}
}

// definePhasesStrength определяет фазы для силовой программы
func (g *StrengthGenerator) definePhasesStrength(totalWeeks int) []models.ProgramPhase {
	// Трёхфазная блочная периодизация:
	// 1. Накопление: объём, 65-75%
	// 2. Трансформация: сила, 75-85%
	// 3. Реализация: пик, 85-100%

	accumWeeks := totalWeeks / 3
	transWeeks := totalWeeks / 3
	_ = totalWeeks - accumWeeks - transWeeks // realWeeks используется в WeekEnd

	return []models.ProgramPhase{
		{
			Name:         "Накопление",
			WeekStart:    1,
			WeekEnd:      accumWeeks,
			Focus:        "Объём, техника, рабочая гипертрофия",
			IntensityMin: 65,
			IntensityMax: 75,
			VolumeLevel:  "high",
		},
		{
			Name:         "Трансформация",
			WeekStart:    accumWeeks + 1,
			WeekEnd:      accumWeeks + transWeeks,
			Focus:        "Конверсия объёма в силу",
			IntensityMin: 75,
			IntensityMax: 85,
			VolumeLevel:  "medium",
		},
		{
			Name:         "Реализация",
			WeekStart:    accumWeeks + transWeeks + 1,
			WeekEnd:      totalWeeks,
			Focus:        "Выход на пик, максимальные веса",
			IntensityMin: 85,
			IntensityMax: 100,
			VolumeLevel:  "low",
		},
	}
}

// generateWeekStrength генерирует неделю для силы
func (g *StrengthGenerator) generateWeekStrength(weekNum int, config StrengthConfig) models.GeneratedWeek {
	phase := g.getPhaseForWeek(weekNum, config.TotalWeeks)
	isDeload := g.isDeloadWeek(weekNum, config.TotalWeeks)

	week := models.GeneratedWeek{
		WeekNum:   weekNum,
		PhaseName: phase,
		IsDeload:  isDeload,
	}

	// Расчёт параметров недели
	if isDeload {
		week.IntensityPercent = 60
		week.VolumePercent = 50
		week.RPETarget = 6
	} else {
		week.IntensityPercent = g.getIntensityForWeek(weekNum, phase, config.TotalWeeks)
		week.VolumePercent = g.getVolumeForPhase(phase)
		week.RPETarget = g.getRPEForPhase(phase)
	}

	// Генерируем дни (силовой сплит)
	dayTypes := g.getStrengthDayTypes(config.DaysPerWeek, config.Focus)
	for dayNum, dayType := range dayTypes {
		day := g.generateDayStrength(dayNum+1, dayType, weekNum, config, isDeload)
		week.Days = append(week.Days, day)
	}

	return week
}

// getStrengthDayTypes возвращает типы дней для силовой программы
func (g *StrengthGenerator) getStrengthDayTypes(daysPerWeek int, focus string) []string {
	switch daysPerWeek {
	case 3:
		// 3 дня: присед, жим, тяга
		return []string{"squat_day", "bench_day", "deadlift_day"}
	case 4:
		// 4 дня: верх/низ x2
		return []string{"squat_day", "bench_day", "deadlift_day", "bench_day_light"}
	default:
		return []string{"squat_day", "bench_day", "deadlift_day"}
	}
}

// generateDayStrength генерирует силовой тренировочный день
func (g *StrengthGenerator) generateDayStrength(dayNum int, dayType string, weekNum int, config StrengthConfig, isDeload bool) models.GeneratedDay {
	day := models.GeneratedDay{
		DayNum: dayNum,
		Name:   g.getDayName(dayType, dayNum),
		Type:   dayType,
	}

	phase := g.getPhaseForWeek(weekNum, config.TotalWeeks)

	// Создаём прогрессии для основных движений
	weightProgs := make(map[string]*progression.WeightProgression)
	for movement, onePM := range g.client.OnePM {
		weightProgs[movement] = progression.NewWeightProgression(onePM)
	}

	// Упражнения в зависимости от типа дня
	exercises := g.getExercisesForDay(dayType, phase, isDeload)

	for orderNum, exDef := range exercises {
		ex := g.createStrengthExercise(exDef, orderNum+1, weekNum, config.TotalWeeks, phase, isDeload, weightProgs)
		day.Exercises = append(day.Exercises, ex)
	}

	// Оценка длительности
	totalSets := 0
	for _, ex := range day.Exercises {
		totalSets += ex.Sets
	}
	day.EstimatedDuration = totalSets * 4 // ~4 мин на подход (с отдыхом)

	return day
}

// ExerciseDef - определение упражнения для силовой программы
type ExerciseDef struct {
	Name        string
	Movement    string  // squat/bench/deadlift/ohp
	IsMain      bool    // Основное соревновательное движение
	IsAccessory bool    // Подсобка
	BasePercent float64 // Базовый % от 1ПМ
}

// getExercisesForDay возвращает упражнения для типа дня
// Сбалансированная силовая программа: основное движение + подсобка для слабых мест + антагонисты
// Адаптировано под пол: женщинам больше ягодиц/ног, меньше верха
func (g *StrengthGenerator) getExercisesForDay(dayType, phase string, isDeload bool) []ExerciseDef {
	isFemale := g.client.Gender == "female" || g.client.Gender == "женский" || g.client.Gender == "ж"

	switch dayType {
	case "squat_day":
		// Присед + ноги/ягодицы
		exercises := []ExerciseDef{
			{Name: "Приседания со штангой", Movement: "squat", IsMain: true, BasePercent: 100},
		}
		if !isDeload {
			if isFemale {
				// Женский вариант: больше ягодиц
				exercises = append(exercises,
					ExerciseDef{Name: "Ягодичный мост со штангой", Movement: "", IsAccessory: true},     // Ягодицы - приоритет
					ExerciseDef{Name: "Болгарские выпады", Movement: "", IsAccessory: true},             // Ноги + ягодицы
					ExerciseDef{Name: "Сгибание ног лёжа", Movement: "", IsAccessory: true},             // Задняя поверхность
					ExerciseDef{Name: "Отведение ноги в кроссовере", Movement: "", IsAccessory: true},   // Ягодицы изоляция
				)
			} else {
				// Мужской вариант
				exercises = append(exercises,
					ExerciseDef{Name: "Жим ногами", Movement: "", IsAccessory: true},           // Ноги без нагрузки на спину
					ExerciseDef{Name: "Сгибание ног лёжа", Movement: "", IsAccessory: true},    // Задняя поверхность
					ExerciseDef{Name: "Жим гантелей сидя", Movement: "ohp", IsAccessory: true, BasePercent: 70}, // Плечи
					ExerciseDef{Name: "Подъём на носки стоя", Movement: "", IsAccessory: true}, // Икры
				)
			}
		}
		return exercises

	case "bench_day":
		// Жим + грудь/трицепс + плечи
		exercises := []ExerciseDef{
			{Name: "Жим штанги лёжа", Movement: "bench", IsMain: true, BasePercent: 100},
		}
		if !isDeload {
			if isFemale {
				// Женский вариант: меньше объёма на грудь, добавляем спину и ягодицы
				exercises = append(exercises,
					ExerciseDef{Name: "Жим гантелей на наклонной", Movement: "bench", IsAccessory: true, BasePercent: 65},
					ExerciseDef{Name: "Тяга верхнего блока", Movement: "", IsAccessory: true},          // Спина
					ExerciseDef{Name: "Разведение гантелей в стороны", Movement: "", IsAccessory: true}, // Плечи
					ExerciseDef{Name: "Гиперэкстензия", Movement: "", IsAccessory: true},                // Поясница + ягодицы
				)
			} else {
				// Мужской вариант
				exercises = append(exercises,
					ExerciseDef{Name: "Жим гантелей на наклонной", Movement: "bench", IsAccessory: true, BasePercent: 65}, // Верх груди
					ExerciseDef{Name: "Жим штанги стоя", Movement: "ohp", IsAccessory: true, BasePercent: 80},             // Плечи
					ExerciseDef{Name: "Французский жим лёжа", Movement: "", IsAccessory: true},                            // Трицепс
					ExerciseDef{Name: "Разведение гантелей в стороны", Movement: "", IsAccessory: true},                   // Средние дельты
				)
			}
		}
		return exercises

	case "bench_day_light":
		// Лёгкий жим + верх спины
		if isFemale {
			// Женский вариант: больше спины и ягодиц
			return []ExerciseDef{
				{Name: "Жим штанги лёжа (темповый)", Movement: "bench", IsMain: true, BasePercent: 75},
				{Name: "Тяга верхнего блока широким хватом", Movement: "", IsAccessory: true},
				{Name: "Тяга горизонтального блока", Movement: "", IsAccessory: true},
				{Name: "Ягодичный мост в тренажёре", Movement: "", IsAccessory: true},
				{Name: "Планка", Movement: "", IsAccessory: true},
			}
		}
		// Мужской вариант
		return []ExerciseDef{
			{Name: "Жим штанги лёжа (темповый)", Movement: "bench", IsMain: true, BasePercent: 75},
			{Name: "Тяга верхнего блока широким хватом", Movement: "", IsAccessory: true},
			{Name: "Тяга гантели в наклоне", Movement: "", IsAccessory: true},
			{Name: "Сгибания на бицепс со штангой", Movement: "", IsAccessory: true},
			{Name: "Молотковые сгибания", Movement: "", IsAccessory: true},
		}

	case "deadlift_day":
		// Тяга + спина + пресс
		exercises := []ExerciseDef{
			{Name: "Становая тяга", Movement: "deadlift", IsMain: true, BasePercent: 100},
		}
		if !isDeload {
			if isFemale {
				// Женский вариант: румынская тяга + больше ягодиц
				exercises = append(exercises,
					ExerciseDef{Name: "Румынская тяга", Movement: "", IsAccessory: true},                // Задняя поверхность + ягодицы
					ExerciseDef{Name: "Тяга верхнего блока", Movement: "", IsAccessory: true},           // Спина
					ExerciseDef{Name: "Ягодичный мост со штангой", Movement: "", IsAccessory: true},     // Ягодицы
					ExerciseDef{Name: "Скручивания на блоке", Movement: "", IsAccessory: true},          // Пресс
				)
			} else {
				// Мужской вариант
				exercises = append(exercises,
					ExerciseDef{Name: "Тяга штанги в наклоне", Movement: "", IsAccessory: true},            // Широчайшие
					ExerciseDef{Name: "Подтягивания", Movement: "", IsAccessory: true},                     // Спина + бицепс
					ExerciseDef{Name: "Сгибания на бицепс с гантелями", Movement: "", IsAccessory: true},   // Бицепс
					ExerciseDef{Name: "Скручивания на блоке", Movement: "", IsAccessory: true},             // Пресс
				)
			}
		}
		return exercises

	default:
		return nil
	}
}

// createStrengthExercise создаёт упражнение для силовой программы
func (g *StrengthGenerator) createStrengthExercise(
	exDef ExerciseDef,
	orderNum, weekNum, totalWeeks int,
	phase string,
	isDeload bool,
	weightProgs map[string]*progression.WeightProgression,
) models.GeneratedExercise {

	ex := models.GeneratedExercise{
		OrderNum:     orderNum,
		ExerciseName: exDef.Name,
	}

	// Для основных движений используем прогрессию
	if exDef.IsMain && exDef.Movement != "" {
		wp, ok := weightProgs[exDef.Movement]
		if ok {
			params := wp.GetStrengthParams(weekNum, phase)

			if isDeload {
				params.Intensity = params.Intensity * 0.7
				params.Weight = wp.CalculateWeight(params.Intensity)
				params.Sets = 3
				params.Reps = 5
			}

			// Применяем базовый процент (для вариаций)
			if exDef.BasePercent < 100 {
				params.Intensity = params.Intensity * exDef.BasePercent / 100
				params.Weight = wp.CalculateWeight(params.Intensity)
			}

			ex.Weight = params.Weight
			ex.WeightPercent = params.Intensity
			ex.Reps = fmt.Sprintf("%d", params.Reps)
			ex.Sets = params.Sets
			ex.RestSeconds = params.RestSeconds
			ex.RPE = params.RPE
		}
	} else if exDef.IsAccessory {
		// Подсобные упражнения
		ex.Sets = g.getAccessorySets(phase, isDeload)
		ex.Reps = g.getAccessoryReps(phase, isDeload)
		ex.RestSeconds = 90
		ex.RPE = 7.5

		// Если есть связь с основным движением
		if exDef.Movement != "" && exDef.BasePercent > 0 {
			if wp, ok := weightProgs[exDef.Movement]; ok {
				baseParams := wp.GetStrengthParams(weekNum, phase)
				adjustedIntensity := baseParams.Intensity * exDef.BasePercent / 100
				ex.Weight = wp.CalculateWeight(adjustedIntensity)
				ex.WeightPercent = adjustedIntensity
			}
		}
	}

	return ex
}

// === Вспомогательные методы ===

func (g *StrengthGenerator) getPhaseForWeek(weekNum, totalWeeks int) string {
	accumWeeks := totalWeeks / 3
	transWeeks := totalWeeks / 3

	if weekNum <= accumWeeks {
		return "accumulation"
	}
	if weekNum <= accumWeeks+transWeeks {
		return "transmutation"
	}
	return "realization"
}

func (g *StrengthGenerator) isDeloadWeek(weekNum, totalWeeks int) bool {
	// Разгрузка каждые 4 недели
	return weekNum%4 == 0 && weekNum < totalWeeks
}

func (g *StrengthGenerator) getIntensityForWeek(weekNum int, phase string, totalWeeks int) float64 {
	switch phase {
	case "accumulation":
		return 65 + float64(weekNum)*2.5 // 65 → 75
	case "transmutation":
		accumWeeks := totalWeeks / 3
		weekInPhase := weekNum - accumWeeks
		return 75 + float64(weekInPhase)*2.5 // 75 → 85
	case "realization":
		accumWeeks := totalWeeks / 3
		transWeeks := totalWeeks / 3
		weekInPhase := weekNum - accumWeeks - transWeeks
		return 85 + float64(weekInPhase)*3 // 85 → 100
	}
	return 75
}

func (g *StrengthGenerator) getVolumeForPhase(phase string) float64 {
	switch phase {
	case "accumulation":
		return 100
	case "transmutation":
		return 80
	case "realization":
		return 60
	}
	return 80
}

func (g *StrengthGenerator) getRPEForPhase(phase string) float64 {
	switch phase {
	case "accumulation":
		return 7.5
	case "transmutation":
		return 8.5
	case "realization":
		return 9.5
	}
	return 8
}

func (g *StrengthGenerator) getDayName(dayType string, dayNum int) string {
	names := map[string]string{
		"squat_day":       "Присед",
		"bench_day":       "Жим лёжа",
		"bench_day_light": "Жим лёжа (лёгкий)",
		"deadlift_day":    "Становая тяга",
	}
	if name, ok := names[dayType]; ok {
		return fmt.Sprintf("День %d — %s", dayNum, name)
	}
	return fmt.Sprintf("День %d", dayNum)
}

// ensureWeekBalance проверяет и корректирует баланс недели
func (g *StrengthGenerator) ensureWeekBalance(week *models.GeneratedWeek) {
	optimizer := models.NewBalanceOptimizer(nil)

	// Собираем все упражнения недели
	var allExercises []models.GeneratedExercise
	usedNames := make([]string, 0)
	for _, day := range week.Days {
		for _, ex := range day.Exercises {
			allExercises = append(allExercises, ex)
			usedNames = append(usedNames, ex.ExerciseName)
		}
	}

	// Анализируем баланс
	balance := models.CalculateBalance(allExercises)
	if balance.OverallScore >= 85 {
		return // Баланс уже хороший
	}

	// Получаем дефициты
	deficits := optimizer.AnalyzeDeficits(balance)
	if len(deficits) == 0 {
		return
	}

	// Добавляем корректирующие упражнения в подходящие дни
	for _, deficit := range deficits {
		if deficit.Priority < 6 {
			continue // Пропускаем низкоприоритетные
		}

		correctives := optimizer.GetCorrectiveExercises(deficit, usedNames)
		if len(correctives) == 0 {
			continue
		}

		// Находим лучший день для упражнения
		dayIdx := g.findBestDayForCategory(week, deficit.Category)
		if dayIdx < 0 || dayIdx >= len(week.Days) {
			continue
		}

		// Добавляем упражнение
		corrEx := correctives[0]
		genEx := models.ConvertCorrectiveToGenerated(corrEx, len(week.Days[dayIdx].Exercises)+1)
		week.Days[dayIdx].Exercises = append(week.Days[dayIdx].Exercises, genEx)
		usedNames = append(usedNames, corrEx.NameRu)
	}
}

// findBestDayForCategory находит лучший день для добавления упражнения
func (g *StrengthGenerator) findBestDayForCategory(week *models.GeneratedWeek, category models.MovementCategory) int {
	// Маппинг категорий на типы дней
	preferredDays := map[models.MovementCategory][]string{
		models.CategoryPush:         {"bench_day", "bench_day_light"},
		models.CategoryPull:         {"deadlift_day", "bench_day_light"},
		models.CategoryQuadDominant: {"squat_day"},
		models.CategoryHipDominant:  {"deadlift_day", "squat_day"},
		models.CategoryCore:         {"squat_day", "deadlift_day"},
	}

	preferred, ok := preferredDays[category]
	if !ok {
		return 0
	}

	for _, pref := range preferred {
		for i, day := range week.Days {
			if day.Type == pref {
				return i
			}
		}
	}

	return 0
}

func (g *StrengthGenerator) getAccessorySets(phase string, isDeload bool) int {
	if isDeload {
		return 2
	}
	switch phase {
	case "accumulation":
		return 4
	case "transmutation":
		return 3
	case "realization":
		return 2
	}
	return 3
}

func (g *StrengthGenerator) getAccessoryReps(phase string, isDeload bool) string {
	if isDeload {
		return "8"
	}
	switch phase {
	case "accumulation":
		return "10-12"
	case "transmutation":
		return "8-10"
	case "realization":
		return "6-8"
	}
	return "8-10"
}

func (g *StrengthGenerator) calculateStats(program *models.GeneratedProgram) models.ProgramStats {
	stats := models.ProgramStats{
		SetsPerMuscle: make(map[models.MuscleGroupExt]int),
	}

	for _, week := range program.Weeks {
		for _, day := range week.Days {
			stats.TotalWorkouts++
			for _, ex := range day.Exercises {
				stats.TotalSets += ex.Sets
				if ex.Weight > 0 {
					reps := 5
					fmt.Sscanf(ex.Reps, "%d", &reps)
					stats.TotalVolume += ex.Weight * float64(ex.Sets*reps)
				}
				stats.SetsPerMuscle[ex.MuscleGroup] += ex.Sets
			}
		}
	}

	if stats.TotalWorkouts > 0 {
		stats.AvgWorkoutDur = (stats.TotalSets * 4) / stats.TotalWorkouts
	}

	// Рассчитываем баланс паттернов движения
	stats.MovementBalance = models.CalculateProgramBalance(program)

	return stats
}
