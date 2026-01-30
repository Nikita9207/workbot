-- Миграция 018: Добавление поля language для мультиязычности
-- Дата: 2026-01-30

-- Добавляем поле language в таблицу clients
ALTER TABLE public.clients
ADD COLUMN IF NOT EXISTS language VARCHAR(2) DEFAULT 'ru';

-- Создаем индекс для быстрого поиска по языку
CREATE INDEX IF NOT EXISTS idx_clients_language ON public.clients(language);

-- Комментарий к колонке
COMMENT ON COLUMN public.clients.language IS 'Язык интерфейса клиента (ru, en)';
