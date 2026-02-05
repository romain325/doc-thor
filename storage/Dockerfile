# =============================================================================
# doc-thor — Storage (Garage)
# =============================================================================
# dxflrs/garage is a single static binary in a scratch image — no shell at all.
# Stage 1 extracts that binary; Stage 2 drops it into Alpine so init.sh can run.
# The binary is fully static (single-layer image, no libc) so Alpine is fine.
#
# Config  : garage.toml is volume-mounted at /etc/garage.toml by docker-compose.
# Data    : persists in /var/lib/garage via the garage-data named volume.
# =============================================================================

# --- stage 1: pull the static garage binary -----------------------------------
FROM dxflrs/garage:v2.1.0 AS garage-bin

# --- stage 2: Alpine shell + garage binary + bootstrap script -----------------
FROM alpine:3.21

# busybox grep lacks -oE; init.sh needs it to parse the node ID from status output.
RUN apk add --no-cache grep

COPY --from=garage-bin /garage  /usr/local/bin/garage
COPY init.sh                    /init.sh
RUN  chmod +x                   /init.sh

ENTRYPOINT ["/init.sh"]
