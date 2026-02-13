# CI/CD для TeamTimeBot (GitHub Actions)

## Что добавлено
- `.github/workflows/ci.yml`:
  - `go mod verify`
  - `go vet ./...`
  - `go test ./...`
  - `go build ./cmd/teamtimebot ./cmd/migrate`
- `.github/workflows/cd.yml`:
  - деплой по SSH при push в `main` или вручную (`workflow_dispatch`)
  - на сервере выполняет `git pull` и `docker compose -f docker-compose.prod.yml up -d --build`
- `Dockerfile` для бота
- `docker-compose.prod.yml` для прода (`db` + `bot`)

## Секреты в GitHub
В репозитории открой `Settings -> Secrets and variables -> Actions` и добавь:
- `SSH_HOST` - IP сервера
- `SSH_USER` - пользователь SSH (например, `ubuntu`)
- `SSH_KEY` - приватный ключ (содержимое файла `~/.ssh/id_ed25519`)
- `SSH_PORT` - обычно `22` (опционально)
- `SERVER_APP_DIR` - путь к проекту на сервере, например `/home/ubuntu/teamtimebot` (опционально)

## Первичная подготовка сервера
1. Установи Docker и Docker Compose plugin.
2. Склонируй репозиторий в `SERVER_APP_DIR`.
3. Создай `.env` рядом с `docker-compose.prod.yml` (можно взять за основу `.env.prod.example`):
   - `BOT_TOKEN=...`
   - `POSTGRES_HOST=db`
   - `POSTGRES_DB=teamtimebot`
   - `POSTGRES_USER=teamtimebot`
   - `POSTGRES_PASSWORD=...`
   - `POSTGRES_PORT=5432`
   - `POSTGRES_SSLMODE=disable`
   - `TZ=UTC`
4. Выполни первый старт вручную:
   - `docker compose -f docker-compose.prod.yml build bot`
   - `docker compose -f docker-compose.prod.yml up -d db`
   - `docker compose -f docker-compose.prod.yml run --rm --entrypoint /migrate bot -dir /migrations up`
   - `docker compose -f docker-compose.prod.yml up -d bot`
5. Примени миграции:
   - команда уже встроена в последовательность выше и в CD workflow.

## Как работает деплой
Каждый push в `main`:
1. GitHub Actions подключается по SSH.
2. Делает `git pull`.
3. Поднимает `db`, применяет миграции, поднимает `bot`.
