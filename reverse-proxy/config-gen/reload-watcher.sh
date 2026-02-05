#!/bin/sh
# Watches the conf.d directory for any file-system event and sends an nginx
# reload signal.  A 1-second sleep after each event gives generate.py (or
# server) time to finish writing before the reload fires.

WATCH_DIR="/etc/nginx/conf.d"

while true; do
    inotifywait -t 60 -e modify,create,delete "$WATCH_DIR" 2>/dev/null
    sleep 1
    nginx -s reload 2>/dev/null || true
done
