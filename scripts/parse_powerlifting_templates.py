#!/usr/bin/env python3
"""
Парсер таблиц пауэрлифтинга (Шейко, Головинский, Русский цикл и др.)
Извлекает шаблоны программ в JSON для использования в генераторе.
"""

import pandas as pd
import json
import re
import os
from pathlib import Path

OUTPUT_DIR = Path("/Users/nikitakrasilnikov/GolandProjects/workbot/data/powerlifting_templates")
BOOKS_DIR = Path("/Users/nikitakrasilnikov/Desktop/Книги")


def parse_sheiko_12_weeks():
    """Парсит Шейко 12 недель для разрядников."""
    file_path = BOOKS_DIR / "sheyko-12_nedel.xls"
    xls = pd.ExcelFile(file_path, engine='xlrd')

    templates = {
        "name": "Шейко 12 недель (разрядники)",
        "author": "Борис Шейко",
        "level": ["2_разряд", "1_разряд", "КМС"],
        "weeks": 12,
        "days_per_week": 3,
        "type": "троеборье",
        "phases": []
    }

    # Парсим каждый план (подготовительный 1, 2, соревновательный)
    for sheet_name in ['План подг. 1', 'План подг. 2', 'План соревн. 3']:
        df = pd.read_excel(xls, sheet_name=sheet_name, header=None)
        phase = parse_sheiko_phase(df, sheet_name)
        if phase:
            templates["phases"].append(phase)

    return templates


def parse_sheiko_phase(df, phase_name):
    """Парсит одну фазу из таблицы Шейко."""
    phase = {
        "name": phase_name,
        "weeks": []
    }

    current_week = None
    current_workout = None

    for idx, row in df.iterrows():
        # Ищем начало недели
        cell0 = str(row[0]) if pd.notna(row[0]) else ""
        cell1 = str(row[1]) if pd.notna(row[1]) else ""

        # Новая неделя
        if "неделя" in cell0.lower():
            week_num = int(re.search(r'(\d+)', cell0).group(1))
            current_week = {
                "week_num": week_num,
                "workouts": []
            }
            phase["weeks"].append(current_week)

        # Новая тренировка
        if "тр." in cell1.lower() and current_week is not None:
            tr_num = int(re.search(r'(\d+)', cell1).group(1))
            current_workout = {
                "workout_num": tr_num,
                "exercises": []
            }
            current_week["workouts"].append(current_workout)

        # Упражнение с процентами
        cell2 = str(row[2]) if pd.notna(row[2]) else ""
        cell3 = str(row[3]) if pd.notna(row[3]) else ""

        if current_workout is not None and cell2 and cell3:
            # Главные упражнения (жим, присед, тяга)
            if any(ex in cell2.lower() for ex in ['жим', 'присед', 'тяга']):
                exercise = parse_sheiko_exercise(cell2, cell3)
                if exercise:
                    current_workout["exercises"].append(exercise)
            # Подсобка (просто название)
            elif cell2 and not cell2.replace('.', '').replace(',', '').replace(' ', '').isdigit():
                # Это подсобное упражнение
                accessory = {
                    "name": cell2.strip(),
                    "type": "accessory",
                    "sets_reps": parse_accessory_sets(row)
                }
                if accessory["sets_reps"]:
                    current_workout["exercises"].append(accessory)

    return phase if phase["weeks"] else None


def parse_sheiko_exercise(name, scheme):
    """
    Парсит упражнение формата: 50% 5x1, 60% 4x2, 70% 3x2, 75% 3x5.
    Возвращает структуру с подходами.
    """
    exercise = {
        "name": name.strip(),
        "type": "competition",
        "sets": []
    }

    # Парсим схему типа "50% 5x1, 60% 4x2, 70% 3x2, 75% 3x5."
    pattern = r'(\d+)%\s*(\d+)x(\d+)'
    matches = re.findall(pattern, scheme)

    for match in matches:
        percent, reps, sets = int(match[0]), int(match[1]), int(match[2])
        exercise["sets"].append({
            "percent": percent,
            "reps": reps,
            "sets": sets
        })

    return exercise if exercise["sets"] else None


def parse_accessory_sets(row):
    """Парсит подходы для подсобки (обычно просто числа повторений)."""
    sets = []
    for i in range(3, min(len(row), 15)):
        val = row[i]
        if pd.notna(val):
            try:
                reps = int(float(val))
                sets.append(reps)
            except:
                pass
    return sets


def parse_sheiko_kms_ms():
    """Парсит Шейко для КМС/МС (4 недели к соревнованиям)."""
    file_path = BOOKS_DIR / "Sheyko_plan_kmc_mc.xls"
    xls = pd.ExcelFile(file_path, engine='xlrd')

    df = pd.read_excel(xls, sheet_name='План', header=None)

    template = {
        "name": "Шейко КМС/МС (4 недели к соревнованиям)",
        "author": "Борис Шейко",
        "level": ["КМС", "МС"],
        "weeks": 4,
        "days_per_week": 4,
        "type": "троеборье",
        "weeks_data": []
    }

    current_week = None
    current_day_exercises = []
    current_date = None

    for idx, row in df.iterrows():
        cell0 = str(row[0]) if pd.notna(row[0]) else ""
        cell1 = row[1]

        # Новая неделя
        if "Неделя" in cell0:
            if current_week and current_day_exercises:
                current_week["days"].append({"exercises": current_day_exercises})
            week_num = int(re.search(r'(\d+)', cell0).group(1))
            current_week = {
                "week_num": week_num,
                "days": []
            }
            template["weeks_data"].append(current_week)
            current_day_exercises = []

        # Новая дата = новый день
        if pd.notna(cell1) and isinstance(cell1, (pd.Timestamp, str)):
            date_str = str(cell1)
            if "2014" in date_str or "2015" in date_str:
                if current_week and current_day_exercises:
                    current_week["days"].append({"exercises": current_day_exercises})
                current_day_exercises = []
                current_date = date_str

        # Упражнение
        exercise_name = str(row[3]) if len(row) > 3 and pd.notna(row[3]) else ""

        if exercise_name and current_week is not None:
            exercise = parse_kms_exercise(row)
            if exercise:
                current_day_exercises.append(exercise)

    # Добавляем последний день
    if current_week and current_day_exercises:
        current_week["days"].append({"exercises": current_day_exercises})

    return template


def parse_kms_exercise(row):
    """
    Парсит упражнение формата КМС/МС: вес х подх х повт
    Например: 90 х1х3 означает 90 кг, 1 подход, 3 повтора
    """
    exercise_name = str(row[3]) if len(row) > 3 and pd.notna(row[3]) else ""
    if not exercise_name:
        return None

    exercise = {
        "name": exercise_name.strip(),
        "sets": []
    }

    # Парсим подходы (колонки 4-15 содержат вес и схему)
    i = 4
    while i < min(len(row) - 1, 16):
        weight = row[i] if pd.notna(row[i]) else None
        scheme = str(row[i + 1]) if i + 1 < len(row) and pd.notna(row[i + 1]) else ""

        if weight and scheme and "х" in scheme.lower():
            # Парсим "х1х3" = 1 подход, 3 повтора
            match = re.search(r'х(\d+)х(\d+)', scheme.lower())
            if match:
                sets, reps = int(match.group(1)), int(match.group(2))
                exercise["sets"].append({
                    "weight": float(weight),
                    "sets": sets,
                    "reps": reps
                })
        i += 2

    return exercise if exercise["sets"] else None


def parse_golovinsky_cycle(filename, cycle_name):
    """Парсит цикл Головинского."""
    file_path = BOOKS_DIR / filename
    xls = pd.ExcelFile(file_path, engine='openpyxl')

    df = pd.read_excel(xls, sheet_name='Цикл', header=None)

    template = {
        "name": f"Головинский {cycle_name}",
        "author": "Головинский",
        "level": ["1_разряд", "КМС", "МС"],
        "type": "троеборье",
        "microcycles": []
    }

    import datetime
    current_micro = None

    for idx, row in df.iterrows():
        cell2 = str(row[2]) if pd.notna(row[2]) else ""
        cell6 = str(row[6]) if len(row) > 6 and pd.notna(row[6]) else ""

        # Новый микроцикл (в колонке 6)
        if "Микроцикл" in cell6:
            micro_match = re.search(r'(\d+)', cell6)
            if micro_match:
                micro_num = int(micro_match.group(1))
                # Извлекаем статистику микроцикла
                tonnage = row[21] if len(row) > 21 and pd.notna(row[21]) else 0
                avg_weight = row[22] if len(row) > 22 and pd.notna(row[22]) else 0
                kps = row[26] if len(row) > 26 and pd.notna(row[26]) else 0

                current_micro = {
                    "microcycle_num": micro_num,
                    "stats": {
                        "tonnage": float(tonnage) if tonnage else 0,
                        "avg_weight": float(avg_weight) if avg_weight else 0,
                        "kps": int(kps) if kps else 0
                    },
                    "days": []
                }
                template["microcycles"].append(current_micro)
            continue

        # Пропускаем заголовки
        if cell2 == "Упражнения" or not cell2:
            continue

        # Упражнение
        if current_micro is not None and cell2 and cell2 != "nan":
            exercise = parse_golovinsky_exercise(row)
            if exercise:
                # Проверяем дату для группировки по дням
                if pd.notna(row[0]) and isinstance(row[0], (pd.Timestamp, datetime.datetime)):
                    date_str = str(row[0]).split()[0]  # Только дата без времени
                    day_found = False
                    for day in current_micro["days"]:
                        if day.get("date") == date_str:
                            day["exercises"].append(exercise)
                            day_found = True
                            break
                    if not day_found:
                        current_micro["days"].append({
                            "date": date_str,
                            "exercises": [exercise]
                        })
                elif current_micro["days"]:
                    # Продолжение того же дня
                    current_micro["days"][-1]["exercises"].append(exercise)

    return template


def parse_golovinsky_exercise(row):
    """Парсит упражнение Головинского.
    Структура колонок: 5=Вес, 6=Повт, 7=Подх, 8=%, затем 9=Вес, 10=Повт, 11=Подх, 12=% и т.д.
    """
    exercise_name = str(row[2]) if len(row) > 2 and pd.notna(row[2]) else ""
    load_type = str(row[1]) if len(row) > 1 and pd.notna(row[1]) else ""

    if not exercise_name or exercise_name == "nan":
        return None

    exercise = {
        "name": exercise_name.strip(),
        "load_type": load_type.strip(),  # Легкая, Средняя, Тяжелая
        "sets": []
    }

    # Парсим подходы: Вес(5), Повт(6), Подх(7), %(8), затем Вес(9), Повт(10), Подх(11), %(12) и т.д.
    col_sets = [(5, 6, 7, 8), (9, 10, 11, 12), (13, 14, 15, 16), (17, 18, 19, 20)]

    for weight_col, reps_col, sets_col, pct_col in col_sets:
        if weight_col < len(row) and pd.notna(row[weight_col]):
            try:
                weight = float(row[weight_col]) if row[weight_col] else 0
                reps = int(row[reps_col]) if reps_col < len(row) and pd.notna(row[reps_col]) else 0
                sets = int(row[sets_col]) if sets_col < len(row) and pd.notna(row[sets_col]) else 0
                pct = float(row[pct_col]) if pct_col < len(row) and pd.notna(row[pct_col]) else 0

                if weight > 0 and reps > 0 and sets > 0:
                    exercise["sets"].append({
                        "weight": weight,
                        "reps": reps,
                        "sets": sets,
                        "percent": pct * 100 if pct < 1 else pct  # Конвертируем 0.7 → 70%
                    })
            except (ValueError, TypeError):
                pass

    # Извлекаем статистику упражнения
    if len(row) > 26:
        try:
            exercise["stats"] = {
                "tonnage": float(row[21]) if pd.notna(row[21]) else 0,
                "avg_weight": float(row[22]) if pd.notna(row[22]) else 0,
                "intensity": float(row[23]) if pd.notna(row[23]) else 0,
                "pm": float(row[24]) if pd.notna(row[24]) else 0,  # 1ПМ для этого упражнения
                "kps": int(row[26]) if pd.notna(row[26]) else 0
            }
        except (ValueError, TypeError):
            pass

    return exercise if exercise["sets"] else None


def parse_russian_cycle():
    """Парсит Русский цикл."""
    file_path = BOOKS_DIR / "Russky-tsikl_-Programma-trenirovok_-Pauerliftin.xls"
    xls = pd.ExcelFile(file_path, engine='xlrd')

    template = {
        "name": "Русский цикл",
        "author": "Народная программа",
        "level": ["2_разряд", "1_разряд", "КМС"],
        "weeks": 12,
        "days_per_week": 2,
        "exercises": {}
    }

    # Парсим каждый лист (присед, жим, тяга)
    for sheet_name in xls.sheet_names:
        if "присед" in sheet_name.lower():
            template["exercises"]["squat"] = parse_russian_cycle_sheet(xls, sheet_name)
        elif "жим" in sheet_name.lower():
            template["exercises"]["bench"] = parse_russian_cycle_sheet(xls, sheet_name)
        elif "станов" in sheet_name.lower():
            template["exercises"]["deadlift"] = parse_russian_cycle_sheet(xls, sheet_name)

    return template


def parse_russian_cycle_sheet(xls, sheet_name):
    """Парсит один лист Русского цикла."""
    df = pd.read_excel(xls, sheet_name=sheet_name, header=None)

    exercise_data = {
        "name": sheet_name,
        "weeks": []
    }

    # Ищем строки с данными по неделям
    for idx, row in df.iterrows():
        cell0 = row[0] if pd.notna(row[0]) else ""

        # Строка с номером недели
        if isinstance(cell0, (int, float)) and 1 <= cell0 <= 15:
            week_num = int(cell0)

            # Собираем данные по весам
            week_data = {
                "week_num": week_num,
                "sets": []
            }

            # Колонки с весами (проценты указаны в заголовке)
            # Обычно: 75, 78, 81, 84, 87, 90, 93, 96, 99, 102, 105, 108, 111, 114
            base_percents = [62.5, 65, 67.5, 70, 72.5, 75, 77.5, 80, 82.5, 85, 87.5, 90, 92.5, 95]

            for i, pct in enumerate(base_percents):
                col_idx = i + 1  # Колонки начинаются с 1
                if col_idx < len(row) and pd.notna(row[col_idx]):
                    val = row[col_idx]
                    if isinstance(val, (int, float)):
                        reps = int(val)
                        week_data["sets"].append({
                            "percent": pct,
                            "reps": reps
                        })
                    elif isinstance(val, str) and "/" in val:
                        # Формат "3/5" = 3 повтора x 5 подходов
                        # Убираем лишние символы типа "(1*)"
                        clean_val = re.sub(r'\([^)]*\)', '', val).strip()
                        parts = clean_val.split("/")
                        try:
                            reps, sets = int(parts[0]), int(parts[1])
                            week_data["sets"].append({
                                "percent": pct,
                                "reps": reps,
                                "sets": sets
                            })
                        except ValueError:
                            pass

            if week_data["sets"]:
                exercise_data["weeks"].append(week_data)

    return exercise_data


def parse_muravyev_cycle():
    """Парсит цикл Муравьёва."""
    file_path = BOOKS_DIR / "Tsikl-Muravyeva.xls"
    xls = pd.ExcelFile(file_path, engine='xlrd')

    template = {
        "name": "Цикл Муравьёва (16 недель)",
        "author": "Муравьёв",
        "level": ["1_разряд", "КМС"],
        "weeks": 16,
        "sheets": []
    }

    for sheet_name in xls.sheet_names:
        df = pd.read_excel(xls, sheet_name=sheet_name, header=None)
        template["sheets"].append({
            "name": sheet_name,
            "data": df.head(30).to_dict()
        })

    return template


def clean_nan(obj):
    """Рекурсивно заменяет NaN на None для JSON совместимости."""
    import math
    if isinstance(obj, dict):
        return {k: clean_nan(v) for k, v in obj.items()}
    elif isinstance(obj, list):
        return [clean_nan(v) for v in obj]
    elif isinstance(obj, float) and math.isnan(obj):
        return None
    return obj


def save_template(template, filename):
    """Сохраняет шаблон в JSON."""
    output_path = OUTPUT_DIR / filename
    cleaned = clean_nan(template)
    with open(output_path, 'w', encoding='utf-8') as f:
        json.dump(cleaned, f, ensure_ascii=False, indent=2, default=str)
    print(f"✅ Сохранено: {output_path}")


def main():
    """Главная функция парсинга."""
    print("🏋️ Парсинг таблиц пауэрлифтинга...\n")

    # Шейко 12 недель
    print("📊 Шейко 12 недель...")
    try:
        template = parse_sheiko_12_weeks()
        save_template(template, "sheiko_12_weeks.json")
    except Exception as e:
        print(f"❌ Ошибка: {e}")

    # Шейко КМС/МС
    print("📊 Шейко КМС/МС...")
    try:
        template = parse_sheiko_kms_ms()
        save_template(template, "sheiko_kms_ms.json")
    except Exception as e:
        print(f"❌ Ошибка: {e}")

    # Головинский циклы
    for filename, name in [("cycle2.xlsm", "Цикл 2"), ("cycle7.xlsm", "Цикл 7"), ("cycle11.xlsm", "Цикл 11")]:
        print(f"📊 Головинский {name}...")
        try:
            template = parse_golovinsky_cycle(filename, name)
            save_template(template, f"golovinsky_{name.replace(' ', '_').lower()}.json")
        except Exception as e:
            print(f"❌ Ошибка: {e}")

    # Русский цикл
    print("📊 Русский цикл...")
    try:
        template = parse_russian_cycle()
        save_template(template, "russian_cycle.json")
    except Exception as e:
        print(f"❌ Ошибка: {e}")

    # Муравьёв
    print("📊 Цикл Муравьёва...")
    try:
        template = parse_muravyev_cycle()
        save_template(template, "muravyev_cycle.json")
    except Exception as e:
        print(f"❌ Ошибка: {e}")

    # Верхошанский - жим лёжа
    print("📊 Верхошанский (жим лёжа)...")
    try:
        template = parse_verkhoshansky_bench()
        save_template(template, "verkhoshansky_bench.json")
    except Exception as e:
        print(f"❌ Ошибка: {e}")

    # Верхошанский - присед
    print("📊 Верхошанский (присед)...")
    try:
        template = parse_verkhoshansky_squat()
        save_template(template, "verkhoshansky_squat.json")
    except Exception as e:
        print(f"❌ Ошибка: {e}")

    print("\n✅ Парсинг завершён!")


def parse_verkhoshansky_bench():
    """Парсит цикл Верхошанского для жима лёжа."""
    file_path = BOOKS_DIR / "Verkhoshansky_-Tsikl_-Pauerlifting-Zhim-Prisyad.xls"
    xls = pd.ExcelFile(file_path, engine='xlrd')

    template = {
        "name": "Верхошанский 6 недель (жим лёжа)",
        "author": "Юрий Верхошанский",
        "level": ["1_разряд", "КМС", "МС"],
        "weeks": 6,
        "days_per_week": 2,
        "type": "жим",
        "weeks_data": []
    }

    df = pd.read_excel(xls, sheet_name="жим лёжа", header=None)

    current_week = None
    week_num = 0

    for idx, row in df.iterrows():
        cell1 = str(row[1]) if pd.notna(row[1]) else ""

        # Ищем строки с номерами недель (Неделя 1, Week 2, Peak Week 6)
        week_match = re.match(r'.*(?:неделя|week)\s*(\d+)', cell1, re.IGNORECASE)
        if week_match:
            # Сначала сохраняем предыдущую неделю если есть данные
            if current_week and (current_week["workouts"][0]["exercises"] or current_week["workouts"][1]["exercises"]):
                template["weeks_data"].append(current_week)

            week_num = int(week_match.group(1))
            current_week = {
                "week_num": week_num,
                "workouts": [
                    {"workout_num": 1, "exercises": []},
                    {"workout_num": 2, "exercises": []}
                ]
            }
            continue

        # Пропускаем заголовок "Подход"
        if "подход" in cell1.lower():
            continue

        # Ищем подходы (числа 1-10, "Set 1", "1" и т.д.)
        set_num = None
        try:
            if isinstance(row[1], (int, float)) and 1 <= row[1] <= 10:
                set_num = int(row[1])
            elif cell1.isdigit() and 1 <= int(cell1) <= 10:
                set_num = int(cell1)
            else:
                # Формат "Set 1", "Set 2"
                set_match = re.match(r'set\s*(\d+)', cell1, re.IGNORECASE)
                if set_match:
                    set_num = int(set_match.group(1))
        except:
            pass

        if current_week and set_num:
            # Лёгкая тренировка (col 4=проценты, col 6=повторения)
            light_pct = row[4] if pd.notna(row[4]) else None
            light_reps = row[6] if pd.notna(row[6]) else None

            # Тяжёлая тренировка (col 9=проценты, col 11=повторения)
            heavy_pct = row[9] if pd.notna(row[9]) else None
            heavy_reps = row[11] if pd.notna(row[11]) else None

            # Добавляем к лёгкой тренировке
            if light_pct is not None and light_reps is not None:
                try:
                    pct = float(light_pct)
                    if pct < 1:  # Если указано как 0.45 вместо 45
                        pct *= 100
                    reps = parse_reps_range(light_reps)

                    bench_ex = find_or_create_exercise(current_week["workouts"][0]["exercises"], "Жим лёжа")
                    bench_ex["sets"].append({
                        "percent": pct,
                        "reps": reps,
                        "sets": 1
                    })
                except (ValueError, TypeError):
                    pass

            # Добавляем к тяжёлой тренировке
            if heavy_pct is not None and heavy_reps is not None:
                try:
                    pct = float(heavy_pct)
                    if pct < 1:
                        pct *= 100
                    reps = parse_reps_range(heavy_reps)

                    bench_ex = find_or_create_exercise(current_week["workouts"][1]["exercises"], "Жим лёжа")
                    bench_ex["sets"].append({
                        "percent": pct,
                        "reps": reps,
                        "sets": 1
                    })
                except (ValueError, TypeError):
                    pass

    # Последняя неделя
    if current_week and (current_week["workouts"][0]["exercises"] or current_week["workouts"][1]["exercises"]):
        template["weeks_data"].append(current_week)

    return template


def parse_verkhoshansky_squat():
    """Парсит цикл Верхошанского для приседа."""
    file_path = BOOKS_DIR / "Verkhoshansky_-Tsikl_-Pauerlifting-Zhim-Prisyad.xls"
    xls = pd.ExcelFile(file_path, engine='xlrd')

    template = {
        "name": "Верхошанский 6 недель (присед)",
        "author": "Юрий Верхошанский",
        "level": ["1_разряд", "КМС", "МС"],
        "weeks": 6,
        "days_per_week": 2,
        "type": "присед",
        "weeks_data": []
    }

    df = pd.read_excel(xls, sheet_name="Приседания", header=None)

    current_week = None
    week_num = 0

    for idx, row in df.iterrows():
        cell1 = str(row[1]) if pd.notna(row[1]) else ""

        # Ищем строки с номерами недель
        week_match = re.match(r'.*(?:неделя|week)\s*(\d+)', cell1, re.IGNORECASE)
        if week_match:
            if current_week and (current_week["workouts"][0]["exercises"] or current_week["workouts"][1]["exercises"]):
                template["weeks_data"].append(current_week)

            week_num = int(week_match.group(1))
            current_week = {
                "week_num": week_num,
                "workouts": [
                    {"workout_num": 1, "exercises": []},
                    {"workout_num": 2, "exercises": []}
                ]
            }
            continue

        # Пропускаем заголовок
        if "подход" in cell1.lower():
            continue

        # Ищем подходы (числа 1-10, "Set 1", "1" и т.д.)
        set_num = None
        try:
            if isinstance(row[1], (int, float)) and 1 <= row[1] <= 10:
                set_num = int(row[1])
            elif cell1.isdigit() and 1 <= int(cell1) <= 10:
                set_num = int(cell1)
            else:
                # Формат "Set 1", "Set 2"
                set_match = re.match(r'set\s*(\d+)', cell1, re.IGNORECASE)
                if set_match:
                    set_num = int(set_match.group(1))
        except:
            pass

        if current_week and set_num:
            light_pct = row[4] if pd.notna(row[4]) else None
            light_reps = row[6] if pd.notna(row[6]) else None
            heavy_pct = row[9] if pd.notna(row[9]) else None
            heavy_reps = row[11] if pd.notna(row[11]) else None

            if light_pct is not None and light_reps is not None:
                try:
                    pct = float(light_pct)
                    if pct < 1:
                        pct *= 100
                    reps = parse_reps_range(light_reps)

                    squat_ex = find_or_create_exercise(current_week["workouts"][0]["exercises"], "Приседания")
                    squat_ex["sets"].append({
                        "percent": pct,
                        "reps": reps,
                        "sets": 1
                    })
                except (ValueError, TypeError):
                    pass

            if heavy_pct is not None and heavy_reps is not None:
                try:
                    pct = float(heavy_pct)
                    if pct < 1:
                        pct *= 100
                    reps = parse_reps_range(heavy_reps)

                    squat_ex = find_or_create_exercise(current_week["workouts"][1]["exercises"], "Приседания")
                    squat_ex["sets"].append({
                        "percent": pct,
                        "reps": reps,
                        "sets": 1
                    })
                except (ValueError, TypeError):
                    pass

    if current_week and (current_week["workouts"][0]["exercises"] or current_week["workouts"][1]["exercises"]):
        template["weeks_data"].append(current_week)

    return template


def find_or_create_exercise(exercises, name):
    """Находит или создаёт упражнение в списке."""
    for ex in exercises:
        if ex["name"] == name:
            return ex

    new_ex = {
        "name": name,
        "type": "competition",
        "sets": []
    }
    exercises.append(new_ex)
    return new_ex


def parse_reps_range(val):
    """Парсит диапазон повторений типа '8-10', '6', '65% X 5'."""
    if isinstance(val, (int, float)):
        return int(val)

    s = str(val).strip()

    # Формат "65% X 5" или "75% X 5"
    if "%" in s.lower() and "x" in s.lower():
        # Берём число после X
        m = re.search(r'x\s*(\d+)', s, re.IGNORECASE)
        if m:
            return int(m.group(1))
        # Если диапазон после X: "50-55% X 8-12"
        m = re.search(r'x\s*(\d+)-(\d+)', s, re.IGNORECASE)
        if m:
            return (int(m.group(1)) + int(m.group(2))) // 2

    # Диапазон "8-10"
    if "-" in s and "%" not in s:
        parts = s.split("-")
        try:
            return (int(parts[0].strip()) + int(parts[1].strip())) // 2
        except:
            pass

    # Просто число
    m = re.search(r'^(\d+)$', s.strip())
    if m:
        return int(m.group(1))

    # Число в конце строки
    m = re.search(r'(\d+)\s*$', s)
    if m:
        return int(m.group(1))

    return 5  # default


if __name__ == "__main__":
    main()
