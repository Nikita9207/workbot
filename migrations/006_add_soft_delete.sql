-- Добавляем поле для мягкого удаления клиентов
ALTER TABLE public.clients ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- Индекс для быстрой фильтрации активных клиентов
CREATE INDEX IF NOT EXISTS idx_clients_deleted_at ON public.clients(deleted_at);

-- Комментарий
COMMENT ON COLUMN public.clients.deleted_at IS 'Дата удаления клиента (soft delete). NULL = активный клиент';
