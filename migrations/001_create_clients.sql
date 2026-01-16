-- Создание/обновление таблицы клиентов для фитнес-бота
-- Безопасно выполнять повторно

-- Создаём таблицу если не существует
CREATE TABLE IF NOT EXISTS public.clients (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    surname VARCHAR(100) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    birth_date VARCHAR(20),
    telegram_id BIGINT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Добавляем колонку telegram_id если её нет
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
        AND table_name = 'clients'
        AND column_name = 'telegram_id'
    ) THEN
        ALTER TABLE public.clients ADD COLUMN telegram_id BIGINT;
    END IF;
END $$;

-- Индексы
CREATE INDEX IF NOT EXISTS idx_clients_phone ON public.clients(phone);
CREATE INDEX IF NOT EXISTS idx_clients_telegram_id ON public.clients(telegram_id);

-- Комментарии
COMMENT ON TABLE public.clients IS 'Таблица клиентов фитнес-тренера';
COMMENT ON COLUMN public.clients.name IS 'Имя клиента';
COMMENT ON COLUMN public.clients.surname IS 'Фамилия клиента';
COMMENT ON COLUMN public.clients.phone IS 'Номер телефона';
COMMENT ON COLUMN public.clients.birth_date IS 'Дата рождения в формате ДД.ММ.ГГГГ';
COMMENT ON COLUMN public.clients.telegram_id IS 'Telegram Chat ID для отправки уведомлений';
