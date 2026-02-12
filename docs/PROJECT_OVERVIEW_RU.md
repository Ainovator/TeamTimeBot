# TeamTimeBot: что здесь и как стартовать

## Важный факт про текущий репозиторий
Сейчас корень проекта - это библиотека `telebot` (`module gopkg.in/telebot.v4`), а не готовое бот-приложение.
Это полезный каркас для Telegram API, но бизнес-логику твоего бота лучше держать отдельно в подпапках.

## Что где находится сейчас
- `bot.go`, `message.go`, `poll.go`, `webhook.go`, `context.go` и похожие файлы: ядро библиотеки Telegram-бота.
- `middleware/`: готовые middleware (логирование, ограничения и т.д.).
- `layout/`: конфиг/парсинг layout для UI-части библиотеки.
- `*_test.go`: тесты самой библиотеки.
- `README.md`: документация по использованию Telebot.

## Что добавлено для старта TeamTimeBot
- `docker-compose.yml`: только Postgres.
- `.env.example`: переменные окружения для compose.
- `db/migrations/00001_init.sql`: `goose`-миграция со схемой под твой кейс:
  - `telegram_groups` (настройки каждой группы),
  - `poll_templates` (шаблоны опросов по группам),
  - `poll_schedules` (расписание публикации опросов),
  - `training_slots` (расписание тренировок).
- `cmd/migrate/main.go`: запуск миграций (`up/down/status`) через `goose`.
- `cmd/teamtimebot/main.go`: точка входа бота.
- `internal/storage/postgres`: `gorm`-модели и CRUD для групп/шаблонов/расписания.
- `internal/telegram/handlers.go`: команды `/setgroup`, `/setpoll`, `/setschedule`.

## Рекомендуемая структура для твоего кода
Чтобы не смешивать библиотеку и приложение:
- `cmd/teamtimebot/main.go`: точка входа приложения.
- `internal/config`: загрузка env/config.
- `internal/storage/postgres`: запросы к БД.
- `internal/service/scheduler`: запуск задач по cron/таймеру (следующий шаг).
- `internal/service/polls`: формирование и отправка опросов (следующий шаг).
- `internal/telegram`: команды/обработчики Telegram.
- `db/migrations`: дальнейшие миграции схемы.

## Как поднять БД
1. `cp .env.example .env`
2. `docker compose up -d`
3. БД: `localhost:${POSTGRES_PORT}` (по умолчанию `5432`)

## Как применить миграции и запустить бота
1. `go run ./cmd/migrate -- up`
2. `go run ./cmd/teamtimebot`
