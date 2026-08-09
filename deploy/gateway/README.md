# Патч для агента zakupki-gateway

**Проблема:** кнопки «Запуск» / «Сохранить и запустить» только показывают URL ЕИС,  
не вызывают `POST /api/v1/searchers/{id}/run` → данные в core не синхронизируются.

**Применить:**

```bash
cd zakupki-gateway
git checkout -b cursor/searchers-run-ui-f74e
git apply ../zakupki-search/deploy/gateway/0001-searchers-run-ui.patch
# commit + push + PR → main
```

**Что меняет:** `ui/searchers.js` — `runSearcher()` дергает search run и обновляет список.

Нужен также патч core (`deploy/core/`) и `SEARCH_URL` / `CORE_URL` в platform.
