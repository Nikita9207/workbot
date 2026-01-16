package bot

import "testing"

func TestValidateWeight(t *testing.T) {
	tests := []struct {
		name    string
		weight  float64
		wantErr bool
	}{
		{"valid weight", 80.0, false},
		{"minimum valid", 0.1, false},
		{"maximum valid", 500.0, false},
		{"zero weight", 0, true},
		{"negative weight", -10, true},
		{"too heavy", 501, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWeight(tt.weight)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateWeight(%v) error = %v, wantErr %v", tt.weight, err, tt.wantErr)
			}
		})
	}
}

func TestValidateReps(t *testing.T) {
	tests := []struct {
		name    string
		reps    int
		wantErr bool
	}{
		{"valid reps", 10, false},
		{"minimum valid", 1, false},
		{"maximum valid", 100, false},
		{"zero reps", 0, true},
		{"negative reps", -5, true},
		{"too many reps", 101, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateReps(tt.reps)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateReps(%v) error = %v, wantErr %v", tt.reps, err, tt.wantErr)
			}
		})
	}
}

func TestValidateSets(t *testing.T) {
	tests := []struct {
		name    string
		sets    int
		wantErr bool
	}{
		{"valid sets", 4, false},
		{"minimum valid", 1, false},
		{"maximum valid", 20, false},
		{"zero sets", 0, true},
		{"negative sets", -1, true},
		{"too many sets", 21, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSets(tt.sets)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSets(%v) error = %v, wantErr %v", tt.sets, err, tt.wantErr)
			}
		})
	}
}

func TestValidateExerciseName(t *testing.T) {
	tests := []struct {
		name    string
		exName  string
		wantErr bool
	}{
		{"valid name", "Жим лежа", false},
		{"minimum length", "Жим", false},
		{"with spaces trimmed", "  Присед  ", false},
		{"empty name", "", true},
		{"too short", "ab", true},
		{"only spaces", "   ", true},
		{"too long", string(make([]byte, 101)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExerciseName(tt.exName)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateExerciseName(%q) error = %v, wantErr %v", tt.exName, err, tt.wantErr)
			}
		})
	}
}

func TestValidateDateString(t *testing.T) {
	tests := []struct {
		name    string
		date    string
		wantErr bool
	}{
		{"full date", "13.01.2026", false},
		{"short date", "13.01", false},
		{"single digit day", "1.01.2026", false},
		{"single digit month", "13.1.2026", false},
		{"both single digit", "1.1", false},
		{"empty date", "", true},
		{"invalid format dash", "2026-01-13", true},
		{"invalid format slash", "13/01/2026", true},
		{"not a date", "hello", true},
		{"only spaces", "   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDateString(tt.date)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDateString(%q) error = %v, wantErr %v", tt.date, err, tt.wantErr)
			}
		})
	}
}

func TestValidateWeeks(t *testing.T) {
	tests := []struct {
		name    string
		weeks   int
		wantErr bool
	}{
		{"valid weeks", 12, false},
		{"minimum valid", 1, false},
		{"maximum valid", 52, false},
		{"zero weeks", 0, true},
		{"negative weeks", -1, true},
		{"too many weeks", 53, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWeeks(tt.weeks)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateWeeks(%v) error = %v, wantErr %v", tt.weeks, err, tt.wantErr)
			}
		})
	}
}

func TestValidateDaysPerWeek(t *testing.T) {
	tests := []struct {
		name    string
		days    int
		wantErr bool
	}{
		{"valid days", 3, false},
		{"minimum valid", 1, false},
		{"maximum valid", 7, false},
		{"zero days", 0, true},
		{"negative days", -1, true},
		{"too many days", 8, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDaysPerWeek(tt.days)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDaysPerWeek(%v) error = %v, wantErr %v", tt.days, err, tt.wantErr)
			}
		})
	}
}

func TestValidate1PM(t *testing.T) {
	tests := []struct {
		name    string
		weight  float64
		wantErr bool
	}{
		{"valid 1PM", 100.0, false},
		{"minimum valid", 0.1, false},
		{"maximum valid", 600.0, false},
		{"zero 1PM", 0, true},
		{"negative 1PM", -10, true},
		{"too heavy 1PM", 601, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate1PM(tt.weight)
			if (err != nil) != tt.wantErr {
				t.Errorf("validate1PM(%v) error = %v, wantErr %v", tt.weight, err, tt.wantErr)
			}
		})
	}
}

func TestValidateIntensityPercent(t *testing.T) {
	tests := []struct {
		name    string
		percent float64
		wantErr bool
	}{
		{"valid percent", 75.0, false},
		{"minimum valid", 0.0, false},
		{"maximum valid", 100.0, false},
		{"negative percent", -1.0, true},
		{"over 100 percent", 101.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIntensityPercent(tt.percent)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateIntensityPercent(%v) error = %v, wantErr %v", tt.percent, err, tt.wantErr)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := ValidationError{Field: "test", Message: "test message"}
	if err.Error() != "test message" {
		t.Errorf("ValidationError.Error() = %q, want %q", err.Error(), "test message")
	}
}
