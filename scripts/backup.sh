#!/bin/sh
# Runs on the server from cron: dumps PostgreSQL into /opt/vuc/backups and
# keeps the newest 14 dumps.
set -eu
umask 077
dir=/opt/vuc/backups
mkdir -p "$dir"
cd /opt/vuc/project
file="$dir/app_db-$(date +%Y-%m-%d_%H-%M).sql.gz"
docker compose -f docker-compose.prod.yaml --env-file ../server.env exec -T postgres \
	pg_dump -U app -d app_db --clean --if-exists | gzip > "$file.part"
mv "$file.part" "$file"
ls -1t "$dir"/app_db-*.sql.gz | tail -n +15 | xargs -r rm --
