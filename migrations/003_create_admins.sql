CREATE TABLE IF NOT EXISTS public.admins (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_admins_telegram_id ON public.admins(telegram_id);
