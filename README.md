# zakupki-search

Микросервис поисковых настроек для ЕИС (`zakupki.gov.ru`).

Пользователь логинится и управляет **своим списком поисковых профилей**.  
Каждый профиль имеет название и JSON-конфиг фильтров ЕИС, из которого потом можно собрать URL поиска и отдать найденные закупки в `zakupki-parser`.

## Стек

- Go HTTP API (`:8091`)
- PostgreSQL — пользователи, сессии, `search_profiles.eis_config` (JSONB)

## Быстрый старт

```bash
docker compose up -d --build
curl -s http://localhost:8091/health
```

Демо-пользователь (создаётся миграцией):

- login: `demo`
- password: `demo`

Локально без Docker (нужен Postgres):

```bash
export DATABASE_URL=postgres://zakupki:zakupki@localhost:5432/zakupki_search?sslmode=disable
go run ./cmd/search
```

## Swagger / OpenAPI

Для других сервисов контракт лежит в `docs/openapi.yaml`.

| URL | Что |
|-----|-----|
| http://localhost:8091/swagger/ | Swagger UI |
| http://localhost:8091/openapi.yaml | Сырой OpenAPI 3.0 |
| http://localhost:8091/docs | редирект на `/swagger/` |

В UI можно вызвать `Authorize` и вставить Bearer token после login.

## API

| Method | Path | Auth | Описание |
|--------|------|------|----------|
| GET | `/health` | — | health |
| POST | `/api/v1/auth/login` | — | `{login,password}` → `{token,user}` |
| POST | `/api/v1/auth/logout` | Bearer | завершить сессию |
| GET | `/api/v1/auth/me` | Bearer | текущий пользователь |
| GET | `/api/v1/search-profiles` | Bearer | список настроек |
| POST | `/api/v1/search-profiles` | Bearer | создать настройку |
| GET | `/api/v1/search-profiles/{id}` | Bearer | получить |
| PUT | `/api/v1/search-profiles/{id}` | Bearer | обновить |
| DELETE | `/api/v1/search-profiles/{id}` | Bearer | удалить |
| GET | `/api/v1/search-profiles/{id}/eis-url` | Bearer | URL/query для ЕИС |

Токен: заголовок `Authorization: Bearer <token>` или cookie `session`.

### Интеграция с core / parser

1. Взять профиль: `GET /api/v1/search-profiles/{id}` или готовый URL: `.../eis-url`
2. Воркер обходит ленту ЕИС (ещё не в этом сервисе)
3. Найденные закупки сохраняются в **zakupki-core** (каталог тендеров)
4. `zakupki-parser` обогащает карточки по `reg_number` из core

`zakupki-search` не хранит список закупок — только настройки поиска.

### Пример

```bash
TOKEN=$(curl -s -X POST http://localhost:8091/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"login":"demo","password":"demo"}' | jq -r .token)

curl -s -X POST http://localhost:8091/api/v1/search-profiles \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "ПО и разработка",
    "description": "Мониторинг 44/223, подача заявок",
    "eis_config": {
      "search_string": "Разработка ПО",
      "morphology": true,
      "fz44": true,
      "fz223": true,
      "stage_af": true,
      "stage_ca": true,
      "stage_pc": false,
      "stage_pa": false,
      "sort_by": "UPDATE_DATE",
      "records_per_page": "_10"
    }
  }'

curl -s http://localhost:8091/api/v1/search-profiles \
  -H "Authorization: Bearer $TOKEN"
```

## Модель `eis_config`

Поля соответствуют query-параметрам  
`/epz/order/extendedsearch/results.html` (см. разбор `Закупки.html` в ZakupkiParser):

- `search_string`, `morphology`, `strict_equal`
- `fz44`, `fz223`, `pp_rf_615`
- этапы: `stage_af`, `stage_ca`, `stage_pc`, `stage_pa`
- `sort_by`, `sort_direction`, `records_per_page`
- цены/даты, `regions`, `okpd2`, `customer_title`

Сервис хранит и отдаёт поисковый конфиг; каталог закупок — в `zakupki-core`.

## Схема БД

- `users` — логин/пароль (bcrypt)
- `sessions` — opaque token (sha256 в БД)
- `search_profiles` — `(user_id, name)` + `eis_config JSONB`
