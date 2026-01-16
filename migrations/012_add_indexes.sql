-- Migration 012: Add indexes for performance optimization

-- Индексы для clients
CREATE INDEX IF NOT EXISTS idx_clients_telegram_id ON public.clients(telegram_id) WHERE telegram_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_clients_deleted_at ON public.clients(deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_clients_active ON public.clients(id) WHERE deleted_at IS NULL;

-- Индексы для appointments
CREATE INDEX IF NOT EXISTS idx_appointments_client_id ON public.appointments(client_id);
CREATE INDEX IF NOT EXISTS idx_appointments_trainer_id ON public.appointments(trainer_id);
CREATE INDEX IF NOT EXISTS idx_appointments_date ON public.appointments(appointment_date);
CREATE INDEX IF NOT EXISTS idx_appointments_status ON public.appointments(status) WHERE status != 'cancelled';
CREATE INDEX IF NOT EXISTS idx_appointments_upcoming ON public.appointments(appointment_date, start_time)
    WHERE appointment_date >= CURRENT_DATE AND status != 'cancelled';

-- Индексы для trainer_schedule
CREATE INDEX IF NOT EXISTS idx_schedule_trainer ON public.trainer_schedule(trainer_id) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_schedule_day ON public.trainer_schedule(day_of_week) WHERE is_active = true;

-- Индексы для exercise_1pm
CREATE INDEX IF NOT EXISTS idx_1pm_client_exercise ON public.exercise_1pm(client_id, exercise_id);
CREATE INDEX IF NOT EXISTS idx_1pm_latest ON public.exercise_1pm(client_id, exercise_id, test_date DESC);

-- Индексы для training_plans
CREATE INDEX IF NOT EXISTS idx_plans_client ON public.training_plans(client_id);
CREATE INDEX IF NOT EXISTS idx_plans_status ON public.training_plans(status);
CREATE INDEX IF NOT EXISTS idx_plans_active ON public.training_plans(client_id, status) WHERE status = 'active';

-- Индексы для mesocycles
CREATE INDEX IF NOT EXISTS idx_mesocycles_plan ON public.mesocycles(training_plan_id);

-- Индексы для microcycles
CREATE INDEX IF NOT EXISTS idx_microcycles_meso ON public.microcycles(mesocycle_id);

-- Индексы для plan_exercises
CREATE INDEX IF NOT EXISTS idx_plan_exercises_micro ON public.plan_exercises(microcycle_id);

-- Индексы для plan_progression
CREATE INDEX IF NOT EXISTS idx_progression_plan ON public.plan_progression(training_plan_id);
CREATE INDEX IF NOT EXISTS idx_progression_exercise ON public.plan_progression(training_plan_id, exercise_id);

-- Индексы для training_logs
CREATE INDEX IF NOT EXISTS idx_logs_client ON public.training_logs(client_id);
CREATE INDEX IF NOT EXISTS idx_logs_date ON public.training_logs(training_date);
CREATE INDEX IF NOT EXISTS idx_logs_client_date ON public.training_logs(client_id, training_date DESC);

-- Индексы для admins
CREATE INDEX IF NOT EXISTS idx_admins_telegram ON public.admins(telegram_id);

-- Индексы для exercises
CREATE INDEX IF NOT EXISTS idx_exercises_name ON public.exercises(name_normalized);
CREATE INDEX IF NOT EXISTS idx_exercises_muscle ON public.exercises(muscle_group);
