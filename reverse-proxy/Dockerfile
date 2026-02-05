FROM nginx:1.27-alpine

# ---------------------------------------------------------------------------
# Runtime dependencies
#   python3 + pip   →  config-gen (generate.py)
#   jinja2          →  template rendering
#   requests        →  HTTP calls to the server API
#   gettext         →  envsubst for the default server-block template
#   inotify-tools   →  inotifywait for the reload watcher
# ---------------------------------------------------------------------------
RUN apk add --no-cache \
        python3 \
        py3-pip \
        gettext \
        inotify-tools \
        py3-jinja2 \
        py3-requests \
        # && pip3 install --no-cache-dir jinja2 requests \
        && rm -rf /var/cache/apk/*

# ---------------------------------------------------------------------------
# Nginx base configuration (static, no env-var interpolation needed).
# The default.conf.tmpl is processed by envsubst at container start.
# ---------------------------------------------------------------------------
COPY nginx/nginx.conf.base      /etc/nginx/nginx.conf
COPY nginx/default.conf.tmpl    /etc/nginx/default.conf.tmpl
RUN  mkdir -p /etc/nginx/conf.d

# ---------------------------------------------------------------------------
# Config-generator: polling script + Jinja2 template + entrypoint.
# ---------------------------------------------------------------------------
COPY config-gen/                /app/config-gen/
RUN  chmod +x /app/config-gen/entrypoint.sh \
        /app/config-gen/reload-watcher.sh

# Logs are tailed by the OTel Collector via a shared volume.
VOLUME /var/log/nginx

EXPOSE 80 443

ENTRYPOINT ["/app/config-gen/entrypoint.sh"]
