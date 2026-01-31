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
echo -e "${YELLOW}[1/5] Сборка для ARM64...${NC}"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o workbot-arm64 ./cmd/main.go

if [ ! -f "workbot-arm64" ]; then
    echo -e "${RED}Ошибка сборки!${NC}"
    exit 1
fi

echo -e "${GREEN}Собрано: $(ls -lh workbot-arm64 | awk '{print $5}')${NC}"

# 2. Копируем на Pi (бинарник в корень проекта для Docker context)
echo -e "${YELLOW}[2/5] Копирование на Pi...${NC}"
scp workbot-arm64 ${PI_USER}@${PI_HOST}:${PI_DIR}/workbot-arm64

# 3. Копируем конфиги
echo -e "${YELLOW}[3/5] Копирование конфигов...${NC}"

# НЕ копируем .env чтобы не перезаписать production конфиг
# if [ -f ".env" ]; then
#     scp .env ${PI_USER}@${PI_HOST}:${PI_DIR}/.env
#     echo "  .env скопирован"
# fi

if [ -f "google-credentials.json" ]; then
    scp google-credentials.json ${PI_USER}@${PI_HOST}:${PI_DIR}/google-credentials.json
    echo "  google-credentials.json скопирован"
fi

# Google OAuth2 credentials
if [ -f "oauth-credentials.json" ]; then
    scp oauth-credentials.json ${PI_USER}@${PI_HOST}:${PI_DIR}/oauth-credentials.json
    echo "  oauth-credentials.json скопирован"
fi

if [ -f "google-token.json" ]; then
    scp google-token.json ${PI_USER}@${PI_HOST}:${PI_DIR}/google-token.json
    echo "  google-token.json скопирован"
fi

# Копируем миграции
scp -r migrations ${PI_USER}@${PI_HOST}:${PI_DIR}/
echo "  migrations скопированы"

# Копируем Dockerfile
scp docker/Dockerfile.prebuilt ${PI_USER}@${PI_HOST}:${PI_DIR}/docker/Dockerfile.prebuilt
echo "  Dockerfile.prebuilt скопирован"

# Копируем данные (шаблоны пауэрлифтинга, упражнения и т.д.)
if [ -d "data" ]; then
    scp -r data ${PI_USER}@${PI_HOST}:${PI_DIR}/
    echo "  data скопированы"
fi

# Копируем AI шаблоны
if [ -d "clients/ai/templates" ]; then
    ssh ${PI_USER}@${PI_HOST} "mkdir -p ${PI_DIR}/clients/ai"
    scp -r clients/ai/templates ${PI_USER}@${PI_HOST}:${PI_DIR}/clients/ai/
    echo "  clients/ai/templates скопированы"
fi

# Копируем локали (i18n)
if [ -d "locales" ]; then
    scp -r locales ${PI_USER}@${PI_HOST}:${PI_DIR}/
    echo "  locales скопированы"
fi

# 4. Применяем миграции
echo -e "${YELLOW}[4/5] Применение миграций...${NC}"
ssh ${PI_USER}@${PI_HOST} << 'EOF'
cd ~/workbot

# Загружаем переменные окружения
if [ -f ".env" ]; then
    export $(grep -v '^#' .env | xargs)
fi

# Применяем все миграции
for migration in migrations/*.sql; do
    if [ -f "$migration" ]; then
        echo "  Применяю: $(basename $migration)"
        PGPASSWORD=$DB_PASSWORD psql -h ${DB_HOST:-localhost} -U ${DB_USER:-workbot} -d ${DB_NAME:-workbot} -f "$migration" 2>/dev/null || true
    fi
done
echo "  Миграции применены"
EOF

# 5. Пересобираем и запускаем через Docker
echo -e "${YELLOW}[5/5] Пересборка и запуск Docker...${NC}"
ssh ${PI_USER}@${PI_HOST} << 'EOF'
cd ~/workbot/docker

# Пересобираем образ (бинарник копируется в Dockerfile)
docker compose build --no-cache workbot

# Перезапускаем контейнер с новым образом
docker compose up -d workbot

sleep 3

# Проверяем
docker compose ps
docker compose logs --tail=20 workbot
EOF

echo -e "${GREEN}=== Deploy завершён! ===${NC}"
echo -e "Логи: ssh ${PI_USER}@${PI_HOST} 'cd ~/workbot/docker && docker compose logs -f workbot'"
