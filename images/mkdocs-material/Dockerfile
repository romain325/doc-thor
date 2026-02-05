# =============================================================================
# doc-thor/builder-mkdocs
# =============================================================================
# Official image for MkDocs-based projects.
# Follows the builder container contract:
#   /repo   — source repository (mounted read-only by the builder)
#   /output — build output      (mounted read-write; collected after exit)
#
# Usage:
#   docker build -t doc-thor/builder-mkdocs .
#
# To add plugins, extend this image:
#   FROM doc-thor/builder-mkdocs
#   RUN pip install --no-cache-dir mkdocs-git-revision-date-plugin
# =============================================================================

FROM python:3.12-alpine

COPY ./requirements.txt /tmp/requirements.txt

RUN pip install --no-cache-dir -r /tmp/requirements.txt

WORKDIR /repo

ENTRYPOINT ["mkdocs", "build", "--site-dir", "/output"]
