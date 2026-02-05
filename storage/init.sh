#!/bin/sh
# =============================================================================
# doc-thor — Garage first-run bootstrap
# =============================================================================
# Runs exactly once.  A marker file (/var/lib/garage/.doc-thor-initialized)
# gates every subsequent container start so the sequence is never repeated.
#
# What it does:
#   1. Start Garage in the background (needs to be running for CLI commands).
#   2. Wait for the process to become ready.
#   3. Assign layout: single zone (dc1), 1 TB, single replica.
#   4. Create access key "doc-thor-key" — credentials are printed to stdout.
#      They appear only once; retrieve them with `docker compose logs garage`.
#   5. Create bucket "doc-thor-docs" and grant read+write to the new key.
#   6. Write the marker, then re-exec `garage server` in the foreground.
# =============================================================================
set -e

MARKER=/var/lib/garage/.doc-thor-initialized

# ---------------------------------------------------------------------------
# Fast path — already bootstrapped.
# ---------------------------------------------------------------------------
if [ -f "$MARKER" ]; then
    exec garage server
fi

echo "============================================================"
echo " doc-thor storage — first-run bootstrap"
echo "============================================================"

# ---------------------------------------------------------------------------
# Start Garage in the background so we can issue CLI commands against it.
# ---------------------------------------------------------------------------
garage server &
GARAGE_PID=$!

# Wait up to 60 s (30 × 2 s) for Garage to accept status queries.
echo "[init] Waiting for Garage to become ready…"
READY=0
for i in $(seq 1 30); do
    if garage status >/dev/null 2>&1; then
        READY=1
        break
    fi
    sleep 2
done

if [ "$READY" -eq 0 ]; then
    echo "[init] ERROR: Garage did not become ready within 60 s." >&2
    kill "$GARAGE_PID" 2>/dev/null
    exit 1
fi
echo "[init] Garage is ready."

# ---------------------------------------------------------------------------
# Resolve this node's ID from the status table (first long hex token).
# ---------------------------------------------------------------------------
NODE_ID=$(garage status 2>/dev/null | grep -oE '[0-9a-f]{16,}' | head -1)

if [ -z "$NODE_ID" ]; then
    echo "[init] ERROR: could not parse node ID from 'garage status'." >&2
    kill "$GARAGE_PID" 2>/dev/null
    exit 1
fi
echo "[init] Node ID: $NODE_ID"

# ---------------------------------------------------------------------------
# Layout — single zone, single replica, 1 TB.
# ---------------------------------------------------------------------------
echo "[init] Applying layout…"
garage layout assign -z dc1 -c 1T "$NODE_ID"
garage layout apply --version 1
echo "[init] Layout applied."

# ---------------------------------------------------------------------------
# Access key — used by builder (write) and server (list/verify).
# Secret key is shown only at creation time; capture it now.
# ---------------------------------------------------------------------------
echo "[init] Creating access key 'doc-thor-key'…"
garage key create doc-thor-key
echo ""
echo "[init] ^^^ Copy Key ID → STORAGE_ACCESS_KEY and Secret Key → STORAGE_SECRET_KEY in deploy/env/.env"
echo ""

# ---------------------------------------------------------------------------
# Bucket + permissions
# ---------------------------------------------------------------------------
echo "[init] Creating bucket 'doc-thor-docs'…"
garage bucket create doc-thor-docs
garage bucket allow doc-thor-docs --read --write --key doc-thor-key
echo "[init] Bucket 'doc-thor-docs' ready (read + write for doc-thor-key)."

# ---------------------------------------------------------------------------
# Mark as bootstrapped — future starts skip everything above.
# ---------------------------------------------------------------------------
touch "$MARKER"

echo "============================================================"
echo " Bootstrap complete."
echo "============================================================"

# Hand over to Garage in the foreground.
wait "$GARAGE_PID"
