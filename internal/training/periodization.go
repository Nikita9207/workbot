package training

import (
	"time"
	"workbot/internal/models"
)

// PeriodizationConfig for generating periodization structure
type PeriodizationConfig struct {
	TotalWeeks      int
	Goal            string // сила, масса, похудение, соревнования
	DeloadFrequency int    // every N weeks
}

// PeriodizationTemplate predefined periodization templates
type PeriodizationTemplate struct {
	Name       string
	Goal       string
	Mesocycles []MesocycleTemplate
}

// MesocycleTemplate template for mesocycle generation
type MesocycleTemplate struct {
	Name             string
	Phase            models.PlanPhase
	WeeksPercent     float64 // % of total weeks
	IntensityPercent int
	VolumePercent    int
}

// Standard periodization templates
var PeriodizationTemplates = map[string]PeriodizationTemplate{
	"strength": {
		Name: "Программа на силу",
		Goal: "сила",
		Mesocycles: []MesocycleTemplate{
			{Name: "Подготовительный", Phase: models.PhaseHypertrophy, WeeksPercent: 0.25, IntensityPercent: 70, VolumePercent: 100},
			{Name: "Базовый силовой", Phase: models.PhaseStrength, WeeksPercent: 0.35, IntensityPercent: 82, VolumePercent: 90},
			{Name: "Интенсивный", Phase: models.PhasePower, WeeksPercent: 0.25, IntensityPercent: 88, VolumePercent: 75},
			{Name: "Пиковый", Phase: models.PhasePeaking, WeeksPercent: 0.15, IntensityPercent: 95, VolumePercent: 50},
		},
	},
	"hypertrophy": {
		Name: "Программа на массу",
		Goal: "масса",
		Mesocycles: []MesocycleTemplate{
			{Name: "Втягивающий", Phase: models.PhaseHypertrophy, WeeksPercent: 0.15, IntensityPercent: 65, VolumePercent: 80},
			{Name: "Накопительный 1", Phase: models.PhaseHypertrophy, WeeksPercent: 0.30, IntensityPercent: 72, VolumePercent: 100},
			{Name: "Накопительный 2", Phase: models.PhaseHypertrophy, WeeksPercent: 0.30, IntensityPercent: 75, VolumePercent: 110},
			{Name: "Интенсивный", Phase: models.PhaseStrength, WeeksPercent: 0.25, IntensityPercent: 80, VolumePercent: 85},
		},
	},
	"weight_loss": {
		Name: "Программа на похудение",
		Goal: "похудение",
		Mesocycles: []MesocycleTemplate{
			{Name: "Адаптационный", Phase: models.PhaseHypertrophy, WeeksPercent: 0.20, IntensityPercent: 60, VolumePercent: 100},
			{Name: "Жиросжигающий 1", Phase: models.PhaseHypertrophy, WeeksPercent: 0.30, IntensityPercent: 65, VolumePercent: 110},
			{Name: "Жиросжигающий 2", Phase: models.PhaseHypertrophy, WeeksPercent: 0.30, IntensityPercent: 68, VolumePercent: 115},
			{Name: "Поддерживающий", Phase: models.PhaseStrength, WeeksPercent: 0.20, IntensityPercent: 75, VolumePercent: 80},
		},
	},
	"competition": {
		Name: "Подготовка к соревнованиям",
		Goal: "соревнования",
		Mesocycles: []MesocycleTemplate{
			{Name: "Общеподготовительный", Phase: models.PhaseHypertrophy, WeeksPercent: 0.25, IntensityPercent: 68, VolumePercent: 100},
			{Name: "Специально-подготовительный", Phase: models.PhaseStrength, WeeksPercent: 0.30, IntensityPercent: 80, VolumePercent: 90},
			{Name: "Предсоревновательный", Phase: models.PhasePower, WeeksPercent: 0.25, IntensityPercent: 88, VolumePercent: 70},
			{Name: "Пиковый", Phase: models.PhasePeaking, WeeksPercent: 0.20, IntensityPercent: 95, VolumePercent: 40},
		},
	},
}

// GenerateMesocycles creates mesocycle structure from template
func GenerateMesocycles(totalWeeks int, goal string, deloadFrequency int) []models.Mesocycle {
	template, ok := PeriodizationTemplates[goal]
	if !ok {
		template = PeriodizationTemplates["strength"] // default
	}

	mesocycles := make([]models.Mesocycle, 0, len(template.Mesocycles))
	currentWeek := 1

	for i, mt := range template.Mesocycles {
		mesoWeeks := int(float64(totalWeeks) * mt.WeeksPercent)
		if mesoWeeks < 2 {
			mesoWeeks = 2
		}
		// Last mesocycle gets remaining weeks
		if i == len(template.Mesocycles)-1 {
			mesoWeeks = totalWeeks - currentWeek + 1
		}

		meso := models.Mesocycle{
			Name:             mt.Name,
			Phase:            mt.Phase,
			WeekStart:        currentWeek,
			WeekEnd:          currentWeek + mesoWeeks - 1,
			IntensityPercent: mt.IntensityPercent,
			VolumePercent:    mt.VolumePercent,
			RPETarget:        models.DefaultPhaseConfigs[mt.Phase].RPETarget,
			OrderNum:         i + 1,
		}

		// Generate microcycles for this mesocycle
		meso.Microcycles = generateMicrocycles(meso, deloadFrequency)

		mesocycles = append(mesocycles, meso)
		currentWeek += mesoWeeks
	}

	return mesocycles
}

// generateMicrocycles creates weekly structure within a mesocycle
func generateMicrocycles(meso models.Mesocycle, deloadFrequency int) []models.Microcycle {
	microcycles := make([]models.Microcycle, 0)

	for week := meso.WeekStart; week <= meso.WeekEnd; week++ {
		isDeload := deloadFrequency > 0 && week%deloadFrequency == 0

		name := ""
		if isDeload {
			name = "Разгрузочная неделя"
		}

		volumeMod := 1.0
		intensityMod := 1.0
		if isDeload {
			volumeMod = 0.5
			intensityMod = 0.65
		}

		microcycles = append(microcycles, models.Microcycle{
			WeekNumber:        week,
			Name:              name,
			IsDeload:          isDeload,
			VolumeModifier:    volumeMod,
			IntensityModifier: intensityMod,
		})
	}

	return microcycles
}

// GenerateFullPeriodization creates complete plan structure
func GenerateFullPeriodization(
	clientID int,
	planName string,
	startDate time.Time,
	totalWeeks int,
	daysPerWeek int,
	goal string,
	deloadFrequency int,
) models.TrainingPlan {
	endDate := startDate.AddDate(0, 0, totalWeeks*7)

	plan := models.TrainingPlan{
		ClientID:    clientID,
		Name:        planName,
		StartDate:   startDate,
		EndDate:     &endDate,
		Status:      models.PlanStatusDraft,
		Goal:        goal,
		DaysPerWeek: daysPerWeek,
		TotalWeeks:  totalWeeks,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	plan.Mesocycles = GenerateMesocycles(totalWeeks, goal, deloadFrequency)

	return plan
}

// GetPhaseForWeek returns the phase for a given week number
func GetPhaseForWeek(mesocycles []models.Mesocycle, week int) models.PlanPhase {
	for _, meso := range mesocycles {
		if week >= meso.WeekStart && week <= meso.WeekEnd {
			return meso.Phase
		}
	}
	return models.PhaseHypertrophy // default
}

// GetMesoForWeek returns the mesocycle for a given week number
func GetMesoForWeek(mesocycles []models.Mesocycle, week int) *models.Mesocycle {
	for i := range mesocycles {
		if week >= mesocycles[i].WeekStart && week <= mesocycles[i].WeekEnd {
			return &mesocycles[i]
		}
	}
	return nil
}

// IsDeloadWeek checks if a week is a deload week
func IsDeloadWeek(week, deloadFrequency int) bool {
	if deloadFrequency <= 0 {
		return false
	}
	return week%deloadFrequency == 0
}

// GetWeekName returns descriptive name for a week
func GetWeekName(week int, mesocycles []models.Mesocycle, deloadFrequency int) string {
	isDeload := IsDeloadWeek(week, deloadFrequency)
	meso := GetMesoForWeek(mesocycles, week)

	if isDeload {
		return "Неделя " + string(rune('0'+week%10)) + " (Разгрузка)"
	}
	if meso != nil {
		return "Неделя " + string(rune('0'+week%10)) + " (" + meso.Phase.NameRu() + ")"
	}
	return "Неделя " + string(rune('0'+week%10))
}

// CompetitionDisciplines - соревновательные дисциплины
var CompetitionDisciplines = []string{
	"Ягодичный мост",
	"Жим лёжа",
	"Становая тяга",
	"Строгий подъём на бицепс",
	"Свободный подъём на бицепс",
}

// GetCompetitionDisciplines returns list of competition exercises
func GetCompetitionDisciplines() []string {
	return CompetitionDisciplines
}

// IsCompetitionDiscipline checks if exercise is a competition discipline
func IsCompetitionDiscipline(exerciseName string) bool {
	for _, d := range CompetitionDisciplines {
		if d == exerciseName {
			return true
		}
	}
	return false
}
