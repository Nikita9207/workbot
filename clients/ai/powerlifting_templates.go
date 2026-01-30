package ai

import (
	"embed"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

//go:embed templates/*.json
var embeddedTemplates embed.FS

// TemplateExerciseSet подход в упражнении
type TemplateExerciseSet struct {
	Percent float64 `json:"percent"` // Процент от 1ПМ
	Reps    int     `json:"reps"`    // Повторения
	Sets    int     `json:"sets"`    // Подходы
	Weight  float64 `json:"weight"`  // Абсолютный вес (если указан)
}

// TemplateExercise упражнение в шаблоне
type TemplateExercise struct {
	Name     string                `json:"name"`
	Type     string                `json:"type"`      // competition, accessory
	LoadType string                `json:"load_type"` // Легкая, Средняя, Тяжелая
	Sets     []TemplateExerciseSet `json:"sets"`
	SetsReps []int                 `json:"sets_reps"` // Для подсобки (просто повторения)
	Stats    *ExerciseStats        `json:"stats,omitempty"`
}

// ExerciseStats статистика упражнения
type ExerciseStats struct {
	Tonnage   float64 `json:"tonnage"`
	AvgWeight float64 `json:"avg_weight"`
	Intensity float64 `json:"intensity"`
	PM        float64 `json:"pm"`  // 1ПМ для этого упражнения
	KPS       int     `json:"kps"` // Количество подъёмов штанги
}

// TemplateWorkout тренировка
type TemplateWorkout struct {
	WorkoutNum int                `json:"workout_num"`
	Date       string             `json:"date,omitempty"`
	Exercises  []TemplateExercise `json:"exercises"`
}

// TemplateWeek неделя
type TemplateWeek struct {
	WeekNum  int               `json:"week_num"`
	Workouts []TemplateWorkout `json:"workouts,omitempty"`
	Days     []TemplateDay     `json:"days,omitempty"` // Альтернативный формат (Шейко КМС)
}

// TemplateDay день тренировки (для Головинского)
type TemplateDay struct {
	Date      string             `json:"date"`
	Exercises []TemplateExercise `json:"exercises"`
}

// TemplateMicrocycle микроцикл (для Головинского)
type TemplateMicrocycle struct {
	MicrocycleNum int                `json:"microcycle_num,omitempty"` // Старый формат
	MicroNum      int                `json:"micro_num,omitempty"`      // Новый формат
	Stats         *CycleStats        `json:"stats,omitempty"`
	Days          []TemplateDay      `json:"days,omitempty"`
	Workouts      []MicrocycleWorkout `json:"workouts,omitempty"` // Новый формат с workouts
}

// MicrocycleWorkout тренировка в микроцикле
type MicrocycleWorkout struct {
	Day       int                `json:"day"`
	Exercises []TemplateExercise `json:"exercises"`
}

// CycleStats статистика цикла
type CycleStats struct {
	Tonnage   float64 `json:"tonnage"`
	AvgWeight float64 `json:"avg_weight"`
	KPS       int     `json:"kps"`
}

// TemplatePhase фаза (для Шейко)
type TemplatePhase struct {
	Name  string         `json:"name"`
	Weeks []TemplateWeek `json:"weeks"`
}

// RussianCycleExercise упражнение в формате Русского цикла
type RussianCycleExercise struct {
	Name  string                 `json:"name"`
	Weeks []RussianCycleWeekData `json:"weeks"`
}

// RussianCycleWeekData данные недели в Русском цикле
type RussianCycleWeekData struct {
	WeekNum     int                   `json:"week_num"`
	TrainingNum int                   `json:"training_num,omitempty"` // 1 или 2 (для Муравьёва)
	Sets        []TemplateExerciseSet `json:"sets"`
}

// RussianCycleExercises структура exercises для Русского цикла
type RussianCycleExercises struct {
	Squat    *RussianCycleExercise `json:"squat,omitempty"`
	Bench    *RussianCycleExercise `json:"bench,omitempty"`
	Deadlift *RussianCycleExercise `json:"deadlift,omitempty"`
}

// PowerliftingTemplate шаблон программы
type PowerliftingTemplate struct {
	Name        string   `json:"name"`
	Author      string   `json:"author"`
	Level       []string `json:"level"` // 2_разряд, 1_разряд, КМС, МС, МСМК
	Weeks       int      `json:"weeks"`
	DaysPerWeek int      `json:"days_per_week"`
	Type        string   `json:"type"` // троеборье, жим, присед, тяга

	// Разные форматы хранения
	Phases      []TemplatePhase        `json:"phases,omitempty"`      // Шейко 12 недель
	WeeksData   []TemplateWeek         `json:"weeks_data,omitempty"`  // Шейко КМС/МС
	Microcycles []TemplateMicrocycle   `json:"microcycles,omitempty"` // Головинский
	Exercises   *RussianCycleExercises `json:"exercises,omitempty"`   // Русский цикл
}

// TemplateManager менеджер шаблонов
type TemplateManager struct {
	templates map[string]*PowerliftingTemplate
}

// NewTemplateManager создаёт менеджер шаблонов
func NewTemplateManager() (*TemplateManager, error) {
	tm := &TemplateManager{
		templates: make(map[string]*PowerliftingTemplate),
	}

	if err := tm.loadTemplates(); err != nil {
		return nil, err
	}

	return tm, nil
}

// loadTemplates загружает все шаблоны из embedded файлов
func (tm *TemplateManager) loadTemplates() error {
	files, err := embeddedTemplates.ReadDir("templates")
	if err != nil {
		return fmt.Errorf("не удалось прочитать embedded шаблоны: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		data, err := embeddedTemplates.ReadFile("templates/" + file.Name())
		if err != nil {
			return fmt.Errorf("не удалось прочитать файл %s: %w", file.Name(), err)
		}

		var template PowerliftingTemplate
		if err := json.Unmarshal(data, &template); err != nil {
			return fmt.Errorf("не удалось распарсить %s: %w", file.Name(), err)
		}

		tm.templates[template.Name] = &template
	}

	return nil
}

// GetTemplate возвращает шаблон по имени
func (tm *TemplateManager) GetTemplate(name string) (*PowerliftingTemplate, bool) {
	t, ok := tm.templates[name]
	return t, ok
}

// ListTemplates возвращает список доступных шаблонов
func (tm *TemplateManager) ListTemplates() []string {
	names := make([]string, 0, len(tm.templates))
	for name := range tm.templates {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetTemplatesForLevel возвращает шаблоны для уровня атлета
func (tm *TemplateManager) GetTemplatesForLevel(level string) []*PowerliftingTemplate {
	var result []*PowerliftingTemplate
	for _, t := range tm.templates {
		for _, l := range t.Level {
			if l == level {
				result = append(result, t)
				break
			}
		}
	}
	return result
}

// LiftType тип дисциплины
type LiftType string

const (
	LiftTypeFull     LiftType = "powerlifting" // Троеборье
	LiftTypeBench    LiftType = "bench"        // Только жим
	LiftTypeSquat    LiftType = "squat"        // Только присед
	LiftTypeDeadlift LiftType = "deadlift"     // Только тяга
	LiftTypeHipThrust LiftType = "hip_thrust"  // Ягодичный мост
)

// AthleteMaxes 1ПМ атлета по упражнениям
type AthleteMaxes struct {
	Squat    float64 // Присед
	Bench    float64 // Жим лёжа
	Deadlift float64 // Становая тяга
	HipThrust float64 // Ягодичный мост
}

// AthleteLevel уровень атлета
type AthleteLevel string

const (
	LevelNovice AthleteLevel = "novice"    // Новичок
	Level3Rank  AthleteLevel = "3_разряд"  // 3 разряд
	Level2Rank  AthleteLevel = "2_разряд"  // 2 разряд
	Level1Rank  AthleteLevel = "1_разряд"  // 1 разряд
	LevelKMS    AthleteLevel = "КМС"       // Кандидат в мастера спорта
	LevelMS     AthleteLevel = "МС"        // Мастер спорта
	LevelMSMC   AthleteLevel = "МСМК"      // Мастер спорта международного класса
)

// GenerationOptions опции генерации программы
type GenerationOptions struct {
	LiftType         LiftType     // Тип дисциплины
	DaysPerWeek      int          // Количество тренировок в неделю (2-4)
	IncludeAccessory bool         // Включать подсобку
	AthleteLevel     AthleteLevel // Уровень атлета (для автовыбора шаблона)
}

// PLGeneratedSet сгенерированный подход (пауэрлифтинг)
type PLGeneratedSet struct {
	Percent  float64 // Процент от 1ПМ
	Reps     int     // Повторения
	Sets     int     // Подходы
	WeightKg float64 // Вес в кг (рассчитанный)
}

// PLGeneratedExercise сгенерированное упражнение (пауэрлифтинг)
type PLGeneratedExercise struct {
	Name       string         // Название
	Type       string         // competition / accessory
	Sets       []PLGeneratedSet // Подходы
	TotalReps  int            // Общее количество повторений
	Tonnage    float64        // Тоннаж
	AvgPercent float64        // Средняя интенсивность
}

// PLGeneratedWorkout сгенерированная тренировка (пауэрлифтинг)
type PLGeneratedWorkout struct {
	DayNum    int                 // День недели
	Name      string              // Название тренировки
	Exercises []PLGeneratedExercise // Упражнения
	TotalKPS  int                 // КПШ за тренировку
	Tonnage   float64             // Тоннаж за тренировку
}

// PLGeneratedWeek сгенерированная неделя (пауэрлифтинг)
type PLGeneratedWeek struct {
	WeekNum   int                // Номер недели
	Phase     string             // Фаза (подготовительная, соревновательная)
	Workouts  []PLGeneratedWorkout // Тренировки
	TotalKPS  int                // КПШ за неделю
	Tonnage   float64            // Тоннаж за неделю
	AvgIntens float64            // Средняя интенсивность
}

// PLGeneratedProgram сгенерированная программа (пауэрлифтинг)
type PLGeneratedProgram struct {
	Name         string           // Название программы
	AthleteMaxes AthleteMaxes     // 1ПМ атлета
	Weeks        []PLGeneratedWeek  // Недели
	TotalKPS     int              // Общий КПШ
	TotalTonnage float64          // Общий тоннаж
	Stats        map[string]int   // Статистика по упражнениям
}

// ProgramGenerator генератор программ
type ProgramGenerator struct {
	tm          *TemplateManager
	roundingKg  float64            // Округление весов (по умолчанию 2.5 кг)
	currentOpts *GenerationOptions // Текущие опции генерации
}

// NewProgramGenerator создаёт генератор программ
func NewProgramGenerator() (*ProgramGenerator, error) {
	tm, err := NewTemplateManager()
	if err != nil {
		return nil, err
	}

	return &ProgramGenerator{
		tm:         tm,
		roundingKg: 2.5,
	}, nil
}

// SetRounding устанавливает шаг округления весов
func (pg *ProgramGenerator) SetRounding(kg float64) {
	pg.roundingKg = kg
}

// roundWeight округляет вес до ближайшего шага
func (pg *ProgramGenerator) roundWeight(weight float64) float64 {
	return math.Round(weight/pg.roundingKg) * pg.roundingKg
}

// getMaxForExercise возвращает 1ПМ для упражнения
// Если основной максимум не указан, пытается рассчитать от других (для подсобки)
func (pg *ProgramGenerator) getMaxForExercise(exerciseName string, maxes AthleteMaxes) float64 {
	name := normalizeExerciseName(exerciseName)

	// Ягодичный мост (Hip Thrust) — соревновательное движение
	if plContains(name, "ягодич") || plContains(name, "мост") || plContains(name, "hip") || plContains(name, "thrust") {
		if maxes.HipThrust > 0 {
			// Вариации ягодичного моста
			if plContains(name, "пауз") {
				return maxes.HipThrust * 0.85 // С паузой — меньше вес
			}
			if plContains(name, "возвыш") || plContains(name, "дефицит") {
				return maxes.HipThrust * 0.80 // С возвышения — меньше вес
			}
			if plContains(name, "резин") {
				return maxes.HipThrust * 0.70 // С резиной — меньше вес штанги
			}
			if plContains(name, "одной") || plContains(name, "single") {
				return maxes.HipThrust * 0.50 // На одной ноге — 50%
			}
			if plContains(name, "пола") || plContains(name, "glute bridge") {
				return maxes.HipThrust * 0.60 // С пола — меньше амплитуда, меньше вес
			}
			return maxes.HipThrust
		}
		// Если HipThrust не указан, но есть присед — можно прикинуть
		// Типичное соотношение: Hip Thrust ~1.2-1.5x от приседа
		if maxes.Squat > 0 {
			return maxes.Squat * 1.3
		}
		return 0
	}

	// Соревновательные движения
	switch {
	case plContains(name, "присед") && !plContains(name, "груди") && !plContains(name, "фронт"):
		if maxes.Squat > 0 {
			return maxes.Squat
		}
		// Если приседа нет, но есть тяга — рассчитываем от тяги (присед как подсобка)
		// Типичное соотношение: присед ~70% от тяги для тяговика
		if maxes.Deadlift > 0 {
			return maxes.Deadlift * 0.70
		}
		return 0
	case plContains(name, "жим") && plContains(name, "лежа"):
		return maxes.Bench
	case (plContains(name, "тяга") || plContains(name, "становая")) && !plContains(name, "блок") && !plContains(name, "прямых"):
		return maxes.Deadlift
	}

	// Вариации приседа (присед на груди, фронтальный и т.д.)
	if plContains(name, "присед") {
		if maxes.Squat > 0 {
			return maxes.Squat * 0.80 // Присед на груди ~80% от классического
		}
		// Если приседа нет, но есть тяга — присед на груди ~55% от тяги
		// (70% от тяги = присед, 80% от приседа = присед на груди → 70%*80% = 56%)
		if maxes.Deadlift > 0 {
			return maxes.Deadlift * 0.55
		}
		return 0
	}

	// Вариации жима (все жимы относятся к жиму лёжа)
	if plContains(name, "жим") {
		if maxes.Bench == 0 {
			return 0
		}
		if plContains(name, "узк") || plContains(name, "средн") {
			return maxes.Bench * 0.85
		}
		if plContains(name, "брус") || plContains(name, "дожим") {
			return maxes.Bench * 1.05 // С бруска можно больше
		}
		if plContains(name, "гантел") {
			return maxes.Bench * 0.45 // На каждую руку
		}
		if plContains(name, "наклон") {
			return maxes.Bench * 0.75
		}
		if plContains(name, "стоя") || plContains(name, "сидя") {
			return maxes.Bench * 0.65
		}
		return maxes.Bench * 0.7 // Остальные жимы
	}

	// Французский жим, трицепс — от жима лёжа
	if plContains(name, "франц") || plContains(name, "трицепс") {
		if maxes.Bench > 0 {
			return maxes.Bench * 0.4
		}
		return 0
	}

	// Бицепс — от жима лёжа (условно)
	if plContains(name, "бицепс") {
		if maxes.Bench > 0 {
			return maxes.Bench * 0.3
		}
		return 0
	}

	// Вариации тяги
	if plContains(name, "тяга") {
		if maxes.Deadlift == 0 {
			return 0
		}
		if plContains(name, "плинт") {
			return maxes.Deadlift * 1.1
		}
		if plContains(name, "колен") {
			return maxes.Deadlift * 0.9
		}
		if plContains(name, "прямых") || plContains(name, "румын") {
			return maxes.Deadlift * 0.7
		}
		return maxes.Deadlift * 0.7
	}

	// Наклоны — подсобка для тяги/приседа
	if plContains(name, "наклон") {
		if maxes.Deadlift > 0 {
			return maxes.Deadlift * 0.4
		}
		if maxes.Squat > 0 {
			return maxes.Squat * 0.4
		}
		return 0
	}

	// Гиперэкстензия — подсобка для спины
	if plContains(name, "гипер") {
		if maxes.Deadlift > 0 {
			return maxes.Deadlift * 0.3
		}
		return 0
	}

	// Тяга блока (верхнего/нижнего) — подсобка для спины
	if plContains(name, "блок") || plContains(name, "широч") {
		if maxes.Deadlift > 0 {
			return maxes.Deadlift * 0.4
		}
		if maxes.Bench > 0 {
			return maxes.Bench * 0.5
		}
		return 0
	}

	// Подтягивания — от жима или тяги
	if plContains(name, "подтяг") {
		// Для подтягиваний вес — это дополнительный вес (отягощение)
		// Обычно ~30-40% от жима
		if maxes.Bench > 0 {
			return maxes.Bench * 0.35
		}
		return 0
	}

	// Разгибания ног — изоляция для квадрицепсов
	if plContains(name, "разгиб") && plContains(name, "ног") {
		if maxes.Squat > 0 {
			return maxes.Squat * 0.3
		}
		return 0
	}

	// Сгибания ног — изоляция для бицепса бедра
	if plContains(name, "сгиб") && plContains(name, "ног") {
		if maxes.Squat > 0 {
			return maxes.Squat * 0.25
		}
		if maxes.Deadlift > 0 {
			return maxes.Deadlift * 0.2
		}
		return 0
	}

	// Разводка гантелей — изоляция для груди
	if plContains(name, "развод") || plContains(name, "разведен") {
		if maxes.Bench > 0 {
			return maxes.Bench * 0.25 // На каждую руку
		}
		return 0
	}

	// Брусья — от жима лёжа
	if plContains(name, "брус") {
		if maxes.Bench > 0 {
			return maxes.Bench * 0.35 // Дополнительный вес
		}
		return 0
	}

	// Пуловер — от жима лёжа
	if plContains(name, "пуловер") {
		if maxes.Bench > 0 {
			return maxes.Bench * 0.4
		}
		return 0
	}

	// Пресс и прочая подсобка — без веса или с минимальным
	return 0
}

// isCompetitionLift проверяет является ли упражнение соревновательным движением
func isCompetitionLift(name string) bool {
	n := normalizeExerciseName(name)
	// Присед (но не на груди)
	if plContains(n, "присед") && !plContains(n, "груди") && !plContains(n, "фронт") {
		return true
	}
	// Жим лёжа (но не стоя, не сидя, не гантели)
	if plContains(n, "жим") && plContains(n, "лежа") {
		return true
	}
	// Становая тяга
	if (plContains(n, "становая") || (plContains(n, "тяга") && !plContains(n, "блок") && !plContains(n, "прямых") && !plContains(n, "румын"))) {
		return true
	}
	return false
}

// GenerateFromTemplate генерирует программу из шаблона
func (pg *ProgramGenerator) GenerateFromTemplate(templateName string, maxes AthleteMaxes) (*PLGeneratedProgram, error) {
	return pg.GenerateFromTemplateWithOptions(templateName, maxes, GenerationOptions{
		LiftType:        LiftTypeFull,
		DaysPerWeek:     0, // использовать как в шаблоне
		IncludeAccessory: true,
	})
}

// GenerateFromTemplateWithOptions генерирует программу с опциями фильтрации
func (pg *ProgramGenerator) GenerateFromTemplateWithOptions(templateName string, maxes AthleteMaxes, opts GenerationOptions) (*PLGeneratedProgram, error) {
	template, ok := pg.tm.GetTemplate(templateName)
	if !ok {
		return nil, fmt.Errorf("шаблон '%s' не найден", templateName)
	}

	program := &PLGeneratedProgram{
		Name:         template.Name,
		AthleteMaxes: maxes,
		Weeks:        make([]PLGeneratedWeek, 0),
		Stats:        make(map[string]int),
	}

	// Сохраняем опции для использования в генерации
	pg.currentOpts = &opts

	// Генерируем в зависимости от формата шаблона
	if len(template.Phases) > 0 {
		// Шейко 12 недель
		pg.generateFromPhases(template, maxes, program)
	} else if len(template.WeeksData) > 0 {
		// Шейко КМС/МС, Верхошанский
		pg.generateFromWeeksData(template, maxes, program)
	} else if len(template.Microcycles) > 0 {
		// Головинский
		pg.generateFromMicrocycles(template, maxes, program)
	} else if template.Exercises != nil {
		// Русский цикл
		pg.generateFromRussianCycle(template, maxes, program)
	}

	// Фильтруем по типу движения
	if opts.LiftType != LiftTypeFull && opts.LiftType != "" {
		pg.filterByLiftType(program, opts.LiftType)
	}

	// Адаптируем под количество тренировок
	if opts.DaysPerWeek > 0 {
		pg.adaptToTrainingDays(program, opts.DaysPerWeek)
	}

	// Подсчитываем общую статистику
	program.TotalKPS = 0
	program.TotalTonnage = 0
	for _, week := range program.Weeks {
		program.TotalKPS += week.TotalKPS
		program.TotalTonnage += week.Tonnage
	}

	pg.currentOpts = nil
	return program, nil
}

// generateFromPhases генерирует из фаз (Шейко 12 недель)
func (pg *ProgramGenerator) generateFromPhases(template *PowerliftingTemplate, maxes AthleteMaxes, program *PLGeneratedProgram) {
	weekNum := 0
	for _, phase := range template.Phases {
		for _, week := range phase.Weeks {
			weekNum++
			genWeek := pg.generateWeek(week, maxes, phase.Name)
			genWeek.WeekNum = weekNum
			program.Weeks = append(program.Weeks, genWeek)
		}
	}
}

// generateFromWeeksData генерирует из недель (Шейко КМС/МС)
func (pg *ProgramGenerator) generateFromWeeksData(template *PowerliftingTemplate, maxes AthleteMaxes, program *PLGeneratedProgram) {
	for _, week := range template.WeeksData {
		genWeek := pg.generateWeekFromKMS(week, maxes)
		program.Weeks = append(program.Weeks, genWeek)
	}
}

// generateFromMicrocycles генерирует из микроциклов (Головинский)
func (pg *ProgramGenerator) generateFromMicrocycles(template *PowerliftingTemplate, maxes AthleteMaxes, program *PLGeneratedProgram) {
	weekNum := 0
	for _, micro := range template.Microcycles {
		weekNum++
		genWeek := pg.generateWeekFromMicrocycle(micro, maxes)
		// Используем номер микроцикла из любого поля
		if micro.MicroNum > 0 {
			genWeek.WeekNum = micro.MicroNum
		} else if micro.MicrocycleNum > 0 {
			genWeek.WeekNum = micro.MicrocycleNum
		} else {
			genWeek.WeekNum = weekNum
		}
		program.Weeks = append(program.Weeks, genWeek)
	}
}

// generateFromRussianCycle генерирует из формата Русского цикла
func (pg *ProgramGenerator) generateFromRussianCycle(template *PowerliftingTemplate, maxes AthleteMaxes, program *PLGeneratedProgram) {
	ex := template.Exercises
	if ex == nil {
		return
	}

	// Определяем количество недель
	numWeeks := template.Weeks
	if numWeeks == 0 {
		numWeeks = 12 // По умолчанию
	}

	// Определяем формат: Муравьёв (2 тренировки/неделю для приседа/жима, 1 для тяги)
	// или классический Русский цикл (1 тренировка/неделю для каждого)
	hasMuravyevFormat := pg.isMuravyevFormat(ex)

	// Создаём недели
	for weekNum := 1; weekNum <= numWeeks; weekNum++ {
		genWeek := PLGeneratedWeek{
			WeekNum: weekNum,
			Phase:   pg.getRussianCyclePhase(weekNum, numWeeks),
		}

		if hasMuravyevFormat {
			// Муравьёв: 3 тренировки в неделю
			// Тренировка I (Пн): Присед + Жим (тренировка 1)
			// Тренировка II (Ср): Тяга
			// Тренировка III (Пт): Присед + Жим (тренировка 2)

			// Тренировка I: Присед (трен.1) + Жим (трен.1)
			workout1 := PLGeneratedWorkout{
				DayNum: 1,
				Name:   "Тренировка I (Пн)",
			}
			if ex.Squat != nil {
				squatEx := pg.generateRussianCycleExerciseByTraining("Присед", ex.Squat, weekNum, 1, maxes.Squat)
				if squatEx != nil {
					workout1.Exercises = append(workout1.Exercises, *squatEx)
					workout1.TotalKPS += squatEx.TotalReps
					workout1.Tonnage += squatEx.Tonnage
				}
			}
			if ex.Bench != nil {
				benchEx := pg.generateRussianCycleExerciseByTraining("Жим лёжа", ex.Bench, weekNum, 1, maxes.Bench)
				if benchEx != nil {
					workout1.Exercises = append(workout1.Exercises, *benchEx)
					workout1.TotalKPS += benchEx.TotalReps
					workout1.Tonnage += benchEx.Tonnage
				}
			}
			if len(workout1.Exercises) > 0 {
				genWeek.Workouts = append(genWeek.Workouts, workout1)
				genWeek.TotalKPS += workout1.TotalKPS
				genWeek.Tonnage += workout1.Tonnage
			}

			// Тренировка II: Тяга (только 1 тренировка в неделю)
			if ex.Deadlift != nil {
				workout2 := PLGeneratedWorkout{
					DayNum: 2,
					Name:   "Тренировка II (Ср)",
				}
				deadliftEx := pg.generateRussianCycleExerciseByTraining("Становая тяга", ex.Deadlift, weekNum, 1, maxes.Deadlift)
				if deadliftEx != nil {
					workout2.Exercises = append(workout2.Exercises, *deadliftEx)
					workout2.TotalKPS += deadliftEx.TotalReps
					workout2.Tonnage += deadliftEx.Tonnage
				}
				if len(workout2.Exercises) > 0 {
					genWeek.Workouts = append(genWeek.Workouts, workout2)
					genWeek.TotalKPS += workout2.TotalKPS
					genWeek.Tonnage += workout2.Tonnage
				}
			}

			// Тренировка III: Присед (трен.2) + Жим (трен.2)
			workout3 := PLGeneratedWorkout{
				DayNum: 3,
				Name:   "Тренировка III (Пт)",
			}
			if ex.Squat != nil {
				squatEx := pg.generateRussianCycleExerciseByTraining("Присед", ex.Squat, weekNum, 2, maxes.Squat)
				if squatEx != nil {
					workout3.Exercises = append(workout3.Exercises, *squatEx)
					workout3.TotalKPS += squatEx.TotalReps
					workout3.Tonnage += squatEx.Tonnage
				}
			}
			if ex.Bench != nil {
				benchEx := pg.generateRussianCycleExerciseByTraining("Жим лёжа", ex.Bench, weekNum, 2, maxes.Bench)
				if benchEx != nil {
					workout3.Exercises = append(workout3.Exercises, *benchEx)
					workout3.TotalKPS += benchEx.TotalReps
					workout3.Tonnage += benchEx.Tonnage
				}
			}
			if len(workout3.Exercises) > 0 {
				genWeek.Workouts = append(genWeek.Workouts, workout3)
				genWeek.TotalKPS += workout3.TotalKPS
				genWeek.Tonnage += workout3.Tonnage
			}
		} else {
			// Классический Русский цикл: 2 тренировки

			// Тренировка 1: Присед + Жим
			workout1 := PLGeneratedWorkout{
				DayNum: 1,
				Name:   "Тренировка 1 (Присед + Жим)",
			}
			if ex.Squat != nil {
				squatEx := pg.generateRussianCycleExercise("Присед", ex.Squat, weekNum, maxes.Squat)
				if squatEx != nil {
					workout1.Exercises = append(workout1.Exercises, *squatEx)
					workout1.TotalKPS += squatEx.TotalReps
					workout1.Tonnage += squatEx.Tonnage
				}
			}
			if ex.Bench != nil {
				benchEx := pg.generateRussianCycleExercise("Жим лёжа", ex.Bench, weekNum, maxes.Bench)
				if benchEx != nil {
					workout1.Exercises = append(workout1.Exercises, *benchEx)
					workout1.TotalKPS += benchEx.TotalReps
					workout1.Tonnage += benchEx.Tonnage
				}
			}
			if len(workout1.Exercises) > 0 {
				genWeek.Workouts = append(genWeek.Workouts, workout1)
				genWeek.TotalKPS += workout1.TotalKPS
				genWeek.Tonnage += workout1.Tonnage
			}

			// Тренировка 2: Тяга
			if ex.Deadlift != nil {
				workout2 := PLGeneratedWorkout{
					DayNum: 2,
					Name:   "Тренировка 2 (Тяга)",
				}
				deadliftEx := pg.generateRussianCycleExercise("Становая тяга", ex.Deadlift, weekNum, maxes.Deadlift)
				if deadliftEx != nil {
					workout2.Exercises = append(workout2.Exercises, *deadliftEx)
					workout2.TotalKPS += deadliftEx.TotalReps
					workout2.Tonnage += deadliftEx.Tonnage
				}
				if len(workout2.Exercises) > 0 {
					genWeek.Workouts = append(genWeek.Workouts, workout2)
					genWeek.TotalKPS += workout2.TotalKPS
					genWeek.Tonnage += workout2.Tonnage
				}
			}
		}

		if len(genWeek.Workouts) > 0 {
			program.Weeks = append(program.Weeks, genWeek)
		}
	}
}

// isMuravyevFormat проверяет, является ли шаблон форматом Муравьёва
// (2 тренировки в неделю для жима/приседа с одинаковым week_num)
func (pg *ProgramGenerator) isMuravyevFormat(ex *RussianCycleExercises) bool {
	if ex == nil || ex.Bench == nil || len(ex.Bench.Weeks) < 2 {
		return false
	}
	// Если есть 2 записи с week_num=1, это Муравьёв
	count := 0
	for _, w := range ex.Bench.Weeks {
		if w.WeekNum == 1 {
			count++
		}
	}
	return count >= 2
}

// generateRussianCycleExerciseByTraining генерирует упражнение с учётом номера тренировки
func (pg *ProgramGenerator) generateRussianCycleExerciseByTraining(name string, rcEx *RussianCycleExercise, weekNum, trainingNum int, oneRM float64) *PLGeneratedExercise {
	if rcEx == nil || len(rcEx.Weeks) == 0 {
		return nil
	}

	// Ищем данные для нужной недели и тренировки
	// В текущем формате JSON записи идут подряд: week_num=1 (трен.1), week_num=1 (трен.2), week_num=2 (трен.1), ...
	var weekData *RussianCycleWeekData
	matchCount := 0
	for i := range rcEx.Weeks {
		if rcEx.Weeks[i].WeekNum == weekNum {
			matchCount++
			if matchCount == trainingNum {
				weekData = &rcEx.Weeks[i]
				break
			}
		}
	}

	if weekData == nil || len(weekData.Sets) == 0 {
		return nil
	}

	return pg.buildExerciseFromWeekData(name, weekData, oneRM)
}

// generateRussianCycleExercise генерирует упражнение из Русского цикла (первая тренировка недели)
func (pg *ProgramGenerator) generateRussianCycleExercise(name string, rcEx *RussianCycleExercise, weekNum int, oneRM float64) *PLGeneratedExercise {
	if rcEx == nil || len(rcEx.Weeks) == 0 {
		return nil
	}

	// Ищем данные для нужной недели (берём первую запись)
	var weekData *RussianCycleWeekData
	for i := range rcEx.Weeks {
		if rcEx.Weeks[i].WeekNum == weekNum {
			weekData = &rcEx.Weeks[i]
			break
		}
	}

	if weekData == nil || len(weekData.Sets) == 0 {
		return nil
	}

	return pg.buildExerciseFromWeekData(name, weekData, oneRM)
}

// buildExerciseFromWeekData создаёт упражнение из данных недели
func (pg *ProgramGenerator) buildExerciseFromWeekData(name string, weekData *RussianCycleWeekData, oneRM float64) *PLGeneratedExercise {
	genEx := &PLGeneratedExercise{
		Name: name,
		Type: "competition",
	}

	for _, set := range weekData.Sets {
		sets := set.Sets
		if sets == 0 {
			sets = 1
		}

		weight := pg.roundWeight(oneRM * set.Percent / 100)

		genSet := PLGeneratedSet{
			Percent:  set.Percent,
			Reps:     set.Reps,
			Sets:     sets,
			WeightKg: weight,
		}

		genEx.Sets = append(genEx.Sets, genSet)

		reps := set.Reps * sets
		genEx.TotalReps += reps
		genEx.Tonnage += weight * float64(reps) / 1000
	}

	if genEx.TotalReps > 0 && oneRM > 0 {
		genEx.AvgPercent = (genEx.Tonnage * 1000 / float64(genEx.TotalReps)) / oneRM * 100
	}

	return genEx
}

// getRussianCyclePhase определяет фазу Русского цикла
func (pg *ProgramGenerator) getRussianCyclePhase(weekNum, totalWeeks int) string {
	progress := float64(weekNum) / float64(totalWeeks)
	switch {
	case progress <= 0.33:
		return "Накопление объёма"
	case progress <= 0.66:
		return "Интенсификация"
	default:
		return "Реализация"
	}
}

// generateWeek генерирует неделю из шаблона Шейко 12 недель
func (pg *ProgramGenerator) generateWeek(week TemplateWeek, maxes AthleteMaxes, phaseName string) PLGeneratedWeek {
	genWeek := PLGeneratedWeek{
		WeekNum: week.WeekNum,
		Phase:   phaseName,
	}

	for _, workout := range week.Workouts {
		genWorkout := pg.generateWorkout(workout, maxes)
		genWeek.Workouts = append(genWeek.Workouts, genWorkout)
		genWeek.TotalKPS += genWorkout.TotalKPS
		genWeek.Tonnage += genWorkout.Tonnage
	}

	// Средняя интенсивность
	if genWeek.TotalKPS > 0 && genWeek.Tonnage > 0 {
		// Упрощённый расчёт
		genWeek.AvgIntens = genWeek.Tonnage / float64(genWeek.TotalKPS)
	}

	return genWeek
}

// generateWorkout генерирует тренировку
func (pg *ProgramGenerator) generateWorkout(workout TemplateWorkout, maxes AthleteMaxes) PLGeneratedWorkout {
	genWorkout := PLGeneratedWorkout{
		DayNum: workout.WorkoutNum,
		Name:   fmt.Sprintf("Тренировка %d", workout.WorkoutNum),
	}

	for _, ex := range workout.Exercises {
		genEx := pg.generateExercise(ex, maxes)
		genWorkout.Exercises = append(genWorkout.Exercises, genEx)
		genWorkout.TotalKPS += genEx.TotalReps
		genWorkout.Tonnage += genEx.Tonnage
	}

	return genWorkout
}

// generateExercise генерирует упражнение
func (pg *ProgramGenerator) generateExercise(ex TemplateExercise, maxes AthleteMaxes) PLGeneratedExercise {
	genEx := PLGeneratedExercise{
		Name: ex.Name,
		Type: ex.Type,
	}

	oneRM := pg.getMaxForExercise(ex.Name, maxes)

	var totalWeight float64
	var totalReps int

	for _, set := range ex.Sets {
		genSet := PLGeneratedSet{
			Percent: set.Percent,
			Reps:    set.Reps,
			Sets:    set.Sets,
		}

		if set.Sets == 0 {
			genSet.Sets = 1
		}

		// Рассчитываем вес
		if set.Percent > 0 && oneRM > 0 {
			genSet.WeightKg = pg.roundWeight(oneRM * set.Percent / 100)
		} else if set.Weight > 0 {
			genSet.WeightKg = pg.roundWeight(set.Weight)
		}

		genEx.Sets = append(genEx.Sets, genSet)

		// Статистика
		repsInSet := genSet.Reps * genSet.Sets
		totalReps += repsInSet
		totalWeight += genSet.WeightKg * float64(repsInSet)
	}

	// Подсобка без процентов
	if len(ex.SetsReps) > 0 {
		for _, reps := range ex.SetsReps {
			genEx.Sets = append(genEx.Sets, PLGeneratedSet{
				Reps: reps,
				Sets: 1,
			})
			totalReps += reps
		}
	}

	genEx.TotalReps = totalReps
	genEx.Tonnage = totalWeight / 1000 // в тоннах

	if totalReps > 0 && totalWeight > 0 {
		genEx.AvgPercent = (totalWeight / float64(totalReps)) / oneRM * 100
	}

	return genEx
}

// generateWeekFromKMS генерирует неделю из формата КМС/МС
func (pg *ProgramGenerator) generateWeekFromKMS(week TemplateWeek, maxes AthleteMaxes) PLGeneratedWeek {
	genWeek := PLGeneratedWeek{
		WeekNum: week.WeekNum,
		Phase:   "Соревновательная подготовка",
	}

	// Формат с Workouts
	if len(week.Workouts) > 0 {
		for _, workout := range week.Workouts {
			genWorkout := pg.generateWorkout(workout, maxes)
			genWeek.Workouts = append(genWeek.Workouts, genWorkout)
			genWeek.TotalKPS += genWorkout.TotalKPS
			genWeek.Tonnage += genWorkout.Tonnage
		}
		return genWeek
	}

	// Формат с Days (Шейко КМС/МС с абсолютными весами)
	if len(week.Days) > 0 {
		for dayIdx, day := range week.Days {
			genWorkout := PLGeneratedWorkout{
				DayNum: dayIdx + 1,
				Name:   fmt.Sprintf("Тренировка %d", dayIdx+1),
			}

			for _, ex := range day.Exercises {
				genEx := pg.generateExerciseWithAbsoluteWeight(ex, maxes)
				genWorkout.Exercises = append(genWorkout.Exercises, genEx)
				genWorkout.TotalKPS += genEx.TotalReps
				genWorkout.Tonnage += genEx.Tonnage
			}

			genWeek.Workouts = append(genWeek.Workouts, genWorkout)
			genWeek.TotalKPS += genWorkout.TotalKPS
			genWeek.Tonnage += genWorkout.Tonnage
		}
	}

	return genWeek
}

// generateExerciseWithAbsoluteWeight генерирует упражнение с абсолютными весами
// Пересчитывает веса пропорционально 1ПМ атлета
func (pg *ProgramGenerator) generateExerciseWithAbsoluteWeight(ex TemplateExercise, maxes AthleteMaxes) PLGeneratedExercise {
	genEx := PLGeneratedExercise{
		Name: ex.Name,
		Type: ex.Type,
	}

	// Определяем базовый 1ПМ из шаблона (по первому тяжёлому подходу)
	var templateMax float64
	for _, set := range ex.Sets {
		if set.Weight > templateMax {
			templateMax = set.Weight
		}
	}

	// Определяем 1ПМ атлета для этого упражнения
	athleteMax := pg.getMaxForExercise(ex.Name, maxes)

	// Коэффициент пересчёта
	ratio := 1.0
	if templateMax > 0 && athleteMax > 0 {
		// Предполагаем что максимальный вес в шаблоне ~90-95% от 1ПМ шаблона
		estimatedTemplateRM := templateMax / 0.9
		ratio = athleteMax / estimatedTemplateRM
	}

	var totalReps int
	var totalWeight float64

	for _, set := range ex.Sets {
		sets := set.Sets
		if sets == 0 {
			sets = 1
		}

		// Пересчитываем вес
		weight := pg.roundWeight(set.Weight * ratio)

		// Рассчитываем процент от 1ПМ атлета
		percent := 0.0
		if athleteMax > 0 {
			percent = weight / athleteMax * 100
		}

		genSet := PLGeneratedSet{
			Percent:  percent,
			Reps:     set.Reps,
			Sets:     sets,
			WeightKg: weight,
		}

		genEx.Sets = append(genEx.Sets, genSet)

		repsInSet := set.Reps * sets
		totalReps += repsInSet
		totalWeight += weight * float64(repsInSet)
	}

	genEx.TotalReps = totalReps
	genEx.Tonnage = totalWeight / 1000

	if totalReps > 0 && athleteMax > 0 {
		genEx.AvgPercent = (totalWeight / float64(totalReps)) / athleteMax * 100
	}

	return genEx
}

// generateWeekFromMicrocycle генерирует неделю из микроцикла Головинского
func (pg *ProgramGenerator) generateWeekFromMicrocycle(micro TemplateMicrocycle, maxes AthleteMaxes) PLGeneratedWeek {
	// Определяем номер микроцикла
	microNum := micro.MicroNum
	if microNum == 0 {
		microNum = micro.MicrocycleNum
	}

	genWeek := PLGeneratedWeek{
		Phase: fmt.Sprintf("Микроцикл %d", microNum),
	}

	// Поддержка нового формата с workouts
	if len(micro.Workouts) > 0 {
		for _, workout := range micro.Workouts {
			genWorkout := PLGeneratedWorkout{
				DayNum: workout.Day,
				Name:   fmt.Sprintf("День %d", workout.Day),
			}

			for _, ex := range workout.Exercises {
				genEx := pg.generateExerciseFromGolovinsky(ex, maxes)
				// Пропускаем упражнения без сетов
				if len(genEx.Sets) == 0 {
					continue
				}
				// НЕ фильтруем здесь по isCompetitionLift — это делается в filterByLiftType
				// чтобы присед мог остаться как подсобка для тяговика
				genWorkout.Exercises = append(genWorkout.Exercises, genEx)
				genWorkout.TotalKPS += genEx.TotalReps
				genWorkout.Tonnage += genEx.Tonnage
			}

			// Пропускаем пустые тренировки
			if len(genWorkout.Exercises) == 0 {
				continue
			}

			genWeek.Workouts = append(genWeek.Workouts, genWorkout)
			genWeek.TotalKPS += genWorkout.TotalKPS
			genWeek.Tonnage += genWorkout.Tonnage
		}
		return genWeek
	}

	// Старый формат с Days
	for dayIdx, day := range micro.Days {
		genWorkout := PLGeneratedWorkout{
			DayNum: dayIdx + 1,
			Name:   fmt.Sprintf("День %d (%s)", dayIdx+1, day.Date),
		}

		for _, ex := range day.Exercises {
			genEx := pg.generateExerciseFromGolovinsky(ex, maxes)
			// Пропускаем упражнения без сетов
			if len(genEx.Sets) == 0 {
				continue
			}
			// НЕ фильтруем здесь — это делается в filterByLiftType
			genWorkout.Exercises = append(genWorkout.Exercises, genEx)
			genWorkout.TotalKPS += genEx.TotalReps
			genWorkout.Tonnage += genEx.Tonnage
		}

		// Пропускаем пустые тренировки
		if len(genWorkout.Exercises) == 0 {
			continue
		}

		genWeek.Workouts = append(genWeek.Workouts, genWorkout)
		genWeek.TotalKPS += genWorkout.TotalKPS
		genWeek.Tonnage += genWorkout.Tonnage
	}

	return genWeek
}

// generateExerciseFromGolovinsky генерирует упражнение из формата Головинского
func (pg *ProgramGenerator) generateExerciseFromGolovinsky(ex TemplateExercise, maxes AthleteMaxes) PLGeneratedExercise {
	genEx := PLGeneratedExercise{
		Name: ex.Name,
		Type: "competition",
	}

	// Определяем 1ПМ из шаблона или по имени
	var oneRM float64
	if ex.Stats != nil && ex.Stats.PM > 0 {
		// Есть 1ПМ в шаблоне — пересчитываем пропорционально
		templatePM := ex.Stats.PM
		athletePM := pg.getMaxForExercise(ex.Name, maxes)
		if athletePM > 0 && templatePM > 0 {
			ratio := athletePM / templatePM
			oneRM = athletePM

			for _, set := range ex.Sets {
				newWeight := pg.roundWeight(set.Weight * ratio)
				sets := set.Sets
				if sets == 0 {
					sets = 1
				}
				genSet := PLGeneratedSet{
					Percent:  set.Percent,
					Reps:     set.Reps,
					Sets:     sets,
					WeightKg: newWeight,
				}
				genEx.Sets = append(genEx.Sets, genSet)
				genEx.TotalReps += set.Reps * sets
				genEx.Tonnage += newWeight * float64(set.Reps*sets) / 1000
			}
		}
	} else {
		// Нет 1ПМ — используем проценты
		oneRM = pg.getMaxForExercise(ex.Name, maxes)
		for _, set := range ex.Sets {
			// Проценты могут быть в формате 0.5 (50%) или 50
			percent := set.Percent
			if percent > 0 && percent < 1 {
				percent = percent * 100 // Конвертируем 0.5 -> 50
			}

			weight := pg.roundWeight(oneRM * percent / 100)
			sets := set.Sets
			if sets == 0 {
				sets = 1
			}
			genSet := PLGeneratedSet{
				Percent:  percent,
				Reps:     set.Reps,
				Sets:     sets,
				WeightKg: weight,
			}
			genEx.Sets = append(genEx.Sets, genSet)
			genEx.TotalReps += set.Reps * sets
			genEx.Tonnage += weight * float64(set.Reps*sets) / 1000
		}
	}

	if genEx.TotalReps > 0 && oneRM > 0 {
		genEx.AvgPercent = (genEx.Tonnage * 1000 / float64(genEx.TotalReps)) / oneRM * 100
	}

	return genEx
}

// ListTemplates возвращает список шаблонов
func (pg *ProgramGenerator) ListTemplates() []string {
	return pg.tm.ListTemplates()
}

// GetTemplatesForLevel возвращает шаблоны для уровня
func (pg *ProgramGenerator) GetTemplatesForLevel(level string) []*PowerliftingTemplate {
	return pg.tm.GetTemplatesForLevel(level)
}

// RecommendTemplate рекомендует лучший шаблон для атлета
func (pg *ProgramGenerator) RecommendTemplate(level AthleteLevel, liftType LiftType) string {
	// Рекомендации на основе уровня атлета
	switch level {
	case LevelNovice, Level3Rank:
		// Для новичков - простые циклы с меньшим объёмом
		return "Русский цикл"

	case Level2Rank, Level1Rank:
		// Для разрядников - Шейко 12 недель
		return "Шейко 12 недель (разрядники)"

	case LevelKMS:
		// Для КМС - Шейко или Головинский
		if liftType == LiftTypeDeadlift {
			return "Головинский Цикл 7" // Хорошо для тяги
		}
		return "Шейко КМС/МС (4 недели к соревнованиям)"

	case LevelMS, LevelMSMC:
		// Для МС/МСМК - Муравьёв или индивидуальный подбор
		return "Цикл Муравьёва (16 недель)"

	default:
		return "Шейко 12 недель (разрядники)"
	}
}

// GetTemplatesByLevel возвращает отсортированные шаблоны для уровня
func (pg *ProgramGenerator) GetTemplatesByLevel(level AthleteLevel) []string {
	// Все шаблоны с приоритетом для уровня
	allTemplates := pg.tm.ListTemplates()

	// Рекомендуемый шаблон первым
	recommended := pg.RecommendTemplate(level, LiftTypeFull)

	var result []string
	result = append(result, recommended)

	for _, t := range allTemplates {
		if t != recommended {
			result = append(result, t)
		}
	}

	return result
}

// GenerateAutomatic генерирует программу автоматически выбирая шаблон
func (pg *ProgramGenerator) GenerateAutomatic(maxes AthleteMaxes, opts GenerationOptions) (*PLGeneratedProgram, error) {
	// Определяем уровень если не указан
	level := opts.AthleteLevel
	if level == "" {
		level = pg.estimateLevel(maxes)
	}

	// Выбираем шаблон
	templateName := pg.RecommendTemplate(level, opts.LiftType)

	return pg.GenerateFromTemplateWithOptions(templateName, maxes, opts)
}

// estimateLevel оценивает уровень атлета по максимумам
func (pg *ProgramGenerator) estimateLevel(maxes AthleteMaxes) AthleteLevel {
	// Примерные границы для мужчин (можно расширить)
	// Жим лёжа как основной показатель
	bench := maxes.Bench

	switch {
	case bench >= 200:
		return LevelMSMC
	case bench >= 170:
		return LevelMS
	case bench >= 145:
		return LevelKMS
	case bench >= 125:
		return Level1Rank
	case bench >= 105:
		return Level2Rank
	case bench >= 85:
		return Level3Rank
	default:
		return LevelNovice
	}
}

// Вспомогательные функции

func normalizeExerciseName(name string) string {
	// Приводим к нижнему регистру для сравнения
	return strings.ToLower(name)
}

func plContains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// classifyExercise определяет к какому типу движения относится упражнение
// Возвращает: squat, bench, deadlift или "" для подсобки
func classifyExercise(name string) LiftType {
	normalized := normalizeExerciseName(name)

	// Ягодичный мост (Hip Thrust)
	if plContains(normalized, "ягодич") || plContains(normalized, "мост") ||
		plContains(normalized, "hip") || plContains(normalized, "thrust") {
		return LiftTypeHipThrust
	}

	// Присед и все вариации приседа
	if plContains(normalized, "присед") {
		return LiftTypeSquat
	}

	// Становая тяга и вариации (но НЕ тяга блока, штанги к поясу и т.д.)
	if plContains(normalized, "становая") {
		return LiftTypeDeadlift
	}
	if plContains(normalized, "тяга") {
		// Это подсобка для спины, не становая
		if plContains(normalized, "блок") || plContains(normalized, "наклон") ||
			plContains(normalized, "штанг") || plContains(normalized, "гантел") ||
			plContains(normalized, "пояс") || plContains(normalized, "горизонт") {
			return "" // подсобка для спины
		}
		// Тяга на прямых, румынская — вариации становой
		if plContains(normalized, "прямых") || plContains(normalized, "румын") ||
			plContains(normalized, "мертв") {
			return LiftTypeDeadlift
		}
		// Остальные "тяга" без уточнения — считаем становой
		return LiftTypeDeadlift
	}

	// Жим лёжа (только соревновательный жим)
	if plContains(normalized, "жим") && plContains(normalized, "лежа") {
		return LiftTypeBench
	}

	// Все остальные жимы, трицепс, бицепс — это подсобка для жима
	// НО они не классифицируются как LiftTypeBench
	return ""
}

// getExerciseMuscleGroup определяет группу мышц упражнения
// upper = верх тела (грудь, плечи, руки)
// lower = низ тела (ноги, ягодицы)
// back = спина
// core = кор
func getExerciseMuscleGroup(name string) string {
	normalized := normalizeExerciseName(name)

	// Низ тела
	if plContains(normalized, "присед") || plContains(normalized, "ног") ||
		plContains(normalized, "выпад") || plContains(normalized, "разгиб") ||
		plContains(normalized, "сгиб") || plContains(normalized, "икр") ||
		plContains(normalized, "ягодиц") {
		return "lower"
	}

	// Спина (включая становую)
	if plContains(normalized, "тяга") || plContains(normalized, "становая") ||
		plContains(normalized, "спин") || plContains(normalized, "широч") ||
		plContains(normalized, "гипер") || plContains(normalized, "подтяг") {
		return "back"
	}

	// Верх тела (грудь, плечи, руки)
	if plContains(normalized, "жим") || plContains(normalized, "трицепс") ||
		plContains(normalized, "бицепс") || plContains(normalized, "франц") ||
		plContains(normalized, "груд") || plContains(normalized, "плеч") ||
		plContains(normalized, "дельт") || plContains(normalized, "развод") ||
		plContains(normalized, "пуловер") {
		return "upper"
	}

	// Кор
	if plContains(normalized, "пресс") || plContains(normalized, "планк") ||
		plContains(normalized, "скруч") {
		return "core"
	}

	return "other"
}

// isExerciseForLiftType проверяет подходит ли упражнение для типа дисциплины
// Логика: оставляем основное движение + релевантную подсобку
func isExerciseForLiftType(name string, liftType LiftType) bool {
	exType := classifyExercise(name)
	muscleGroup := getExerciseMuscleGroup(name)

	switch liftType {
	case LiftTypeBench:
		// Жимовик: жим лёжа + весь верх тела + кор
		// Убираем: соревновательный присед, становую, ноги, hip thrust
		if exType == LiftTypeSquat || exType == LiftTypeDeadlift || exType == LiftTypeHipThrust {
			return false
		}
		if muscleGroup == "lower" {
			return false
		}
		return true

	case LiftTypeSquat:
		// Приседатель: присед + низ тела + спина + кор
		// Убираем: соревновательный жим лёжа, становую, верх тела
		if exType == LiftTypeBench || exType == LiftTypeDeadlift {
			return false
		}
		if muscleGroup == "upper" {
			return false
		}
		return true

	case LiftTypeDeadlift:
		// Тяговик: становая + спина + низ тела (включая присед как подсобку!) + кор
		// Убираем: только соревновательный жим лёжа и верх тела
		if exType == LiftTypeBench {
			return false
		}
		if muscleGroup == "upper" {
			return false
		}
		// Присед разрешён как подсобка для тяги!
		return true

	case LiftTypeHipThrust:
		// Hip Thrust: ягодичный мост + задняя цепь + спина + кор + верх тела (поддерживающий)
		// Для hip thrust программы оставляем ВСЁ — это полноценная 4-дневная программа
		return true
	}

	return true // Троеборье — всё включаем
}

// filterByLiftType фильтрует программу оставляя только нужные движения
func (pg *ProgramGenerator) filterByLiftType(program *PLGeneratedProgram, liftType LiftType) {
	for weekIdx := range program.Weeks {
		week := &program.Weeks[weekIdx]
		week.TotalKPS = 0
		week.Tonnage = 0

		for workoutIdx := range week.Workouts {
			workout := &week.Workouts[workoutIdx]

			// Фильтруем упражнения
			var filteredExercises []PLGeneratedExercise
			for _, ex := range workout.Exercises {
				if isExerciseForLiftType(ex.Name, liftType) {
					filteredExercises = append(filteredExercises, ex)
				}
			}

			// Пересчитываем статистику
			workout.Exercises = filteredExercises
			workout.TotalKPS = 0
			workout.Tonnage = 0
			for _, ex := range workout.Exercises {
				workout.TotalKPS += ex.TotalReps
				workout.Tonnage += ex.Tonnage
			}

			week.TotalKPS += workout.TotalKPS
			week.Tonnage += workout.Tonnage
		}

		// Убираем пустые тренировки
		var filteredWorkouts []PLGeneratedWorkout
		for _, w := range week.Workouts {
			if len(w.Exercises) > 0 {
				filteredWorkouts = append(filteredWorkouts, w)
			}
		}
		week.Workouts = filteredWorkouts
	}

	// Убираем пустые недели
	var filteredWeeks []PLGeneratedWeek
	for _, w := range program.Weeks {
		if len(w.Workouts) > 0 {
			filteredWeeks = append(filteredWeeks, w)
		}
	}
	program.Weeks = filteredWeeks

	// Обновляем название
	switch liftType {
	case LiftTypeBench:
		program.Name += " (только жим)"
	case LiftTypeSquat:
		program.Name += " (только присед)"
	case LiftTypeDeadlift:
		program.Name += " (только тяга)"
	case LiftTypeHipThrust:
		program.Name += " (ягодичный мост)"
	}
}

// GetTemplatesForLiftType возвращает шаблоны для типа дисциплины
func (tm *TemplateManager) GetTemplatesForLiftType(liftType LiftType) []*PowerliftingTemplate {
	var result []*PowerliftingTemplate
	for _, t := range tm.templates {
		// Проверяем совместимость шаблона с типом
		if t.Type == string(liftType) || t.Type == "троеборье" || t.Type == "" {
			result = append(result, t)
		}
	}
	return result
}

// adaptToTrainingDays адаптирует программу под количество тренировок в неделю
func (pg *ProgramGenerator) adaptToTrainingDays(program *PLGeneratedProgram, targetDays int) {
	if targetDays <= 0 || targetDays > 6 {
		return // Не адаптируем
	}

	for weekIdx := range program.Weeks {
		week := &program.Weeks[weekIdx]
		currentDays := len(week.Workouts)

		if currentDays == targetDays {
			continue // Уже нужное количество
		}

		if currentDays > targetDays {
			// Нужно уменьшить — объединяем тренировки
			week.Workouts = pg.mergeWorkouts(week.Workouts, targetDays)
		} else {
			// Нужно увеличить — разделяем тренировки
			week.Workouts = pg.splitWorkouts(week.Workouts, targetDays)
		}

		// Пересчитываем статистику недели
		week.TotalKPS = 0
		week.Tonnage = 0
		for _, w := range week.Workouts {
			week.TotalKPS += w.TotalKPS
			week.Tonnage += w.Tonnage
		}
	}
}

// mergeWorkouts объединяет тренировки до нужного количества
func (pg *ProgramGenerator) mergeWorkouts(workouts []PLGeneratedWorkout, targetDays int) []PLGeneratedWorkout {
	if len(workouts) <= targetDays {
		return workouts
	}

	result := make([]PLGeneratedWorkout, targetDays)

	// Распределяем упражнения по тренировкам
	exercisesPerDay := (len(workouts) + targetDays - 1) / targetDays

	exerciseIdx := 0
	for dayIdx := 0; dayIdx < targetDays; dayIdx++ {
		result[dayIdx] = PLGeneratedWorkout{
			DayNum: dayIdx + 1,
			Name:   fmt.Sprintf("Тренировка %d", dayIdx+1),
		}

		// Берём упражнения из нескольких исходных тренировок
		for i := 0; i < exercisesPerDay && exerciseIdx < len(workouts); i++ {
			for _, ex := range workouts[exerciseIdx].Exercises {
				result[dayIdx].Exercises = append(result[dayIdx].Exercises, ex)
				result[dayIdx].TotalKPS += ex.TotalReps
				result[dayIdx].Tonnage += ex.Tonnage
			}
			exerciseIdx++
		}
	}

	return result
}

// splitWorkouts разделяет тренировки на большее количество
func (pg *ProgramGenerator) splitWorkouts(workouts []PLGeneratedWorkout, targetDays int) []PLGeneratedWorkout {
	if len(workouts) >= targetDays {
		return workouts
	}

	// Собираем все упражнения
	var allExercises []PLGeneratedExercise
	for _, w := range workouts {
		allExercises = append(allExercises, w.Exercises...)
	}

	if len(allExercises) == 0 {
		return workouts
	}

	// Распределяем по новому количеству дней
	result := make([]PLGeneratedWorkout, targetDays)
	exercisesPerDay := (len(allExercises) + targetDays - 1) / targetDays

	exerciseIdx := 0
	for dayIdx := 0; dayIdx < targetDays && exerciseIdx < len(allExercises); dayIdx++ {
		result[dayIdx] = PLGeneratedWorkout{
			DayNum: dayIdx + 1,
			Name:   fmt.Sprintf("Тренировка %d", dayIdx+1),
		}

		for i := 0; i < exercisesPerDay && exerciseIdx < len(allExercises); i++ {
			ex := allExercises[exerciseIdx]
			result[dayIdx].Exercises = append(result[dayIdx].Exercises, ex)
			result[dayIdx].TotalKPS += ex.TotalReps
			result[dayIdx].Tonnage += ex.Tonnage
			exerciseIdx++
		}
	}

	// Убираем пустые тренировки
	var filtered []PLGeneratedWorkout
	for _, w := range result {
		if len(w.Exercises) > 0 {
			filtered = append(filtered, w)
		}
	}

	return filtered
}

// ========================================
// ВАЛИДАТОР ПРОГРАММЫ
// ========================================

// PLValidationResult результат валидации программы пауэрлифтинга
type PLValidationResult struct {
	IsValid  bool
	Warnings []string
	Errors   []string
	Stats    PLProgramStats
}

// PLProgramStats статистика программы пауэрлифтинга
type PLProgramStats struct {
	TotalWeeks     int
	TotalWorkouts  int
	TotalKPS       int
	TotalTonnage   float64
	AvgKPSPerWeek  float64
	AvgTonnageWeek float64
	AvgIntensity   float64
	KPSByExercise  map[string]int
}

// ValidateProgram проверяет программу на соответствие методике
func (pg *ProgramGenerator) ValidateProgram(program *PLGeneratedProgram) PLValidationResult {
	result := PLValidationResult{
		IsValid: true,
		Stats: PLProgramStats{
			KPSByExercise: make(map[string]int),
		},
	}

	// Считаем статистику
	for _, week := range program.Weeks {
		result.Stats.TotalWeeks++
		result.Stats.TotalKPS += week.TotalKPS
		result.Stats.TotalTonnage += week.Tonnage

		for _, workout := range week.Workouts {
			result.Stats.TotalWorkouts++

			for _, ex := range workout.Exercises {
				result.Stats.KPSByExercise[ex.Name] += ex.TotalReps
			}
		}
	}

	if result.Stats.TotalWeeks > 0 {
		result.Stats.AvgKPSPerWeek = float64(result.Stats.TotalKPS) / float64(result.Stats.TotalWeeks)
		result.Stats.AvgTonnageWeek = result.Stats.TotalTonnage / float64(result.Stats.TotalWeeks)
	}

	// Проверки по методике Шейко
	// КПШ в неделю для жима: 50-100 для разрядников, 70-120 для КМС/МС
	if result.Stats.AvgKPSPerWeek < 40 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Низкий КПШ в неделю: %.0f (рекомендуется 50-100)", result.Stats.AvgKPSPerWeek))
	}
	if result.Stats.AvgKPSPerWeek > 150 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Высокий КПШ в неделю: %.0f (риск перетренированности)", result.Stats.AvgKPSPerWeek))
	}

	// Проверка наличия соревновательного движения
	hasCompetition := false
	for name := range result.Stats.KPSByExercise {
		normalized := normalizeExerciseName(name)
		// Жим лёжа (включая вариации с бруском)
		if (plContains(normalized, "жим лёжа") || plContains(normalized, "жим лежа") ||
			(plContains(normalized, "жим") && !plContains(normalized, "стоя") && !plContains(normalized, "сидя"))) {
			hasCompetition = true
			break
		}
		// Присед
		if plContains(normalized, "присед") {
			hasCompetition = true
			break
		}
		// Становая тяга (исключая тягу блока и в наклоне)
		if plContains(normalized, "тяга") && !plContains(normalized, "блок") &&
			!plContains(normalized, "наклон") && !plContains(normalized, "гантел") {
			hasCompetition = true
			break
		}
	}

	if !hasCompetition {
		result.Errors = append(result.Errors, "Не найдено соревновательное движение!")
		result.IsValid = false
	}

	return result
}

// FormatProgram форматирует программу для вывода
func FormatPLProgram(program *PLGeneratedProgram) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("ПРОГРАММА: %s\n\n", program.Name))
	sb.WriteString("1ПМ атлета:\n")
	sb.WriteString(fmt.Sprintf("• Присед: %.1f кг\n", program.AthleteMaxes.Squat))
	sb.WriteString(fmt.Sprintf("• Жим лёжа: %.1f кг\n", program.AthleteMaxes.Bench))
	sb.WriteString(fmt.Sprintf("• Тяга: %.1f кг\n\n", program.AthleteMaxes.Deadlift))

	for _, week := range program.Weeks {
		sb.WriteString("┌─────────────────────────────────\n")
		sb.WriteString(fmt.Sprintf("│ НЕДЕЛЯ %d (%s)\n", week.WeekNum, week.Phase))
		sb.WriteString(fmt.Sprintf("│ КПШ: %d | Тоннаж: %.1f т\n", week.TotalKPS, week.Tonnage))
		sb.WriteString("└─────────────────────────────────\n\n")

		for _, workout := range week.Workouts {
			sb.WriteString(fmt.Sprintf("━━━ %s ━━━\n\n", workout.Name))

			for i, ex := range workout.Exercises {
				sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, ex.Name))

				for _, set := range ex.Sets {
					if set.WeightKg > 0 {
						sb.WriteString(fmt.Sprintf("   %.0f кг × %d повт × %d подх (%.0f%%)\n",
							set.WeightKg, set.Reps, set.Sets, set.Percent))
					} else {
						sb.WriteString(fmt.Sprintf("   %d повт × %d подх\n", set.Reps, set.Sets))
					}
				}
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("══════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf("ИТОГО: КПШ %d | Тоннаж %.1f т\n",
		program.TotalKPS, program.TotalTonnage))

	return sb.String()
}

// ========================================
// РЕДАКТОР ПРОГРАММЫ
// ========================================

// ProgramEditor редактор программы для тренера
type ProgramEditor struct {
	program *PLGeneratedProgram
}

// NewProgramEditor создаёт редактор для программы
func NewProgramEditor(program *PLGeneratedProgram) *ProgramEditor {
	return &ProgramEditor{program: program}
}

// GetProgram возвращает программу
func (pe *ProgramEditor) GetProgram() *PLGeneratedProgram {
	return pe.program
}

// ReplaceExercise заменяет упражнение на всех тренировках
func (pe *ProgramEditor) ReplaceExercise(oldName, newName string, newOneRM float64) {
	for weekIdx := range pe.program.Weeks {
		for workoutIdx := range pe.program.Weeks[weekIdx].Workouts {
			for exIdx := range pe.program.Weeks[weekIdx].Workouts[workoutIdx].Exercises {
				ex := &pe.program.Weeks[weekIdx].Workouts[workoutIdx].Exercises[exIdx]
				if normalizeExerciseName(ex.Name) == normalizeExerciseName(oldName) {
					ex.Name = newName
					// Пересчитываем веса если есть новый 1ПМ
					if newOneRM > 0 {
						pe.recalculateExerciseWeights(ex, newOneRM)
					}
				}
			}
		}
	}
	pe.RecalculateStats()
}

// recalculateExerciseWeights пересчитывает веса упражнения
func (pe *ProgramEditor) recalculateExerciseWeights(ex *PLGeneratedExercise, oneRM float64) {
	ex.Tonnage = 0
	ex.TotalReps = 0

	for setIdx := range ex.Sets {
		set := &ex.Sets[setIdx]
		if set.Percent > 0 {
			set.WeightKg = math.Round(oneRM*set.Percent/100/2.5) * 2.5
		}
		reps := set.Reps * set.Sets
		ex.TotalReps += reps
		ex.Tonnage += set.WeightKg * float64(reps) / 1000
	}

	if ex.TotalReps > 0 && oneRM > 0 {
		ex.AvgPercent = (ex.Tonnage * 1000 / float64(ex.TotalReps)) / oneRM * 100
	}
}

// ModifySet модифицирует подход в упражнении
func (pe *ProgramEditor) ModifySet(weekNum, workoutNum, exerciseIdx, setIdx int, percent float64, reps, sets int) error {
	if weekNum < 1 || weekNum > len(pe.program.Weeks) {
		return fmt.Errorf("неверный номер недели: %d", weekNum)
	}

	week := &pe.program.Weeks[weekNum-1]
	if workoutNum < 1 || workoutNum > len(week.Workouts) {
		return fmt.Errorf("неверный номер тренировки: %d", workoutNum)
	}

	workout := &week.Workouts[workoutNum-1]
	if exerciseIdx < 0 || exerciseIdx >= len(workout.Exercises) {
		return fmt.Errorf("неверный индекс упражнения: %d", exerciseIdx)
	}

	ex := &workout.Exercises[exerciseIdx]
	if setIdx < 0 || setIdx >= len(ex.Sets) {
		return fmt.Errorf("неверный индекс подхода: %d", setIdx)
	}

	set := &ex.Sets[setIdx]
	if percent > 0 {
		set.Percent = percent
	}
	if reps > 0 {
		set.Reps = reps
	}
	if sets > 0 {
		set.Sets = sets
	}

	pe.RecalculateStats()
	return nil
}

// AddExercise добавляет упражнение в тренировку
func (pe *ProgramEditor) AddExercise(weekNum, workoutNum int, exercise PLGeneratedExercise) error {
	if weekNum < 1 || weekNum > len(pe.program.Weeks) {
		return fmt.Errorf("неверный номер недели: %d", weekNum)
	}

	week := &pe.program.Weeks[weekNum-1]
	if workoutNum < 1 || workoutNum > len(week.Workouts) {
		return fmt.Errorf("неверный номер тренировки: %d", workoutNum)
	}

	workout := &week.Workouts[workoutNum-1]
	workout.Exercises = append(workout.Exercises, exercise)

	pe.RecalculateStats()
	return nil
}

// RemoveExercise удаляет упражнение из тренировки
func (pe *ProgramEditor) RemoveExercise(weekNum, workoutNum, exerciseIdx int) error {
	if weekNum < 1 || weekNum > len(pe.program.Weeks) {
		return fmt.Errorf("неверный номер недели: %d", weekNum)
	}

	week := &pe.program.Weeks[weekNum-1]
	if workoutNum < 1 || workoutNum > len(week.Workouts) {
		return fmt.Errorf("неверный номер тренировки: %d", workoutNum)
	}

	workout := &week.Workouts[workoutNum-1]
	if exerciseIdx < 0 || exerciseIdx >= len(workout.Exercises) {
		return fmt.Errorf("неверный индекс упражнения: %d", exerciseIdx)
	}

	workout.Exercises = append(workout.Exercises[:exerciseIdx], workout.Exercises[exerciseIdx+1:]...)

	pe.RecalculateStats()
	return nil
}

// ScaleIntensity масштабирует интенсивность всей программы
func (pe *ProgramEditor) ScaleIntensity(factor float64) {
	for weekIdx := range pe.program.Weeks {
		for workoutIdx := range pe.program.Weeks[weekIdx].Workouts {
			for exIdx := range pe.program.Weeks[weekIdx].Workouts[workoutIdx].Exercises {
				ex := &pe.program.Weeks[weekIdx].Workouts[workoutIdx].Exercises[exIdx]
				for setIdx := range ex.Sets {
					set := &ex.Sets[setIdx]
					set.Percent *= factor
					if set.WeightKg > 0 && set.Percent > 0 {
						// Пересчитываем вес
						oneRM := set.WeightKg / (set.Percent / factor / 100)
						set.WeightKg = math.Round(oneRM*set.Percent/100/2.5) * 2.5
					}
				}
			}
		}
	}
	pe.RecalculateStats()
}

// RecalculateStats пересчитывает всю статистику программы
func (pe *ProgramEditor) RecalculateStats() {
	pe.program.TotalKPS = 0
	pe.program.TotalTonnage = 0
	pe.program.Stats = make(map[string]int)

	for weekIdx := range pe.program.Weeks {
		week := &pe.program.Weeks[weekIdx]
		week.TotalKPS = 0
		week.Tonnage = 0

		for workoutIdx := range week.Workouts {
			workout := &week.Workouts[workoutIdx]
			workout.TotalKPS = 0
			workout.Tonnage = 0

			for exIdx := range workout.Exercises {
				ex := &workout.Exercises[exIdx]

				// Пересчитываем статистику упражнения
				ex.TotalReps = 0
				ex.Tonnage = 0
				for _, set := range ex.Sets {
					reps := set.Reps * set.Sets
					ex.TotalReps += reps
					ex.Tonnage += set.WeightKg * float64(reps) / 1000
				}

				workout.TotalKPS += ex.TotalReps
				workout.Tonnage += ex.Tonnage
				pe.program.Stats[ex.Name] += ex.TotalReps
			}

			week.TotalKPS += workout.TotalKPS
			week.Tonnage += workout.Tonnage
		}

		pe.program.TotalKPS += week.TotalKPS
		pe.program.TotalTonnage += week.Tonnage
	}
}

// GetWeekSummary возвращает краткую статистику недели
func (pe *ProgramEditor) GetWeekSummary(weekNum int) (kps int, tonnage float64, err error) {
	if weekNum < 1 || weekNum > len(pe.program.Weeks) {
		return 0, 0, fmt.Errorf("неверный номер недели: %d", weekNum)
	}
	week := pe.program.Weeks[weekNum-1]
	return week.TotalKPS, week.Tonnage, nil
}

// ExportToJSON экспортирует программу в JSON
func (pe *ProgramEditor) ExportToJSON() ([]byte, error) {
	return json.MarshalIndent(pe.program, "", "  ")
}

// ImportFromJSON импортирует программу из JSON
func (pe *ProgramEditor) ImportFromJSON(data []byte) error {
	var program PLGeneratedProgram
	if err := json.Unmarshal(data, &program); err != nil {
		return err
	}
	pe.program = &program
	pe.RecalculateStats()
	return nil
}

// FormatWeekCompact компактный формат недели
func FormatWeekCompact(week PLGeneratedWeek) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Неделя %d | КПШ: %d | Тоннаж: %.1f т\n",
		week.WeekNum, week.TotalKPS, week.Tonnage))

	for _, workout := range week.Workouts {
		sb.WriteString(fmt.Sprintf("  %s:\n", workout.Name))
		for _, ex := range workout.Exercises {
			sb.WriteString(fmt.Sprintf("    %s: ", ex.Name))
			for i, set := range ex.Sets {
				if i > 0 {
					sb.WriteString(", ")
				}
				if set.WeightKg > 0 {
					sb.WriteString(fmt.Sprintf("%.0f×%d×%d", set.WeightKg, set.Reps, set.Sets))
				}
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
