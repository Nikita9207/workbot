package training

import (
	"workbot/internal/models"
)

// ProgressionConfig holds parameters for generating progression
type ProgressionConfig struct {
	TotalWeeks          int     // общее количество недель
	DeloadFrequency     int     // каждые N недель делаем разгрузку
	CompoundIncrement   float64 // прирост кг/неделю для базовых упражнений
	IsolationIncrement  float64 // прирост кг/неделю для изоляции
	StartIntensity      float64 // начальная интенсивность % от 1ПМ
	PeakIntensity       float64 // пиковая интенсивность % от 1ПМ
	DeloadIntensityMult float64 // множитель интенсивности на разгрузке (0.6-0.7)
	DeloadVolumeMult    float64 // множитель объёма на разгрузке (0.5)
}

// DefaultProgressionConfig returns standard progression config
func DefaultProgressionConfig() ProgressionConfig {
	return ProgressionConfig{
		TotalWeeks:          12,
		DeloadFrequency:     4, // deload каждую 4-ю неделю
		CompoundIncrement:   2.5,
		IsolationIncrement:  1.0,
		StartIntensity:      65,
		PeakIntensity:       85,
		DeloadIntensityMult: 0.65,
		DeloadVolumeMult:    0.5,
	}
}

// StrengthProgressionConfig for strength-focused programs
func StrengthProgressionConfig() ProgressionConfig {
	return ProgressionConfig{
		TotalWeeks:          12,
		DeloadFrequency:     4,
		CompoundIncrement:   2.5,
		IsolationIncrement:  1.0,
		StartIntensity:      75,
		PeakIntensity:       92,
		DeloadIntensityMult: 0.60,
		DeloadVolumeMult:    0.5,
	}
}

// HypertrophyProgressionConfig for hypertrophy-focused programs
func HypertrophyProgressionConfig() ProgressionConfig {
	return ProgressionConfig{
		TotalWeeks:          12,
		DeloadFrequency:     4,
		CompoundIncrement:   1.5,
		IsolationIncrement:  0.5,
		StartIntensity:      60,
		PeakIntensity:       75,
		DeloadIntensityMult: 0.65,
		DeloadVolumeMult:    0.5,
	}
}

// GenerateProgression creates a week-by-week progression table
func GenerateProgression(
	exercises []models.Exercise,
	client1PMs map[int]float64, // exerciseID -> 1PM
	mesocycles []models.Mesocycle,
	config ProgressionConfig,
) []models.Progression {
	var progression []models.Progression

	for _, ex := range exercises {
		onePM := client1PMs[ex.ID]
		if onePM == 0 {
			continue // skip exercises without 1PM
		}

		for week := 1; week <= config.TotalWeeks; week++ {
			isDeload := config.DeloadFrequency > 0 && week%config.DeloadFrequency == 0

			// Find mesocycle for this week
			var currentMeso *models.Mesocycle
			for i := range mesocycles {
				if week >= mesocycles[i].WeekStart && week <= mesocycles[i].WeekEnd {
					currentMeso = &mesocycles[i]
					break
				}
			}

			var sets, reps int
			var intensity float64

			if isDeload {
				// Deload week: reduced volume and intensity
				sets = 2
				reps = 6
				if currentMeso != nil {
					intensity = float64(currentMeso.IntensityPercent) * config.DeloadIntensityMult
				} else {
					intensity = config.StartIntensity * config.DeloadIntensityMult
				}
			} else if currentMeso != nil {
				// Use mesocycle phase parameters
				phaseConfig := models.DefaultPhaseConfigs[currentMeso.Phase]
				sets = (phaseConfig.SetsRange[0] + phaseConfig.SetsRange[1]) / 2
				reps = (phaseConfig.RepsRange[0] + phaseConfig.RepsRange[1]) / 2

				// Linear progression within mesocycle
				weekInMeso := week - currentMeso.WeekStart
				mesoWeeks := currentMeso.WeekEnd - currentMeso.WeekStart + 1
				// Deload weeks don't count for progression
				effectiveWeek := weekInMeso
				deloadsInMeso := 0
				for w := currentMeso.WeekStart; w < week; w++ {
					if config.DeloadFrequency > 0 && w%config.DeloadFrequency == 0 {
						deloadsInMeso++
					}
				}
				effectiveWeek -= deloadsInMeso

				intensityRange := phaseConfig.IntensityRange[1] - phaseConfig.IntensityRange[0]
				effectiveWeeks := mesoWeeks - (mesoWeeks / config.DeloadFrequency)
				if effectiveWeeks < 1 {
					effectiveWeeks = 1
				}
				intensityStep := intensityRange / float64(effectiveWeeks)
				intensity = phaseConfig.IntensityRange[0] + intensityStep*float64(effectiveWeek)

				// Cap at mesocycle target
				if intensity > float64(currentMeso.IntensityPercent) {
					intensity = float64(currentMeso.IntensityPercent)
				}
			} else {
				// Default linear progression
				weekInBlock := (week - 1) % config.DeloadFrequency
				sets = 4
				reps = 8
				intensityRange := config.PeakIntensity - config.StartIntensity
				intensity = config.StartIntensity + float64(weekInBlock)*(intensityRange/float64(config.DeloadFrequency-1))
			}

			// Adjust sets for compound vs isolation
			if ex.MovementType == "isolation" {
				sets = sets - 1
				if sets < 2 {
					sets = 2
				}
				reps = reps + 2 // Higher reps for isolation
			}

			weight := CalculateWorkingWeight(onePM, intensity)

			progression = append(progression, models.Progression{
				ExerciseID:       ex.ID,
				ExerciseName:     ex.Name,
				MuscleGroup:      ex.MuscleGroup,
				WeekNumber:       week,
				Sets:             sets,
				Reps:             reps,
				WeightKg:         weight,
				IntensityPercent: intensity,
				IsDeload:         isDeload,
			})
		}
	}

	return progression
}

// GenerateSimpleProgression creates progression without mesocycles
func GenerateSimpleProgression(
	exercises []models.Exercise,
	client1PMs map[int]float64,
	config ProgressionConfig,
) []models.Progression {
	return GenerateProgression(exercises, client1PMs, nil, config)
}

// CalculateWeeklyTonnage calculates total tonnage for a week from progression
func CalculateWeeklyTonnage(progression []models.Progression, week int) float64 {
	var total float64
	for _, p := range progression {
		if p.WeekNumber == week {
			total += float64(p.Sets) * float64(p.Reps) * p.WeightKg
		}
	}
	return total
}

// GroupProgressionByWeek groups progression data by week number
func GroupProgressionByWeek(progression []models.Progression) []models.WeekProgression {
	weekMap := make(map[int]*models.WeekProgression)

	for _, p := range progression {
		if _, ok := weekMap[p.WeekNumber]; !ok {
			weekMap[p.WeekNumber] = &models.WeekProgression{
				WeekNumber: p.WeekNumber,
				IsDeload:   p.IsDeload,
				Exercises:  []models.Progression{},
			}
		}
		weekMap[p.WeekNumber].Exercises = append(weekMap[p.WeekNumber].Exercises, p)
	}

	// Convert to slice and sort by week
	result := make([]models.WeekProgression, 0, len(weekMap))
	for week := 1; week <= len(weekMap); week++ {
		if wp, ok := weekMap[week]; ok {
			result = append(result, *wp)
		}
	}
	return result
}

// GroupProgressionByExercise groups progression data by exercise
func GroupProgressionByExercise(progression []models.Progression) map[string][]models.Progression {
	result := make(map[string][]models.Progression)
	for _, p := range progression {
		result[p.ExerciseName] = append(result[p.ExerciseName], p)
	}
	return result
}

// BuildProgressionTable builds a full table for Excel export
func BuildProgressionTable(
	planName, clientName string,
	totalWeeks int,
	progression []models.Progression,
) models.ProgressionTable {
	// Get unique exercise names in order
	exerciseOrder := make([]string, 0)
	seen := make(map[string]bool)
	for _, p := range progression {
		if !seen[p.ExerciseName] {
			seen[p.ExerciseName] = true
			exerciseOrder = append(exerciseOrder, p.ExerciseName)
		}
	}

	return models.ProgressionTable{
		PlanName:      planName,
		ClientName:    clientName,
		TotalWeeks:    totalWeeks,
		ExerciseNames: exerciseOrder,
		Weeks:         GroupProgressionByWeek(progression),
		ByExercise:    GroupProgressionByExercise(progression),
	}
}
