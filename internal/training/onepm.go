package training

import "math"

// Calculate1PM calculates 1RM using specified formula
// weight: weight lifted in kg
// reps: number of repetitions performed
// method: "brzycki", "epley", or "average"
func Calculate1PM(weight float64, reps int, method string) float64 {
	if reps <= 0 || weight <= 0 {
		return 0
	}
	if reps == 1 {
		return weight
	}

	switch method {
	case "brzycki":
		return calculateBrzycki(weight, reps)
	case "epley":
		return calculateEpley(weight, reps)
	case "average":
		brzycki := calculateBrzycki(weight, reps)
		epley := calculateEpley(weight, reps)
		return math.Round((brzycki+epley)/2*100) / 100
	default:
		return calculateBrzycki(weight, reps)
	}
}

// calculateBrzycki Brzycki formula: 1RM = weight * (36 / (37 - reps))
// Most accurate for reps < 10
func calculateBrzycki(weight float64, reps int) float64 {
	if reps >= 37 {
		return weight * 1.0 // cap at reps = 36
	}
	return math.Round(weight*(36.0/float64(37-reps))*100) / 100
}

// calculateEpley Epley formula: 1RM = weight * (1 + 0.0333 * reps)
// Good for higher rep ranges
func calculateEpley(weight float64, reps int) float64 {
	return math.Round(weight*(1+0.0333*float64(reps))*100) / 100
}

// CalculateWorkingWeight calculates the working weight from 1PM and percentage
// Rounds to nearest 0.5kg for practical use
func CalculateWorkingWeight(onePM float64, intensityPercent float64) float64 {
	if onePM <= 0 || intensityPercent <= 0 {
		return 0
	}
	raw := onePM * intensityPercent / 100
	// Round to nearest 0.5 kg
	return math.Round(raw*2) / 2
}

// CalculateWorkingWeightRound rounds to specified increment (e.g., 2.5kg for plates)
func CalculateWorkingWeightRound(onePM float64, intensityPercent float64, increment float64) float64 {
	if onePM <= 0 || intensityPercent <= 0 || increment <= 0 {
		return 0
	}
	raw := onePM * intensityPercent / 100
	return math.Round(raw/increment) * increment
}

// IntensityToReps returns approximate reps achievable at given intensity
var IntensityToReps = map[int]int{
	100: 1,
	97:  2,
	94:  3,
	91:  4,
	88:  5,
	85:  6,
	82:  7,
	79:  8,
	76:  9,
	73:  10,
	70:  11,
	67:  12,
	64:  14,
	61:  16,
	58:  18,
	55:  20,
}

// RepsToIntensity returns recommended intensity % for target reps
var RepsToIntensity = map[int]float64{
	1:  100,
	2:  97,
	3:  94,
	4:  91,
	5:  88,
	6:  85,
	7:  82,
	8:  79,
	9:  76,
	10: 73,
	11: 70,
	12: 67,
	14: 64,
	16: 61,
	18: 58,
	20: 55,
}

// GetIntensityForReps returns recommended intensity % for target reps
func GetIntensityForReps(targetReps int) float64 {
	if intensity, ok := RepsToIntensity[targetReps]; ok {
		return intensity
	}
	// Approximate using formula: intensity ≈ 100 - (reps * 3)
	if targetReps < 1 {
		return 100
	}
	if targetReps > 20 {
		return 50
	}
	return float64(100 - targetReps*3)
}

// GetRepsForIntensity returns approximate reps for given intensity
func GetRepsForIntensity(intensity float64) int {
	if intensity >= 100 {
		return 1
	}
	if intensity <= 55 {
		return 20
	}
	// Find closest match
	intIntensity := int(math.Round(intensity))
	if reps, ok := IntensityToReps[intIntensity]; ok {
		return reps
	}
	// Approximate: reps ≈ (100 - intensity) / 3
	return int(math.Round((100 - intensity) / 3))
}

// EstimateRPE estimates RPE based on weight relative to 1PM and reps
func EstimateRPE(weight, onePM float64, reps int) float64 {
	if onePM <= 0 || weight <= 0 || reps <= 0 {
		return 0
	}
	intensity := (weight / onePM) * 100
	// RPE ≈ 10 - (max_reps - actual_reps)
	maxReps := GetRepsForIntensity(intensity)
	rpe := 10.0 - float64(maxReps-reps)
	if rpe < 1 {
		return 1
	}
	if rpe > 10 {
		return 10
	}
	return math.Round(rpe*10) / 10
}

// CalcMethodName returns human-readable method name in Russian
func CalcMethodName(method string) string {
	switch method {
	case "brzycki":
		return "Формула Бжицки"
	case "epley":
		return "Формула Эпли"
	case "average":
		return "Среднее (Бжицки + Эпли)"
	case "manual":
		return "Ручной ввод"
	default:
		return method
	}
}
