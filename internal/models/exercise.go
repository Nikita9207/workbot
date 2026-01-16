package models

import "time"

// Exercise represents an exercise from the catalog
type Exercise struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	NameNormalized string    `json:"name_normalized"`
	MuscleGroup    string    `json:"muscle_group"`  // грудь, спина, ноги, плечи, руки, кор
	MovementType   string    `json:"movement_type"` // compound, isolation
	Equipment      string    `json:"equipment"`     // штанга, гантели, тренажёр, собственный вес
	IsTrackable1PM bool      `json:"is_trackable_1pm"`
	CreatedAt      time.Time `json:"created_at"`
}

// Exercise1PM represents a single 1RM record for a client
type Exercise1PM struct {
	ID           int       `json:"id"`
	ClientID     int       `json:"client_id"`
	ExerciseID   int       `json:"exercise_id"`
	ExerciseName string    `json:"exercise_name"` // joined from exercises
	OnePMKg      float64   `json:"one_pm_kg"`
	TestDate     time.Time `json:"test_date"`
	CalcMethod   string    `json:"calc_method"` // manual, brzycki, epley, average
	SourceWeight float64   `json:"source_weight"`
	SourceReps   int       `json:"source_reps"`
	Notes        string    `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
	CreatedBy    int64     `json:"created_by"`
}

// Exercise1PMHistory holds history of 1PM changes for an exercise
type Exercise1PMHistory struct {
	ExerciseID   int           `json:"exercise_id"`
	ExerciseName string        `json:"exercise_name"`
	MuscleGroup  string        `json:"muscle_group"`
	Records      []Exercise1PM `json:"records"`
	InitialPM    float64       `json:"initial_pm"`
	CurrentPM    float64       `json:"current_pm"`
	GainKg       float64       `json:"gain_kg"`
	GainPercent  float64       `json:"gain_percent"`
}

// Client1PMSummary holds all 1PM data for a client
type Client1PMSummary struct {
	ClientID     int                  `json:"client_id"`
	ClientName   string               `json:"client_name"`
	Exercises    []Exercise1PMHistory `json:"exercises"`
	LastTestDate time.Time            `json:"last_test_date"`
}

// ExerciseListItem for selection in bot
type ExerciseListItem struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	MuscleGroup string `json:"muscle_group"`
	Has1PM      bool   `json:"has_1pm"`
	Current1PM  float64 `json:"current_1pm"`
}
