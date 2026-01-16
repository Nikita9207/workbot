# Деплой WorkBot на Raspberry Pi

## Архитектура

```
Raspberry Pi (бот + файлы)
├── Docker
│   ├── workbot     (Telegram бот)
│   ├── postgres    (База данных)
│   └── syncthing   (Синхронизация файлов)
│
└── /data/          (Excel файлы)
    ├── Журнал.xlsx
    └── Клиенты/
        ├── Иван_Петров/
        └── ...

        ↕ Syncthing (автосинхронизация)

Mac (копия для просмотра)
└── ~/Desktop/Работа/
    ├── Журнал.xlsx
    └── Клиенты/
```

## Шаг 1: Подготовка Raspberry Pi

### 1.1 Установка Raspberry Pi OS

1. Скачайте [Raspberry Pi Imager](https://www.raspberrypi.com/software/)
2. Запишите **Raspberry Pi OS Lite (64-bit)** на SD карту
3. В настройках включите SSH и Wi-Fi

### 1.2 Подключение по SSH

```bash
ssh pi@raspberrypi.local
# или по IP: ssh pi@192.168.1.XXX
```

## Шаг 2: Установка Docker

На Raspberry Pi выполните:

```bash
# Скачиваем и запускаем скрипт установки
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Добавляем пользователя в группу docker
sudo usermod -aG docker $USER

# Перезаходим для применения изменений
exit
# Заново подключаемся по SSH
ssh pi@raspberrypi.local

# Проверяем
docker --version
```

## Шаг 3: Копирование проекта

### Вариант A: Через Git (рекомендуется)

```bash
cd ~
git clone https://github.com/YOUR_USERNAME/workbot.git
cd workbot
```

### Вариант B: Через SCP (с Mac)

```bash
# На Mac:
scp -r /Users/nikitakrasilnikov/GolandProjects/workbot pi@raspberrypi.local:~/
```

## Шаг 4: Настройка переменных окружения

```bash
cd ~/workbot
cp .env.example .env
nano .env
```

Заполните:
```env
BOT_TOKEN=123456:ABC-DEF...
DB_USER=workbot
DB_PASSWORD=ваш_безопасный_пароль
DB_NAME=workbot
GROQ_API_KEY=gsk_...
WORK_DIR=/data
```

## Шаг 5: Запуск

```bash
cd ~/workbot/docker
docker compose up -d
```

Проверка:
```bash
docker compose ps
docker compose logs -f workbot
```

## Шаг 6: Настройка Syncthing

### 6.1 На Raspberry Pi

1. Откройте в браузере: `http://raspberrypi.local:8384`
2. При первом запуске задайте пароль для Web UI
3. Перейдите в **Actions → Settings → GUI**
4. Скопируйте **Device ID** (понадобится для Mac)

### 6.2 На Mac

1. Установите Syncthing:
   ```bash
   brew install syncthing
   brew services start syncthing
   ```

2. Откройте: `http://localhost:8384`

3. **Add Remote Device**:
   - Вставьте Device ID от Raspberry Pi
   - Дайте имя (например "Raspberry Pi")

4. На Raspberry Pi:
   - Примите запрос на подключение от Mac

### 6.3 Настройка папки синхронизации

На **Raspberry Pi** (Web UI):

1. **Add Folder**:
   - Folder Label: `WorkBot Data`
   - Folder Path: `/data`
   - Share with: выберите Mac

На **Mac** (Web UI):

1. Примите папку
2. Укажите путь: `~/Desktop/Работа`

## Полезные команды

```bash
# Логи бота
docker compose logs -f workbot

# Перезапуск бота
docker compose restart workbot

# Остановка всего
docker compose down

# Обновление (после git pull)
docker compose build --no-cache workbot
docker compose up -d

# Бэкап
./deploy/backup.sh
```

## Автозапуск после перезагрузки

Docker уже настроен на автозапуск (`restart: unless-stopped`).

Для проверки:
```bash
sudo reboot
# После перезагрузки:
docker compose ps
```

## Мониторинг

### Проверка синхронизации
- Raspberry Pi: `http://raspberrypi.local:8384`
- Mac: `http://localhost:8384`

### Проверка бота
```bash
docker compose logs --tail=50 workbot
```

## Устранение проблем

### Бот не запускается
```bash
docker compose logs workbot
# Проверьте .env файл
```

### Syncthing не синхронизирует
1. Проверьте что оба устройства онлайн
2. Проверьте что папка расшарена
3. Проверьте права доступа: `ls -la /data`

### Нет места на диске
```bash
# Очистка Docker
docker system prune -a
```
