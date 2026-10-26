# Task Manager API

REST API для управления задачами в командах: роли (owner / admin / member), история изменений задач,
комментарии, кеш списка задач в Redis и SQL-отчет по команде.

**Стек:** Go, MySQL, Redis, Docker Compose, OpenAPI.

## Запуск

```bash
cp .env.example .env    # задайте JWT_SECRET: openssl rand -hex 32
docker compose up -d --build
```

- API: `http://localhost:8080/api/v1`
- Swagger UI: `http://localhost:8080/swagger/`

Миграции (`migrations/`, [goose](https://github.com/pressly/goose)) применяются автоматически при старте.

## Эндпоинты

Все, кроме регистрации и входа, требуют заголовок `Authorization: Bearer <token>`.

| Метод | Путь | Что делает |
|---|---|---|
| POST | `/register` | регистрация |
| POST | `/login` | вход, выдает JWT |
| POST | `/teams` | создать команду (вы — owner) |
| GET | `/teams` | мои команды |
| POST | `/teams/{id}/invite` | добавить пользователя в команду (owner / admin) |
| GET | `/teams/{id}/stats` | отчет по команде (owner / admin) |
| POST | `/tasks` | создать задачу |
| GET | `/tasks?team_id=&status=&assignee_id=&limit=&offset=&cursor=` | задачи команды |
| PUT | `/tasks/{id}` | обновить задачу |
| GET | `/tasks/{id}/history` | история изменений |
| POST | `/tasks/{id}/comments` | добавить комментарий |
| GET | `/tasks/{id}/comments` | комментарии |

## Пример

```bash
API=http://localhost:8080/api/v1
curl -X POST $API/register -d '{"email":"alice@example.com","password":"s3cret-pass","name":"Alice"}'
TOKEN=$(curl -s -X POST $API/login -d '{"email":"alice@example.com","password":"s3cret-pass"}' | jq -r .access_token)

curl -X POST $API/teams -H "Authorization: Bearer $TOKEN" -d '{"name":"Backend"}'
curl -X POST $API/tasks -H "Authorization: Bearer $TOKEN" -d '{"team_id":1,"title":"Настроить CI"}'
curl "$API/tasks?team_id=1&status=todo" -H "Authorization: Bearer $TOKEN"
```

## Права

| | owner | admin | member |
|---|---|---|---|
| Видеть команду, задачи, историю, комментарии; создавать задачи и комментировать | ✓ | ✓ | ✓ |
| Редактировать любую задачу команды | ✓ | ✓ | — |
| Приглашать (admin назначает только owner) | ✓ | ✓ | — |
| Отчет по команде | ✓ | ✓ | — |

Участник может менять свою задачу (title, description, status, assignee), а назначенную ему — только status.

## Конфигурация

Переменные окружения из `.env`, обязательные: `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `REDIS_ADDR`,
`JWT_SECRET` (не короче 32 символов) и `MYSQL_ROOT_PASSWORD` для контейнера MySQL. Остальные (`JWT_TTL`,
`CACHE_TTL`, `RATE_LIMIT_RPS`, `RATE_LIMIT_BURST` и т. д.) имеют значения по умолчанию — см. `.env.example`.

## Тесты

```bash
make test-unit          # права доступа, без БД
make test-integration   # SQL-отчет на MySQL (нужен docker compose up -d mysql)
```
