-- Таблица для записей на тренировки
CREATE TABLE IF NOT EXISTS public.appointments (
    id SERIAL PRIMARY KEY,
    client_id INTEGER NOT NULL REFERENCES public.clients(id) ON DELETE CASCADE,
    trainer_id BIGINT NOT NULL REFERENCES public.admins(telegram_id) ON DELETE CASCADE,
    appointment_date DATE NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    status VARCHAR(20) DEFAULT 'scheduled', -- scheduled, confirmed, cancelled, completed
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Индексы для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_appointments_client ON public.appointments(client_id);
CREATE INDEX IF NOT EXISTS idx_appointments_trainer ON public.appointments(trainer_id);
CREATE INDEX IF NOT EXISTS idx_appointments_date ON public.appointments(appointment_date);
CREATE INDEX IF NOT EXISTS idx_appointments_status ON public.appointments(status);

-- Таблица для расписания тренера (доступные слоты)
CREATE TABLE IF NOT EXISTS public.trainer_schedule (
    id SERIAL PRIMARY KEY,
    trainer_id BIGINT NOT NULL REFERENCES public.admins(telegram_id) ON DELETE CASCADE,
    day_of_week INTEGER NOT NULL CHECK (day_of_week >= 0 AND day_of_week <= 6), -- 0=Вс, 1=Пн, ..., 6=Сб
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    slot_duration INTEGER DEFAULT 60, -- длительность слота в минутах
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Уникальный индекс для расписания
CREATE UNIQUE INDEX IF NOT EXISTS idx_trainer_schedule_unique
ON public.trainer_schedule(trainer_id, day_of_week, start_time);
