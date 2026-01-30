-- Предотвращение двойного бронирования на одно время
-- Добавляем уникальный индекс: один тренер не может иметь две записи на одну дату и время (кроме отменённых)
CREATE UNIQUE INDEX IF NOT EXISTS idx_appointments_unique_slot
ON public.appointments(trainer_id, appointment_date, start_time)
WHERE status != 'cancelled';

-- Комментарий: partial index исключает отменённые записи из проверки уникальности
