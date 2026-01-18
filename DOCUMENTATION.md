# WorkBot - Полная документация проекта

## Оглавление

1. [Обзор проекта](#1-обзор-проекта)
2. [Структура директорий](#2-структура-директорий)
3. [Архитектура и компоненты](#3-архитектура-и-компоненты)
4. [База данных](#4-база-данных)
5. [Telegram бот](#5-telegram-бот)
6. [AI интеграции](#6-ai-интеграции)
7. [Google интеграции](#7-google-интеграции)
8. [Алгоритмы тренировок](#8-алгоритмы-тренировок)
9. [Excel интеграция](#9-excel-интеграция)
10. [Конфигурация](#10-конфигурация)
11. [Развёртывание](#11-развёртывание)
12. [Потоки данных](#12-потоки-данных)

---

## 1. Обзор проекта

**WorkBot** — это комплексная система управления персональными тренировками на базе Telegram бота. Система предназначена для персональных тренеров и включает:

- Telegram бот для взаимодействия с клиентами и тренером
- PostgreSQL база данных для хранения всех данных
- Google Sheets & Calendar интеграция для синхронизации
- AI-генерация тренировочных программ (Ollama + Groq Whisper)
- Система периодизации и прогрессии нагрузок
- RAG (Retrieval-Augmented Generation) база знаний

### Технологический стек

| Компонент | Технология |
|-----------|------------|
| Язык | Go 1.24.1 |
| База данных | PostgreSQL 15 |
| Telegram API | go-telegram-bot-api v5 |
| Google APIs | Sheets v4, Calendar v3, Drive v3 |
| AI/LLM | Ollama (локальный), Groq Whisper |
| Excel | excelize v2 |
| Контейнеризация | Docker + Docker Compose |

---

## 2. Структура директорий

```
workbot/
├── cmd/                           # Точки входа приложений
│   ├── main.go                    # Главный Telegram бот
│   ├── program_generator/         # CLI генератор программ
│   ├── indexer/                   # Индексатор документов (RAG)
│   └── test_generator/            # Тестовая генерация
│
├── clients/                       # Клиенты внешних сервисов
│   ├── ai/                        # AI/LLM клиенты
│   │   ├── groq.go               # Ollama клиент (OpenAI-совместимый API)
│   │   ├── whisper.go            # Groq Whisper (транскрипция голоса)
│   │   ├── trainer.go            # TrainerAI - высокоуровневый помощник
│   │   ├── program_generator*.go # Генераторы программ (v1, v2)
│   │   ├── validator*.go         # Валидаторы программ
│   │   └── prompts*.go           # Системные промпты
│   └── knowledge/                 # RAG база знаний
│       └── store.go              # In-memory хранилище с поиском
│
├── internal/                      # Внутренние пакеты
│   ├── bot/                       # Логика Telegram бота
│   │   ├── bot.go                # Структура Bot, инициализация
│   │   ├── handlers.go           # Обработчики команд клиентов
│   │   ├── ai_handlers.go        # AI-обработчики (админ)
│   │   ├── admin*.go             # Админ-панель
│   │   ├── booking_handlers.go   # Система бронирования
│   │   ├── feedback_handlers.go  # Сбор обратной связи
│   │   ├── plan_handlers.go      # Управление планами
│   │   ├── schedule_handlers.go  # Расписание
│   │   ├── onepm_handlers.go     # Отслеживание 1ПМ
│   │   ├── calendar_widget.go    # Визуальный календарь
│   │   └── registration.go       # Регистрация клиентов
│   │
│   ├── models/                    # Модели данных
│   │   ├── types.go              # Базовые модели
│   │   ├── program.go            # Program, Workout, WorkoutExercise
│   │   ├── periodization.go      # Макро/Мезо/Микроциклы
│   │   ├── exercise.go           # Упражнения
│   │   └── training_plan.go      # Тренировочные планы
│   │
│   ├── repository/                # Слой доступа к данным
│   │   ├── repository.go         # Агрегатор репозиториев
│   │   ├── client_repo.go        # Операции с клиентами
│   │   ├── exercise_repo.go      # Операции с упражнениями
│   │   ├── appointment_repo.go   # Операции с записями
│   │   ├── plan_repo.go          # Операции с планами
│   │   └── schedule_repo.go      # Операции с расписанием
│   │
│   ├── config/                    # Конфигурация
│   │   └── config.go             # Загрузка из .env
│   │
│   ├── excel/                     # Работа с Excel
│   │   ├── unified.go            # Единый журнал тренировок
│   │   ├── journal_file.go       # Экспорт журнала
│   │   ├── client_workbook.go    # Книги клиентов
│   │   ├── program_export.go     # Экспорт программ
│   │   ├── watcher.go            # Мониторинг файлов
│   │   └── sync.go               # Синхронизация
│   │
│   ├── training/                  # Алгоритмы тренировок
│   │   ├── onepm.go              # Расчёт 1ПМ
│   │   ├── periodization.go      # Шаблоны периодизации
│   │   └── progression.go        # Прогрессия нагрузок
│   │
│   ├── calendar/                  # ICS календарь
│   │   └── ics.go                # Генерация ICS файлов
│   │
│   ├── gsheets/                   # Google Sheets
│   │   └── client.go             # API клиент
│   │
│   └── gcalendar/                 # Google Calendar
│       └── client.go             # API клиент
│
├── migrations/                    # SQL миграции (15 файлов)
│   ├── 001_create_clients.sql
│   ├── ...
│   └── 015_add_google_event_id.sql
│
├── docker/                        # Docker конфигурация
│   └── docker-compose.yml
│
├── knowledge.json                 # Индекс базы знаний (~220MB)
├── go.mod                         # Go модули
└── .env                           # Конфигурация (не в git)
```

---

## 3. Архитектура и компоненты

### 3.1 Диаграмма компонентов

```
┌─────────────────────────────────────────────────────────────┐
│                    TELEGRAM КЛИЕНТЫ                         │
│              (Клиенты и Тренер/Админ)                       │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                   TELEGRAM BOT API                          │
│            (Long polling, обработка updates)                │
└───────────────────────────┬─────────────────────────────────┘
                            │
            ┌───────────────┼───────────────┐
            ▼               ▼               ▼
     ┌──────────┐    ┌──────────┐    ┌──────────┐
     │ handlers │    │ai_handlers│   │ booking  │
     │ (клиент) │    │  (админ)  │   │ handlers │
     └────┬─────┘    └────┬─────┘   └────┬─────┘
          │               │              │
          └───────────────┼──────────────┘
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                      BOT CORE                               │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │ Telegram API│ │  AI Client  │ │  Knowledge Store    │   │
│  └─────────────┘ └─────────────┘ └─────────────────────┘   │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │Google Sheets│ │Google Cal   │ │   Excel Watcher     │   │
│  └─────────────┘ └─────────────┘ └─────────────────────┘   │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                   REPOSITORY LAYER                          │
│  ClientRepo │ ExerciseRepo │ PlanRepo │ AppointmentRepo    │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                     POSTGRESQL                              │
│        (clients, exercises, plans, appointments...)         │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 Структура Bot

```go
type Bot struct {
    api            *tgbotapi.BotAPI      // Telegram API клиент
    db             *sql.DB               // PostgreSQL соединение
    aiClient       *ai.Client            // Ollama клиент
    whisperClient  *ai.WhisperClient     // Groq Whisper
    knowledgeStore *knowledge.Store      // RAG база знаний
    config         *config.Config        // Конфигурация
    sheetsClient   *gsheets.Client       // Google Sheets
    calendarClient *gcalendar.Client     // Google Calendar
}
```

---

## 4. База данных

### 4.1 Схема базы данных

#### Таблица `clients` - Клиенты
```sql
CREATE TABLE clients (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    surname VARCHAR(100),
    phone VARCHAR(20),
    birth_date DATE,
    telegram_id BIGINT UNIQUE,
    goal TEXT,                    -- Цель тренировок
    training_plan TEXT,           -- Текущий план
    notes TEXT,
    google_sheet_id VARCHAR(255), -- ID Google таблицы
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP          -- Soft delete
);
```

#### Таблица `admins` - Администраторы
```sql
CREATE TABLE admins (
    telegram_id BIGINT PRIMARY KEY,
    created_at TIMESTAMP DEFAULT NOW()
);
```

#### Таблица `appointments` - Записи на тренировки
```sql
CREATE TABLE appointments (
    id SERIAL PRIMARY KEY,
    client_id INTEGER REFERENCES clients(id),
    trainer_id BIGINT,            -- Telegram ID тренера
    appointment_date DATE,
    start_time TIME,
    end_time TIME,
    status VARCHAR(20),           -- scheduled, confirmed, completed, cancelled
    google_event_id VARCHAR(255), -- ID события Google Calendar
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

#### Таблица `exercises` - Упражнения
```sql
CREATE TABLE exercises (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    name_normalized VARCHAR(200), -- Нормализованное название
    muscle_group VARCHAR(50),     -- chest, back, legs, shoulders, arms, core
    movement_type VARCHAR(50),    -- compound, isolation
    equipment VARCHAR(100),       -- barbell, dumbbell, machine, bodyweight
    is_trackable_1pm BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW()
);
```

#### Таблица `training_plans` - Тренировочные планы
```sql
CREATE TABLE training_plans (
    id SERIAL PRIMARY KEY,
    client_id INTEGER REFERENCES clients(id),
    name VARCHAR(200),
    description TEXT,
    start_date DATE,
    end_date DATE,
    status VARCHAR(20),           -- draft, active, completed
    goal VARCHAR(50),             -- strength, hypertrophy, weight_loss, competition
    days_per_week INTEGER,
    total_weeks INTEGER,
    ai_generated BOOLEAN DEFAULT false,
    ai_prompt TEXT,
    created_by BIGINT,            -- Telegram ID создателя
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP
);
```

#### Таблицы периодизации

```sql
-- Макроциклы (годовые периоды)
CREATE TABLE macrocycles (
    id SERIAL PRIMARY KEY,
    plan_id INTEGER REFERENCES training_plans(id),
    name VARCHAR(100),
    goal VARCHAR(50),
    start_date DATE,
    end_date DATE
);

-- Мезоциклы (4-6 недельные блоки)
CREATE TABLE mesocycles (
    id SERIAL PRIMARY KEY,
    macrocycle_id INTEGER REFERENCES macrocycles(id),
    phase VARCHAR(50),            -- hypertrophy, strength, power, peaking, deload
    week_start INTEGER,
    week_end INTEGER,
    intensity_percent DECIMAL(5,2),
    volume_percent DECIMAL(5,2),
    rpe_target DECIMAL(3,1)
);

-- Микроциклы (недельные планы)
CREATE TABLE microcycles (
    id SERIAL PRIMARY KEY,
    mesocycle_id INTEGER REFERENCES mesocycles(id),
    week_number INTEGER,
    is_deload BOOLEAN DEFAULT false,
    volume_modifier DECIMAL(5,2) DEFAULT 1.0,
    intensity_modifier DECIMAL(5,2) DEFAULT 1.0
);

-- Упражнения в плане
CREATE TABLE plan_exercises (
    id SERIAL PRIMARY KEY,
    microcycle_id INTEGER REFERENCES microcycles(id),
    exercise_id INTEGER REFERENCES exercises(id),
    day_of_week INTEGER,          -- 1-7
    sets INTEGER,
    reps VARCHAR(20),             -- "8-12" или "5"
    intensity_percent DECIMAL(5,2),
    rest_seconds INTEGER,
    notes TEXT,
    order_index INTEGER
);

-- Прогрессия нагрузок
CREATE TABLE progression (
    id SERIAL PRIMARY KEY,
    microcycle_id INTEGER REFERENCES microcycles(id),
    exercise_id INTEGER REFERENCES exercises(id),
    week_number INTEGER,
    weight DECIMAL(6,2),
    sets INTEGER,
    reps INTEGER,
    rpe DECIMAL(3,1)
);
```

#### Таблица `training_logs` - Логи тренировок
```sql
CREATE TABLE training_logs (
    id SERIAL PRIMARY KEY,
    client_id INTEGER REFERENCES clients(id),
    plan_exercise_id INTEGER REFERENCES plan_exercises(id),
    exercise_name VARCHAR(200),
    sets_completed INTEGER,
    reps_completed INTEGER,
    weight_used DECIMAL(6,2),
    tonnage DECIMAL(10,2),        -- sets * reps * weight
    status VARCHAR(20),           -- completed, partial, skipped
    rating INTEGER,               -- 1-5 оценка
    feedback TEXT,
    completed_at TIMESTAMP DEFAULT NOW()
);
```

### 4.2 Индексы

```sql
CREATE INDEX idx_clients_telegram_id ON clients(telegram_id);
CREATE INDEX idx_clients_phone ON clients(phone);
CREATE INDEX idx_appointments_client_id ON appointments(client_id);
CREATE INDEX idx_appointments_date ON appointments(appointment_date);
CREATE INDEX idx_appointments_google_event_id ON appointments(google_event_id);
CREATE INDEX idx_exercises_muscle_group ON exercises(muscle_group);
CREATE INDEX idx_training_plans_client_id ON training_plans(client_id);
```

---

## 5. Telegram бот

### 5.1 Команды клиента

| Команда/Текст | Описание |
|---------------|----------|
| `/start` | Начало работы, регистрация или главное меню |
| "Записаться на тренировку" | Открывает визуальный календарь для записи |
| "Мои записи" | Показывает предстоящие записи |
| "Мои тренировки" | История тренировок |
| "Обратная связь" | Отправка текстового или голосового фидбэка |
| "Экспорт в календарь" | Экспорт записей в ICS формат |

### 5.2 Команды админа (тренера)

| Кнопка | Описание |
|--------|----------|
| "AI: Сгенерировать тренировку" | Генерация одной тренировки |
| "AI: План с прогрессией" | 12-недельный план с прогрессией |
| "AI: Методики" | Информация о методиках тренировок |
| "AI: К соревнованиям" | Подготовка к соревнованиям |
| "AI: План на неделю" | Недельный план |
| "AI: Годовой план" | Годовой периодизированный план |
| "AI: Задать вопрос" | Вопрос с RAG-контекстом |
| "Управление клиентами" | CRUD операции с клиентами |
| "Управление программами" | Создание/редактирование программ |
| "1ПМ клиентов" | Отслеживание максимумов |

### 5.3 Машина состояний

```
┌─────────────────────────────────────────────────────────────┐
│                    НАЧАЛЬНОЕ СОСТОЯНИЕ                      │
│                      (нет состояния)                        │
└───────────────────────────┬─────────────────────────────────┘
                            │
            ┌───────────────┼───────────────┐
            ▼               ▼               ▼
     ┌──────────┐    ┌──────────┐    ┌──────────┐
     │reg_name  │    │booking_  │    │feedback_ │
     │          │    │date      │    │text      │
     └────┬─────┘    └────┬─────┘    └────┬─────┘
          ▼               ▼               ▼
     ┌──────────┐    ┌──────────┐    ┌──────────┐
     │reg_phone │    │booking_  │    │ (сброс)  │
     │          │    │time      │    │          │
     └────┬─────┘    └────┬─────┘    └──────────┘
          ▼               ▼
     ┌──────────┐    ┌──────────┐
     │reg_birth │    │booking_  │
     │          │    │confirm   │
     └────┬─────┘    └────┬─────┘
          ▼               ▼
     ┌──────────┐    ┌──────────┐
     │ (сброс)  │    │ (сброс)  │
     └──────────┘    └──────────┘
```

### 5.4 Визуальный календарь

Календарь реализован через inline-кнопки Telegram:

```
      📅 Январь 2026
Пн  Вт  Ср  Чт  Пт  Сб  Вс
          1   2   3   4   5
 6   7   8   9  10  11  12
13  14  15  16  17 [18] 19   ← Сегодня выделено
20  21  22  23  24  25  26
27  28  29  30  31
    ◀️         ▶️
        ❌ Отмена
```

**Callback данные:**
- `cal_day_18.01.2026` — выбор даты
- `cal_prev_01.2026` — предыдущий месяц
- `cal_next_01.2026` — следующий месяц
- `cal_cancel` — отмена

### 5.5 Обработка голосовых сообщений

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ Голосовое   │────▶│  Telegram   │────▶│  Скачать    │
│ сообщение   │     │  File API   │     │  .ogg файл  │
└─────────────┘     └─────────────┘     └──────┬──────┘
                                               │
                                               ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Сохранить  │◀────│  Groq API   │◀────│  Отправить  │
│  feedback   │     │  Whisper    │     │  на Groq    │
└─────────────┘     └─────────────┘     └─────────────┘
```

---

## 6. AI интеграции

### 6.1 Ollama (локальный LLM)

**Файл:** `clients/ai/groq.go`

```go
type Client struct {
    baseURL string  // http://localhost:11434
    model   string  // gemma2:9b-instruct-q4_K_M
    client  *http.Client
}

// OpenAI-совместимый API
func (c *Client) Chat(systemPrompt, userMessage string) (string, error)
func (c *Client) IsAvailable() bool
```

**Использование:**
- Генерация тренировок
- Создание программ
- Ответы на вопросы (с RAG-контекстом)
- Анализ прогресса клиента

**Таймаут:** 10 минут (для длинных генераций)

### 6.2 TrainerAI

**Файл:** `clients/ai/trainer.go`

```go
type TrainerAI struct {
    client *Client
}

type ClientProfile struct {
    Name         string
    Age          int
    Level        string   // beginner, intermediate, advanced
    Goal         string   // strength, hypertrophy, weight_loss
    Restrictions []string
    Equipment    []string
}

func (t *TrainerAI) GenerateTraining(profile ClientProfile) (*Training, error)
func (t *TrainerAI) GenerateWeekPlan(profile ClientProfile) (*WeekPlan, error)
func (t *TrainerAI) GenerateYearPlan(profile ClientProfile) (*YearPlan, error)
```

### 6.3 Groq Whisper

**Файл:** `clients/ai/whisper.go`

```go
type WhisperClient struct {
    apiKey string
}

func (w *WhisperClient) Transcribe(audioPath string) (string, error)
func (w *WhisperClient) TranscribeFromURL(url string) (string, error)
```

**Модель:** `whisper-large-v3-turbo`
**Язык:** Русский (автоопределение)

### 6.4 RAG База знаний

**Файл:** `clients/knowledge/store.go`

```go
type Store struct {
    documents []Document
    mu        sync.RWMutex
}

type Document struct {
    ID       string
    Title    string
    Content  string
    Keywords []string
    Source   string
}

func (s *Store) Load(path string) error
func (s *Store) Search(query string, topK int) []Document
func (s *Store) Count() int
```

**Формат индекса:** JSON (`knowledge.json`, ~220MB)

**Алгоритм поиска:**
1. Токенизация запроса
2. TF-IDF скоринг по ключевым словам
3. Возврат топ-K релевантных документов
4. Добавление в контекст промпта

---

## 7. Google интеграции

### 7.1 Google Sheets

**Файл:** `internal/gsheets/client.go`

```go
type Client struct {
    service  *sheets.Service
    folderID string
}

func NewClient(credentialsPath, folderID string) (*Client, error)
func (c *Client) CreateClientSheet(client *models.Client) (string, error)
func (c *Client) UpdateTrainings(sheetID string, trainings []models.Training) error
func (c *Client) GetClientData(sheetID string) (*models.ClientData, error)
```

**Структура таблицы клиента:**
- Лист "Тренировки" — история тренировок
- Лист "Анкета" — данные клиента
- Лист "Статистика" — графики прогресса

### 7.2 Google Calendar

**Файл:** `internal/gcalendar/client.go`

```go
type Client struct {
    service    *calendar.Service
    calendarID string
}

type Event struct {
    ID          string
    Summary     string
    Description string
    Location    string
    StartTime   time.Time
    EndTime     time.Time
    Attendees   []string
    Reminders   []int     // минуты до события
}

func NewClient(credentialsPath, calendarID string) (*Client, error)
func (c *Client) CreateEvent(event Event) (string, error)
func (c *Client) CreateTrainingEvent(clientName string, startTime time.Time, durationMinutes int, clientEmail string) (string, error)
func (c *Client) UpdateEvent(eventID string, event Event) error
func (c *Client) DeleteEvent(eventID string) error
func (c *Client) GetUpcomingEvents(maxResults int64) ([]Event, error)
func (c *Client) GetBusySlots(startDate, endDate time.Time) ([]BusySlot, error)
```

**Настройка напоминаний:**
- За 1 час до тренировки
- За 15 минут до тренировки

### 7.3 Аутентификация

Используется **Service Account** (сервисный аккаунт):

1. Создать проект в Google Cloud Console
2. Включить APIs: Sheets, Calendar, Drive
3. Создать Service Account
4. Скачать JSON ключ → `google-credentials.json`
5. Дать доступ Service Account к календарю

---

## 8. Алгоритмы тренировок

### 8.1 Расчёт 1ПМ (одноповторный максимум)

**Файл:** `internal/training/onepm.go`

```go
// Формула Brzycki (лучше для <10 повторений)
func Brzycki(weight float64, reps int) float64 {
    return weight * (36.0 / (37.0 - float64(reps)))
}

// Формула Epley (лучше для >10 повторений)
func Epley(weight float64, reps int) float64 {
    return weight * (1.0 + 0.0333*float64(reps))
}

// Среднее двух формул
func Calculate1RM(weight float64, reps int) float64 {
    return (Brzycki(weight, reps) + Epley(weight, reps)) / 2
}

// Рабочий вес от 1ПМ
func WorkingWeight(oneRM float64, intensityPercent float64) float64 {
    return math.Round(oneRM*intensityPercent/100*2) / 2 // округление до 0.5кг
}
```

**Таблица интенсивность → повторения:**

| Интенсивность | Повторения |
|---------------|------------|
| 100% | 1 |
| 97% | 2 |
| 94% | 3 |
| 91% | 4 |
| 88% | 5 |
| 85% | 6 |
| 82% | 7 |
| 79% | 8 |
| 76% | 9 |
| 73% | 10 |
| 70% | 11 |
| 67% | 12 |

### 8.2 Периодизация

**Файл:** `internal/training/periodization.go`

#### Шаблоны периодизации:

**1. Силовая программа (12 недель):**
```
Недели 1-3:  Гипертрофия  │ 70% │ 100% объём │ RPE 7
Недели 4-7:  Сила         │ 82% │ 90% объём  │ RPE 8
Недели 8-10: Мощность     │ 88% │ 75% объём  │ RPE 8.5
Недели 11-12: Пиковая     │ 95% │ 50% объём  │ RPE 9.5
```

**2. Гипертрофия (12 недель):**
```
Недели 1-2:  Вводный      │ 65% │ 80% объём  │ RPE 6
Недели 3-5:  Накопление 1 │ 70% │ 100% объём │ RPE 7
Недели 6-8:  Накопление 2 │ 72% │ 110% объём │ RPE 7.5
Недели 9-12: Интенсив     │ 75% │ 100% объём │ RPE 8
```

**3. Жиросжигание (12 недель):**
```
Недели 1-2:  Адаптация    │ 60% │ 80% объём  │ RPE 6
Недели 3-6:  Жиросжиг 1   │ 65% │ 100% объём │ RPE 7
Недели 7-10: Жиросжиг 2   │ 70% │ 110% объём │ RPE 7.5
Недели 11-12: Поддержка   │ 65% │ 90% объём  │ RPE 7
```

**4. Подготовка к соревнованиям (16 недель):**
```
Недели 1-4:  Общий        │ 70% │ 100% объём │ RPE 7
Недели 5-9:  Специальный  │ 80% │ 90% объём  │ RPE 8
Недели 10-13: Предсоревн. │ 88% │ 75% объём  │ RPE 8.5
Недели 14-16: Пиковая     │ 95% │ 50% объём  │ RPE 9.5
```

### 8.3 Прогрессия нагрузок

**Файл:** `internal/training/progression.go`

```go
type ProgressionConfig struct {
    WeeklyWeightIncrement float64 // кг в неделю
    DeloadFrequency       int     // каждые N недель
    DeloadIntensity       float64 // % от рабочего веса
    DeloadVolume          float64 // % от объёма
}

// Стандартные настройки
var DefaultProgression = ProgressionConfig{
    WeeklyWeightIncrement: 2.5,  // базовые упражнения
    DeloadFrequency:       4,
    DeloadIntensity:       0.65, // -35%
    DeloadVolume:          0.50, // -50%
}

var IsolationProgression = ProgressionConfig{
    WeeklyWeightIncrement: 1.0,  // изолирующие упражнения
    DeloadFrequency:       4,
    DeloadIntensity:       0.70,
    DeloadVolume:          0.60,
}
```

**Логика прогрессии:**
1. Каждую неделю: +2.5кг (базовые) или +1кг (изоляция)
2. Каждую 4-ю неделю: разгрузка (-35% интенсивность, -50% объём)
3. При провале: повтор веса следующую неделю
4. При 3 провалах подряд: снижение на 10% и новый цикл

### 8.4 Расчёт тоннажа

```go
func Tonnage(sets, reps int, weight float64) float64 {
    return float64(sets) * float64(reps) * weight
}

// Пример: 4x8x80кг = 2560кг тоннажа
```

---

## 9. Excel интеграция

### 9.1 Единый журнал тренировок

**Файл:** `internal/excel/unified.go`

Создаёт единую таблицу со всеми тренировками всех клиентов:

| Дата | Клиент | Упражнение | Подходы | Повторения | Вес | Тоннаж | Статус | Оценка | Комментарий |
|------|--------|------------|---------|------------|-----|--------|--------|--------|-------------|
| 18.01.2026 | Иванов И. | Жим лёжа | 4 | 8 | 80 | 2560 | ✅ | 5 | Отлично |

**Цветовое кодирование:**
- 🟢 Зелёный — выполнено
- 🔴 Красный — пропущено
- 🔵 Синий — разгрузка
- ⚪ Серый — запланировано

### 9.2 Мониторинг файлов

**Файл:** `internal/excel/watcher.go`

```go
type Watcher struct {
    dir      string
    interval time.Duration
    onChange func(path string)
}

func NewWatcher(dir string, interval time.Duration) *Watcher
func (w *Watcher) Start(ctx context.Context)
func (w *Watcher) Stop()
```

**Функциональность:**
- Отслеживает изменения в Excel файлах
- Парсит новые тренировки
- Отправляет уведомления в Telegram
- Двусторонняя синхронизация

### 9.3 Экспорт программ

**Файл:** `internal/excel/program_export.go`

Экспортирует тренировочную программу в Excel:

**Лист "Обзор":**
- Название программы
- Цель
- Длительность
- Дни в неделю

**Лист "Неделя 1", "Неделя 2", ...:**
| День | Упражнение | Подходы | Повторения | Интенсивность | Вес | Отдых |
|------|------------|---------|------------|---------------|-----|-------|
| Пн | Приседания | 4 | 6 | 82% | 120 | 180с |

---

## 10. Конфигурация

### 10.1 Переменные окружения

```bash
# Telegram
BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrsTUVwxyz

# База данных
DB_HOST=localhost
DB_PORT=5432
DB_USER=workbot
DB_PASSWORD=secure_password
DB_NAME=workbot

# Рабочие директории
WORK_DIR=/data
CLIENTS_DIR=/data/Клиенты

# Google APIs
GOOGLE_CREDENTIALS_PATH=/app/google-credentials.json
GOOGLE_DRIVE_FOLDER_ID=1abc...xyz
GOOGLE_CALENDAR_ID=primary  # или email@gmail.com

# AI/LLM
OLLAMA_URL=http://localhost:11434
OLLAMA_MODEL=gemma2:9b-instruct-q4_K_M

# Groq (транскрипция голоса)
GROQ_API_KEY=gsk_...

# RAG
RAG_INDEX_PATH=/data/knowledge.json
```

### 10.2 Файл конфигурации

**Файл:** `internal/config/config.go`

```go
type Config struct {
    // Telegram
    BotToken string

    // Database
    DBHost     string
    DBPort     string
    DBUser     string
    DBPassword string
    DBName     string

    // Directories
    WorkDir     string
    ClientsDir  string
    JournalPath string

    // Google
    GoogleCredentialsPath string
    GoogleDriveFolderID   string
    GoogleCalendarID      string

    // AI
    OllamaURL   string
    OllamaModel string
    GroqAPIKey  string

    // RAG
    RAGIndexPath string
}

func Load() (*Config, error) {
    // Загружает из .env и переменных окружения
    // .env имеет приоритет над переменными окружения
}

func (c *Config) DSN() string {
    // Возвращает строку подключения к PostgreSQL
}
```

---

## 11. Развёртывание

### 11.1 Docker Compose

**Файл:** `docker/docker-compose.yml`

```yaml
services:
  postgres:
    image: postgres:15-alpine
    container_name: workbot_postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: ${DB_USER:-workbot}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: ${DB_NAME:-workbot}
      TZ: Europe/Moscow
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER:-workbot}"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - workbot_net

  workbot:
    build:
      context: ..
      dockerfile: docker/Dockerfile.prebuilt
    container_name: workbot_bot
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      BOT_TOKEN: ${BOT_TOKEN}
      DB_HOST: postgres
      DB_PORT: 5432
      DB_USER: ${DB_USER:-workbot}
      DB_PASSWORD: ${DB_PASSWORD}
      DB_NAME: ${DB_NAME:-workbot}
      GROQ_API_KEY: ${GROQ_API_KEY}
      GOOGLE_CREDENTIALS_PATH: /app/google-credentials.json
      GOOGLE_DRIVE_FOLDER_ID: ${GOOGLE_DRIVE_FOLDER_ID}
      GOOGLE_CALENDAR_ID: ${GOOGLE_CALENDAR_ID:-primary}
      OLLAMA_URL: ${OLLAMA_URL:-http://localhost:11434}
      OLLAMA_MODEL: ${OLLAMA_MODEL:-gemma2:9b-instruct-q4_K_M}
      TZ: Europe/Moscow
    volumes:
      - workbot_data:/data
      - ./google-credentials.json:/app/google-credentials.json:ro
    networks:
      - workbot_net

volumes:
  postgres_data:
  workbot_data:

networks:
  workbot_net:
    driver: bridge
```

### 11.2 Команды развёртывания

```bash
# Сборка и запуск
cd docker
docker-compose up -d --build

# Просмотр логов
docker-compose logs -f workbot

# Перезапуск
docker-compose restart workbot

# Остановка
docker-compose down

# Применение миграций
docker exec -i workbot_postgres psql -U workbot -d workbot < migrations/001_create_clients.sql
```

### 11.3 Скрипт деплоя

**Файл:** `deploy.sh`

```bash
#!/bin/bash
set -e

# Сборка для ARM64 (Raspberry Pi)
GOOS=linux GOARCH=arm64 go build -o workbot-arm64 ./cmd/main.go

# Копирование на Pi
scp workbot-arm64 pi@192.168.1.135:~/workbot/
scp .env pi@192.168.1.135:~/workbot/

# Перезапуск на Pi
ssh pi@192.168.1.135 "cd ~/workbot && docker-compose up -d --build"
```

---

## 12. Потоки данных

### 12.1 Регистрация клиента

```
Telegram /start
      │
      ▼
┌─────────────────┐
│ Новый клиент?   │──Да──▶ Запрос имени
└────────┬────────┘              │
         │Нет                    ▼
         ▼                 Запрос телефона
   Главное меню                  │
                                 ▼
                           Запрос даты рождения
                                 │
                                 ▼
                           Сохранение в БД
                                 │
                                 ▼
                     Создание Google Sheet
                                 │
                                 ▼
                           Главное меню
```

### 12.2 Запись на тренировку

```
"Записаться на тренировку"
         │
         ▼
┌─────────────────┐
│ Показать        │
│ визуальный      │
│ календарь       │
└────────┬────────┘
         │
         ▼
   Выбор даты (inline кнопка)
         │
         ▼
┌─────────────────┐
│ Получить        │
│ свободные слоты │
└────────┬────────┘
         │
         ▼
   Выбор времени
         │
         ▼
   Подтверждение
         │
         ▼
┌─────────────────┐
│ Создать запись  │
│ в БД            │
└────────┬────────┘
         │
         ├──────────────────────────────┐
         ▼                              ▼
┌─────────────────┐          ┌─────────────────┐
│ Создать событие │          │ Уведомить       │
│ Google Calendar │          │ тренера         │
└────────┬────────┘          └─────────────────┘
         │
         ▼
   Подтверждение клиенту
```

### 12.3 Генерация тренировки (AI)

```
Админ: "AI: Сгенерировать тренировку"
         │
         ▼
   Выбор клиента
         │
         ▼
   Ввод параметров (тип, интенсивность, длительность)
         │
         ▼
┌─────────────────┐
│ Загрузка        │
│ профиля клиента │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ RAG: поиск      │
│ релевантных     │
│ документов      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Ollama:         │
│ генерация       │
│ тренировки      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Валидация       │
│ программы       │
└────────┬────────┘
         │
         ├──────────────────────────────┐
         ▼                              ▼
┌─────────────────┐          ┌─────────────────┐
│ Сохранение в БД │          │ Экспорт в       │
│                 │          │ Google Sheets   │
└────────┬────────┘          └─────────────────┘
         │
         ▼
   Отправка клиенту в Telegram
```

### 12.4 Голосовой фидбэк

```
Клиент: голосовое сообщение
         │
         ▼
┌─────────────────┐
│ Telegram API:   │
│ получить        │
│ file_path       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Скачать .ogg    │
│ файл            │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Groq Whisper:   │
│ транскрипция    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Сохранить       │
│ в training_logs │
└────────┬────────┘
         │
         ├──────────────────────────────┐
         ▼                              ▼
   Подтверждение клиенту      Уведомление тренеру
```

---

## Приложения

### A. Полезные SQL запросы

```sql
-- Все записи клиента
SELECT * FROM appointments
WHERE client_id = 1
ORDER BY appointment_date DESC;

-- Статистика тренировок за месяц
SELECT
    c.name,
    COUNT(*) as trainings,
    SUM(tl.tonnage) as total_tonnage
FROM training_logs tl
JOIN clients c ON tl.client_id = c.id
WHERE tl.completed_at >= NOW() - INTERVAL '30 days'
GROUP BY c.id, c.name;

-- Прогресс по упражнению
SELECT
    week_number,
    weight,
    sets,
    reps
FROM progression
WHERE exercise_id = 1
ORDER BY week_number;
```

### B. Команды администрирования

```bash
# Бэкап базы данных
docker exec workbot_postgres pg_dump -U workbot workbot > backup.sql

# Восстановление
docker exec -i workbot_postgres psql -U workbot workbot < backup.sql

# Просмотр логов бота
docker logs -f workbot_bot --tail 100

# Подключение к базе
docker exec -it workbot_postgres psql -U workbot workbot
```

### C. Структура Google Credentials

```json
{
  "type": "service_account",
  "project_id": "workbot-XXXXXX",
  "private_key_id": "...",
  "private_key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n",
  "client_email": "workbot-service@workbot-XXXXXX.iam.gserviceaccount.com",
  "client_id": "...",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token"
}
```

---

**Версия документации:** 1.0
**Дата обновления:** 18 января 2026
**Автор:** Claude Code
