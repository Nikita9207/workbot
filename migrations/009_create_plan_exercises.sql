-- Migration 009: Create plan exercises and progression tables

-- Упражнения в тренировочных днях
CREATE TABLE IF NOT EXISTS public.plan_exercises (
    id SERIAL PRIMARY KEY,
    microcycle_id INTEGER NOT NULL REFERENCES public.microcycles(id) ON DELETE CASCADE,
    exercise_id INTEGER NOT NULL REFERENCES public.exercises(id),
    day_of_week INTEGER NOT NULL CHECK (day_of_week >= 1 AND day_of_week <= 7), -- 1=Пн, 7=Вс
    order_num INTEGER NOT NULL DEFAULT 1,
    sets INTEGER NOT NULL DEFAULT 4,
    reps_min INTEGER NOT NULL DEFAULT 8,
    reps_max INTEGER NOT NULL DEFAULT 12,
    intensity_percent DECIMAL(4,1), -- % от 1ПМ
    rpe_target DECIMAL(3,1),
    rest_seconds INTEGER DEFAULT 90,
    tempo VARCHAR(20), -- "3-1-2-0" формат
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_plan_exercises_microcycle ON public.plan_exercises(microcycle_id);
CREATE INDEX IF NOT EXISTS idx_plan_exercises_day ON public.plan_exercises(day_of_week);
CREATE INDEX IF NOT EXISTS idx_plan_exercises_exercise ON public.plan_exercises(exercise_id);

-- Таблица прогрессии (главная таблица с весами по неделям)
CREATE TABLE IF NOT EXISTS public.plan_progression (
    id SERIAL PRIMARY KEY,
    training_plan_id INTEGER NOT NULL REFERENCES public.training_plans(id) ON DELETE CASCADE,
    exercise_id INTEGER NOT NULL REFERENCES public.exercises(id),
    week_number INTEGER NOT NULL,
    day_of_week INTEGER, -- опционально: день недели
    sets INTEGER NOT NULL,
    reps INTEGER NOT NULL,
    weight_kg DECIMAL(6,2), -- рассчитанный или заданный вес
    intensity_percent DECIMAL(4,1), -- % от 1ПМ
    is_deload BOOLEAN DEFAULT false,
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(training_plan_id, exercise_id, week_number)
);

CREATE INDEX IF NOT EXISTS idx_plan_progression_plan ON public.plan_progression(training_plan_id);
CREATE INDEX IF NOT EXISTS idx_plan_progression_week ON public.plan_progression(week_number);
CREATE INDEX IF NOT EXISTS idx_plan_progression_exercise ON public.plan_progression(exercise_id);

-- Шаблоны тренировочных дней (для повторного использования)
CREATE TABLE IF NOT EXISTS public.training_day_templates (
    id SERIAL PRIMARY KEY,
    training_plan_id INTEGER NOT NULL REFERENCES public.training_plans(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL, -- "День А - Верх Push", "День Б - Низ"
    day_type VARCHAR(50), -- push, pull, legs, upper, lower, fullbody
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_day_templates_plan ON public.training_day_templates(training_plan_id);

-- Упражнения в шаблонах дней
CREATE TABLE IF NOT EXISTS public.template_exercises (
    id SERIAL PRIMARY KEY,
    template_id INTEGER NOT NULL REFERENCES public.training_day_templates(id) ON DELETE CASCADE,
    exercise_id INTEGER NOT NULL REFERENCES public.exercises(id),
    order_num INTEGER NOT NULL DEFAULT 1,
    sets_default INTEGER NOT NULL DEFAULT 4,
    reps_min_default INTEGER NOT NULL DEFAULT 8,
    reps_max_default INTEGER NOT NULL DEFAULT 12,
    rest_seconds_default INTEGER DEFAULT 90,
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_template_exercises_template ON public.template_exercises(template_id);
