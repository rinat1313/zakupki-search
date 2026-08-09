# Wiring `SEARCH_URL` для zakupki-platform

Раздел «Поисковики» в UI ходит через **gateway**. Gateway читает env `SEARCH_URL`
и проксирует `/api/v1/auth`, `/api/v1/searchers*` в этот сервис.

Без `SEARCH_URL` UI покажет: `search service not configured (SEARCH_URL)`.

## Куда прописывать

Не в образ `zakupki-search`, а в **compose оркестратора** `zakupki-platform`
(файл `docker-compose.search.yml`), чтобы при `./up.sh` переменная уже была
в контейнере gateway.

## Патч

Применить из корня `zakupki-platform`:

```bash
git apply ../zakupki-search/deploy/platform/0001-wire-gateway-search-url.patch
# или вручную вставить фрагмент ниже
```

Суть изменений:

1. `docker-compose.search.yml` — override gateway:
   ```yaml
   gateway:
     environment:
       SEARCH_URL: http://search:8093
     depends_on:
       search:
         condition: service_started
   ```
2. `docker-compose.host.yml` (если `ZAKUPKI_HOST_NET=1`):
   `SEARCH_URL: http://127.0.0.1:8093`
3. Комментарий в `.env.example`

После merge в `zakupki-platform` main достаточно `./up.sh` (с sibling search) —
ручной `export SEARCH_URL=...` не нужен.
