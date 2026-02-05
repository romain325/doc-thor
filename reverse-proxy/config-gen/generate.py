#!/usr/bin/env python3
"""Polls the server API for project/version metadata and renders one Nginx
server-block config file per project into /etc/nginx/conf.d/.

Files are written only when their content actually changes so the reload
watcher does not fire on every poll cycle.  Configs for projects that have
disappeared are pruned automatically.  The 00-default.conf file (webapp,
written by entrypoint) is never touched.
"""

import hashlib
import logging
import os
import time
from pathlib import Path

import requests
from jinja2 import Environment, FileSystemLoader

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
log = logging.getLogger(__name__)

SERVER_URL    = os.environ["SERVER_URL"].rstrip("/")
NGINX_TOKEN   = os.environ["NGINX_TOKEN"]
BASE_DOMAIN   = os.environ["BASE_DOMAIN"]
STORAGE_URL    = os.environ["STORAGE_URL"].rstrip("/")
STORAGE_BUCKET = os.environ["STORAGE_BUCKET"]
POLL_INTERVAL  = int(os.environ.get("POLL_INTERVAL", "10"))

CONF_DIR      = Path("/etc/nginx/conf.d")
TEMPLATE_DIR  = Path(__file__).resolve().parent / "templates"


def _template():
    env = Environment(loader=FileSystemLoader(str(TEMPLATE_DIR)), keep_trailing_newline=True)
    return env.get_template("server-block.j2")


def _fetch_projects() -> list[dict]:
    """GET /projects  →  [{slug, versions[], latest}, ...]"""
    resp = requests.get(
        f"{SERVER_URL}/projects",
        headers={"Authorization": f"Bearer {NGINX_TOKEN}"},
        timeout=5,
    )
    resp.raise_for_status()
    return resp.json()


def _content_hash(text: str) -> str:
    return hashlib.sha256(text.encode()).hexdigest()[:16]


def sync(template, projects: list[dict]):
    active: set[str] = set()

    for project in projects:
        slug  = project["slug"]
        fname = f"{slug}.conf"
        active.add(fname)

        rendered = template.render(
            project=project,
            base_domain=BASE_DOMAIN,
            storage_url=STORAGE_URL,
            storage_bucket=STORAGE_BUCKET,
        )

        dest = CONF_DIR / fname
        if dest.exists() and dest.read_text() == rendered:
            continue

        dest.write_text(rendered)
        log.info("wrote %s (hash %s)", dest, _content_hash(rendered))

    # Prune configs for projects that no longer exist.
    for f in CONF_DIR.glob("*.conf"):
        if f.name.startswith("00-"):
            continue          # entrypoint-managed
        if f.name not in active:
            f.unlink()
            log.info("removed %s", f)


def main():
    template = _template()
    while True:
        try:
            projects = _fetch_projects()
            sync(template, projects)
        except Exception as exc:  # noqa: BLE001
            log.warning("sync cycle failed: %s", exc)
        time.sleep(POLL_INTERVAL)


if __name__ == "__main__":
    main()
