-- Миграция 013: Таблица программ тренировок
-- Полные тренировочные программы для клиентов

-- Таблица программ
CREATE TABLE IF NOT EXISTS public.training_programs (
    id SERIAL PRIMARY KEY,
    client_id INTEGER NOT NULL REFERENCES public.clients(id),
    name VARCHAR(200) NOT NULL,
    goal VARCHAR(500),
    description TEXT,
    total_weeks INTEGER NOT NULL DEFAULT 4,
    days_per_week INTEGER NOT NULL DEFAULT 3,
    start_date DATE NOT NULL DEFAULT CURRENT_DATE,
    end_date DATE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    current_week INTEGER NOT NULL DEFAULT 1,
    file_path VARCHAR(500),
    ai_generated BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Таблица тренировок в программе
CREATE TABLE IF NOT EXISTS public.program_workouts (
    id SERIAL PRIMARY KEY,
    program_id INTEGER NOT NULL REFERENCES public.training_programs(id) ON DELETE CASCADE,
    week_num INTEGER NOT NULL,
    day_num INTEGER NOT NULL,
    order_in_week INTEGER NOT NULL DEFAULT 1,
    name VARCHAR(200) NOT NULL,
    planned_date DATE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    notes TEXT,
    feedback TEXT,
    completed_at TIMESTAMP,
    sent_at TIMESTAMP
);

-- Таблица упражнений в тренировке
CREATE TABLE IF NOT EXISTS public.workout_exercises (
    id SERIAL PRIMARY KEY,
    workout_id INTEGER NOT NULL REFERENCES public.program_workouts(id) ON DELETE CASCADE,
    order_num INTEGER NOT NULL DEFAULT 1,
    exercise_name VARCHAR(200) NOT NULL,
    sets INTEGER NOT NULL DEFAULT 3,
    reps VARCHAR(50) NOT NULL DEFAULT '10',
    weight DECIMAL(6,2),
    weight_percent DECIMAL(5,2),
    rest_seconds INTEGER DEFAULT 90,
    tempo VARCHAR(20),
    rpe DECIMAL(3,1),
    notes TEXT,
    -- Фактические результаты
    actual_sets INTEGER,
    actual_reps INTEGER,
    actual_weight DECIMAL(6,2),
    actual_rpe DECIMAL(3,1),
    completed BOOLEAN DEFAULT false
);

-- Таблица анкет клиентов
CREATE TABLE IF NOT EXISTS public.client_forms (
    id SERIAL PRIMARY KEY,
    client_id INTEGER REFERENCES public.clients(id),
    telegram_id BIGINT,
    name VARCHAR(100),
    surname VARCHAR(100),
    phone VARCHAR(50),
    birth_date VARCHAR(20),
    gender VARCHAR(20),
    height INTEGER,
    weight DECIMAL(5,2),
    goal VARCHAR(200),
    goal_details TEXT,
    experience VARCHAR(50),
    experience_years DECIMAL(4,1),
    training_days INTEGER,
    injuries TEXT,
    equipment TEXT,
    preferences TEXT,
    notes TEXT,
    file_path VARCHAR(500),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Индексы для оптимизации
CREATE INDEX IF NOT EXISTS idx_programs_client_id ON public.training_programs(client_id);
CREATE INDEX IF NOT EXISTS idx_programs_status ON public.training_programs(status);
CREATE INDEX IF NOT EXISTS idx_programs_active ON public.training_programs(client_id, status) WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_workouts_program_id ON public.program_workouts(program_id);
CREATE INDEX IF NOT EXISTS idx_workouts_status ON public.program_workouts(status);
CREATE INDEX IF NOT EXISTS idx_workouts_pending ON public.program_workouts(program_id, status) WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_exercises_workout_id ON public.workout_exercises(workout_id);

CREATE INDEX IF NOT EXISTS idx_forms_client_id ON public.client_forms(client_id);
CREATE INDEX IF NOT EXISTS idx_forms_telegram_id ON public.client_forms(telegram_id);

-- Комментарии
COMMENT ON TABLE public.training_programs IS 'Программы тренировок клиентов';
COMMENT ON TABLE public.program_workouts IS 'Отдельные тренировки в программе';
COMMENT ON TABLE public.workout_exercises IS 'Упражнения в тренировке';
COMMENT ON TABLE public.client_forms IS 'Анкеты клиентов';

COMMENT ON COLUMN public.training_programs.status IS 'active, completed, paused';
COMMENT ON COLUMN public.program_workouts.status IS 'pending, sent, completed, skipped';
