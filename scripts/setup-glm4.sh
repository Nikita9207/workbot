#!/bin/bash
# Скрипт установки GLM-4.7-Flash для Ollama
# ==========================================

MODEL_PATH="$HOME/.ollama/gguf/GLM-4.7-Flash-Q4_K_M.gguf"
MODEL_NAME="glm4-flash"

echo "🔧 Настройка GLM-4.7-Flash для Ollama"
echo ""

# Проверяем наличие файла
if [ ! -f "$MODEL_PATH" ]; then
    echo "❌ Файл модели не найден: $MODEL_PATH"
    echo ""
    echo "Скачайте модель командой:"
    echo "curl -L -o $MODEL_PATH \\"
    echo "  'https://huggingface.co/unsloth/GLM-4.7-Flash-GGUF/resolve/main/GLM-4.7-Flash-Q4_K_M.gguf'"
    exit 1
fi

echo "✅ Файл модели найден: $(du -h "$MODEL_PATH" | cut -f1)"

# Создаём Modelfile
MODELFILE="/tmp/Modelfile.glm4"
cat > "$MODELFILE" << 'EOF'
FROM ~/.ollama/gguf/GLM-4.7-Flash-Q4_K_M.gguf

TEMPLATE """{{- if .System }}{{ .System }}

{{ end }}{{- range .Messages }}{{- if eq .Role "user" }}[用户]
{{ .Content }}

[助手]
{{ else if eq .Role "assistant" }}{{ .Content }}

{{ end }}{{- end }}"""

PARAMETER temperature 0.7
PARAMETER num_ctx 32768
PARAMETER stop "[用户]"
PARAMETER stop "</s>"

SYSTEM """You are GLM-4, a helpful AI assistant. You respond in the same language as the user's query. For Russian queries, respond in Russian. Always provide accurate, helpful responses. When asked to generate JSON, return ONLY valid JSON without any additional text."""
EOF

echo "📝 Modelfile создан"

# Проверяем что Ollama запущен
if ! curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo "⚠️ Ollama не запущен. Запускаю..."
    ollama serve &
    sleep 3
fi

# Создаём модель в Ollama
echo "🔨 Создаю модель $MODEL_NAME в Ollama..."
ollama create "$MODEL_NAME" -f "$MODELFILE"

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Модель $MODEL_NAME успешно создана!"
    echo ""
    echo "Проверка:"
    ollama list | grep "$MODEL_NAME"
    echo ""
    echo "Тест модели:"
    echo "  ollama run $MODEL_NAME 'Привет! Расскажи о себе кратко.'"
    echo ""
    echo "Использование в plancli:"
    echo "  ./plancli -ai ollama -ollama-model $MODEL_NAME -client 'Тест' -goal strength -weeks 4 -days 3"
else
    echo "❌ Ошибка создания модели"
    exit 1
fi
