#!/bin/sh
set -eu

GEN=/etc/nginx/generated
mkdir -p "$GEN"

NETSTORAGE_HOSTNAME="${NETSTORAGE_HOSTNAME:-example-storage.ns.edgekey.net}"
NETSTORAGE_HOST_HEADER="${NETSTORAGE_HOST_HEADER:-$NETSTORAGE_HOSTNAME}"
NETSTORAGE_SCHEME="${NETSTORAGE_SCHEME:-https}"
MODE="${NGINX_DOMAIN_MODE:-local}"

export NETSTORAGE_HOSTNAME NETSTORAGE_HOST_HEADER NETSTORAGE_SCHEME

envsubst '${NETSTORAGE_HOSTNAME}' \
  < /etc/nginx/snippets/netstorage-upstream.conf.template \
  > "$GEN/netstorage-upstream.conf"

envsubst '${NETSTORAGE_HOST_HEADER} ${NETSTORAGE_SCHEME}' \
  < /etc/nginx/snippets/netstorage-location.conf.template \
  > "$GEN/netstorage-location.conf"

case "$MODE" in
  local)
    cp /etc/nginx/conf.d/domains-local.conf "$GEN/domains-active.conf"
    ;;
  netstorage)
    cp /etc/nginx/conf.d/domains.conf "$GEN/domains-active.conf"
    ;;
  *)
    echo "NGINX_DOMAIN_MODE must be 'local' or 'netstorage' (got: $MODE)" >&2
    exit 1
    ;;
esac

exec nginx -g 'daemon off;'
