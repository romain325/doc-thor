# Builder images

Catalog of all available builder images. Each entry lists the image, its base, and
what it ships with.

---

## mkdocs-material

**Image:** `romain325/doc-thor-builder-mkdocs`
**Base:** `python:3.12-alpine`

MkDocs project builder. Includes the Material theme and a set of common plugins out
of the box.

| Dependency | Role |
|------------|------|
| `mkdocs` | Core documentation generator |
| `mkdocs-material` | Material Design theme |
| `mkdocs-awesome-pages-plugin` | Automatic nav from directory layout |
| `mkdocs-mermaid2-plugin` | Mermaid diagram support |
| `mkdocs-windmill` | Windmill theme |
| `pymdown_extensions` | Extended Markdown syntax |
| `Pygments` | Syntax highlighting |
| `Jinja2` | Template engine |
| `MarkupSafe` | Safe string handling (Jinja2 dep) |
| `packaging` | Version/requirement parsing |
