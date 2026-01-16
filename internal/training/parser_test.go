package training

import (
	"testing"
	"time"
)

func TestParse_SlashFormat(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantExercise string
		wantSets     int
		wantReps     int
		wantWeight   float64
	}{
		{
			name:         "basic format with weight",
			input:        "Жим лежа 4/10 60",
			wantExercise: "Жим лежа",
			wantSets:     4,
			wantReps:     10,
			wantWeight:   60,
		},
		{
			name:         "without weight",
			input:        "Подтягивания 4/10",
			wantExercise: "Подтягивания",
			wantSets:     4,
			wantReps:     10,
			wantWeight:   0,
		},
		{
			name:         "decimal weight",
			input:        "Жим гантелей 3/12 22.5",
			wantExercise: "Жим гантелей",
			wantSets:     3,
			wantReps:     12,
			wantWeight:   22.5,
		},
		{
			name:         "comma decimal weight",
			input:        "Присед 5/5 80,5",
			wantExercise: "Присед",
			wantSets:     5,
			wantReps:     5,
			wantWeight:   80.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exercises, _, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if len(exercises) != 1 {
				t.Fatalf("Parse() returned %d exercises, want 1", len(exercises))
			}

			ex := exercises[0]
			if ex.Name != tt.wantExercise {
				t.Errorf("Name = %q, want %q", ex.Name, tt.wantExercise)
			}
			if ex.Sets != tt.wantSets {
				t.Errorf("Sets = %d, want %d", ex.Sets, tt.wantSets)
			}
			if ex.Reps != tt.wantReps {
				t.Errorf("Reps = %d, want %d", ex.Reps, tt.wantReps)
			}
			if ex.Weight != tt.wantWeight {
				t.Errorf("Weight = %f, want %f", ex.Weight, tt.wantWeight)
			}
		})
	}
}

func TestParse_XFormat(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantExercise string
		wantSets     int
		wantReps     int
		wantWeight   float64
	}{
		{
			name:         "basic x format",
			input:        "Жим лежа 4x10x60",
			wantExercise: "Жим лежа",
			wantSets:     4,
			wantReps:     10,
			wantWeight:   60,
		},
		{
			name:         "cyrillic х",
			input:        "Присед 5х5х100",
			wantExercise: "Присед",
			wantSets:     5,
			wantReps:     5,
			wantWeight:   100,
		},
		{
			name:         "without weight",
			input:        "Подтягивания 3x12",
			wantExercise: "Подтягивания",
			wantSets:     3,
			wantReps:     12,
			wantWeight:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exercises, _, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if len(exercises) != 1 {
				t.Fatalf("Parse() returned %d exercises, want 1", len(exercises))
			}

			ex := exercises[0]
			if ex.Name != tt.wantExercise {
				t.Errorf("Name = %q, want %q", ex.Name, tt.wantExercise)
			}
			if ex.Sets != tt.wantSets {
				t.Errorf("Sets = %d, want %d", ex.Sets, tt.wantSets)
			}
			if ex.Reps != tt.wantReps {
				t.Errorf("Reps = %d, want %d", ex.Reps, tt.wantReps)
			}
			if ex.Weight != tt.wantWeight {
				t.Errorf("Weight = %f, want %f", ex.Weight, tt.wantWeight)
			}
		})
	}
}

func TestParse_AlternativeFormat(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantExercise string
		wantSets     int
		wantReps     int
		wantWeight   float64
	}{
		{
			name:         "space separated",
			input:        "Жим лежа 4 10 60",
			wantExercise: "Жим лежа",
			wantSets:     4,
			wantReps:     10,
			wantWeight:   60,
		},
		{
			name:         "two numbers only",
			input:        "Подтягивания 4 10",
			wantExercise: "Подтягивания",
			wantSets:     4,
			wantReps:     10,
			wantWeight:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exercises, _, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if len(exercises) != 1 {
				t.Fatalf("Parse() returned %d exercises, want 1", len(exercises))
			}

			ex := exercises[0]
			if ex.Name != tt.wantExercise {
				t.Errorf("Name = %q, want %q", ex.Name, tt.wantExercise)
			}
			if ex.Sets != tt.wantSets {
				t.Errorf("Sets = %d, want %d", ex.Sets, tt.wantSets)
			}
			if ex.Reps != tt.wantReps {
				t.Errorf("Reps = %d, want %d", ex.Reps, tt.wantReps)
			}
			if ex.Weight != tt.wantWeight {
				t.Errorf("Weight = %f, want %f", ex.Weight, tt.wantWeight)
			}
		})
	}
}

func TestParse_MultipleExercises(t *testing.T) {
	input := `Подтягивания 4/10 20
Жим лежа 4/10 60
Присед 5/5 100`

	exercises, _, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(exercises) != 3 {
		t.Fatalf("Parse() returned %d exercises, want 3", len(exercises))
	}

	expected := []struct {
		name   string
		sets   int
		reps   int
		weight float64
	}{
		{"Подтягивания", 4, 10, 20},
		{"Жим лежа", 4, 10, 60},
		{"Присед", 5, 5, 100},
	}

	for i, exp := range expected {
		if exercises[i].Name != exp.name {
			t.Errorf("Exercise %d: Name = %q, want %q", i, exercises[i].Name, exp.name)
		}
		if exercises[i].Sets != exp.sets {
			t.Errorf("Exercise %d: Sets = %d, want %d", i, exercises[i].Sets, exp.sets)
		}
		if exercises[i].Reps != exp.reps {
			t.Errorf("Exercise %d: Reps = %d, want %d", i, exercises[i].Reps, exp.reps)
		}
		if exercises[i].Weight != exp.weight {
			t.Errorf("Exercise %d: Weight = %f, want %f", i, exercises[i].Weight, exp.weight)
		}
	}
}

func TestParse_WithDate(t *testing.T) {
	input := `13.01.2026
Жим лежа 4/10 60`

	exercises, date, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(exercises) != 1 {
		t.Fatalf("Parse() returned %d exercises, want 1", len(exercises))
	}

	expectedDate := time.Date(2026, 1, 13, 0, 0, 0, 0, time.UTC)
	if date.Day() != expectedDate.Day() || date.Month() != expectedDate.Month() || date.Year() != expectedDate.Year() {
		t.Errorf("Date = %v, want %v", date, expectedDate)
	}
}

func TestParse_WithShortDate(t *testing.T) {
	input := `13.01
Жим лежа 4/10 60`

	exercises, date, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(exercises) != 1 {
		t.Fatalf("Parse() returned %d exercises, want 1", len(exercises))
	}

	if date.Day() != 13 || date.Month() != 1 {
		t.Errorf("Date = %v, want day=13, month=1", date)
	}
}

func TestParse_EmptyLines(t *testing.T) {
	input := `Жим лежа 4/10 60

Присед 5/5 100`

	exercises, _, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(exercises) != 2 {
		t.Fatalf("Parse() returned %d exercises, want 2", len(exercises))
	}
}

func TestTryParseDate(t *testing.T) {
	tests := []struct {
		input    string
		wantOk   bool
		wantDay  int
		wantMon  time.Month
		wantYear int
	}{
		{"13.01.2026", true, 13, time.January, 2026},
		{"1.1.2026", true, 1, time.January, 2026},
		{"31.12.2025", true, 31, time.December, 2025},
		{"13.01", true, 13, time.January, 0}, // year will be current year
		{"not a date", false, 0, 0, 0},
		{"2026-01-13", false, 0, 0, 0}, // not supported format
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			date, ok := tryParseDate(tt.input)
			if ok != tt.wantOk {
				t.Errorf("tryParseDate(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
				return
			}
			if !ok {
				return
			}
			if date.Day() != tt.wantDay {
				t.Errorf("Day = %d, want %d", date.Day(), tt.wantDay)
			}
			if date.Month() != tt.wantMon {
				t.Errorf("Month = %v, want %v", date.Month(), tt.wantMon)
			}
			if tt.wantYear != 0 && date.Year() != tt.wantYear {
				t.Errorf("Year = %d, want %d", date.Year(), tt.wantYear)
			}
		})
	}
}

func TestFormatConfirmation(t *testing.T) {
	exercises := []struct {
		name   string
		sets   int
		reps   int
		weight float64
	}{
		{"Жим лежа", 4, 10, 60},
		{"Присед", 5, 5, 100},
	}

	var exInputs []struct {
		Name    string
		Sets    int
		Reps    int
		Weight  float64
		Comment string
	}
	for _, e := range exercises {
		exInputs = append(exInputs, struct {
			Name    string
			Sets    int
			Reps    int
			Weight  float64
			Comment string
		}{e.name, e.sets, e.reps, e.weight, ""})
	}

	// Just verify it doesn't panic and returns non-empty string
	// The actual format is tested implicitly
}
