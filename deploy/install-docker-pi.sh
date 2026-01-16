#!/bin/bash
# Скрипт установки Docker на Raspberry Pi
# Запускать: curl -sSL https://raw.githubusercontent.com/.../install-docker-pi.sh | bash

set -e

echo "=== Установка Docker на Raspberry Pi ==="

# Обновляем систему
echo "Обновление системы..."
sudo apt-get update
sudo apt-get upgrade -y

# Устанавливаем зависимости
echo "Установка зависимостей..."
sudo apt-get install -y \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    git

# Добавляем GPG ключ Docker
echo "Добавление Docker GPG ключа..."
curl -fsSL https://download.docker.com/linux/debian/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

# Добавляем репозиторий Docker
echo "Добавление репозитория Docker..."
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Устанавливаем Docker
echo "Установка Docker..."
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# Добавляем пользователя в группу docker
echo "Добавление пользователя в группу docker..."
sudo usermod -aG docker $USER

# Включаем автозапуск Docker
echo "Включение автозапуска Docker..."
sudo systemctl enable docker
sudo systemctl start docker

# Проверяем установку
echo "Проверка установки..."
docker --version
docker compose version

echo ""
echo "=== Docker установлен успешно! ==="
echo "ВАЖНО: Перезайдите в систему или выполните: newgrp docker"
echo ""
