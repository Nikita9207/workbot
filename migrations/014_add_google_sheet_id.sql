-- Добавляем поле для хранения ID Google таблицы клиента
ALTER TABLE clients ADD COLUMN IF NOT EXISTS google_sheet_id VARCHAR(255);

-- Индекс для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_clients_google_sheet_id ON clients(google_sheet_id) WHERE google_sheet_id IS NOT NULL;
