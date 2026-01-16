#!/bin/bash
# Скрипт деплоя workbot на Raspberry Pi
set -e

WORKBOT_DIR="${HOME}/workbot"
REPO_URL="https://github.com/YOUR_USERNAME/workbot.git"  # Заменить на свой репозиторий

echo "=== Деплой WorkBot на Raspberry Pi ==="

# Проверяем Docker
if ! command -v docker &> /dev/null; then
    echo "Docker не установлен! Запустите сначала install-docker-pi.sh"
    exit 1
fi

# Создаём директорию
mkdir -p "$WORKBOT_DIR"
cd "$WORKBOT_DIR"

# Клонируем или обновляем репозиторий
if [ -d ".git" ]; then
    echo "Обновление репозитория..."
    git pull
else
    echo "Клонирование репозитория..."
    # Если репозитория нет, копируем файлы вручную
    echo "Репозиторий не настроен. Скопируйте файлы проекта в $WORKBOT_DIR"
fi

# Проверяем наличие .env
if [ ! -f ".env" ]; then
    echo ""
    echo "Создайте файл .env с настройками:"
    echo "cp .env.example .env"
    echo "nano .env"
    echo ""

    # Создаём .env.example если его нет
    if [ ! -f ".env.example" ]; then
        cat > .env.example << 'EOF'
# Telegram Bot
BOT_TOKEN=your_telegram_bot_token

# PostgreSQL
DB_USER=workbot
DB_PASSWORD=your_secure_password
DB_NAME=workbot

# Groq AI (опционально)
GROQ_API_KEY=your_groq_api_key

# Рабочая директория (внутри контейнера)
WORK_DIR=/data
EOF
    fi
    exit 1
fi

# Останавливаем старые контейнеры
echo "Останавливаем старые контейнеры..."
cd docker
docker compose down 2>/dev/null || true

# Собираем и запускаем
echo "Собираем и запускаем контейнеры..."
docker compose build --no-cache
docker compose up -d

# Проверяем статус
echo ""
echo "Статус контейнеров:"
docker compose ps

echo ""
echo "=== Деплой завершён! ==="
echo ""
echo "Syncthing Web UI: http://$(hostname -I | awk '{print $1}'):8384"
echo ""
echo "Полезные команды:"
echo "  docker compose logs -f workbot  # Логи бота"
echo "  docker compose logs -f postgres # Логи БД"
echo "  docker compose restart workbot  # Перезапуск бота"
echo ""
