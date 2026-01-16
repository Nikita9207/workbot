package models

import "time"

// Training представляет тренировку из общего листа Excel
type Training struct {
	RowNum      int
	ClientID    int
	Date        string
	Time        string
	Description string
	Sent        bool
}

// ClientTraining представляет упражнение с персонального листа клиента
type ClientTraining struct {
	SheetName     string
	RowNum        int
	ClientID      int
	Date          string
	TrainingNum   int
	Exercise      string
	Sets          int
	Reps          int
	Weight        float64
	Tonnage       float64
	Status        string
	Feedback      string
	Rating        int
	CompletedDate string
	CompletedTime string
	TotalTonnage  float64
	TotalTrains   int
	Sent          bool
}

// TrainingGroup группирует упражнения одной тренировки
type TrainingGroup struct {
	ClientID    int
	TelegramID  int64
	SheetName   string
	TrainingNum int
	Date        string
	Exercises   []ClientTraining
}

// ExerciseInput представляет введённое упражнение из Telegram
type ExerciseInput struct {
	Name    string
	Sets    int
	Reps    int
	Weight  float64
	Comment string
}

// DayData содержит данные о тренировке за день для календаря
type DayData struct {
	Tonnage float64
	Status  string
}

// Client представляет клиента из БД
type Client struct {
	ID         int
	Name       string
	Surname    string
	Phone      string
	BirthDate  string
	TelegramID int64
	CreatedAt  *time.Time
}
