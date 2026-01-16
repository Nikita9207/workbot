-- Добавляем поле цели в таблицу клиентов
ALTER TABLE public.clients ADD COLUMN IF NOT EXISTS goal TEXT;
ALTER TABLE public.clients ADD COLUMN IF NOT EXISTS training_plan TEXT;
ALTER TABLE public.clients ADD COLUMN IF NOT EXISTS notes TEXT;

-- Таблица для истории целей клиента
CREATE TABLE IF NOT EXISTS public.client_goals (
    id SERIAL PRIMARY KEY,
    client_id INTEGER NOT NULL REFERENCES public.clients(id) ON DELETE CASCADE,
    goal TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'active', -- active, achieved, cancelled
    created_at TIMESTAMP DEFAULT NOW(),
    achieved_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_client_goals_client ON public.client_goals(client_id);
