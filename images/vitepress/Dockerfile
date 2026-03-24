# =============================================================================
# romain325/doc-thor-builder-vuepress
# =============================================================================
# Official image for VuePress-based projects.
# Follows the builder container contract:
#   /repo   — source repository (mounted read-only by the builder)
#   /output — build output      (mounted read-write; collected after exit)
# =============================================================================

FROM node:25-alpine

WORKDIR /repo

RUN npm add -D vitepress@next 

ENTRYPOINT ["sh", "-c", "npx vitepress build && cp -r .vitepress/dist/* /output"]
