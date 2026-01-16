-- Migration 007: Create exercises catalog and 1PM tracking tables

-- Каталог упражнений
CREATE TABLE IF NOT EXISTS public.exercises (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    name_normalized VARCHAR(255) NOT NULL, -- lowercase для поиска
    muscle_group VARCHAR(100), -- грудь, спина, ноги, плечи, руки, кор
    movement_type VARCHAR(50), -- compound, isolation
    equipment VARCHAR(100), -- штанга, гантели, тренажёр, собственный вес
    is_trackable_1pm BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(name_normalized)
);

CREATE INDEX IF NOT EXISTS idx_exercises_name ON public.exercises(name_normalized);
CREATE INDEX IF NOT EXISTS idx_exercises_muscle_group ON public.exercises(muscle_group);

-- История 1ПМ клиентов
CREATE TABLE IF NOT EXISTS public.exercise_1pm (
    id SERIAL PRIMARY KEY,
    client_id INTEGER NOT NULL REFERENCES public.clients(id) ON DELETE CASCADE,
    exercise_id INTEGER NOT NULL REFERENCES public.exercises(id) ON DELETE CASCADE,
    one_pm_kg DECIMAL(6,2) NOT NULL,
    test_date DATE NOT NULL,
    calc_method VARCHAR(20) DEFAULT 'manual', -- manual, brzycki, epley, average
    source_weight DECIMAL(6,2), -- вес для расчёта (если не manual)
    source_reps INTEGER, -- повторы для расчёта
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    created_by BIGINT REFERENCES public.admins(telegram_id)
);

CREATE INDEX IF NOT EXISTS idx_exercise_1pm_client ON public.exercise_1pm(client_id);
CREATE INDEX IF NOT EXISTS idx_exercise_1pm_exercise ON public.exercise_1pm(exercise_id);
CREATE INDEX IF NOT EXISTS idx_exercise_1pm_date ON public.exercise_1pm(test_date DESC);

-- Базовые упражнения для старта
INSERT INTO public.exercises (name, name_normalized, muscle_group, movement_type, equipment) VALUES
    ('Жим лёжа', 'жим лёжа', 'грудь', 'compound', 'штанга'),
    ('Жим лёжа на наклонной', 'жим лёжа на наклонной', 'грудь', 'compound', 'штанга'),
    ('Жим гантелей лёжа', 'жим гантелей лёжа', 'грудь', 'compound', 'гантели'),
    ('Приседания со штангой', 'приседания со штангой', 'ноги', 'compound', 'штанга'),
    ('Фронтальные приседания', 'фронтальные приседания', 'ноги', 'compound', 'штанга'),
    ('Становая тяга', 'становая тяга', 'спина', 'compound', 'штанга'),
    ('Румынская тяга', 'румынская тяга', 'ноги', 'compound', 'штанга'),
    ('Тяга штанги в наклоне', 'тяга штанги в наклоне', 'спина', 'compound', 'штанга'),
    ('Подтягивания', 'подтягивания', 'спина', 'compound', 'собственный вес'),
    ('Тяга верхнего блока', 'тяга верхнего блока', 'спина', 'compound', 'тренажёр'),
    ('Жим стоя', 'жим стоя', 'плечи', 'compound', 'штанга'),
    ('Жим гантелей сидя', 'жим гантелей сидя', 'плечи', 'compound', 'гантели'),
    ('Отжимания на брусьях', 'отжимания на брусьях', 'грудь', 'compound', 'собственный вес'),
    ('Выпады', 'выпады', 'ноги', 'compound', 'гантели'),
    ('Жим ногами', 'жим ногами', 'ноги', 'compound', 'тренажёр'),
    ('Сгибания на бицепс', 'сгибания на бицепс', 'руки', 'isolation', 'штанга'),
    ('Французский жим', 'французский жим', 'руки', 'isolation', 'штанга'),
    ('Разгибания на трицепс', 'разгибания на трицепс', 'руки', 'isolation', 'тренажёр'),
    ('Разведения гантелей', 'разведения гантелей', 'грудь', 'isolation', 'гантели'),
    ('Махи гантелями в стороны', 'махи гантелями в стороны', 'плечи', 'isolation', 'гантели'),
    ('Тяга гантели в наклоне', 'тяга гантели в наклоне', 'спина', 'compound', 'гантели'),
    ('Гиперэкстензия', 'гиперэкстензия', 'спина', 'isolation', 'тренажёр'),
    ('Планка', 'планка', 'кор', 'isolation', 'собственный вес'),
    ('Скручивания', 'скручивания', 'кор', 'isolation', 'собственный вес'),
    ('Сгибания ног', 'сгибания ног', 'ноги', 'isolation', 'тренажёр'),
    ('Разгибания ног', 'разгибания ног', 'ноги', 'isolation', 'тренажёр'),
    -- Соревновательные дисциплины
    ('Ягодичный мост', 'ягодичный мост', 'ноги', 'compound', 'штанга'),
    ('Строгий подъём на бицепс', 'строгий подъём на бицепс', 'руки', 'isolation', 'штанга'),
    ('Свободный подъём на бицепс', 'свободный подъём на бицепс', 'руки', 'isolation', 'штанга')
ON CONFLICT (name_normalized) DO NOTHING;
