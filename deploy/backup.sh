#!/bin/bash
# Скрипт резервного копирования данных workbot
set -e

BACKUP_DIR="${HOME}/workbot_backups"
DATE=$(date +%Y%m%d_%H%M%S)

echo "=== Резервное копирование WorkBot ==="

mkdir -p "$BACKUP_DIR"

# Бэкап PostgreSQL
echo "Создание дампа PostgreSQL..."
docker exec workbot_postgres pg_dump -U workbot workbot > "$BACKUP_DIR/db_$DATE.sql"

# Бэкап Excel файлов
echo "Копирование Excel файлов..."
docker cp workbot_bot:/data "$BACKUP_DIR/data_$DATE"

# Сжимаем
echo "Сжатие архива..."
tar -czf "$BACKUP_DIR/workbot_backup_$DATE.tar.gz" \
    -C "$BACKUP_DIR" \
    "db_$DATE.sql" \
    "data_$DATE"

# Удаляем временные файлы
rm -f "$BACKUP_DIR/db_$DATE.sql"
rm -rf "$BACKUP_DIR/data_$DATE"

# Удаляем старые бэкапы (старше 7 дней)
find "$BACKUP_DIR" -name "workbot_backup_*.tar.gz" -mtime +7 -delete

echo ""
echo "Бэкап создан: $BACKUP_DIR/workbot_backup_$DATE.tar.gz"
echo ""
