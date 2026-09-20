# Как устроен Conversation Summarizer

Приложенный файл Задание.md задаёт Go-микросервис с PostgreSQL, Redis, OpenAI-compatible LLM, Docker Compose, Swagger, slog, миграциями, retry, ограничением истории, метриками, unit-тестами и Gitea CI/CD.

## Полный путь запроса

1. HTTP handler принимает и валидирует JSON.
2. Service проверяет Redis, если это не regenerate.
3. История ограничивается по MAX_MESSAGES и MAX_CONTENT_LENGTH.
4. LLM client вызывает `/chat/completions`, делает retry для сети, timeout, 429 и 5xx.
5. Ответ модели декодируется и проверяется по status, sentiment, priority.
6. Repository сохраняет новую версию в PostgreSQL.
7. Результат кладётся в Redis с TTL и возвращается клиенту.

## Cache и база

Обычный GET: Redis -> response. При miss: Redis -> PostgreSQL -> Redis -> response. Redis ускоряет чтение, но PostgreSQL остаётся источником истины.

Каждый create и regenerate создаёт новую строку. Старые версии сохраняются. Исходные приватные messages не пишутся в базу, поэтому regenerate принимает актуальную историю в body.

## Алгоритмы

- Последние N сообщений выбираются срезом; проверка входа O(n), копирование O(N).
- Текст ограничивается с конца, чтобы сохранить свежий контекст; UTF-8 проверяется после обрезки.
- Retry использует задержки 1, 2 и 4 секунды и ограниченное число попыток.
- Redis даёт ожидаемое O(1) чтение key/value.
- Индекс PostgreSQL `(conversation_id, created_at DESC, id DESC)` ускоряет последнюю версию и историю.

## Gitea CI/CD

`.gitea/workflows/ci-cd.yml` выполняет test, Docker image build и ручной deploy через `workflow_dispatch` на self-hosted runner. До deploy нужны Go, Docker, runner, DEPLOY_PATH и секреты. Адрес Gitea нельзя выдумать: remote добавляется после получения URL организации.

## Следующие шаги

1. Запустить `go test ./...` и `docker compose up --build` на машине с Go и Docker.
2. Добавить cursor pagination и Idempotency-Key.
3. Добавить Kafka event `conversation.closed` с защитой от повторной обработки.
4. Использовать готовый Next.js UI и NestJS/Prisma слой из apps/web и apps/api; затем добавить Kafka, если нужен асинхронный сценарий.
