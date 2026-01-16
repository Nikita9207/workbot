package models

import "time"

// TrainingLog represents a single completed exercise
type TrainingLog struct {
	ID              int       `json:"id"`
	ClientID        int       `json:"client_id"`
	TrainingPlanID  *int      `json:"training_plan_id"`
	PlanExerciseID  *int      `json:"plan_exercise_id"`
	ExerciseID      int       `json:"exercise_id"`
	ExerciseName    string    `json:"exercise_name"` // joined
	MuscleGroup     string    `json:"muscle_group"`  // joined
	TrainingDate    time.Time `json:"training_date"`
	WeekNumber      int       `json:"week_number"`
	DayOfWeek       int       `json:"day_of_week"`
	SetsPlanned     int       `json:"sets_planned"`
	SetsCompleted   int       `json:"sets_completed"`
	RepsPlanned     int       `json:"reps_planned"`
	RepsCompleted   int       `json:"reps_completed"`
	WeightPlanned   float64   `json:"weight_planned"`
	WeightKg        float64   `json:"weight_kg"`
	TonnageKg       float64   `json:"tonnage_kg"` // computed
	RPETarget       float64   `json:"rpe_target"`
	RPEActual       float64   `json:"rpe_actual"`
	Status          string    `json:"status"` // completed, partial, skipped, modified
	Notes           string    `json:"notes"`
	CreatedAt       time.Time `json:"created_at"`
}

// VolumeAnalytics represents pre-computed volume statistics
type VolumeAnalytics struct {
	ID             int       `json:"id"`
	ClientID       int       `json:"client_id"`
	TrainingPlanID *int      `json:"training_plan_id"`
	WeekNumber     int       `json:"week_number"`
	MuscleGroup    string    `json:"muscle_group"`
	TotalSets      int       `json:"total_sets"`
	TotalReps      int       `json:"total_reps"`
	TotalTonnage   float64   `json:"total_tonnage"`
	AvgIntensity   float64   `json:"avg_intensity"`
	ComputedAt     time.Time `json:"computed_at"`
}

// WeeklySummary represents weekly training summary
type WeeklySummary struct {
	ID                 int       `json:"id"`
	ClientID           int       `json:"client_id"`
	TrainingPlanID     *int      `json:"training_plan_id"`
	WeekNumber         int       `json:"week_number"`
	WeekStartDate      time.Time `json:"week_start_date"`
	TrainingsPlanned   int       `json:"trainings_planned"`
	TrainingsCompleted int       `json:"trainings_completed"`
	TotalTonnage       float64   `json:"total_tonnage"`
	TotalSets          int       `json:"total_sets"`
	TotalReps          int       `json:"total_reps"`
	AvgRPE             float64   `json:"avg_rpe"`
	CompliancePercent  float64   `json:"compliance_percent"` // % выполнения плана
	Notes              string    `json:"notes"`
	ComputedAt         time.Time `json:"computed_at"`
}

// ExerciseProgress represents progress tracking for an exercise
type ExerciseProgress struct {
	ID           int       `json:"id"`
	ClientID     int       `json:"client_id"`
	ExerciseID   int       `json:"exercise_id"`
	ExerciseName string    `json:"exercise_name"`
	RecordDate   time.Time `json:"record_date"`
	BestWeight   float64   `json:"best_weight"`
	BestReps     int       `json:"best_reps"`
	Estimated1PM float64   `json:"estimated_1pm"`
	TotalVolume  int       `json:"total_volume"` // sets * reps за день
	TotalTonnage float64   `json:"total_tonnage"`
	CreatedAt    time.Time `json:"created_at"`
}

// ClientAnalytics aggregated analytics for a client
type ClientAnalytics struct {
	ClientID          int                       `json:"client_id"`
	ClientName        string                    `json:"client_name"`
	TotalTrainings    int                       `json:"total_trainings"`
	TotalTonnage      float64                   `json:"total_tonnage"`
	AvgTrainingsWeek  float64                   `json:"avg_trainings_week"`
	WeeklyVolume      []VolumeAnalytics         `json:"weekly_volume"`
	MuscleGroupVolume map[string]float64        `json:"muscle_group_volume"` // group -> total sets
	TopExercises      []ExerciseProgress        `json:"top_exercises"`
	ProgressTrend     []WeeklySummary           `json:"progress_trend"`
}

// MuscleGroupSummary volume per muscle group
type MuscleGroupSummary struct {
	MuscleGroup  string  `json:"muscle_group"`
	TotalSets    int     `json:"total_sets"`
	TotalReps    int     `json:"total_reps"`
	TotalTonnage float64 `json:"total_tonnage"`
	AvgIntensity float64 `json:"avg_intensity"`
}

// WeeklyMuscleVolume volume breakdown by muscle group per week
type WeeklyMuscleVolume struct {
	WeekNumber int                  `json:"week_number"`
	ByMuscle   []MuscleGroupSummary `json:"by_muscle"`
	TotalSets  int                  `json:"total_sets"`
}

// TonnageChartData for generating tonnage chart
type TonnageChartData struct {
	Weeks       []int     `json:"weeks"`
	Tonnage     []float64 `json:"tonnage"`
	Labels      []string  `json:"labels"`
}

// PMProgressChartData for generating 1PM progress chart
type PMProgressChartData struct {
	Dates    []time.Time `json:"dates"`
	Values   []float64   `json:"values"`
	Exercise string      `json:"exercise"`
}
