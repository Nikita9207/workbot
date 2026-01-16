-- Migration 008: Create training plans and periodization tables

-- Тренировочные планы
CREATE TABLE IF NOT EXISTS public.training_plans (
    id SERIAL PRIMARY KEY,
    client_id INTEGER NOT NULL REFERENCES public.clients(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    start_date DATE NOT NULL,
    end_date DATE,
    status VARCHAR(20) DEFAULT 'active', -- draft, active, completed, archived
    goal VARCHAR(100), -- сила, масса, похудение, выносливость, соревнования
    days_per_week INTEGER DEFAULT 3,
    total_weeks INTEGER DEFAULT 12,
    ai_generated BOOLEAN DEFAULT false,
    ai_prompt TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_by BIGINT REFERENCES public.admins(telegram_id)
);

CREATE INDEX IF NOT EXISTS idx_training_plans_client ON public.training_plans(client_id);
CREATE INDEX IF NOT EXISTS idx_training_plans_status ON public.training_plans(status);

-- Макроциклы (годовой уровень)
CREATE TABLE IF NOT EXISTS public.macrocycles (
    id SERIAL PRIMARY KEY,
    training_plan_id INTEGER NOT NULL REFERENCES public.training_plans(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    main_goal VARCHAR(255),
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_macrocycles_plan ON public.macrocycles(training_plan_id);

-- Мезоциклы (4-6 недель)
CREATE TABLE IF NOT EXISTS public.mesocycles (
    id SERIAL PRIMARY KEY,
    macrocycle_id INTEGER REFERENCES public.macrocycles(id) ON DELETE CASCADE,
    training_plan_id INTEGER NOT NULL REFERENCES public.training_plans(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL, -- "Втягивающий", "Базовый", "Пиковый"
    phase VARCHAR(50) NOT NULL, -- hypertrophy, strength, power, peaking, deload
    week_start INTEGER NOT NULL,
    week_end INTEGER NOT NULL,
    volume_percent INTEGER DEFAULT 100,
    intensity_percent INTEGER DEFAULT 70, -- целевая интенсивность % от 1ПМ
    rpe_target DECIMAL(3,1) DEFAULT 7.5,
    notes TEXT,
    order_num INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mesocycles_plan ON public.mesocycles(training_plan_id);
CREATE INDEX IF NOT EXISTS idx_mesocycles_macrocycle ON public.mesocycles(macrocycle_id);

-- Микроциклы (недели)
CREATE TABLE IF NOT EXISTS public.microcycles (
    id SERIAL PRIMARY KEY,
    mesocycle_id INTEGER NOT NULL REFERENCES public.mesocycles(id) ON DELETE CASCADE,
    week_number INTEGER NOT NULL, -- абсолютный номер недели в плане
    name VARCHAR(100),
    is_deload BOOLEAN DEFAULT false,
    volume_modifier DECIMAL(3,2) DEFAULT 1.0, -- 0.5 для deload
    intensity_modifier DECIMAL(3,2) DEFAULT 1.0,
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_microcycles_mesocycle ON public.microcycles(mesocycle_id);
CREATE INDEX IF NOT EXISTS idx_microcycles_week ON public.microcycles(week_number);
