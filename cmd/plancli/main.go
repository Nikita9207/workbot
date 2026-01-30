package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"workbot/clients/ai"
	"workbot/clients/knowledge"
	"workbot/internal/gsheets"
	"workbot/internal/models"
	"workbot/internal/training"
)

// knowledgeStore глобальное хранилище знаний
var knowledgeStore *knowledge.Store
var knowledgePath string

// loadEnvFile загружает переменные окружения из .env файла
func loadEnvFile() {
	// Пробуем загрузить .env из текущей директории
	envPaths := []string{".env", "../.env", "../../.env"}

	for _, path := range envPaths {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			// Пропускаем пустые строки и комментарии
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			// Парсим KEY=VALUE
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Убираем кавычки если есть
			value = strings.Trim(value, `"'`)
			// Устанавливаем только если не установлено
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
		}
		log.Printf("📄 Загружены переменные из %s", path)
		return
	}
}

// loadKnowledgeIfNeeded ленивая загрузка базы знаний
func loadKnowledgeIfNeeded() {
	if knowledgeStore != nil {
		return // Уже загружено
	}

	knowledgeStore = knowledge.NewStore()
	fmt.Println("📚 Загружаю базу знаний...")
	start := time.Now()
	if err := knowledgeStore.Load(knowledgePath); err != nil {
		log.Printf("⚠️ База знаний не загружена: %v", err)
		knowledgeStore = nil
	} else {
		log.Printf("📚 Загружено %d документов за %.1f сек", knowledgeStore.Count(), time.Since(start).Seconds())
	}
}

// Соревновательные дисциплины
const (
	DisciplineBenchPress  = "bench_press"  // Жим лёжа
	DisciplineDeadlift    = "deadlift"     // Становая тяга
	DisciplineGluteBridge = "glute_bridge" // Ягодичный мост
	DisciplineStrictCurl  = "strict_curl"  // Строгий подъём на бицепс
)

// Цели тренировок
const (
	GoalStrength    = "strength"
	GoalHypertrophy = "hypertrophy"
	GoalWeightLoss  = "weight_loss"
	GoalCompetition = "competition"
)

func main() {
	// Загружаем переменные из .env
	loadEnvFile()

	// Флаги командной строки
	jsonFile := flag.String("json", "", "Путь к JSON файлу с планом (ручной режим)")
	credentials := flag.String("creds", "google-credentials.json", "Путь к Google credentials")
	folderID := flag.String("folder", "", "ID папки Google Drive")

	// Режим генерации
	interactive := flag.Bool("i", false, "Интерактивный режим (пошаговый ввод)")
	generate := flag.Bool("generate", false, "Режим генерации плана")

	// Параметры для генерации
	clientName := flag.String("client", "", "Имя клиента")
	goal := flag.String("goal", "", "Цель: strength, hypertrophy, weight_loss, competition")
	disciplines := flag.String("disciplines", "", "Дисциплины через запятую: bench_press,deadlift,glute_bridge,strict_curl")
	compDate := flag.String("date", "", "Дата соревнований/проходки (DD.MM.YYYY)")
	weeks := flag.Int("weeks", 0, "Количество недель (если не указана дата)")
	days := flag.Int("days", 3, "Тренировок в неделю")
	onePMData := flag.String("1pm", "", "1ПМ данные (текущий): 'Жим лёжа:100,Становая тяга:150'")
	targetPMData := flag.String("target", "", "Целевой 1ПМ на соревнования: 'Жим лёжа:110,Становая тяга:160'")

	// AI параметры
	ollamaURL := flag.String("ollama-url", "http://localhost:11434", "URL Ollama сервера")
	ollamaModel := flag.String("ollama-model", "", "Модель Ollama: gemma2 (по умолчанию), glm4-flash")
	knowledgePathFlag := flag.String("knowledge", "knowledge.json", "Путь к базе знаний")
	aiProvider := flag.String("ai", "auto", "AI провайдер: auto, ollama, openrouter")
	useGLM := flag.Bool("glm", false, "Использовать GLM-4.7-Flash вместо Gemma2")

	// Инкрементальная генерация
	batchWeeks := flag.Int("batch", 0, "Генерировать по N недель (4 = месяц). 0 = всё сразу")
	continueGen := flag.String("continue", "", "Продолжить генерацию для клиента")
	listStates := flag.Bool("list-states", false, "Показать сохранённые состояния генерации")

	// Примеры
	example := flag.Bool("example", false, "Показать пример JSON")
	exampleComp := flag.Bool("example-comp", false, "Показать пример соревновательной подготовки")
	calcWeeks := flag.String("calc-weeks", "", "Рассчитать недели до даты (DD.MM.YYYY)")

	flag.Parse()

	// Сохраняем путь для ленивой загрузки
	knowledgePath = *knowledgePathFlag

	// Примеры и справка (без загрузки базы знаний)
	if *example {
		printExample()
		return
	}
	if *exampleComp {
		printCompetitionExample()
		return
	}
	if *calcWeeks != "" {
		calculateWeeksToDate(*calcWeeks)
		return
	}

	// Список сохранённых состояний
	if *listStates {
		listSavedStates()
		return
	}

	// Выбор модели
	modelToUse := *ollamaModel
	if *useGLM && modelToUse == "" {
		modelToUse = ai.ModelGLM4Flash
	}

	// Продолжение генерации
	if *continueGen != "" {
		runContinueMode(*continueGen, *credentials, *folderID, *ollamaURL, modelToUse, *aiProvider)
		return
	}

	// Интерактивный режим
	if *interactive {
		runInteractiveMode(*credentials, *folderID, *ollamaURL, modelToUse, *aiProvider, *batchWeeks)
		return
	}

	// Режим генерации из флагов
	if *generate || *clientName != "" {
		runGenerateMode(*clientName, *goal, *disciplines, *compDate, *weeks, *days, *onePMData, *targetPMData, *credentials, *folderID, *ollamaURL, modelToUse, *aiProvider, *batchWeeks)
		return
	}

	// Режим из JSON файла
	if *jsonFile != "" {
		runJSONMode(*jsonFile, *credentials, *folderID)
		return
	}

	// Справка
	printUsage()
}

// listSavedStates показывает сохранённые состояния генерации
func listSavedStates() {
	states, err := ai.ListSavedStates()
	if err != nil {
		log.Printf("Ошибка: %v", err)
		return
	}

	if len(states) == 0 {
		fmt.Println("Нет сохранённых состояний генерации")
		return
	}

	fmt.Println("📋 Сохранённые состояния генерации:")
	for _, clientName := range states {
		state, err := ai.LoadState(clientName)
		if err != nil || state == nil {
			continue
		}
		fmt.Printf("   • %s: %d/%d недель (%.0f%%) — %s\n",
			state.Request.ClientName,
			state.LastWeekNum, state.TotalWeeks,
			state.GetProgress(),
			state.Status)
	}
	fmt.Println()
	fmt.Println("Для продолжения: ./plancli -continue \"Имя_клиента\"")
}

// runContinueMode продолжает генерацию с сохранённого состояния
func runContinueMode(clientName, credentials, folderID, ollamaURL, ollamaModel, aiProvider string) {
	state, err := ai.LoadState(clientName)
	if err != nil {
		log.Fatalf("Ошибка загрузки состояния: %v", err)
	}
	if state == nil {
		log.Fatalf("Состояние для клиента '%s' не найдено. Используйте -list-states", clientName)
	}

	if state.IsComplete() {
		fmt.Printf("✅ Генерация для %s уже завершена (%d недель)\n", clientName, state.TotalWeeks)
		fmt.Print("Создать Google Sheets? [Y/n]: ")
		reader := bufio.NewReader(os.Stdin)
		confirm, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(confirm)) != "n" {
			// Создаём AI клиент для сборки плана
			cfg := ai.ProviderConfig{
				Provider:    ai.Provider(aiProvider),
				OllamaURL:   ollamaURL,
				OllamaModel: ollamaModel,
			}
			aiClient, _ := ai.NewAIClient(cfg)
			generator := ai.NewProgramGeneratorV3(aiClient)
			plan := generator.BuildPlanFromState(state)
			createGoogleSheet(plan, credentials, folderID)
		}
		return
	}

	fmt.Printf("📋 Продолжаю генерацию для %s\n", state.Request.ClientName)
	fmt.Printf("   Прогресс: %d/%d недель (%.0f%%)\n", state.LastWeekNum, state.TotalWeeks, state.GetProgress())

	runIncrementalGeneration(state, credentials, folderID, ollamaURL, ollamaModel, aiProvider)
}

// runIncrementalGeneration запускает инкрементальную генерацию
func runIncrementalGeneration(state *ai.PlanGenerationState, credentials, folderID, ollamaURL, ollamaModel, aiProvider string) {
	// Ленивая загрузка базы знаний
	loadKnowledgeIfNeeded()

	// Создаём AI клиент
	cfg := ai.ProviderConfig{
		Provider:    ai.Provider(aiProvider),
		OllamaURL:   ollamaURL,
		OllamaModel: ollamaModel,
	}
	aiClient, err := ai.NewAIClient(cfg)
	if err != nil {
		log.Printf("⚠️ %v, использую Ollama", err)
		aiClient = ai.NewClientWithURL(ollamaURL, ollamaModel)
	}

	generator := ai.NewProgramGeneratorV3(aiClient)
	reader := bufio.NewReader(os.Stdin)

	for !state.IsComplete() {
		startWeek, endWeek, _ := state.GetNextBatchRange()
		fmt.Printf("\n🚀 Генерирую недели %d-%d...\n", startWeek, endWeek)

		_, err := generator.GenerateBatch(state)
		if err != nil {
			log.Printf("❌ Ошибка: %v", err)
			// Сохраняем состояние даже при ошибке
			state.SaveState()
			fmt.Println("💾 Состояние сохранено. Можно продолжить позже.")
			return
		}

		// Сохраняем состояние после каждого батча
		if err := state.SaveState(); err != nil {
			log.Printf("⚠️ Ошибка сохранения состояния: %v", err)
		}

		fmt.Printf("✅ Сгенерировано %d/%d недель (%.0f%%)\n",
			state.LastWeekNum, state.TotalWeeks, state.GetProgress())

		if !state.IsComplete() {
			fmt.Print("\n⏸️  Продолжить генерацию? [Y/n]: ")
			confirm, _ := reader.ReadString('\n')
			if strings.ToLower(strings.TrimSpace(confirm)) == "n" {
				fmt.Println("💾 Состояние сохранено. Для продолжения:")
				fmt.Printf("   ./plancli -continue \"%s\"\n", state.Request.ClientName)
				return
			}
		}
	}

	// Генерация завершена
	fmt.Println("\n✅ Генерация завершена!")
	plan := generator.BuildPlanFromState(state)
	printPlanSummary(plan)

	// Удаляем файл состояния
	ai.DeleteState(state.Request.ClientName)

	fmt.Print("\n📤 Создать таблицу в Google Sheets? [Y/n]: ")
	confirm, _ := reader.ReadString('\n')
	if strings.ToLower(strings.TrimSpace(confirm)) != "n" {
		createGoogleSheet(plan, credentials, folderID)
	}
}

func runInteractiveMode(credentials, folderID, ollamaURL, ollamaModel, aiProvider string, batchWeeks int) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║   ГЕНЕРАТОР ТРЕНИРОВОЧНЫХ ПЛАНОВ       ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Println()

	// Имя клиента
	fmt.Print("👤 Имя клиента: ")
	clientName, _ := reader.ReadString('\n')
	clientName = strings.TrimSpace(clientName)
	if clientName == "" {
		clientName = "Клиент"
	}

	// Пол
	fmt.Println("\n⚧ Пол:")
	fmt.Println("  1. Мужской")
	fmt.Println("  2. Женский")
	fmt.Print("Выбор [1-2]: ")
	genderChoice, _ := reader.ReadString('\n')
	genderChoice = strings.TrimSpace(genderChoice)
	var gender string
	switch genderChoice {
	case "2", "ж", "f", "female":
		gender = "female"
	default:
		gender = "male"
	}

	// Возраст
	fmt.Print("\n🎂 Возраст [25]: ")
	ageInput, _ := reader.ReadString('\n')
	ageInput = strings.TrimSpace(ageInput)
	age := 25
	if ageInput != "" {
		age, _ = strconv.Atoi(ageInput)
	}

	// Вес
	fmt.Print("\n⚖️ Вес (кг) [75]: ")
	weightInput, _ := reader.ReadString('\n')
	weightInput = strings.TrimSpace(weightInput)
	weight := 75.0
	if weightInput != "" {
		weight, _ = strconv.ParseFloat(weightInput, 64)
	}

	// Рост
	fmt.Print("\n📏 Рост (см) [175]: ")
	heightInput, _ := reader.ReadString('\n')
	heightInput = strings.TrimSpace(heightInput)
	height := 175.0
	if heightInput != "" {
		height, _ = strconv.ParseFloat(heightInput, 64)
	}

	// Уровень подготовки
	fmt.Println("\n💪 Уровень подготовки:")
	fmt.Println("  1. Новичок (< 1 года)")
	fmt.Println("  2. Средний (1-3 года)")
	fmt.Println("  3. Продвинутый (> 3 лет)")
	fmt.Print("Выбор [1-3]: ")
	expChoice, _ := reader.ReadString('\n')
	expChoice = strings.TrimSpace(expChoice)
	var experience string
	switch expChoice {
	case "1":
		experience = "beginner"
	case "3":
		experience = "advanced"
	default:
		experience = "intermediate"
	}

	// Цель
	fmt.Println("\n🎯 Выберите цель:")
	fmt.Println("  1. Сила (strength)")
	fmt.Println("  2. Масса (hypertrophy)")
	fmt.Println("  3. Похудение (weight_loss)")
	fmt.Println("  4. Соревнования (competition)")
	fmt.Print("Выбор [1-4]: ")
	goalChoice, _ := reader.ReadString('\n')
	goalChoice = strings.TrimSpace(goalChoice)

	var goal string
	switch goalChoice {
	case "1", "strength":
		goal = GoalStrength
	case "2", "hypertrophy":
		goal = GoalHypertrophy
	case "3", "weight_loss":
		goal = GoalWeightLoss
	case "4", "competition":
		goal = GoalCompetition
	default:
		goal = GoalStrength
	}

	var disciplines []string
	var compDate string
	var totalWeeks int

	// Для соревнований — дополнительные вопросы
	if goal == GoalCompetition {
		fmt.Println("\n🏆 СОРЕВНОВАТЕЛЬНАЯ ПОДГОТОВКА")

		fmt.Println("\n📋 Выберите дисциплины (через запятую):")
		fmt.Println("  1. Жим лёжа (bench_press)")
		fmt.Println("  2. Становая тяга (deadlift)")
		fmt.Println("  3. Ягодичный мост (glute_bridge)")
		fmt.Println("  4. Строгий подъём на бицепс (strict_curl)")
		fmt.Print("Выбор [например: 1,3,4]: ")
		discChoice, _ := reader.ReadString('\n')
		discChoice = strings.TrimSpace(discChoice)

		for _, d := range strings.Split(discChoice, ",") {
			d = strings.TrimSpace(d)
			switch d {
			case "1", "bench_press":
				disciplines = append(disciplines, DisciplineBenchPress)
			case "2", "deadlift":
				disciplines = append(disciplines, DisciplineDeadlift)
			case "3", "glute_bridge":
				disciplines = append(disciplines, DisciplineGluteBridge)
			case "4", "strict_curl":
				disciplines = append(disciplines, DisciplineStrictCurl)
			}
		}

		for {
			fmt.Print("\n📅 Дата соревнований (DD.MM.YYYY): ")
			compDate, _ = reader.ReadString('\n')
			compDate = strings.TrimSpace(compDate)

			if compDate == "" {
				break // Пропустить ввод даты
			}

			parsed, err := time.Parse("02.01.2006", compDate)
			if err != nil {
				fmt.Println("   ❌ Неверный формат даты. Используйте DD.MM.YYYY")
				continue
			}

			daysUntil := int(parsed.Sub(time.Now()).Hours() / 24)

			// Проверка на прошедшую дату
			if daysUntil < 0 {
				fmt.Printf("   ❌ Дата %s уже прошла! Введите будущую дату.\n", compDate)
				continue
			}

			// Проверка на слишком близкую дату
			if daysUntil < 14 {
				fmt.Printf("   ⚠️ Всего %d дней — слишком мало для подготовки!\n", daysUntil)
				fmt.Print("   Продолжить? [y/N]: ")
				confirm, _ := reader.ReadString('\n')
				if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
					continue
				}
			}

			// Проверка на слишком далёкую дату
			totalWeeks = int(math.Ceil(float64(daysUntil) / 7))
			if totalWeeks > 52 {
				fmt.Printf("   ⚠️ %d недель — очень долгая подготовка!\n", totalWeeks)
				fmt.Print("   Продолжить? [y/N]: ")
				confirm, _ := reader.ReadString('\n')
				if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
					continue
				}
			}

			fmt.Printf("   ✅ %d дней (~%d недель) до соревнований\n", daysUntil, totalWeeks)
			break
		}
	}

	// Количество недель (если не соревнования или дата не указана)
	if totalWeeks == 0 {
		fmt.Print("\n📅 Количество недель [8]: ")
		weeksInput, _ := reader.ReadString('\n')
		weeksInput = strings.TrimSpace(weeksInput)
		if weeksInput == "" {
			totalWeeks = 8
		} else {
			totalWeeks, _ = strconv.Atoi(weeksInput)
		}
	}

	// Дни в неделю
	fmt.Print("\n🏋️ Тренировок в неделю [3]: ")
	daysInput, _ := reader.ReadString('\n')
	daysInput = strings.TrimSpace(daysInput)
	daysPerWeek := 3
	if daysInput != "" {
		daysPerWeek, _ = strconv.Atoi(daysInput)
	}

	// 1ПМ данные
	onePM := make(map[string]float64)
	targetPM := make(map[string]float64)
	fmt.Println("\n💪 Введите текущий 1ПМ:")

	exerciseNames := getExercisesForDisciplines(disciplines, goal)
	for _, exName := range exerciseNames {
		fmt.Printf("   %s (кг): ", exName)
		weightInput, _ := reader.ReadString('\n')
		weightInput = strings.TrimSpace(weightInput)
		if weightInput != "" {
			if w, err := strconv.ParseFloat(weightInput, 64); err == nil && w > 0 {
				onePM[exName] = w
			}
		}
	}

	// Целевой 1ПМ для соревнований
	if goal == GoalCompetition && len(onePM) > 0 {
		fmt.Println("\n🎯 Введите ЦЕЛЕВОЙ 1ПМ на соревнования (какой хотите пожать):")
		for exName, currentPM := range onePM {
			fmt.Printf("   %s (текущий: %.0f кг, цель): ", exName, currentPM)
			targetInput, _ := reader.ReadString('\n')
			targetInput = strings.TrimSpace(targetInput)
			if targetInput != "" {
				if w, err := strconv.ParseFloat(targetInput, 64); err == nil && w > 0 {
					targetPM[exName] = w
				}
			}
		}
	}

	// Дополнительные упражнения
	fmt.Println("\n   Дополнительные упражнения (формат: Название:вес, пустая строка для завершения):")
	for {
		fmt.Print("   > ")
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		parts := strings.Split(line, ":")
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			if w, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil && w > 0 {
				onePM[name] = w
			}
		}
	}

	// Спрашиваем про режим генерации если много недель
	useBatchMode := batchWeeks > 0
	actualBatchSize := batchWeeks
	if totalWeeks > 8 && !useBatchMode {
		fmt.Printf("\n📊 Программа на %d недель. Как генерировать?\n", totalWeeks)
		fmt.Println("  1. Всё сразу (может занять 15-30 минут)")
		fmt.Println("  2. По месяцам (4 недели за раз, можно прерывать)")
		fmt.Print("Выбор [1-2, по умолчанию 2]: ")
		modeInput, _ := reader.ReadString('\n')
		modeInput = strings.TrimSpace(modeInput)
		if modeInput != "1" {
			useBatchMode = true
			actualBatchSize = 4
		}
	}

	// Инкрементальная генерация
	if useBatchMode && actualBatchSize > 0 {
		fmt.Printf("\n⏳ Запускаю инкрементальную генерацию (по %d недель)...\n", actualBatchSize)
		fmt.Printf("   Клиент: %s, %s, %d лет, %.0f кг, %.0f см, %s\n", clientName, gender, age, weight, height, experience)

		// Формируем запрос
		var compDateParsed *time.Time
		if compDate != "" {
			if parsed, err := time.Parse("02.01.2006", compDate); err == nil {
				compDateParsed = &parsed
			}
		}

		req := ai.ProgramRequestV3{
			ClientName:      clientName,
			Gender:          gender,
			Age:             age,
			Weight:          weight,
			Height:          height,
			Experience:      experience,
			Goal:            goal,
			Disciplines:     disciplines,
			DaysPerWeek:     daysPerWeek,
			TotalWeeks:      totalWeeks,
			OnePMData:       onePM,
			TargetOnePM:     targetPM,
			Equipment:       "full_gym",
			CompetitionDate: compDateParsed,
		}

		state := ai.NewPlanGenerationState(req, actualBatchSize)
		runIncrementalGeneration(state, credentials, folderID, ollamaURL, ollamaModel, aiProvider)
		return
	}

	// Обычная генерация (всё сразу)
	fmt.Println("\n⏳ Генерирую персональный план через AI...")
	fmt.Printf("   Клиент: %s, %s, %d лет, %.0f кг, %.0f см, %s\n", clientName, gender, age, weight, height, experience)
	fmt.Println("   (это может занять 1-3 минуты)")

	plan, err := generatePlanWithAI(clientName, gender, age, weight, height, experience, goal, disciplines, compDate, totalWeeks, daysPerWeek, onePM, targetPM, ollamaURL, ollamaModel, aiProvider)
	if err != nil {
		fmt.Printf("\n❌ Ошибка генерации AI: %v\n", err)
		fmt.Println("   Использую резервную генерацию...")
		plan = generatePlan(clientName, goal, disciplines, compDate, totalWeeks, daysPerWeek, onePM)
	}

	// Показать план
	printPlanSummary(plan)

	// Спросить подтверждение
	fmt.Print("\n📤 Создать таблицу в Google Sheets? [Y/n]: ")
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))

	if confirm == "" || confirm == "y" || confirm == "yes" || confirm == "д" || confirm == "да" {
		createGoogleSheet(plan, credentials, folderID)
	} else {
		// Сохранить в JSON
		fmt.Print("💾 Сохранить в JSON? [Y/n]: ")
		saveConfirm, _ := reader.ReadString('\n')
		saveConfirm = strings.TrimSpace(strings.ToLower(saveConfirm))

		if saveConfirm == "" || saveConfirm == "y" || saveConfirm == "yes" {
			filename := fmt.Sprintf("plan_%s_%s.json", strings.ReplaceAll(clientName, " ", "_"), time.Now().Format("20060102"))
			savePlanToJSON(plan, filename)
		}
	}
}

func runGenerateMode(clientName, goal, disciplinesStr, compDate string, weeks, days int, onePMStr, targetPMStr, credentials, folderID, ollamaURL, ollamaModel, aiProvider string, batchWeeks int) {
	if clientName == "" {
		log.Fatal("Укажите имя клиента: -client \"Имя\"")
	}
	if goal == "" {
		goal = GoalStrength
	}

	// Парсинг дисциплин
	var disciplines []string
	if disciplinesStr != "" {
		for _, d := range strings.Split(disciplinesStr, ",") {
			disciplines = append(disciplines, strings.TrimSpace(d))
		}
	}

	// Рассчёт недель из даты
	totalWeeks := weeks
	if compDate != "" && totalWeeks == 0 {
		parsed, err := time.Parse("02.01.2006", compDate)
		if err != nil {
			log.Fatalf("❌ Неверный формат даты: %s (используйте DD.MM.YYYY)", compDate)
		}
		daysUntil := int(parsed.Sub(time.Now()).Hours() / 24)
		if daysUntil < 0 {
			log.Fatalf("❌ Дата %s уже прошла! Укажите будущую дату.", compDate)
		}
		if daysUntil < 14 {
			fmt.Printf("⚠️ Внимание: всего %d дней до соревнований — очень мало!\n", daysUntil)
		}
		totalWeeks = int(math.Ceil(float64(daysUntil) / 7))
		if totalWeeks > 52 {
			fmt.Printf("⚠️ Внимание: %d недель — очень длинная подготовка!\n", totalWeeks)
		}
	}
	if totalWeeks == 0 {
		totalWeeks = 8
	}

	// Парсинг 1ПМ
	onePM := make(map[string]float64)
	if onePMStr != "" {
		for _, item := range strings.Split(onePMStr, ",") {
			parts := strings.Split(item, ":")
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				if w, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
					onePM[name] = w
				}
			}
		}
	}

	// Парсинг целевого 1ПМ
	targetPM := make(map[string]float64)
	if targetPMStr != "" {
		for _, item := range strings.Split(targetPMStr, ",") {
			parts := strings.Split(item, ":")
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				if w, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
					targetPM[name] = w
				}
			}
		}
	}

	// Инкрементальная генерация если указан batch
	if batchWeeks > 0 {
		fmt.Printf("📋 Запускаю инкрементальную генерацию для %s (по %d недель)...\n", clientName, batchWeeks)

		var compDateParsed *time.Time
		if compDate != "" {
			if parsed, err := time.Parse("02.01.2006", compDate); err == nil {
				compDateParsed = &parsed
			}
		}

		req := ai.ProgramRequestV3{
			ClientName:      clientName,
			Gender:          "male",
			Age:             25,
			Weight:          75,
			Height:          175,
			Experience:      "intermediate",
			Goal:            goal,
			Disciplines:     disciplines,
			DaysPerWeek:     days,
			TotalWeeks:      totalWeeks,
			OnePMData:       onePM,
			TargetOnePM:     targetPM,
			Equipment:       "full_gym",
			CompetitionDate: compDateParsed,
		}

		state := ai.NewPlanGenerationState(req, batchWeeks)
		runIncrementalGeneration(state, credentials, folderID, ollamaURL, ollamaModel, aiProvider)
		return
	}

	fmt.Printf("📋 Генерирую план для %s через AI...\n", clientName)
	// Для режима из флагов — дефолтные значения антропометрии (используйте -i для персональных)
	plan, err := generatePlanWithAI(clientName, "male", 25, 75, 175, "intermediate", goal, disciplines, compDate, totalWeeks, days, onePM, targetPM, ollamaURL, ollamaModel, aiProvider)
	if err != nil {
		fmt.Printf("❌ Ошибка AI: %v\n", err)
		fmt.Println("   Использую резервную генерацию...")
		plan = generatePlan(clientName, goal, disciplines, compDate, totalWeeks, days, onePM)
	}

	printPlanSummary(plan)
	createGoogleSheet(plan, credentials, folderID)
}

// generatePlanWithAI генерирует план через AI (Ollama или OpenRouter)
func generatePlanWithAI(clientName, gender string, age int, weight, height float64, experience, goal string, disciplines []string, compDate string, totalWeeks, daysPerWeek int, onePM, targetPM map[string]float64, ollamaURL, ollamaModel, providerStr string) (*models.TrainingPlan, error) {
	// Ленивая загрузка базы знаний
	loadKnowledgeIfNeeded()

	// Определяем провайдер
	provider := ai.Provider(providerStr)
	cfg := ai.ProviderConfig{
		Provider:    provider,
		OllamaURL:   ollamaURL,
		OllamaModel: ollamaModel,
	}

	// Создаём AI клиент
	aiClient, err := ai.NewAIClient(cfg)
	if err != nil {
		// Fallback на Ollama если Zhipu недоступен
		log.Printf("⚠️ %v, использую Ollama", err)
		aiClient = ai.NewClientWithURL(ollamaURL, ollamaModel)
	}

	// Показываем какой провайдер используется
	if provider == ai.ProviderAuto || provider == "" {
		provider = ai.GetDefaultProvider()
	}
	fmt.Printf("   🤖 AI: %s\n", ai.GetProviderName(provider))

	// Создаём генератор
	generator := ai.NewProgramGeneratorV3(aiClient)

	// Ищем релевантный контекст из базы знаний
	var knowledgeContext string
	if knowledgeStore != nil && knowledgeStore.IsLoaded() {
		// Формируем поисковый запрос на основе цели и параметров
		searchQuery := buildKnowledgeQuery(goal, disciplines, experience, totalWeeks)
		knowledgeContext = knowledgeStore.GetContext(searchQuery, 5) // top 5 релевантных чанков
		if knowledgeContext != "" {
			log.Printf("📚 Найден контекст из базы знаний (%d символов)", len(knowledgeContext))
		}
	}

	// Формируем запрос с полной антропометрией
	req := ai.ProgramRequestV3{
		ClientName:       clientName,
		Gender:           gender,
		Age:              age,
		Weight:           weight,
		Height:           height,
		Experience:       experience,
		Goal:             goal,
		Disciplines:      disciplines, // Соревновательные дисциплины!
		DaysPerWeek:      daysPerWeek,
		TotalWeeks:       totalWeeks,
		OnePMData:        onePM,
		TargetOnePM:      targetPM, // Целевой 1ПМ на соревнования!
		Equipment:        "full_gym",
		KnowledgeContext: knowledgeContext, // Контекст из книг!
	}

	// Парсим дату соревнований
	if compDate != "" {
		if parsed, err := time.Parse("02.01.2006", compDate); err == nil {
			req.CompetitionDate = &parsed
		}
	}

	// Генерируем программу
	result, err := generator.GenerateProgram(req)
	if err != nil {
		return nil, fmt.Errorf("генерация: %w", err)
	}

	// Проверяем валидацию
	if result.Validation != nil && !result.Validation.IsValid {
		fmt.Println("\n⚠️ Валидация:")
		for _, e := range result.Validation.Errors {
			fmt.Printf("   ❌ %s\n", e)
		}
		for _, w := range result.Validation.Warnings {
			fmt.Printf("   ⚠️ %s\n", w)
		}
	}

	// Добавляем метаданные
	plan := result.Plan
	plan.ClientName = clientName
	endDate := time.Now().AddDate(0, 0, totalWeeks*7)
	plan.StartDate = time.Now()
	plan.EndDate = &endDate

	return plan, nil
}

// buildKnowledgeQuery формирует поисковый запрос для базы знаний
func buildKnowledgeQuery(goal string, disciplines []string, experience string, totalWeeks int) string {
	var parts []string

	// Основа — цель тренировок
	switch goal {
	case GoalStrength:
		parts = append(parts, "сила силовая тренировка программа периодизация")
	case GoalHypertrophy:
		parts = append(parts, "гипертрофия масса объём тренировка мышцы")
	case GoalWeightLoss:
		parts = append(parts, "похудение жиросжигание кардио метаболизм")
	case GoalCompetition:
		parts = append(parts, "соревнования пик форма подводка периодизация сила")
	}

	// Добавляем дисциплины
	for _, d := range disciplines {
		switch d {
		case DisciplineBenchPress:
			parts = append(parts, "жим лёжа грудные техника")
		case DisciplineDeadlift:
			parts = append(parts, "становая тяга спина техника")
		case DisciplineGluteBridge:
			parts = append(parts, "ягодичный мост ягодицы")
		case DisciplineStrictCurl:
			parts = append(parts, "бицепс подъём штанги")
		}
	}

	// Добавляем опыт
	switch experience {
	case "beginner":
		parts = append(parts, "новичок начинающий линейная прогрессия")
	case "intermediate":
		parts = append(parts, "средний уровень блочная периодизация")
	case "advanced":
		parts = append(parts, "продвинутый волновая периодизация интенсификация")
	}

	// Длительность программы
	if totalWeeks >= 12 {
		parts = append(parts, "макроцикл мезоцикл долгосрочная")
	} else if totalWeeks >= 8 {
		parts = append(parts, "мезоцикл среднесрочная")
	} else {
		parts = append(parts, "микроцикл краткосрочная пик")
	}

	return strings.Join(parts, " ")
}

func generatePlan(clientName, goal string, disciplines []string, compDate string, totalWeeks, daysPerWeek int, onePM map[string]float64) *models.TrainingPlan {
	// Название программы
	programName := getProgramName(goal, disciplines)

	// Генерация периодизации
	periodization := training.GenerateFullPeriodization(
		0, // clientID (не используется для CLI)
		programName,
		time.Now(),
		totalWeeks,
		daysPerWeek,
		goal,
		4, // deload каждые 4 недели
	)

	// Создаём план
	endDate := time.Now().AddDate(0, 0, totalWeeks*7)
	plan := &models.TrainingPlan{
		ClientName:  clientName,
		Name:        programName,
		Goal:        goal,
		TotalWeeks:  totalWeeks,
		DaysPerWeek: daysPerWeek,
		StartDate:   time.Now(),
		EndDate:     &endDate,
		OnePMData:   onePM,
		Mesocycles:  periodization.Mesocycles,
	}

	// Генерируем недели с упражнениями
	exercises := getExercisesForPlan(disciplines, goal, onePM)
	plan.Weeks = generateWeeksWithExercises(periodization.Mesocycles, exercises, daysPerWeek, onePM)

	return plan
}

func generateWeeksWithExercises(mesocycles []models.Mesocycle, exercises []models.Exercise, daysPerWeek int, onePM map[string]float64) []models.TrainingWeek {
	var weeks []models.TrainingWeek

	for _, meso := range mesocycles {
		for _, micro := range meso.Microcycles {
			week := models.TrainingWeek{
				WeekNum:          micro.WeekNumber,
				Phase:            meso.Phase,
				IntensityPercent: float64(meso.IntensityPercent) * micro.IntensityModifier,
				VolumePercent:    float64(meso.VolumePercent) * micro.VolumeModifier,
				RPETarget:        meso.RPETarget,
				IsDeload:         micro.IsDeload,
			}

			// Распределяем упражнения по дням
			week.Workouts = distributeExercisesToDays(exercises, daysPerWeek, week.IntensityPercent, week.IsDeload, onePM)

			weeks = append(weeks, week)
		}
	}

	return weeks
}

func distributeExercisesToDays(exercises []models.Exercise, daysPerWeek int, intensity float64, isDeload bool, onePM map[string]float64) []models.DayWorkout {
	var workouts []models.DayWorkout

	// Группируем упражнения по мышечным группам
	groups := make(map[string][]models.Exercise)
	for _, ex := range exercises {
		groups[ex.MuscleGroup] = append(groups[ex.MuscleGroup], ex)
	}

	// Распределяем по сплиту
	splits := getSplitForDays(daysPerWeek)

	for dayNum, split := range splits {
		workout := models.DayWorkout{
			DayNum:       dayNum + 1,
			Name:         split.Name,
			Type:         split.Type,
			MuscleGroups: split.MuscleGroups,
		}

		// Добавляем упражнения для этого дня
		for _, muscleGroup := range split.MuscleGroups {
			for _, ex := range groups[muscleGroup] {
				sets, reps := getSetsRepsForPhase(intensity, isDeload)
				weightKg := 0.0
				weightPercent := intensity

				if pm, ok := onePM[ex.Name]; ok {
					weightKg = training.CalculateWorkingWeightRound(pm, intensity, 2.5)
				}

				workoutEx := models.WorkoutExerciseV2{
					ExerciseName:  ex.Name,
					MuscleGroup:   ex.MuscleGroup,
					Sets:          sets,
					Reps:          reps,
					WeightPercent: weightPercent,
					WeightKg:      weightKg,
					RestSeconds:   getRestForIntensity(intensity),
					RPE:           getRPEForIntensity(intensity),
				}
				workout.Exercises = append(workout.Exercises, workoutEx)
			}
		}

		workouts = append(workouts, workout)
	}

	return workouts
}

type DaySplit struct {
	Name         string
	Type         string
	MuscleGroups []string
}

func getSplitForDays(days int) []DaySplit {
	switch days {
	case 2:
		return []DaySplit{
			{Name: "Верх", Type: "upper", MuscleGroups: []string{"Грудь", "Спина", "Плечи", "Бицепс", "Трицепс"}},
			{Name: "Низ", Type: "lower", MuscleGroups: []string{"Квадрицепс", "Ягодицы", "Бицепс бедра", "Икры"}},
		}
	case 3:
		return []DaySplit{
			{Name: "Толкающие", Type: "push", MuscleGroups: []string{"Грудь", "Плечи", "Трицепс"}},
			{Name: "Тянущие", Type: "pull", MuscleGroups: []string{"Спина", "Бицепс", "Бицепс бедра"}},
			{Name: "Ноги", Type: "legs", MuscleGroups: []string{"Квадрицепс", "Ягодицы", "Икры"}},
		}
	case 4:
		return []DaySplit{
			{Name: "Верх А", Type: "upper", MuscleGroups: []string{"Грудь", "Спина", "Плечи"}},
			{Name: "Низ А", Type: "lower", MuscleGroups: []string{"Квадрицепс", "Ягодицы"}},
			{Name: "Верх Б", Type: "upper", MuscleGroups: []string{"Спина", "Бицепс", "Трицепс"}},
			{Name: "Низ Б", Type: "lower", MuscleGroups: []string{"Бицепс бедра", "Ягодицы", "Икры"}},
		}
	case 5:
		return []DaySplit{
			{Name: "Грудь", Type: "push", MuscleGroups: []string{"Грудь", "Трицепс"}},
			{Name: "Спина", Type: "pull", MuscleGroups: []string{"Спина", "Бицепс"}},
			{Name: "Ноги", Type: "legs", MuscleGroups: []string{"Квадрицепс", "Бицепс бедра", "Икры"}},
			{Name: "Плечи", Type: "push", MuscleGroups: []string{"Плечи", "Трицепс"}},
			{Name: "Ягодицы", Type: "legs", MuscleGroups: []string{"Ягодицы", "Бицепс бедра"}},
		}
	default: // 6 дней
		return []DaySplit{
			{Name: "Толкающие А", Type: "push", MuscleGroups: []string{"Грудь", "Плечи", "Трицепс"}},
			{Name: "Тянущие А", Type: "pull", MuscleGroups: []string{"Спина", "Бицепс"}},
			{Name: "Ноги А", Type: "legs", MuscleGroups: []string{"Квадрицепс", "Ягодицы"}},
			{Name: "Толкающие Б", Type: "push", MuscleGroups: []string{"Грудь", "Плечи", "Трицепс"}},
			{Name: "Тянущие Б", Type: "pull", MuscleGroups: []string{"Спина", "Бицепс бедра"}},
			{Name: "Ноги Б", Type: "legs", MuscleGroups: []string{"Ягодицы", "Икры"}},
		}
	}
}

func getSetsRepsForPhase(intensity float64, isDeload bool) (int, string) {
	if isDeload {
		return 2, "8-10"
	}

	switch {
	case intensity >= 90:
		return 5, "1-3"
	case intensity >= 85:
		return 4, "3-5"
	case intensity >= 80:
		return 4, "4-6"
	case intensity >= 75:
		return 4, "5-8"
	case intensity >= 70:
		return 4, "6-8"
	default:
		return 3, "8-12"
	}
}

func getRestForIntensity(intensity float64) int {
	switch {
	case intensity >= 90:
		return 300
	case intensity >= 85:
		return 240
	case intensity >= 80:
		return 180
	case intensity >= 75:
		return 150
	default:
		return 120
	}
}

func getRPEForIntensity(intensity float64) float64 {
	switch {
	case intensity >= 95:
		return 10
	case intensity >= 90:
		return 9.5
	case intensity >= 85:
		return 9
	case intensity >= 80:
		return 8.5
	case intensity >= 75:
		return 8
	case intensity >= 70:
		return 7.5
	default:
		return 7
	}
}

func getExercisesForPlan(disciplines []string, goal string, onePM map[string]float64) []models.Exercise {
	// ВСЕГДА берём полный набор упражнений
	exercises := getDefaultExercises(goal)

	// Добавляем соревновательные упражнения если их нет
	for _, d := range disciplines {
		var name, muscleGroup string
		switch d {
		case DisciplineBenchPress:
			name, muscleGroup = "Жим лёжа", "Грудь"
		case DisciplineDeadlift:
			name, muscleGroup = "Становая тяга", "Спина"
		case DisciplineGluteBridge:
			name, muscleGroup = "Ягодичный мост со штангой", "Ягодицы"
		case DisciplineStrictCurl:
			name, muscleGroup = "Строгий подъём на бицепс", "Бицепс"
		}
		if name != "" {
			found := false
			for _, ex := range exercises {
				if ex.Name == name {
					found = true
					break
				}
			}
			if !found {
				exercises = append(exercises, models.Exercise{Name: name, MuscleGroup: muscleGroup, MovementType: "compound"})
			}
		}
	}

	// Добавляем упражнения из 1ПМ данных если их нет
	for name := range onePM {
		found := false
		for _, ex := range exercises {
			if ex.Name == name {
				found = true
				break
			}
		}
		if !found {
			exercises = append(exercises, models.Exercise{
				Name:        name,
				MuscleGroup: guessMuscleGroup(name),
			})
		}
	}

	return exercises
}

func guessMuscleGroup(name string) string {
	nameLower := strings.ToLower(name)
	switch {
	case strings.Contains(nameLower, "жим лёжа") || strings.Contains(nameLower, "жим лежа"):
		return "Грудь"
	case strings.Contains(nameLower, "становая"):
		return "Спина"
	case strings.Contains(nameLower, "присед") || strings.Contains(nameLower, "приседания"):
		return "Квадрицепс"
	case strings.Contains(nameLower, "ягодичный") || strings.Contains(nameLower, "мост"):
		return "Ягодицы"
	case strings.Contains(nameLower, "бицепс") || strings.Contains(nameLower, "подъём на бицепс"):
		return "Бицепс"
	case strings.Contains(nameLower, "трицепс") || strings.Contains(nameLower, "французский"):
		return "Трицепс"
	case strings.Contains(nameLower, "тяга") && strings.Contains(nameLower, "наклон"):
		return "Спина"
	case strings.Contains(nameLower, "жим") && strings.Contains(nameLower, "плеч"):
		return "Плечи"
	default:
		return "Другое"
	}
}

func getDefaultExercises(goal string) []models.Exercise {
	// Полноценные списки упражнений для каждой мышечной группы (6-8 упражнений на тренировку)
	exercises := []models.Exercise{
		// ГРУДЬ (Push)
		{Name: "Жим лёжа", MuscleGroup: "Грудь", MovementType: "compound"},
		{Name: "Жим гантелей на наклонной", MuscleGroup: "Грудь", MovementType: "compound"},
		{Name: "Разводка гантелей лёжа", MuscleGroup: "Грудь", MovementType: "isolation"},
		{Name: "Отжимания на брусьях", MuscleGroup: "Грудь", MovementType: "compound"},

		// ПЛЕЧИ (Push)
		{Name: "Жим стоя", MuscleGroup: "Плечи", MovementType: "compound"},
		{Name: "Жим гантелей сидя", MuscleGroup: "Плечи", MovementType: "compound"},
		{Name: "Махи гантелей в стороны", MuscleGroup: "Плечи", MovementType: "isolation"},
		{Name: "Махи гантелей перед собой", MuscleGroup: "Плечи", MovementType: "isolation"},

		// ТРИЦЕПС (Push)
		{Name: "Французский жим лёжа", MuscleGroup: "Трицепс", MovementType: "isolation"},
		{Name: "Разгибания на трицепс на блоке", MuscleGroup: "Трицепс", MovementType: "isolation"},
		{Name: "Жим узким хватом", MuscleGroup: "Трицепс", MovementType: "compound"},

		// СПИНА (Pull)
		{Name: "Становая тяга", MuscleGroup: "Спина", MovementType: "compound"},
		{Name: "Тяга штанги в наклоне", MuscleGroup: "Спина", MovementType: "compound"},
		{Name: "Подтягивания", MuscleGroup: "Спина", MovementType: "compound"},
		{Name: "Тяга верхнего блока", MuscleGroup: "Спина", MovementType: "compound"},
		{Name: "Тяга гантели одной рукой", MuscleGroup: "Спина", MovementType: "compound"},
		{Name: "Тяга нижнего блока", MuscleGroup: "Спина", MovementType: "compound"},

		// БИЦЕПС (Pull)
		{Name: "Подъём штанги на бицепс", MuscleGroup: "Бицепс", MovementType: "isolation"},
		{Name: "Молотки с гантелями", MuscleGroup: "Бицепс", MovementType: "isolation"},
		{Name: "Сгибания на скамье Скотта", MuscleGroup: "Бицепс", MovementType: "isolation"},

		// КВАДРИЦЕПС (Legs)
		{Name: "Приседания со штангой", MuscleGroup: "Квадрицепс", MovementType: "compound"},
		{Name: "Жим ногами", MuscleGroup: "Квадрицепс", MovementType: "compound"},
		{Name: "Выпады с гантелями", MuscleGroup: "Квадрицепс", MovementType: "compound"},
		{Name: "Разгибания ног в тренажёре", MuscleGroup: "Квадрицепс", MovementType: "isolation"},
		{Name: "Гакк-приседания", MuscleGroup: "Квадрицепс", MovementType: "compound"},

		// БИЦЕПС БЕДРА (Pull/Legs)
		{Name: "Румынская тяга", MuscleGroup: "Бицепс бедра", MovementType: "compound"},
		{Name: "Сгибания ног лёжа", MuscleGroup: "Бицепс бедра", MovementType: "isolation"},
		{Name: "Сгибания ног сидя", MuscleGroup: "Бицепс бедра", MovementType: "isolation"},

		// ЯГОДИЦЫ (Legs)
		{Name: "Ягодичный мост со штангой", MuscleGroup: "Ягодицы", MovementType: "compound"},
		{Name: "Болгарские выпады", MuscleGroup: "Ягодицы", MovementType: "compound"},
		{Name: "Отведение ноги в кроссовере", MuscleGroup: "Ягодицы", MovementType: "isolation"},

		// ИКРЫ (Legs)
		{Name: "Подъём на носки стоя", MuscleGroup: "Икры", MovementType: "isolation"},
		{Name: "Подъём на носки сидя", MuscleGroup: "Икры", MovementType: "isolation"},
	}

	// Для силы — убираем часть изоляции, добавляем базу
	if goal == GoalStrength || goal == GoalCompetition {
		// Добавляем вариации базовых
		exercises = append(exercises,
			models.Exercise{Name: "Жим лёжа с паузой", MuscleGroup: "Грудь", MovementType: "compound"},
			models.Exercise{Name: "Становая тяга с дефицитом", MuscleGroup: "Спина", MovementType: "compound"},
			models.Exercise{Name: "Приседания с паузой", MuscleGroup: "Квадрицепс", MovementType: "compound"},
		)
	}

	return exercises
}

func getExercisesForDisciplines(disciplines []string, goal string) []string {
	var names []string

	for _, d := range disciplines {
		switch d {
		case DisciplineBenchPress:
			names = append(names, "Жим лёжа")
		case DisciplineDeadlift:
			names = append(names, "Становая тяга")
		case DisciplineGluteBridge:
			names = append(names, "Ягодичный мост со штангой")
		case DisciplineStrictCurl:
			names = append(names, "Строгий подъём на бицепс")
		}
	}

	if len(names) == 0 {
		// Базовые упражнения по цели
		switch goal {
		case GoalStrength, GoalCompetition:
			names = []string{"Жим лёжа", "Приседания со штангой", "Становая тяга"}
		case GoalHypertrophy:
			names = []string{"Жим лёжа", "Приседания", "Тяга в наклоне", "Жим стоя"}
		default:
			names = []string{"Жим лёжа", "Приседания", "Становая тяга"}
		}
	}

	return names
}

func getProgramName(goal string, disciplines []string) string {
	if len(disciplines) > 0 {
		var discNames []string
		for _, d := range disciplines {
			switch d {
			case DisciplineBenchPress:
				discNames = append(discNames, "Жим")
			case DisciplineDeadlift:
				discNames = append(discNames, "Тяга")
			case DisciplineGluteBridge:
				discNames = append(discNames, "Мост")
			case DisciplineStrictCurl:
				discNames = append(discNames, "Бицепс")
			}
		}
		return "Подготовка: " + strings.Join(discNames, " + ")
	}

	switch goal {
	case GoalStrength:
		return "Силовая программа"
	case GoalHypertrophy:
		return "Программа на массу"
	case GoalWeightLoss:
		return "Программа похудения"
	case GoalCompetition:
		return "Соревновательная подготовка"
	default:
		return "Тренировочная программа"
	}
}

func printPlanSummary(plan *models.TrainingPlan) {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Printf("║  %s\n", plan.Name)
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Printf("👤 Клиент: %s\n", plan.ClientName)
	fmt.Printf("🎯 Цель: %s\n", plan.Goal)
	fmt.Printf("📅 Недель: %d | Дней/нед: %d\n", plan.TotalWeeks, plan.DaysPerWeek)
	fmt.Printf("📆 Период: %s — %s\n", plan.StartDate.Format("02.01.2006"), plan.EndDate.Format("02.01.2006"))

	if len(plan.OnePMData) > 0 {
		fmt.Println("\n💪 1ПМ:")
		for name, weight := range plan.OnePMData {
			fmt.Printf("   • %s: %.1f кг\n", name, weight)
		}
	}

	fmt.Println("\n📊 Периодизация:")
	for _, meso := range plan.Mesocycles {
		deload := ""
		for _, micro := range meso.Microcycles {
			if micro.IsDeload {
				deload = " (с разгрузкой)"
				break
			}
		}
		fmt.Printf("   Нед %d-%d: %s — %d%% интенс.%s\n",
			meso.WeekStart, meso.WeekEnd, meso.Name, meso.IntensityPercent, deload)
	}

	if len(plan.Weeks) > 0 && len(plan.Weeks[0].Workouts) > 0 {
		fmt.Println("\n🏋️ Неделя 1:")
		for _, workout := range plan.Weeks[0].Workouts {
			fmt.Printf("   День %d: %s\n", workout.DayNum, workout.Name)
			for _, ex := range workout.Exercises {
				weight := ""
				if ex.WeightKg > 0 {
					weight = fmt.Sprintf(" @ %.1f кг", ex.WeightKg)
				} else if ex.WeightPercent > 0 {
					weight = fmt.Sprintf(" @ %.0f%%", ex.WeightPercent)
				}
				fmt.Printf("      • %s: %dx%s%s\n", ex.ExerciseName, ex.Sets, ex.Reps, weight)
			}
		}
	}
}

func createGoogleSheet(plan *models.TrainingPlan, credentials, folderID string) {
	fmt.Println("\n⏳ Создаю Google Sheets...")

	client, err := gsheets.NewClient(credentials, folderID)
	if err != nil {
		log.Printf("❌ Ошибка подключения к Google Sheets: %v", err)
		fmt.Println("   Убедитесь, что файл credentials существует и настроен правильно")
		return
	}

	// Конвертируем в формат gsheets
	programData := convertPlanToGSheets(plan)

	spreadsheetID, err := client.CreateProgramSpreadsheet(programData)
	if err != nil {
		log.Printf("❌ Ошибка создания таблицы: %v", err)
		return
	}

	url := gsheets.GetSpreadsheetURL(spreadsheetID)
	fmt.Println()
	fmt.Println("✅ Таблица создана!")
	fmt.Printf("📊 URL: %s\n", url)
}

func convertPlanToGSheets(plan *models.TrainingPlan) gsheets.ProgramData {
	program := gsheets.ProgramData{
		ClientName:  plan.ClientName,
		ProgramName: plan.Name,
		Goal:        plan.Goal,
		TotalWeeks:  plan.TotalWeeks,
		DaysPerWeek: plan.DaysPerWeek,
		CreatedAt:   time.Now().Format("02.01.2006"),
		OnePMData:   plan.OnePMData,
		Period:      fmt.Sprintf("%s — %s", plan.StartDate.Format("02.01.2006"), plan.EndDate.Format("02.01.2006")),
	}

	for _, week := range plan.Weeks {
		weekData := gsheets.WeekData{
			WeekNum:          week.WeekNum,
			Phase:            string(week.Phase),
			IntensityPercent: week.IntensityPercent,
			VolumePercent:    week.VolumePercent,
			RPETarget:        week.RPETarget,
			IsDeload:         week.IsDeload,
		}

		if week.IsDeload {
			weekData.Focus = "Разгрузка"
		} else {
			weekData.Focus = getPhaseFocus(week.Phase)
		}

		for _, workout := range week.Workouts {
			workoutData := gsheets.WorkoutData{
				DayNum:       workout.DayNum,
				Name:         workout.Name,
				Type:         workout.Type,
				MuscleGroups: workout.MuscleGroups,
			}

			for i, ex := range workout.Exercises {
				workoutData.Exercises = append(workoutData.Exercises, gsheets.ExerciseData{
					OrderNum:      i + 1,
					Name:          ex.ExerciseName,
					MuscleGroup:   ex.MuscleGroup,
					Sets:          ex.Sets,
					Reps:          ex.Reps,
					WeightPercent: ex.WeightPercent,
					WeightKg:      ex.WeightKg,
					RestSeconds:   ex.RestSeconds,
					RPE:           ex.RPE,
				})
			}

			weekData.Workouts = append(weekData.Workouts, workoutData)
		}

		program.Weeks = append(program.Weeks, weekData)
	}

	return program
}

func getPhaseFocus(phase models.PlanPhase) string {
	switch phase {
	case models.PhaseHypertrophy:
		return "Гипертрофия"
	case models.PhaseStrength:
		return "Сила"
	case models.PhasePower:
		return "Мощность"
	case models.PhasePeaking:
		return "Пик"
	case models.PhaseDeload:
		return "Разгрузка"
	case models.PhaseAccumulation:
		return "Накопление"
	case models.PhaseTransmutation:
		return "Трансформация"
	case models.PhaseRealization:
		return "Реализация"
	default:
		return "Базовый"
	}
}

func savePlanToJSON(plan *models.TrainingPlan, filename string) {
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		log.Printf("Ошибка сериализации: %v", err)
		return
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		log.Printf("Ошибка записи файла: %v", err)
		return
	}

	fmt.Printf("💾 Сохранено в %s\n", filename)
}

func runJSONMode(jsonFile, credentials, folderID string) {
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		log.Fatalf("Ошибка чтения файла: %v", err)
	}

	var plan models.TrainingPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		log.Fatalf("Ошибка парсинга JSON: %v", err)
	}

	printPlanSummary(&plan)
	createGoogleSheet(&plan, credentials, folderID)
}

func printUsage() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║   PLANCLI - Генератор тренировочных планов                 ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("ИСПОЛЬЗОВАНИЕ:")
	fmt.Println()
	fmt.Println("  Интерактивный режим (рекомендуется):")
	fmt.Println("    plancli -i")
	fmt.Println()
	fmt.Println("  Быстрая генерация:")
	fmt.Println("    plancli -client \"Иван Иванов\" -goal competition \\")
	fmt.Println("            -disciplines bench_press,glute_bridge \\")
	fmt.Println("            -date 15.03.2026 \\")
	fmt.Println("            -1pm \"Жим лёжа:100,Ягодичный мост:180\" \\")
	fmt.Println("            -target \"Жим лёжа:110,Ягодичный мост:200\"")
	fmt.Println()
	fmt.Println("  Из JSON файла:")
	fmt.Println("    plancli -json plan.json")
	fmt.Println()
	fmt.Println("ФЛАГИ:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("ЦЕЛИ (-goal):")
	fmt.Println("  strength     - Сила")
	fmt.Println("  hypertrophy  - Масса")
	fmt.Println("  weight_loss  - Похудение")
	fmt.Println("  competition  - Соревнования")
	fmt.Println()
	fmt.Println("ДИСЦИПЛИНЫ (-disciplines):")
	fmt.Println("  bench_press  - Жим лёжа")
	fmt.Println("  deadlift     - Становая тяга")
	fmt.Println("  glute_bridge - Ягодичный мост")
	fmt.Println("  strict_curl  - Строгий подъём на бицепс")
	fmt.Println()
	fmt.Println("ПРИМЕРЫ:")
	fmt.Println("  plancli -example        # Пример JSON")
	fmt.Println("  plancli -example-comp   # Пример соревновательной подготовки")
	fmt.Println("  plancli -calc-weeks 15.03.2026  # Рассчитать недели")
}

func calculateWeeksToDate(dateStr string) {
	compDate, err := time.Parse("02.01.2006", dateStr)
	if err != nil {
		log.Fatalf("Ошибка парсинга даты (формат DD.MM.YYYY): %v", err)
	}

	now := time.Now()
	days := compDate.Sub(now).Hours() / 24
	weeks := int(math.Ceil(days / 7))

	fmt.Printf("📅 Дата соревнований: %s\n", compDate.Format("02.01.2006 (Mon)"))
	fmt.Printf("📆 Сегодня: %s\n", now.Format("02.01.2006 (Mon)"))
	fmt.Printf("⏱️  До соревнований: %d дней (~%d недель)\n", int(days), weeks)
	fmt.Println()

	if weeks <= 0 {
		fmt.Println("⚠️  Дата уже прошла!")
		return
	}

	fmt.Println("Рекомендуемая структура:")
	if weeks >= 12 {
		fmt.Printf("  Накопление:    %d недель (65-75%%)\n", weeks/2)
		fmt.Printf("  Трансформация: %d недель (75-85%%)\n", weeks/3)
		fmt.Printf("  Реализация:    %d недель (85-95%%)\n", weeks/6)
		fmt.Println("  Разгрузка:     1 неделя")
	} else if weeks >= 8 {
		fmt.Printf("  Накопление:    %d недель\n", weeks/2)
		fmt.Printf("  Реализация:    %d недель\n", weeks/2-1)
		fmt.Println("  Разгрузка:     1 неделя")
	} else if weeks >= 4 {
		fmt.Printf("  Интенсификация: %d недель\n", weeks-1)
		fmt.Println("  Разгрузка:      1 неделя")
	} else {
		fmt.Println("  ⚠️  Мало времени! Поддерживающий режим.")
	}
}

func printExample() {
	fmt.Println("Пример JSON для стандартной программы — используйте -example-comp для соревнований")
}

func printCompetitionExample() {
	fmt.Println("Для соревновательной подготовки используйте интерактивный режим:")
	fmt.Println()
	fmt.Println("  plancli -i")
	fmt.Println()
	fmt.Println("Или командную строку:")
	fmt.Println()
	fmt.Println("  plancli -client \"Мария Сидорова\" \\")
	fmt.Println("          -goal competition \\")
	fmt.Println("          -disciplines bench_press,glute_bridge,strict_curl \\")
	fmt.Println("          -date 15.03.2026 \\")
	fmt.Println("          -days 4 \\")
	fmt.Println("          -1pm \"Жим лёжа:70,Ягодичный мост:165,Строгий подъём на бицепс:32\"")
}
