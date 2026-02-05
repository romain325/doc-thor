#!/bin/sh
set -e

# ---------------------------------------------------------------------------
# 1. Render the default (catch-all) server block from its template.
#    Only ${BASE_DOMAIN} is substituted; bare nginx variables
#    ($host, $remote_addr …) are intentionally left alone.
# ---------------------------------------------------------------------------
envsubst '${BASE_DOMAIN}' \
    < /etc/nginx/default.conf.tmpl \
    > /etc/nginx/conf.d/00-default.conf

# ---------------------------------------------------------------------------
# 2. Start the reload watcher in the background.  It will begin watching
#    conf.d immediately; if nginx is not yet running the reload signal is a
#    harmless no-op.
# ---------------------------------------------------------------------------
/app/config-gen/reload-watcher.sh &

# ---------------------------------------------------------------------------
# 3. Start the polling config-generator in the background.  It fetches the
#    project list from the server API and writes per-project server blocks
#    into conf.d.  The first cycle runs before nginx is up, so the initial
#    set of configs is already on disk when nginx starts.
# ---------------------------------------------------------------------------
python3 /app/config-gen/generate.py &

# ---------------------------------------------------------------------------
# 4. Exec nginx in the foreground so the container stays alive and PID 1
#    can forward signals (SIGTERM, SIGQUIT) cleanly.
# ---------------------------------------------------------------------------
exec nginx -g 'daemon off;'
