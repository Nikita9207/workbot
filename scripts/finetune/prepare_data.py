#!/usr/bin/env python3
"""
Подготовка данных из knowledge.json для fine-tuning LLM
Конвертирует документы в формат JSONL для MLX LoRA

Использование:
    python prepare_data.py --input knowledge.json --output training_data.jsonl
"""

import json
import argparse
import random
from pathlib import Path
from typing import List, Dict, Any

def load_knowledge(input_path: str) -> List[Dict[str, Any]]:
    """Загружает knowledge.json"""
    with open(input_path, 'r', encoding='utf-8') as f:
        data = json.load(f)
    return data.get('documents', [])

def create_training_examples(documents: List[Dict[str, Any]],
                            system_prompt: str = None) -> List[Dict[str, str]]:
    """
    Создаёт обучающие примеры из документов.

    Формат для MLX:
    {"text": "<s>[INST] <<SYS>>\nSystem prompt\n<</SYS>>\n\nUser message [/INST] Assistant response </s>"}

    Или простой формат:
    {"prompt": "...", "completion": "..."}
    """
    examples = []

    if system_prompt is None:
        system_prompt = """Ты - эксперт по силовым тренировкам и спортивной науке.
Отвечай на вопросы о тренировках, используя научные данные и практический опыт.
Давай конкретные рекомендации по упражнениям, подходам, повторениям и периодизации."""

    for doc in documents:
        content = doc.get('content', '').strip()
        metadata = doc.get('metadata', {})
        source_file = metadata.get('file', 'unknown')

        if not content or len(content) < 100:
            continue

        # Разбиваем длинный контент на части
        chunks = split_into_chunks(content, max_length=2000)

        for i, chunk in enumerate(chunks):
            # Создаём Q&A пары на основе контента
            qa_pairs = generate_qa_pairs(chunk, source_file)

            for question, answer in qa_pairs:
                # Формат для MLX LoRA (Llama-style)
                example = {
                    "text": f"<s>[INST] <<SYS>>\n{system_prompt}\n<</SYS>>\n\n{question} [/INST] {answer} </s>"
                }
                examples.append(example)

                # Альтернативный формат (prompt/completion)
                # examples.append({
                #     "prompt": f"{system_prompt}\n\nВопрос: {question}",
                #     "completion": answer
                # })

    return examples

def split_into_chunks(text: str, max_length: int = 2000) -> List[str]:
    """Разбивает текст на части по абзацам"""
    paragraphs = text.split('\n\n')
    chunks = []
    current_chunk = ""

    for para in paragraphs:
        if len(current_chunk) + len(para) < max_length:
            current_chunk += para + "\n\n"
        else:
            if current_chunk:
                chunks.append(current_chunk.strip())
            current_chunk = para + "\n\n"

    if current_chunk:
        chunks.append(current_chunk.strip())

    return chunks

def generate_qa_pairs(content: str, source: str) -> List[tuple]:
    """
    Генерирует пары вопрос-ответ из контента.
    В реальном проекте можно использовать LLM для генерации вопросов.
    """
    pairs = []

    # Простой подход: контент как ответ на общий вопрос
    topic = extract_topic(content)

    questions = [
        f"Расскажи о {topic}",
        f"Что важно знать о {topic}?",
        f"Как применять знания о {topic} в тренировках?",
    ]

    # Выбираем один вопрос случайно для разнообразия
    question = random.choice(questions)
    pairs.append((question, content))

    return pairs

def extract_topic(content: str) -> str:
    """Извлекает тему из контента (упрощённо - первое предложение)"""
    first_line = content.split('\n')[0].strip()
    # Берём первые 50 символов как тему
    topic = first_line[:50].lower()
    if topic.endswith('.'):
        topic = topic[:-1]
    return topic if topic else "тренировках"

def save_jsonl(examples: List[Dict], output_path: str):
    """Сохраняет в JSONL формат"""
    with open(output_path, 'w', encoding='utf-8') as f:
        for example in examples:
            f.write(json.dumps(example, ensure_ascii=False) + '\n')

def split_train_valid(examples: List[Dict], valid_ratio: float = 0.1) -> tuple:
    """Разделяет на train и validation"""
    random.shuffle(examples)
    split_idx = int(len(examples) * (1 - valid_ratio))
    return examples[:split_idx], examples[split_idx:]

def main():
    parser = argparse.ArgumentParser(description='Подготовка данных для fine-tuning')
    parser.add_argument('--input', '-i', default='knowledge.json',
                       help='Путь к knowledge.json')
    parser.add_argument('--output', '-o', default='training_data',
                       help='Префикс для выходных файлов')
    parser.add_argument('--valid-ratio', type=float, default=0.1,
                       help='Доля валидационных данных (default: 0.1)')
    parser.add_argument('--system-prompt', '-s', default=None,
                       help='Системный промпт для обучения')

    args = parser.parse_args()

    print(f"Загружаю {args.input}...")
    documents = load_knowledge(args.input)
    print(f"Загружено {len(documents)} документов")

    print("Создаю обучающие примеры...")
    examples = create_training_examples(documents, args.system_prompt)
    print(f"Создано {len(examples)} примеров")

    if not examples:
        print("Ошибка: не удалось создать примеры!")
        return

    # Разделяем на train/valid
    train_examples, valid_examples = split_train_valid(examples, args.valid_ratio)

    # Сохраняем
    train_path = f"{args.output}_train.jsonl"
    valid_path = f"{args.output}_valid.jsonl"

    save_jsonl(train_examples, train_path)
    save_jsonl(valid_examples, valid_path)

    print(f"\nСохранено:")
    print(f"  Train: {train_path} ({len(train_examples)} примеров)")
    print(f"  Valid: {valid_path} ({len(valid_examples)} примеров)")

    # Показываем пример
    print("\nПример обучающих данных:")
    print("-" * 50)
    sample = train_examples[0]['text'][:500]
    print(sample + "...")

if __name__ == '__main__':
    main()
