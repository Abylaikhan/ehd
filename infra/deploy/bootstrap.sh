#!/usr/bin/env bash
# Идемпотентный провизионинг + деплой ЕХД на чистый Ubuntu 24.04.
# Запускается на сервере из каталога проекта (/opt/ehd2). Безопасен для повторного запуска.
set -euo pipefail

DOMAIN_WEB="ehd.makeit.kz"
DOMAIN_API="apiehd.makeit.kz"
COMPOSE="docker compose -f docker-compose.prod.yml --env-file .env.prod"

log() { echo -e "\n\033[1;36m>>> $*\033[0m"; }

# --- 1. Swap (сборка Nuxt на 2GB RAM) ---
if ! swapon --show | grep -q '/swapfile'; then
  log "Создаю swap 2G"
  fallocate -l 2G /swapfile || dd if=/dev/zero of=/swapfile bs=1M count=2048
  chmod 600 /swapfile
  mkswap /swapfile
  swapon /swapfile
  grep -q '/swapfile' /etc/fstab || echo '/swapfile none swap sw 0 0' >> /etc/fstab
else
  log "Swap уже есть"
fi

# --- 2. Docker + compose ---
if ! command -v docker >/dev/null 2>&1; then
  log "Устанавливаю Docker"
  export DEBIAN_FRONTEND=noninteractive
  curl -fsSL https://get.docker.com | sh
else
  log "Docker уже установлен: $(docker --version)"
fi
systemctl enable --now docker >/dev/null 2>&1 || true

# --- 3. .env.prod со свежими секретами (генерируется один раз) ---
if [ ! -f .env.prod ]; then
  log "Генерирую .env.prod со свежими секретами"
  PG_PW=$(openssl rand -hex 24)
  CH_ADMIN_PW=$(openssl rand -hex 24)
  ADMIN_PW="$(openssl rand -base64 12 | tr -d '/+=')Aa9!"
  ENC_KEY=$(openssl rand -hex 32)
  HMAC_KEY=$(openssl rand -hex 32)
  cat > .env.prod <<EOF
POSTGRES_DB=ehd
POSTGRES_USER=ehd
POSTGRES_PASSWORD=${PG_PW}
CLICKHOUSE_ADMIN_PASSWORD=${CH_ADMIN_PW}
CLICKHOUSE_SRC_DB=ehd_src
CLICKHOUSE_RO_USER=reporter_ro
CLICKHOUSE_RO_PASSWORD=reporter_ro_local
ADMIN_LOGIN=admin
ADMIN_PASSWORD=${ADMIN_PW}
ADMIN_EMAIL=admin@${DOMAIN_WEB}
AUTH_ENC_KEY=${ENC_KEY}
AUTH_HMAC_KEY=${HMAC_KEY}
SESSION_TTL=3h
TEMP_PASSWORD_TTL=72h
MAX_FAILED_ATTEMPTS=3
EOF
  chmod 600 .env.prod
else
  log ".env.prod уже существует — секреты не трогаю"
fi

# --- 4. nginx: стартовать с bootstrap-конфигом (HTTP), убрать https-конфиг если остался ---
log "Готовлю nginx bootstrap-конфиг"
mkdir -p infra/nginx/certbot-www
# на первом заходе (нет сертификата) — работаем по HTTP через bootstrap.conf
if [ ! -f /etc/letsencrypt/live/${DOMAIN_WEB}/fullchain.pem ]; then
  rm -f infra/nginx/conf.d/ehd-https.conf
  cp infra/nginx/bootstrap.conf infra/nginx/conf.d/bootstrap.conf
fi

# --- 5. Сборка и запуск стека ---
log "Собираю образы (может занять несколько минут)"
$COMPOSE build
log "Поднимаю стек"
$COMPOSE up -d

# --- 6. TLS: выпуск сертификата, если его ещё нет ---
if [ ! -f /etc/letsencrypt/live/${DOMAIN_WEB}/fullchain.pem ]; then
  log "Жду готовности nginx перед выпуском сертификата"
  sleep 8
  ADMIN_EMAIL=$(grep '^ADMIN_EMAIL=' .env.prod | cut -d= -f2-)
  log "Выпускаю сертификат Let's Encrypt для ${DOMAIN_WEB}, ${DOMAIN_API}"
  docker run --rm \
    -v /etc/letsencrypt:/etc/letsencrypt \
    -v "$(pwd)/infra/nginx/certbot-www:/var/www/certbot" \
    certbot/certbot certonly --webroot -w /var/www/certbot \
    --email "${ADMIN_EMAIL}" --agree-tos --no-eff-email --non-interactive \
    -d "${DOMAIN_WEB}" -d "${DOMAIN_API}"
else
  log "Сертификат уже выпущен"
fi

# --- 7. Переключить nginx на HTTPS-конфиг ---
if [ -f /etc/letsencrypt/live/${DOMAIN_WEB}/fullchain.pem ]; then
  log "Активирую HTTPS-конфиг nginx"
  cp infra/nginx/ehd-https.conf infra/nginx/conf.d/ehd-https.conf
  rm -f infra/nginx/conf.d/bootstrap.conf
  $COMPOSE exec -T nginx nginx -t && $COMPOSE exec -T nginx nginx -s reload
fi

# --- 8. Cron авто-обновления сертификата ---
RENEW="$(pwd)/infra/deploy/renew.sh"
if ! crontab -l 2>/dev/null | grep -q "$RENEW"; then
  log "Ставлю cron авто-обновления TLS (еженедельно)"
  ( crontab -l 2>/dev/null; echo "17 3 * * 1 $RENEW >> /var/log/ehd-certbot.log 2>&1" ) | crontab -
fi

log "Готово. Статус:"
$COMPOSE ps
echo
log "Учётные данные администратора (сохраните):"
echo "  URL:   https://${DOMAIN_WEB}"
echo "  login: $(grep '^ADMIN_LOGIN=' .env.prod | cut -d= -f2-)"
echo "  pass:  $(grep '^ADMIN_PASSWORD=' .env.prod | cut -d= -f2-)"
