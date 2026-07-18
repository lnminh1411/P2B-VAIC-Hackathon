#!/bin/sh
set -eu

case "${P2B_PROCESS:-api}" in
  api)
    exec /usr/local/bin/p2b-service
    ;;
  worker)
    exec /usr/local/bin/p2b-worker
    ;;
  crawler)
    exec /usr/local/bin/p2b-crawler
    ;;
  migrate)
    exec /usr/local/bin/p2b-migrate
    ;;
  *)
    echo "Unsupported P2B_PROCESS" >&2
    exit 64
    ;;
esac
