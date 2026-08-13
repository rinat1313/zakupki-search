# Почему в UI нет тендеров + что чинить

## Короткий диагноз

Сохранение поисковика в UI **только пишет фильтры** в Postgres `zakupki-search`.  
Тендеры живут в **`zakupki-core`**. Сейчас цепочка обрывана в двух местах:

1. **`zakupki-gateway` UI** — кнопки «Запуск» / «Сохранить и запустить» **не вызывают** `POST /api/v1/searchers/{id}/run`. Они только показывают URL ЕИС.
2. **`zakupki-core`** — нет API, которое search вызывает при run:
   - `GET/POST …/categories/by-search-config/{id}`
   - `POST …/categories/by-search-config/{id}/sync`
   - фильтр `GET /api/v1/tenders?search_config_id=`

Поэтому в СУБД core ничего не появляется, список пустой.

## Ожидаемый поток

```
UI save searcher     → search DB (фильтры)
UI Run               → search POST /searchers/{id}/run
                     → EIS scrape (HTML)
                     → core POST …/by-search-config/{id}/sync  (upsert + ingest enqueue)
UI list tenders      → core GET /tenders?search_config_id={id}
```

## Патчи в этом репо

| Файл | Для репо |
|------|----------|
| `deploy/core/0001-search-config-sync.patch` | **zakupki-core** |
| `deploy/gateway/0001-searchers-run-ui.patch` | **zakupki-gateway** |

Применить:

```bash
cd ../zakupki-core
git apply ../zakupki-search/deploy/core/0001-search-config-sync.patch

cd ../zakupki-gateway
git apply ../zakupki-search/deploy/gateway/0001-searchers-run-ui.patch
```

Также нужен рабочий `CORE_URL` у search (`http://core:8080` в platform compose) и `SEARCH_URL` у gateway.

## Проверка после merge

1. `./up.sh` (platform) — подняты core + search + gateway  
2. UI → Поисковики → demo/demo → создать поиск с широкими фильтрами  
3. **Запустить** → alert с `found_count > 0` и `status=done`  
4. В списке появляются карточки; в core БД есть строки в `tenders` + `tender_categories`  
5. Ingest worker дособирает карточки (docs/progress)

## Что уже сделано в zakupki-search

- `run` шлёт в core `title`, `config_version`, `items`, `enqueue`
- без `CORE_URL` run/tenders возвращают явную ошибку, а не пустой успех
- legacy `POST /api/v1/search-profiles/{id}/run` → тот же handler
