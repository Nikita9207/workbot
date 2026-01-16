-- Migration 011: Add competition discipline exercises

-- Соревновательные дисциплины
INSERT INTO public.exercises (name, name_normalized, muscle_group, movement_type, equipment) VALUES
    ('Ягодичный мост', 'ягодичный мост', 'ноги', 'compound', 'штанга'),
    ('Строгий подъём на бицепс', 'строгий подъём на бицепс', 'руки', 'isolation', 'штанга'),
    ('Свободный подъём на бицепс', 'свободный подъём на бицепс', 'руки', 'isolation', 'штанга')
ON CONFLICT (name_normalized) DO NOTHING;

-- Жим лёжа и Становая тяга уже есть в базе
