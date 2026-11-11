# Сервер: развёртывание и обслуживание

Адрес: https://37-46-129-246.sslip.io (VDS в Москве, Ubuntu 24.04). Автоматического деплоя (CI/CD) нет —
обновление запускается руками скриптом с машины разработчика.

## Устройство

- `docker-compose.prod.yaml`: Caddy принимает 80 и 443 и сам получает сертификат Let's Encrypt для адреса из
  `VUC_DOMAIN`; приложение и PostgreSQL наружу не открыты. Приложение запущено с `VUC_TRUST_PROXY=1` —
  cookie получают `Secure` за прокси.
- Файрвол пропускает только 22, 80, 443. Вход на сервер — только по SSH-ключу, у каждого разработчика свой.

| Что | Где на сервере |
| --- | --- |
| Код | `/opt/vuc/project` — копия ветки на момент деплоя, коммит в `REVISION`; прежняя — `/opt/vuc/project.old` |
| Настройки | `/opt/vuc/server.env`: `VUC_DOMAIN`, `DB_PASSWORD`, `VUC_ADMIN_USER`, `VUC_ADMIN_PASSWORD`. Не в git |
| Ключ GigaChat и сертификат Минцифры | `/opt/vuc/.secrets/llm.env`, `russian_trusted_root_ca_pem.crt` |
| Копии базы | `/opt/vuc/backups/`, cron `/etc/cron.d/vuc-backup` в 03:30 по Москве, хранятся 14 (`scripts/backup.sh`) |

Дальше `ssh vuc` — это `ssh -i <ваш ключ> root@37.46.129.246`, а `c` — команда compose на сервере:

```bash
cd /opt/vuc/project && c="docker compose -f docker-compose.prod.yaml --env-file ../server.env"
```

## Доступ новому разработчику

Разработчик присылает публичный ключ (`~/.ssh/id_ed25519.pub`), тот, у кого уже есть доступ, добавляет его:

```bash
ssh vuc 'cat >> /root/.ssh/authorized_keys' < его_ключ.pub
```

## Обновить сервер

Из корня репозитория, после того как нужное влито в `main`:

```bash
git fetch && VUC_HOST=root@37.46.129.246 VUC_SSH_KEY=~/.ssh/<ваш ключ> ./scripts/deploy.sh origin/main
```

Скрипт берёт ветку через `git archive`, по SSH распаковывает её в `/opt/vuc/project`, на сервере
`docker compose` собирает образ и перезапускает контейнеры; миграции применяются сами. `server.env` и
секреты не трогаются. Можно выкатить и другую ветку или коммит. При запуске программа сама докладывает в
GigaChat недостающие выводы — не деплоить во время прогона на том же ключе (будет 429).

## Преподаватели

Преподаватель регистрируется по одноразовому коду, код привязан к почте и действует 24 часа:

```bash
$c exec app vuc -create-vuc "Название ВУЦ"                 # один раз
$c exec app vuc -invite-teacher почта@example.ru -vuc "Название ВУЦ"
```

Пароль администратора (для приглашений через API): `grep ADMIN_PASSWORD /opt/vuc/server.env`.

## Журнал, копии, восстановление

```bash
$c ps && $c logs --tail 50 app
/opt/vuc/project/scripts/backup.sh                         # копия прямо сейчас
gunzip -c /opt/vuc/backups/<файл>.sql.gz | $c exec -T postgres psql -U app -d app_db
```

Демо-данные (`$c --profile tools run --rm seed`) заливаются только в пустую базу.
