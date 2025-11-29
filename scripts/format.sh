#!/bin/bash

# Цвета для вывода
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Начинаем форматирование проекта...${NC}"

# Проверяем наличие goimports
if ! command -v goimports &> /dev/null
then
    echo -e "${YELLOW}goimports не найден. Устанавливаем...${NC}"
    go install golang.org/x/tools/cmd/goimports@latest
    echo -e "${GREEN}goimports успешно установлен${NC}"
fi

# Форматируем все .go файлы с помощью goimports
echo -e "${YELLOW}Запускаем goimports...${NC}"
goimports -w -local github.com/Popolzen/shortener .

# Дополнительно запускаем gofmt с упрощением кода
echo -e "${YELLOW}Запускаем gofmt для дополнительной оптимизации...${NC}"
gofmt -w -s .

# Проверяем результат
echo -e "${GREEN}✓ Форматирование завершено!${NC}"

# Показываем файлы, которые были изменены
echo -e "${YELLOW}Проверяем изменения с помощью git...${NC}"
git diff --name-only | grep "\.go$" || echo -e "${GREEN}Нет изменений в .go файлах${NC}"

echo -e "${GREEN}Готово! Все файлы отформатированы.${NC}"