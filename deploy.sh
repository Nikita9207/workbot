#!/bin/bash

# ============================================
# Deploy скрипт для Raspberry Pi
# ============================================

set -e

# Конфигурация Pi
PI_HOST="${PI_HOST:-192.168.1.135}"
PI_USER="${PI_USER:-nikitakrasilnikov}"
PI_DIR="${PI_DIR:-/home/nikitakrasilnikov/workbot}"

# Цвета
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}=== Deploy workbot на Raspberry Pi ===${NC}"

# 1. Собираем для ARM64
echo -e "${YELLOW}[1/4] Сборка для ARM64...${NC}"
GOOS=linux GOARCH=arm64 go build -o workbot-arm64 ./cmd/main.go

if [ ! -f "workbot-arm64" ]; then
    echo -e "${RED}Ошибка сборки!${NC}"
    exit 1
fi

echo -e "${GREEN}Собрано: $(ls -lh workbot-arm64 | awk '{print $5}')${NC}"

# 2. Копируем на Pi
echo -e "${YELLOW}[2/4] Копирование на Pi...${NC}"
scp workbot-arm64 ${PI_USER}@${PI_HOST}:${PI_DIR}/workbot-new

# 3. Копируем конфиги
echo -e "${YELLOW}[3/4] Копирование конфигов...${NC}"

if [ -f ".env" ]; then
    scp .env ${PI_USER}@${PI_HOST}:${PI_DIR}/.env
    echo "  .env скопирован"
fi

if [ -f "google-credentials.json" ]; then
    scp google-credentials.json ${PI_USER}@${PI_HOST}:${PI_DIR}/google-credentials.json
    echo "  google-credentials.json скопирован"
fi

# Копируем миграции
scp -r migrations ${PI_USER}@${PI_HOST}:${PI_DIR}/
echo "  migrations скопированы"

# 4. Перезапускаем на Pi
echo -e "${YELLOW}[4/4] Перезапуск на Pi...${NC}"
ssh ${PI_USER}@${PI_HOST} << 'EOF'
cd ~/workbot

# Останавливаем старый процесс
if pgrep -f "workbot" > /dev/null; then
    echo "Останавливаю старый процесс..."
    pkill -f "workbot" || true
    sleep 2
fi

# Заменяем бинарник
mv workbot-new workbot
chmod +x workbot

# Запускаем
echo "Запускаю бота..."
nohup ./workbot > workbot.log 2>&1 &

sleep 2

# Проверяем
if pgrep -f "workbot" > /dev/null; then
    echo "Бот запущен! PID: $(pgrep -f workbot)"
else
    echo "ОШИБКА: бот не запустился!"
    tail -20 workbot.log
    exit 1
fi
EOF

echo -e "${GREEN}=== Deploy завершён! ===${NC}"
echo -e "Логи: ssh ${PI_USER}@${PI_HOST} 'tail -f ${PI_DIR}/workbot.log'"
