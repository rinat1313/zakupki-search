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
export EIS_CA_DIR=certs   # сертификаты Минцифры для zakupki.gov.ru
go run ./cmd/search
```

### TLS / ЕИС (важно с 2026)

`zakupki.gov.ru` перешёл на CA Минцифры. В образе и в `certs/` лежат Root + Sub CA 2022/2024.

| Env | Назначение |
|-----|------------|
| `EIS_CA_DIR` | каталог с PEM (`certs` / `/app/certs`) |
| `EIS_TLS_INSECURE` | по умолчанию `true` — fallback для цепочек Минцифры/ГОСТ |

Без этого `POST …/run` падает с `x509: certificate signed by unknown authority`.

## Swagger / OpenAPI

Для других сервисов контракт лежит в `docs/openapi.yaml`.

| URL | Что |
|-----|-----|
| http://localhost:8091/swagger/ | Swagger UI |
| http://localhost:8091/openapi.yaml | Сырой OpenAPI 3.0 |
| http://localhost:8091/docs | редирект на `/swagger/` |

В UI можно вызвать `Authorize` и вставить Bearer token после login.

## API для gateway UI (`/api/v1/searchers`)

Контракт под `zakupki-gateway` → `SEARCH_URL` (`ui/searchers.js`):

| Method | Path | Описание |
|--------|------|----------|
| POST | `/api/v1/auth/login` | `{login,password}` → `{token,user}` (`user.name`) |
| GET/POST | `/api/v1/searchers` | список / создать (`name`, `config`, `auto_ai`) |
| GET/PUT/DELETE | `/api/v1/searchers/{id}` | CRUD |
| PUT | `/api/v1/searchers/{id}/auto-ai` | `{enabled}` — AI на этот поиск |
| POST | `/api/v1/searchers/{id}/run` | ЕИС → sync в core (`search_config_id`) |
| GET | `/api/v1/searchers/{id}/tenders` | прокси списка из core |

Env:

- `CORE_URL` (например `http://127.0.0.1:8080`) — **обязателен** для run и списка тендеров
- на gateway: `SEARCH_URL` → этот сервис (в Docker-сети platform: `http://search:8093`)

Чтобы `SEARCH_URL` был уже прописан при `./up.sh` платформы, см. [`deploy/platform/`](deploy/platform/).

Сохранение фильтров в UI **не наполняет** БД. Нужен **Запуск** (`POST …/run`) → ЕИС → sync в core.  
Если список пустой — см. [`deploy/AGENT_HANDOFF_TENDERS.md`](deploy/AGENT_HANDOFF_TENDERS.md).

Legacy: `/api/v1/search-profiles` (+ `eis-url`, `…/run`).

Токен: заголовок `Authorization: Bearer <token>` или cookie `session`.

### Интеграция с core / parser

1. Сохранить профиль (фильтры) в search
2. `POST /api/v1/searchers/{id}/run` — обход ЕИС + sync в core (`search_config_id`)
3. Core upsert тендеров и ставит ingest job (сбор карточек / docs)
4. UI читает `GET /api/v1/tenders?search_config_id=…` из core

`zakupki-search` не хранит список закупок — только настройки поиска.

### Ключ связи с zakupki-core

| Где | Имя поля | Смысл |
|-----|----------|--------|
| `zakupki-search` | `id` / `profile_id` | PK конфигурации поиска (стабильный UUID) |
| `zakupki-search` | `config_version` | версия фильтров; +1 при смене `eis_config` |
| `zakupki-core` | `search_profile_id` | ссылка на настройку у тендера |
| `zakupki-core` | `synced_config_version` (рекоменд.) | какая версия уже применена к списку |

### Актуализация списка при смене настроек

1. Пользователь меняет фильтры → у профиля растёт `config_version`.
2. Worker заново обходит ЕИС по новому `eis-url`.
3. В **zakupki-core** уходит **полный snapshot** хитов для `search_profile_id`:
   - upsert новых/обновлённых;
   - удалить (или отвязать) тендеры профиля, которых больше нет в выдаче.
4. Core запоминает `synced_config_version`.

Рекомендуемая уникальность хита в core: `(search_profile_id, source_site, reg_number)`.  
Контракт sync описан в OpenAPI как `CoreProfileSyncRequest` (реализуется в core).

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
