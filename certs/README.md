# TLS trust for zakupki.gov.ru (ЕИС)

С 4 июля 2026 ЕИС использует сертификаты **Минцифры** (*Russian Trusted Root CA* /
*Russian Trusted Sub CA*). Их нет в стандартном Debian/Mozilla trust store —
без них Go даёт:

`tls: failed to verify certificate: x509: certificate signed by unknown authority`

Файлы здесь скачаны с https://gu-st.ru/content/Other/doc/ (официальная выдача).

Dockerfile ставит их в `/usr/local/share/ca-certificates/` и копирует в `/app/certs`.
Клиент ЕИС дополнительно читает `EIS_CA_DIR` (по умолчанию `certs` / `/app/certs`).

Крайний случай: `EIS_TLS_INSECURE=true` (InsecureSkipVerify) — только для отладки.
