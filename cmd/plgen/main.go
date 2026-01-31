package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"workbot/clients/ai"
)

func main() {
	// Флаги
	listTemplates := flag.Bool("list", false, "Показать доступные шаблоны")
	templateName := flag.String("template", "", "Имя шаблона для генерации")
	squat := flag.Float64("squat", 0, "1ПМ приседа (кг)")
	bench := flag.Float64("bench", 0, "1ПМ жима лёжа (кг)")
	deadlift := flag.Float64("deadlift", 0, "1ПМ становой тяги (кг)")
	hipThrust := flag.Float64("hipthrust", 0, "1ПМ ягодичного моста (кг)")
	liftType := flag.String("lift", "full", "Тип дисциплины: full (троеборье), bench (жим), squat (присед), deadlift (тяга), hipthrust (ягодичный мост)")
	daysPerWeek := flag.Int("days", 0, "Количество тренировок в неделю (2-4, 0 = как в шаблоне)")
	autoSelect := flag.Bool("auto", false, "Автоматически выбрать шаблон по уровню атлета")
	output := flag.String("output", "", "Файл для сохранения программы (markdown)")

	flag.Parse()

	// Создаём генератор
	gen, err := ai.NewProgramGenerator()
	if err != nil {
		fmt.Printf("❌ Ошибка инициализации: %v\n", err)
		os.Exit(1)
	}

	// Показать список шаблонов
	if *listTemplates {
		fmt.Println("📋 Доступные шаблоны программ:")
		for i, name := range gen.ListTemplates() {
			fmt.Printf("%d. %s\n", i+1, name)
		}
		return
	}

	// Интерактивный режим если параметры не указаны
	if (*templateName == "" && !*autoSelect) || (*bench == 0 && *hipThrust == 0) {
		runInteractive(gen)
		return
	}

	// Генерация программы
	maxes := ai.AthleteMaxes{
		Squat:    *squat,
		Bench:    *bench,
		Deadlift: *deadlift,
		HipThrust: *hipThrust,
	}

	// Определяем тип дисциплины
	var lt ai.LiftType
	switch *liftType {
	case "bench", "жим":
		lt = ai.LiftTypeBench
	case "squat", "присед":
		lt = ai.LiftTypeSquat
	case "deadlift", "тяга":
		lt = ai.LiftTypeDeadlift
	case "hipthrust", "hip_thrust", "ягодичный", "мост":
		lt = ai.LiftTypeHipThrust
	default:
		lt = ai.LiftTypeFull
	}

	opts := ai.GenerationOptions{
		LiftType:         lt,
		DaysPerWeek:      *daysPerWeek,
		IncludeAccessory: true,
	}

	var program *ai.PLGeneratedProgram
	if *autoSelect {
		// Автоматический выбор шаблона
		program, err = gen.GenerateAutomatic(maxes, opts)
	} else {
		program, err = gen.GenerateFromTemplateWithOptions(*templateName, maxes, opts)
	}
	if err != nil {
		fmt.Printf("❌ Ошибка генерации: %v\n", err)
		os.Exit(1)
	}

	// Валидация
	validation := gen.ValidateProgram(program)
	if !validation.IsValid {
		fmt.Println("⚠️ Программа содержит ошибки:")
		for _, e := range validation.Errors {
			fmt.Printf("  ❌ %s\n", e)
		}
	}
	if len(validation.Warnings) > 0 {
		fmt.Println("⚠️ Предупреждения:")
		for _, w := range validation.Warnings {
			fmt.Printf("  ⚠️ %s\n", w)
		}
	}

	// Форматируем вывод
	formatted := ai.FormatPLProgram(program)

	if *output != "" {
		err := os.WriteFile(*output, []byte(formatted), 0644)
		if err != nil {
			fmt.Printf("❌ Ошибка записи файла: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Программа сохранена в %s\n", *output)
	} else {
		fmt.Println(formatted)
	}

	// Статистика
	fmt.Println("\n📊 Статистика программы:")
	fmt.Printf("  Недель: %d\n", validation.Stats.TotalWeeks)
	fmt.Printf("  Тренировок: %d\n", validation.Stats.TotalWorkouts)
	fmt.Printf("  Общий КПШ: %d\n", validation.Stats.TotalKPS)
	fmt.Printf("  Общий тоннаж: %.1f т\n", validation.Stats.TotalTonnage)
	fmt.Printf("  Средний КПШ/неделю: %.0f\n", validation.Stats.AvgKPSPerWeek)
}

func runInteractive(gen *ai.ProgramGenerator) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("🏋️ Генератор программ пауэрлифтинга")
	fmt.Println("====================================")

	// Выбор шаблона
	templates := gen.ListTemplates()
	fmt.Println("Доступные шаблоны:")
	for i, name := range templates {
		fmt.Printf("  %d. %s\n", i+1, name)
	}

	fmt.Print("\nВыберите шаблон [номер]: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	templateIdx, err := strconv.Atoi(input)
	if err != nil || templateIdx < 1 || templateIdx > len(templates) {
		fmt.Println("❌ Неверный выбор")
		return
	}
	templateName := templates[templateIdx-1]

	// Выбор дисциплины
	fmt.Println("\nВыберите дисциплину:")
	fmt.Println("  1. Троеборье (присед + жим + тяга)")
	fmt.Println("  2. Только жим лёжа")
	fmt.Println("  3. Только присед")
	fmt.Println("  4. Только становая тяга")
	fmt.Println("  5. Ягодичный мост (Hip Thrust)")
	fmt.Print("Выбор [1]: ")
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var liftType ai.LiftType
	switch input {
	case "2":
		liftType = ai.LiftTypeBench
	case "3":
		liftType = ai.LiftTypeSquat
	case "4":
		liftType = ai.LiftTypeDeadlift
	case "5":
		liftType = ai.LiftTypeHipThrust
	default:
		liftType = ai.LiftTypeFull
	}

	// Ввод 1ПМ в зависимости от дисциплины
	fmt.Println("\nВведите ваши максимумы (1ПМ):")

	var bench, squat, deadlift, hipThrust float64

	switch liftType {
	case ai.LiftTypeBench:
		fmt.Print("Жим лёжа (кг): ")
		input, _ = reader.ReadString('\n')
		bench, err = strconv.ParseFloat(strings.TrimSpace(input), 64)
		if err != nil || bench <= 0 {
			fmt.Println("❌ Неверное значение")
			return
		}
	case ai.LiftTypeSquat:
		fmt.Print("Присед (кг): ")
		input, _ = reader.ReadString('\n')
		squat, err = strconv.ParseFloat(strings.TrimSpace(input), 64)
		if err != nil || squat <= 0 {
			fmt.Println("❌ Неверное значение")
			return
		}
	case ai.LiftTypeDeadlift:
		fmt.Print("Становая тяга (кг): ")
		input, _ = reader.ReadString('\n')
		deadlift, err = strconv.ParseFloat(strings.TrimSpace(input), 64)
		if err != nil || deadlift <= 0 {
			fmt.Println("❌ Неверное значение")
			return
		}
	case ai.LiftTypeHipThrust:
		fmt.Print("Ягодичный мост (кг): ")
		input, _ = reader.ReadString('\n')
		hipThrust, err = strconv.ParseFloat(strings.TrimSpace(input), 64)
		if err != nil || hipThrust <= 0 {
			fmt.Println("❌ Неверное значение")
			return
		}
		// Опционально: жим для верхней части тела
		fmt.Print("Жим лёжа (кг, опционально, Enter для пропуска): ")
		input, _ = reader.ReadString('\n')
		if strings.TrimSpace(input) != "" {
			bench, _ = strconv.ParseFloat(strings.TrimSpace(input), 64)
		}
	default:
		// Троеборье - все три
		fmt.Print("Жим лёжа (кг): ")
		input, _ = reader.ReadString('\n')
		bench, err = strconv.ParseFloat(strings.TrimSpace(input), 64)
		if err != nil || bench <= 0 {
			fmt.Println("❌ Неверное значение")
			return
		}

		fmt.Print("Присед (кг): ")
		input, _ = reader.ReadString('\n')
		squat, _ = strconv.ParseFloat(strings.TrimSpace(input), 64)

		fmt.Print("Становая тяга (кг): ")
		input, _ = reader.ReadString('\n')
		deadlift, _ = strconv.ParseFloat(strings.TrimSpace(input), 64)
	}

	// Выбор количества тренировок в неделю
	fmt.Println("\nКоличество тренировок в неделю:")
	fmt.Println("  0. Как в шаблоне (по умолчанию)")
	fmt.Println("  2. 2 тренировки")
	fmt.Println("  3. 3 тренировки")
	fmt.Println("  4. 4 тренировки")
	fmt.Print("Выбор [0]: ")
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	daysPerWeek := 0
	if input != "" {
		daysPerWeek, _ = strconv.Atoi(input)
		if daysPerWeek < 0 || daysPerWeek > 6 {
			daysPerWeek = 0
		}
	}

	// Генерация
	fmt.Println("\n⏳ Генерирую программу...")

	maxes := ai.AthleteMaxes{
		Squat:    squat,
		Bench:    bench,
		Deadlift: deadlift,
		HipThrust: hipThrust,
	}

	opts := ai.GenerationOptions{
		LiftType:         liftType,
		DaysPerWeek:      daysPerWeek,
		IncludeAccessory: true,
	}

	program, err := gen.GenerateFromTemplateWithOptions(templateName, maxes, opts)
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}

	// Валидация
	validation := gen.ValidateProgram(program)

	// Вывод краткой статистики
	fmt.Println("\n✅ Программа сгенерирована!")
	fmt.Printf("\n📊 Статистика:\n")
	fmt.Printf("  Шаблон: %s\n", program.Name)
	fmt.Printf("  Недель: %d\n", len(program.Weeks))
	fmt.Printf("  Общий КПШ: %d\n", program.TotalKPS)
	fmt.Printf("  Общий тоннаж: %.1f т\n", program.TotalTonnage)
	fmt.Printf("  Средний КПШ/неделю: %.0f\n", validation.Stats.AvgKPSPerWeek)

	if len(validation.Warnings) > 0 {
		fmt.Println("\n⚠️ Предупреждения:")
		for _, w := range validation.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}

	// Предложение сохранить
	fmt.Print("\nСохранить программу? [y/N]: ")
	input, _ = reader.ReadString('\n')
	if strings.ToLower(strings.TrimSpace(input)) == "y" {
		fmt.Print("Имя файла [program.md]: ")
		input, _ = reader.ReadString('\n')
		filename := strings.TrimSpace(input)
		if filename == "" {
			filename = "program.md"
		}

		formatted := ai.FormatPLProgram(program)
		err := os.WriteFile(filename, []byte(formatted), 0644)
		if err != nil {
			fmt.Printf("❌ Ошибка: %v\n", err)
			return
		}
		fmt.Printf("✅ Сохранено в %s\n", filename)
	}

	// Показать первую неделю
	fmt.Print("\nПоказать первую неделю? [Y/n]: ")
	input, _ = reader.ReadString('\n')
	if strings.ToLower(strings.TrimSpace(input)) != "n" && len(program.Weeks) > 0 {
		fmt.Println("\n" + ai.FormatWeekCompact(program.Weeks[0]))
	}
}
