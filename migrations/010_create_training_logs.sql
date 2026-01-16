-- Migration 010: Create training logs and analytics tables

-- Логи выполненных тренировок
CREATE TABLE IF NOT EXISTS public.training_logs (
    id SERIAL PRIMARY KEY,
    client_id INTEGER NOT NULL REFERENCES public.clients(id) ON DELETE CASCADE,
    training_plan_id INTEGER REFERENCES public.training_plans(id) ON DELETE SET NULL,
    plan_exercise_id INTEGER REFERENCES public.plan_exercises(id) ON DELETE SET NULL,
    exercise_id INTEGER NOT NULL REFERENCES public.exercises(id),
    training_date DATE NOT NULL,
    week_number INTEGER,
    day_of_week INTEGER,
    sets_planned INTEGER,
    sets_completed INTEGER NOT NULL,
    reps_planned INTEGER,
    reps_completed INTEGER NOT NULL,
    weight_planned DECIMAL(6,2),
    weight_kg DECIMAL(6,2),
    tonnage_kg DECIMAL(10,2) GENERATED ALWAYS AS (sets_completed * reps_completed * COALESCE(weight_kg, 0)) STORED,
    rpe_target DECIMAL(3,1),
    rpe_actual DECIMAL(3,1),
    status VARCHAR(20) DEFAULT 'completed', -- completed, partial, skipped, modified
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_training_logs_client ON public.training_logs(client_id);
CREATE INDEX IF NOT EXISTS idx_training_logs_plan ON public.training_logs(training_plan_id);
CREATE INDEX IF NOT EXISTS idx_training_logs_date ON public.training_logs(training_date DESC);
CREATE INDEX IF NOT EXISTS idx_training_logs_exercise ON public.training_logs(exercise_id);
CREATE INDEX IF NOT EXISTS idx_training_logs_week ON public.training_logs(week_number);

-- Аналитика объёмов по группам мышц
CREATE TABLE IF NOT EXISTS public.volume_analytics (
    id SERIAL PRIMARY KEY,
    client_id INTEGER NOT NULL REFERENCES public.clients(id) ON DELETE CASCADE,
    training_plan_id INTEGER REFERENCES public.training_plans(id) ON DELETE CASCADE,
    week_number INTEGER NOT NULL,
    muscle_group VARCHAR(100) NOT NULL,
    total_sets INTEGER NOT NULL DEFAULT 0,
    total_reps INTEGER NOT NULL DEFAULT 0,
    total_tonnage DECIMAL(12,2) NOT NULL DEFAULT 0,
    avg_intensity DECIMAL(4,1),
    computed_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(client_id, training_plan_id, week_number, muscle_group)
);

CREATE INDEX IF NOT EXISTS idx_volume_analytics_client ON public.volume_analytics(client_id);
CREATE INDEX IF NOT EXISTS idx_volume_analytics_plan ON public.volume_analytics(training_plan_id);
CREATE INDEX IF NOT EXISTS idx_volume_analytics_week ON public.volume_analytics(week_number);

-- Недельные итоги
CREATE TABLE IF NOT EXISTS public.weekly_summaries (
    id SERIAL PRIMARY KEY,
    client_id INTEGER NOT NULL REFERENCES public.clients(id) ON DELETE CASCADE,
    training_plan_id INTEGER REFERENCES public.training_plans(id) ON DELETE CASCADE,
    week_number INTEGER NOT NULL,
    week_start_date DATE NOT NULL,
    trainings_planned INTEGER DEFAULT 0,
    trainings_completed INTEGER DEFAULT 0,
    total_tonnage DECIMAL(12,2) DEFAULT 0,
    total_sets INTEGER DEFAULT 0,
    total_reps INTEGER DEFAULT 0,
    avg_rpe DECIMAL(3,1),
    compliance_percent DECIMAL(5,2), -- % выполнения плана
    notes TEXT,
    computed_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(client_id, training_plan_id, week_number)
);

CREATE INDEX IF NOT EXISTS idx_weekly_summaries_client ON public.weekly_summaries(client_id);
CREATE INDEX IF NOT EXISTS idx_weekly_summaries_plan ON public.weekly_summaries(training_plan_id);

-- Прогресс по ключевым упражнениям (для графиков)
CREATE TABLE IF NOT EXISTS public.exercise_progress (
    id SERIAL PRIMARY KEY,
    client_id INTEGER NOT NULL REFERENCES public.clients(id) ON DELETE CASCADE,
    exercise_id INTEGER NOT NULL REFERENCES public.exercises(id),
    record_date DATE NOT NULL,
    best_weight DECIMAL(6,2),
    best_reps INTEGER,
    estimated_1pm DECIMAL(6,2),
    total_volume INTEGER, -- сеты * повторы за день
    total_tonnage DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(client_id, exercise_id, record_date)
);

CREATE INDEX IF NOT EXISTS idx_exercise_progress_client ON public.exercise_progress(client_id);
CREATE INDEX IF NOT EXISTS idx_exercise_progress_exercise ON public.exercise_progress(exercise_id);
CREATE INDEX IF NOT EXISTS idx_exercise_progress_date ON public.exercise_progress(record_date DESC);
