#!/bin/bash

# Скрипт для нагрузочного тестирования URL Shortener
# Использование: ./scripts/load_test.sh [количество_запросов]

REQUESTS=${1:-1000}
BASE_URL="http://localhost:8080"
CONCURRENT=10

echo "🚀 Запуск нагрузочного тестирования..."
echo "📊 Количество запросов: $REQUESTS"
echo "⚡ Параллельных соединений: $CONCURRENT"
echo ""

# Функция для создания короткой ссылки
create_short_url() {
    local id=$1
    curl -s -X POST "$BASE_URL/" \
        -H "Content-Type: text/plain" \
        -d "https://example.com/page/$id" \
        > /dev/null
}

# Функция для создания короткой ссылки через JSON API
create_short_url_json() {
    local id=$1
    curl -s -X POST "$BASE_URL/api/shorten" \
        -H "Content-Type: application/json" \
        -d "{\"url\":\"https://example.com/page/$id\"}" \
        > /dev/null
}

# Функция для batch создания
create_batch() {
    local batch_size=10
    local json="["
    for i in $(seq 1 $batch_size); do
        if [ $i -gt 1 ]; then
            json="$json,"
        fi
        json="$json{\"correlation_id\":\"id_$i\",\"original_url\":\"https://example.com/batch/$i\"}"
    done
    json="$json]"
    
    curl -s -X POST "$BASE_URL/api/shorten/batch" \
        -H "Content-Type: application/json" \
        -d "$json" \
        > /dev/null
}

echo "1️⃣  Тест POST / (text/plain)..."
for i in $(seq 1 $((REQUESTS / 3))); do
    create_short_url $i &
    if [ $((i % CONCURRENT)) -eq 0 ]; then
        wait
    fi
done
wait
echo "✅ Завершено"

echo ""
echo "2️⃣  Тест POST /api/shorten (JSON)..."
for i in $(seq 1 $((REQUESTS / 3))); do
    create_short_url_json $i &
    if [ $((i % CONCURRENT)) -eq 0 ]; then
        wait
    fi
done
wait
echo "✅ Завершено"

echo ""
echo "3️⃣  Тест POST /api/shorten/batch..."
for i in $(seq 1 $((REQUESTS / 30))); do
    create_batch &
    if [ $((i % CONCURRENT)) -eq 0 ]; then
        wait
    fi
done
wait
echo "✅ Завершено"

echo ""
echo "🎉 Нагрузочное тестирование завершено!"