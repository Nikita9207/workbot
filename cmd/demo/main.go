package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"workbot/internal/generator"
	"workbot/internal/generator/formatter"
	"workbot/internal/models"
)

func main() {
	scenario := flag.Int("scenario", 0, "Номер сценария (1-4), 0 = все")
	outputDir := flag.String("output", "./test_programs", "Директория для вывода")
	flag.Parse()

	// Определяем путь к данным
	execPath, _ := os.Executable()
	dataDir := filepath.Join(filepath.Dir(execPath), "..", "..", "data")

	// Если запускаем через go run, используем относительный путь
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		dataDir = "./data"
	}

	// Создаём selector
	selector, err := generator.NewExerciseSelector(dataDir)
	if err != nil {
		fmt.Printf("Ошибка загрузки упражнений: %v\n", err)
		fmt.Println("Используем встроенные данные...")
	}

	// Создаём форматтер
	telegramFormatter := formatter.NewTelegramFormatter()

	// Создаём директорию для вывода
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Printf("Ошибка создания директории: %v\n", err)
		return
	}

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║        ДЕМОНСТРАЦИЯ ГЕНЕРАТОРА ТРЕНИРОВОЧНЫХ ПРОГРАММ        ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	scenarios := []int{1, 2, 3, 4}
	if *scenario > 0 && *scenario <= 4 {
		scenarios = []int{*scenario}
	}

	for _, s := range scenarios {
		runScenario(s, selector, telegramFormatter, *outputDir)
		fmt.Println()
	}

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("Программы сохранены в: %s\n", *outputDir)
}

func runScenario(num int, selector *generator.ExerciseSelector, formatter *formatter.TelegramFormatter, outputDir string) {
	switch num {
	case 1:
		runScenario1(selector, formatter, outputDir)
	case 2:
		runScenario2(selector, formatter, outputDir)
	case 3:
		runScenario3(selector, formatter, outputDir)
	case 4:
		runScenario4(selector, formatter, outputDir)
	}
}

// Сценарий 1: Домашние тренировки TRX + гири (гипертрофия)
func runScenario1(selector *generator.ExerciseSelector, fmt_ *formatter.TelegramFormatter, outputDir string) {
	fmt.Println("┌──────────────────────────────────────────────────────────────┐")
	fmt.Println("│ СЦЕНАРИЙ 1: Домашние тренировки TRX + гири (гипертрофия)     │")
	fmt.Println("└──────────────────────────────────────────────────────────────┘")

	client := &models.ClientProfile{
		ID:         1,
		Name:       "Анна Петрова",
		Gender:     "female",
		Age:        32,
		Weight:     65,
		Height:     168,
		Experience: models.ExpIntermediate,
		Constraints: []models.ClientConstraint{
			{BodyZone: models.ZoneKnee, Severity: models.SeverityRelative, Notes: "Дискомфорт в коленях"},
		},
		AvailableEquip: []models.EquipmentType{
			models.EquipmentTRX,
			models.EquipmentKettlebell,
			models.EquipmentBodyweight,
		},
		AvailableKBWeights: []float64{8, 12},
		Location:           models.LocationHome,
	}

	fmt.Printf("Клиент: %s, %d лет, %.0f кг\n", client.Name, client.Age, client.Weight)
	fmt.Printf("Уровень: %s\n", client.Experience)
	fmt.Printf("Оборудование: TRX, гири %v кг\n", client.AvailableKBWeights)
	fmt.Printf("Ограничения: дискомфорт в коленях\n")
	fmt.Println()

	gen := generator.NewHypertrophyGenerator(selector, client)
	program, err := gen.Generate(generator.HypertrophyConfig{
		TotalWeeks:  9,
		DaysPerWeek: 3,
		Split:       "fullbody",
	})

	if err != nil {
		fmt.Printf("Ошибка генерации: %v\n", err)
		return
	}

	// Форматируем и выводим
	output := fmt_.FormatProgram(program)
	fmt.Println(output)

	// Сохраняем в файл
	saveToFile(outputDir, "scenario1_hypertrophy_home.txt", output)
}

// Сценарий 2: Зал полное оборудование (гипертрофия)
func runScenario2(selector *generator.ExerciseSelector, fmt_ *formatter.TelegramFormatter, outputDir string) {
	fmt.Println("┌──────────────────────────────────────────────────────────────┐")
	fmt.Println("│ СЦЕНАРИЙ 2: Зал полное оборудование (гипертрофия)            │")
	fmt.Println("└──────────────────────────────────────────────────────────────┘")

	client := &models.ClientProfile{
		ID:         2,
		Name:       "Михаил Сидоров",
		Gender:     "male",
		Age:        28,
		Weight:     82,
		Height:     180,
		Experience: models.ExpAdvanced,
		Constraints: []models.ClientConstraint{}, // Без ограничений
		AvailableEquip: []models.EquipmentType{
			models.EquipmentBarbell,
			models.EquipmentDumbbell,
			models.EquipmentCable,
			models.EquipmentMachine,
			models.EquipmentPullupBar,
			models.EquipmentBench,
			models.EquipmentRack,
		},
		Location: models.LocationGym,
		OnePM: map[string]float64{
			"squat":    140,
			"bench":    100,
			"deadlift": 180,
			"ohp":      65,
		},
	}

	fmt.Printf("Клиент: %s, %d лет, %.0f кг\n", client.Name, client.Age, client.Weight)
	fmt.Printf("Уровень: %s\n", client.Experience)
	fmt.Printf("1ПМ: Присед %.0f кг, Жим %.0f кг, Тяга %.0f кг\n",
		client.OnePM["squat"], client.OnePM["bench"], client.OnePM["deadlift"])
	fmt.Println()

	gen := generator.NewHypertrophyGenerator(selector, client)
	program, err := gen.Generate(generator.HypertrophyConfig{
		TotalWeeks:  12,
		DaysPerWeek: 4,
		Split:       "upper_lower",
	})

	if err != nil {
		fmt.Printf("Ошибка генерации: %v\n", err)
		return
	}

	output := fmt_.FormatProgram(program)
	fmt.Println(output)
	saveToFile(outputDir, "scenario2_hypertrophy_gym.txt", output)
}

// Сценарий 3: Только гири (сила)
func runScenario3(selector *generator.ExerciseSelector, fmt_ *formatter.TelegramFormatter, outputDir string) {
	fmt.Println("┌──────────────────────────────────────────────────────────────┐")
	fmt.Println("│ СЦЕНАРИЙ 3: Только гири (сила)                               │")
	fmt.Println("└──────────────────────────────────────────────────────────────┘")

	client := &models.ClientProfile{
		ID:         3,
		Name:       "Алексей Волков",
		Gender:     "male",
		Age:        35,
		Weight:     90,
		Height:     178,
		Experience: models.ExpIntermediate,
		Constraints: []models.ClientConstraint{
			{BodyZone: models.ZoneLowerBack, Severity: models.SeverityRelative, Notes: "Протрузия L4-L5"},
		},
		AvailableEquip: []models.EquipmentType{
			models.EquipmentKettlebell,
			models.EquipmentBodyweight,
		},
		AvailableKBWeights: []float64{16, 24, 32},
		Location:           models.LocationHome,
	}

	fmt.Printf("Клиент: %s, %d лет, %.0f кг\n", client.Name, client.Age, client.Weight)
	fmt.Printf("Уровень: %s\n", client.Experience)
	fmt.Printf("Оборудование: гири %v кг\n", client.AvailableKBWeights)
	fmt.Printf("Ограничения: протрузия L4-L5\n")
	fmt.Println()

	gen := generator.NewStrengthGenerator(selector, client)
	program, err := gen.Generate(generator.StrengthConfig{
		TotalWeeks:  8,
		DaysPerWeek: 3,
		Focus:       "all",
	})

	if err != nil {
		fmt.Printf("Ошибка генерации: %v\n", err)
		return
	}

	output := fmt_.FormatProgram(program)
	fmt.Println(output)
	saveToFile(outputDir, "scenario3_strength_kettlebell.txt", output)
}

// Сценарий 4: Hyrox подготовка
func runScenario4(selector *generator.ExerciseSelector, fmt_ *formatter.TelegramFormatter, outputDir string) {
	fmt.Println("┌──────────────────────────────────────────────────────────────┐")
	fmt.Println("│ СЦЕНАРИЙ 4: Hyrox подготовка                                 │")
	fmt.Println("└──────────────────────────────────────────────────────────────┘")

	client := &models.ClientProfile{
		ID:         4,
		Name:       "Екатерина Морозова",
		Gender:     "female",
		Age:        29,
		Weight:     62,
		Height:     170,
		Experience: models.ExpIntermediate,
		Constraints: []models.ClientConstraint{}, // Без ограничений
		AvailableEquip: []models.EquipmentType{
			models.EquipmentBarbell,
			models.EquipmentDumbbell,
			models.EquipmentKettlebell,
			models.EquipmentSkiErg,
			models.EquipmentRowErg,
			models.EquipmentAssaultBike,
			models.EquipmentSled,
			models.EquipmentWallBall,
			models.EquipmentSandbag,
			models.EquipmentPullupBar,
		},
		AvailableKBWeights: []float64{12, 16, 20},
		Location:           models.LocationGym,
		OnePM: map[string]float64{
			"squat":    80,
			"bench":    50,
			"deadlift": 100,
		},
	}

	fmt.Printf("Клиент: %s, %d лет, %.0f кг\n", client.Name, client.Age, client.Weight)
	fmt.Printf("Уровень: %s\n", client.Experience)
	fmt.Printf("Цель: Hyrox соревнования\n")
	fmt.Printf("Оборудование: Зал с Ski Erg, Assault Bike, Sled, Гребля\n")
	fmt.Println()

	gen := generator.NewHyroxGenerator(selector, client)
	program, err := gen.Generate(generator.HyroxConfig{
		TotalWeeks:      12,
		DaysPerWeek:     4,
		CompetitionDate: true,
	})

	if err != nil {
		fmt.Printf("Ошибка генерации: %v\n", err)
		return
	}

	output := fmt_.FormatProgram(program)
	fmt.Println(output)
	saveToFile(outputDir, "scenario4_hyrox.txt", output)
}

func saveToFile(dir, filename, content string) {
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		fmt.Printf("Ошибка сохранения %s: %v\n", filename, err)
	} else {
		fmt.Printf("✓ Сохранено: %s\n", path)
	}
}
