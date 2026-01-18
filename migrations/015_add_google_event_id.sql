-- Добавляем поле для хранения Google Calendar Event ID
ALTER TABLE public.appointments ADD COLUMN IF NOT EXISTS google_event_id VARCHAR(255);

-- Индекс для быстрого поиска по Google Event ID
CREATE INDEX IF NOT EXISTS idx_appointments_google_event_id ON public.appointments(google_event_id);
