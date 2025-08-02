# Архиватор файлов по ссылкам

REST API сервис для создания архивов из файлов по ссылкам. Реализован на Go с использованием Gin framework.

## Архитектура

### Структура проекта
```
/cmd/server/main.go         # Точка входа
/internal/api/              # HTTP API (Gin handlers)
/internal/service/          # Бизнес-логика (архивы, скачивание)
/internal/model/            # Модели данных
/internal/taskmgr/          # Менеджер задач
/internal/config/           # Конфиг
/config.yaml                # Конфигурация
/archives/                  # Временные архивы (не в git)
```

### Используемые паттерны
- **Dependency Injection**: TaskManager внедряется в API handlers
- **Clean Architecture**: разделение слоёв (API, бизнес-логика, модели)
- **Worker Pool**: ограничение на 3 одновременные архивирования
- **Repository Pattern**: TaskManager управляет состоянием задач
- **Middleware**: логирование запросов (встроено в Gin)

## Запуск

```bash
go mod tidy
go run ./cmd/server/main.go
```

Сервер запустится на порту 8080 (настраивается в config.yaml).

## API

### Создание задачи
```bash
curl -X POST http://localhost:8080/tasks
```

**Ответ:**
```json
{
  "id": "c0e1b2d3-...",
  "files": [],
  "created_at": "2024-01-01T12:00:00Z",
  "status": "pending"
}
```

### Добавление файла
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/file.pdf"}' \
  http://localhost:8080/tasks/{task_id}/files
```

### Получение статуса
```bash
curl http://localhost:8080/tasks/{task_id}/status
```

**Ответ после архивации:**
```json
{
  "id": "c0e1b2d3-...",
  "files": [
    {
      "url": "https://example.com/file.pdf",
      "success": true
    }
  ],
  "created_at": "2024-01-01T12:00:00Z",
  "status": "done",
  "archive_url": "/archives/c0e1b2d3-....zip"
}
```

### Скачивание архива
```bash
curl -O http://localhost:8080/archives/{task_id}.zip
```

## Ограничения

- **Не более 3 файлов в задаче**
- **Не более 3 одновременных задач**
- **Разрешённые типы:** .pdf, .jpeg, .jpg
- **Максимальный размер файла:** 20 МБ
- **Таймаут скачивания:** 30 секунд

## Обработка ошибок

### Ошибки валидации
```json
{
  "error": "file extension not allowed: .exe"
}
```

### Ошибки скачивания
В статусе задачи для каждого файла указывается:
```json
{
  "url": "https://example.com/file.pdf",
  "success": false,
  "reason": "https://example.com/file.pdf (HTTP 404)"
}
```

### Ограничения
```json
{
  "error": "too many active tasks"
}
```

## Конфигурация

Файл `config.yaml`:
```yaml
server:
  port: 8080
files:
  allowed_extensions:
    - .pdf
    - .jpeg
```

## Особенности реализации

### Асинхронная архивация
- После добавления 3-го файла автоматически запускается архивация
- Статус задачи обновляется: `pending` → `in_progress` → `done`
- Ошибки скачивания не прерывают архивацию остальных файлов

### Безопасность
- Валидация URL (только http/https)
- Проверка Content-Type при скачивании
- Ограничение размера файлов
- Фильтрация расширений

### Производительность
- Семафор для ограничения одновременных архивирований
- Автоматическая очистка старых архивов (раз в час)
- Логирование всех запросов

## Тестирование

### Пример полного цикла
```bash
# 1. Создать задачу
TASK_ID=$(curl -s -X POST http://localhost:8080/tasks | jq -r '.id')

# 2. Добавить файлы
curl -X POST -H "Content-Type: application/json" \
  -d '{"url":"https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf"}' \
  http://localhost:8080/tasks/$TASK_ID/files

# 3. Проверить статус
curl http://localhost:8080/tasks/$TASK_ID/status

# 4. Скачать архив (когда готов)
curl -O http://localhost:8080/archives/$TASK_ID.zip
```

## Зависимости

- **Gin**: веб-фреймворк
- **UUID**: генерация уникальных идентификаторов
- **YAML**: конфигурация

Все зависимости минимальны и стандартны для Go-экосистемы. 