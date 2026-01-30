-- Миграция 017: Добавление таблицы прогресса клиентов и полей для напоминаний

-- Таблица для отслеживания прогресса клиентов (вес, замеры, фото)
CREATE TABLE IF NOT EXISTS public.client_progress (
    id SERIAL PRIMARY KEY,
    client_id INTEGER NOT NULL REFERENCES public.clients(id) ON DELETE CASCADE,
    record_date DATE NOT NULL DEFAULT CURRENT_DATE,

    -- Вес
    weight DECIMAL(5,2), -- кг, до 999.99
    body_fat DECIMAL(4,2), -- % жира, до 99.99

    -- Замеры в см
    chest DECIMAL(5,1),
    waist DECIMAL(5,1),
    hips DECIMAL(5,1),
    biceps DECIMAL(4,1),
    thigh DECIMAL(5,1),

    -- Фото (Telegram file_id)
    photo_file_id VARCHAR(255),

    -- Заметки
    notes TEXT,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Уникальность: один клиент - одна запись в день
    CONSTRAINT unique_client_date UNIQUE (client_id, record_date)
);

-- Индексы для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_client_progress_client_id ON public.client_progress(client_id);
CREATE INDEX IF NOT EXISTS idx_client_progress_date ON public.client_progress(record_date DESC);

-- Добавление полей напоминаний в таблицу appointments
ALTER TABLE public.appointments
ADD COLUMN IF NOT EXISTS reminder_1day_sent BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS reminder_1hour_sent BOOLEAN DEFAULT FALSE;

-- Индекс для быстрого поиска записей для напоминаний
CREATE INDEX IF NOT EXISTS idx_appointments_reminders
ON public.appointments(appointment_date, status)
WHERE reminder_1day_sent = FALSE OR reminder_1hour_sent = FALSE;

-- Комментарии к таблице
COMMENT ON TABLE public.client_progress IS 'Отслеживание прогресса клиентов: вес, замеры тела, фото';
COMMENT ON COLUMN public.client_progress.weight IS 'Вес клиента в кг';
COMMENT ON COLUMN public.client_progress.body_fat IS 'Процент жира в теле';
COMMENT ON COLUMN public.client_progress.chest IS 'Обхват груди в см';
COMMENT ON COLUMN public.client_progress.waist IS 'Обхват талии в см';
COMMENT ON COLUMN public.client_progress.hips IS 'Обхват бёдер в см';
COMMENT ON COLUMN public.client_progress.biceps IS 'Обхват бицепса в см';
COMMENT ON COLUMN public.client_progress.thigh IS 'Обхват бедра в см';
COMMENT ON COLUMN public.client_progress.photo_file_id IS 'Telegram file_id для фото прогресса';
COMMENT ON COLUMN public.appointments.reminder_1day_sent IS 'Отправлено ли напоминание за 1 день';
COMMENT ON COLUMN public.appointments.reminder_1hour_sent IS 'Отправлено ли напоминание за 1 час';
