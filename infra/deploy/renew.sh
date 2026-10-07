#!/usr/bin/env bash
# Обновление TLS-сертификата Let's Encrypt + reload nginx. Вызывается из cron.
set -euo pipefail
cd "$(dirname "$0")/../.."   # -> корень проекта (/opt/ehd2)

docker run --rm \
  -v /etc/letsencrypt:/etc/letsencrypt \
  -v "$(pwd)/infra/nginx/certbot-www:/var/www/certbot" \
  certbot/certbot renew --webroot -w /var/www/certbot --quiet

docker compose -f docker-compose.prod.yml --env-file .env.prod exec -T nginx nginx -s reload || true
