-- Добавление telegram_id в таблицу clients
ALTER TABLE public.clients ADD COLUMN IF NOT EXISTS telegram_id BIGINT;

-- Индекс для быстрого поиска по telegram_id
CREATE INDEX IF NOT EXISTS idx_clients_telegram_id ON public.clients(telegram_id);

-- Таблица тренировок
CREATE TABLE IF NOT EXISTS public.trainings (
    id SERIAL PRIMARY KEY,
    client_id INTEGER NOT NULL REFERENCES public.clients(id),
    training_date DATE NOT NULL,
    training_time TIME,
    description TEXT,
    sent BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Индексы для тренировок
CREATE INDEX IF NOT EXISTS idx_trainings_client_id ON public.trainings(client_id);
CREATE INDEX IF NOT EXISTS idx_trainings_date ON public.trainings(training_date);
CREATE INDEX IF NOT EXISTS idx_trainings_sent ON public.trainings(sent);

-- Комментарии
COMMENT ON TABLE public.trainings IS 'Расписание тренировок клиентов';
COMMENT ON COLUMN public.trainings.client_id IS 'ID клиента из таблицы clients';
COMMENT ON COLUMN public.trainings.training_date IS 'Дата тренировки';
COMMENT ON COLUMN public.trainings.training_time IS 'Время тренировки';
COMMENT ON COLUMN public.trainings.description IS 'Описание тренировки';
COMMENT ON COLUMN public.trainings.sent IS 'Флаг отправки уведомления клиенту';
