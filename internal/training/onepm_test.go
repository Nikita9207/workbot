package training

import (
	"math"
	"testing"
)

func TestCalculate1PM_Brzycki(t *testing.T) {
	tests := []struct {
		name   string
		weight float64
		reps   int
		want   float64
	}{
		{"100kg x 5", 100, 5, 112.5},  // 100 * (36 / 32) = 112.5
		{"80kg x 10", 80, 10, 106.67}, // 80 * (36 / 27) ≈ 106.67
		{"60kg x 3", 60, 3, 63.53},    // 60 * (36 / 34) ≈ 63.53
		{"1 rep is same as weight", 100, 1, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Calculate1PM(tt.weight, tt.reps, "brzycki")
			if math.Abs(got-tt.want) > 0.1 {
				t.Errorf("Calculate1PM(%v, %v, brzycki) = %v, want %v", tt.weight, tt.reps, got, tt.want)
			}
		})
	}
}

func TestCalculate1PM_Epley(t *testing.T) {
	tests := []struct {
		name   string
		weight float64
		reps   int
		want   float64
	}{
		{"100kg x 5", 100, 5, 116.65}, // 100 * (1 + 0.0333 * 5) = 116.65
		{"80kg x 10", 80, 10, 106.64}, // 80 * (1 + 0.333) = 106.64
		{"1 rep is same as weight", 100, 1, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Calculate1PM(tt.weight, tt.reps, "epley")
			if math.Abs(got-tt.want) > 0.1 {
				t.Errorf("Calculate1PM(%v, %v, epley) = %v, want %v", tt.weight, tt.reps, got, tt.want)
			}
		})
	}
}

func TestCalculate1PM_Average(t *testing.T) {
	// Average of Brzycki and Epley
	weight := 100.0
	reps := 5
	brzycki := Calculate1PM(weight, reps, "brzycki")
	epley := Calculate1PM(weight, reps, "epley")
	expectedAvg := (brzycki + epley) / 2

	got := Calculate1PM(weight, reps, "average")
	if math.Abs(got-expectedAvg) > 0.1 {
		t.Errorf("Calculate1PM average = %v, want %v", got, expectedAvg)
	}
}

func TestCalculate1PM_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		weight float64
		reps   int
		want   float64
	}{
		{"zero weight", 0, 5, 0},
		{"zero reps", 100, 0, 0},
		{"negative weight", -100, 5, 0},
		{"negative reps", 100, -5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Calculate1PM(tt.weight, tt.reps, "brzycki")
			if got != tt.want {
				t.Errorf("Calculate1PM(%v, %v) = %v, want %v", tt.weight, tt.reps, got, tt.want)
			}
		})
	}
}

func TestCalculate1PM_DefaultMethod(t *testing.T) {
	// Unknown method should default to Brzycki
	got := Calculate1PM(100, 5, "unknown")
	expected := Calculate1PM(100, 5, "brzycki")
	if got != expected {
		t.Errorf("Unknown method = %v, want %v (brzycki default)", got, expected)
	}
}

func TestCalculateWorkingWeight(t *testing.T) {
	tests := []struct {
		name    string
		onePM   float64
		percent float64
		want    float64
	}{
		{"75% of 100kg", 100, 75, 75.0},
		{"80% of 100kg", 100, 80, 80.0},
		{"85% of 100kg", 100, 85, 85.0},
		{"75% of 112.5kg", 112.5, 75, 84.5},
		{"round to 0.5", 100, 77, 77.0},
		{"round to 0.5 down", 100, 77.2, 77.0},
		{"round to 0.5 up", 100, 77.3, 77.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateWorkingWeight(tt.onePM, tt.percent)
			if got != tt.want {
				t.Errorf("CalculateWorkingWeight(%v, %v) = %v, want %v", tt.onePM, tt.percent, got, tt.want)
			}
		})
	}
}

func TestCalculateWorkingWeight_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		onePM   float64
		percent float64
		want    float64
	}{
		{"zero 1PM", 0, 75, 0},
		{"zero percent", 100, 0, 0},
		{"negative 1PM", -100, 75, 0},
		{"negative percent", 100, -75, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateWorkingWeight(tt.onePM, tt.percent)
			if got != tt.want {
				t.Errorf("CalculateWorkingWeight(%v, %v) = %v, want %v", tt.onePM, tt.percent, got, tt.want)
			}
		})
	}
}

func TestCalculateWorkingWeightRound(t *testing.T) {
	tests := []struct {
		name      string
		onePM     float64
		percent   float64
		increment float64
		want      float64
	}{
		{"round to 2.5kg", 100, 77, 2.5, 77.5},
		{"round to 5kg", 100, 77, 5, 75},
		{"round to 2.5kg up", 100, 78, 2.5, 77.5},
		{"round to 2.5kg down", 100, 76, 2.5, 75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateWorkingWeightRound(tt.onePM, tt.percent, tt.increment)
			if got != tt.want {
				t.Errorf("CalculateWorkingWeightRound(%v, %v, %v) = %v, want %v",
					tt.onePM, tt.percent, tt.increment, got, tt.want)
			}
		})
	}
}

func TestGetIntensityForReps(t *testing.T) {
	tests := []struct {
		reps     int
		expected float64
	}{
		{1, 100},
		{5, 88},
		{10, 73},
		{12, 67},
		{0, 100},  // edge case
		{25, 50},  // beyond table
		{15, 55},  // approximate
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := GetIntensityForReps(tt.reps)
			if got != tt.expected {
				t.Errorf("GetIntensityForReps(%d) = %v, want %v", tt.reps, got, tt.expected)
			}
		})
	}
}

func TestGetRepsForIntensity(t *testing.T) {
	tests := []struct {
		intensity float64
		expected  int
	}{
		{100, 1},
		{88, 5},
		{73, 10},
		{105, 1},  // above 100
		{50, 20},  // below 55
		{90, 3},   // approximate
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := GetRepsForIntensity(tt.intensity)
			if got != tt.expected {
				t.Errorf("GetRepsForIntensity(%v) = %d, want %d", tt.intensity, got, tt.expected)
			}
		})
	}
}

func TestEstimateRPE(t *testing.T) {
	tests := []struct {
		name   string
		weight float64
		onePM  float64
		reps   int
		minRPE float64
		maxRPE float64
	}{
		{"heavy set", 90, 100, 3, 7, 10},
		{"moderate set", 75, 100, 8, 6, 10},
		{"light set", 50, 100, 15, 1, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateRPE(tt.weight, tt.onePM, tt.reps)
			if got < tt.minRPE || got > tt.maxRPE {
				t.Errorf("EstimateRPE(%v, %v, %v) = %v, want between %v and %v",
					tt.weight, tt.onePM, tt.reps, got, tt.minRPE, tt.maxRPE)
			}
		})
	}
}

func TestEstimateRPE_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		weight float64
		onePM  float64
		reps   int
		want   float64
	}{
		{"zero weight", 0, 100, 5, 0},
		{"zero 1PM", 100, 0, 5, 0},
		{"zero reps", 100, 100, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateRPE(tt.weight, tt.onePM, tt.reps)
			if got != tt.want {
				t.Errorf("EstimateRPE(%v, %v, %v) = %v, want %v",
					tt.weight, tt.onePM, tt.reps, got, tt.want)
			}
		})
	}
}

func TestCalcMethodName(t *testing.T) {
	tests := []struct {
		method   string
		expected string
	}{
		{"brzycki", "Формула Бжицки"},
		{"epley", "Формула Эпли"},
		{"average", "Среднее (Бжицки + Эпли)"},
		{"manual", "Ручной ввод"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			got := CalcMethodName(tt.method)
			if got != tt.expected {
				t.Errorf("CalcMethodName(%q) = %q, want %q", tt.method, got, tt.expected)
			}
		})
	}
}
