# Патч для агента zakupki-core

**Проблема:** UI «Поисковики» пустой — search при `POST …/run` зовёт API, которого в core нет.

**Применить:**

```bash
cd zakupki-core
git checkout -b cursor/search-config-sync-f74e
git apply ../zakupki-search/deploy/core/0001-search-config-sync.patch
go build ./...
# commit + push + PR → main
```

**Что добавляет патч:**

1. Колонки `categories.search_config_id`, `synced_config_version` (+ unique index)
2. `GET /api/v1/categories/by-search-config/{id}`
3. `POST /api/v1/categories/by-search-config/{id}/sync`  
   body: `{ title, config_version, items[{reg_number,source_site,notice_url,law,object_name}], enqueue }`  
   → upsert tenders, link category, optional ingest job
4. `POST /api/v1/categories` принимает `search_config_id`
5. `GET /api/v1/tenders?search_config_id=` фильтрует список

Без этого патча sync из search всегда 404, тендеры в БД не появятся.
