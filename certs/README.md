# TLS trust for zakupki.gov.ru (ЕИС)

С 2026 ЕИС на сертификатах Минцифры. В пуле нужны RSA Root и **оба**
Sub CA (2022 + 2024). Один старый Sub даёт:

`x509: … (possibly because of "crypto/rsa: verification error" while trying
to verify candidate authority certificate "Russian Trusted Sub CA")`

Файлы из официальных zip gu-st.ru. Дублируются в `internal/eissearch/certs/`
для `go:embed`.

По умолчанию `EIS_TLS_INSECURE=true` (fallback). Строго: `EIS_TLS_INSECURE=false`.
